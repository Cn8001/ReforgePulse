// Package journal is the change log. A change is written before ReforgePulse
// touches the system and finished with its outcome after, so a crash halfway
// still leaves a trace.
package journal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/store"
)

type Journal struct {
	db  *sql.DB
	now func() time.Time
}

// New keeps the log in db's changes table.
func New(db *sql.DB) *Journal { return &Journal{db: db, now: time.Now} }

// Begin records a change about to be applied and returns its ID.
func (j *Journal) Begin(ctx context.Context, c domain.Change) (string, error) {
	c.At = j.now().UTC()
	c.ID = shortID(c)
	_, err := j.db.ExecContext(ctx, `INSERT INTO changes
		(id, at, action, summary, target, before, after, reversible, reverts, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.At.Format(store.Stamp), c.Action, c.Summary, c.Target, c.Before, c.After, c.Reversible, c.Reverts, domain.Started)
	if err != nil {
		return "", fmt.Errorf("record change: %w", err)
	}
	return c.ID, nil
}

// Finish records how the change went: applied when err is nil, else failed.
// after says what the change actually did, such as the bytes it freed.
func (j *Journal) Finish(ctx context.Context, id, after string, err error) error {
	status, msg := domain.Applied, ""
	if err != nil {
		status, msg = domain.Failed, err.Error()
	}
	_, dbErr := j.db.ExecContext(ctx, `UPDATE changes
		SET status = ?, error = ?, after = CASE WHEN ? = '' THEN after ELSE ? END
		WHERE id = ?`, status, msg, after, after, id)
	if dbErr != nil {
		return fmt.Errorf("finish change %s: %w", id, dbErr)
	}
	return nil
}

// Log returns every change, newest first. A change an applied revert undid
// shows as reverted.
func (j *Journal) Log(ctx context.Context) ([]domain.Change, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT c.id, c.at, c.action, c.summary, c.target, c.before, c.after,
		c.reversible, c.reverts, c.status, c.error,
		EXISTS (SELECT 1 FROM changes r WHERE r.reverts = c.id AND r.status = 'applied')
		FROM changes c ORDER BY c.seq DESC`)
	if err != nil {
		return nil, fmt.Errorf("read change log: %w", err)
	}
	defer rows.Close()
	out := []domain.Change{} // an empty list, not null, for the UI and agents
	for rows.Next() {
		var c domain.Change
		var at string
		var undone bool
		if err := rows.Scan(&c.ID, &at, &c.Action, &c.Summary, &c.Target, &c.Before, &c.After,
			&c.Reversible, &c.Reverts, &c.Status, &c.Error, &undone); err != nil {
			return nil, fmt.Errorf("read change: %w", err)
		}
		c.At, _ = time.Parse(store.Stamp, at)
		if undone && c.Status == domain.Applied {
			c.Status = domain.Reverted
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read change log: %w", err)
	}
	return out, nil
}

// ErrRevertAlone is refused on purpose: without the entry that undid it, the
// original change would look applied again and could be reverted twice.
var ErrRevertAlone = errors.New("a revert goes with the change it undid; delete that change instead")

// Delete removes these changes, and with each the entries that revert it.
// It returns how many changes went.
func (j *Journal) Delete(ctx context.Context, ids ...string) (int, error) {
	return j.remove(ctx, func(tx *sql.Tx) error {
		for _, id := range ids {
			var reverts string
			var origKept bool
			err := tx.QueryRowContext(ctx, `SELECT reverts,
				EXISTS (SELECT 1 FROM changes o WHERE o.id = c.reverts)
				FROM changes c WHERE id = ?`, id).Scan(&reverts, &origKept)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("change %s not found", id)
			}
			if err != nil {
				return fmt.Errorf("read change %s: %w", id, err)
			}
			if reverts != "" && origKept && !slices.Contains(ids, reverts) {
				return fmt.Errorf("delete %s: %w", id, ErrRevertAlone)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO drop_ids VALUES (?)`, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBefore removes changes begun before t, with what reverts them.
func (j *Journal) DeleteBefore(ctx context.Context, t time.Time) (int, error) {
	return j.remove(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO drop_ids SELECT id FROM changes WHERE at < ?`, t.UTC().Format(store.Stamp))
		return err
	})
}

// Prune keeps the newest keep changes, so the log cannot grow without end.
func (j *Journal) Prune(ctx context.Context, keep int) (int, error) {
	return j.remove(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO drop_ids
			SELECT id FROM changes ORDER BY seq DESC LIMIT -1 OFFSET ?`, keep)
		return err
	})
}

// remove deletes what pick puts in drop_ids, plus every entry that reverts
// one of them, in one transaction.
func (j *Journal) remove(ctx context.Context, pick func(*sql.Tx) error) (int, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin delete: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS drop_ids (id TEXT PRIMARY KEY ON CONFLICT IGNORE);
		DELETE FROM drop_ids`); err != nil {
		return 0, fmt.Errorf("prepare delete: %w", err)
	}
	if err := pick(tx); err != nil {
		return 0, err
	}
	// Reverts follow their originals out, and a revert can itself be undone.
	for {
		res, err := tx.ExecContext(ctx, `INSERT INTO drop_ids
			SELECT id FROM changes WHERE reverts IN (SELECT id FROM drop_ids) AND id NOT IN (SELECT id FROM drop_ids)`)
		if err != nil {
			return 0, fmt.Errorf("follow reverts: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			break
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM changes WHERE id IN (SELECT id FROM drop_ids)`)
	if err != nil {
		return 0, fmt.Errorf("delete changes: %w", err)
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit delete: %w", err)
	}
	return int(n), nil
}

// shortID hashes the change's content, like a short git hash.
func shortID(c domain.Change) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%d|%s|%s|%s|%s|%s", c.At.UnixNano(), c.Summary, c.Target, c.Before, c.After, c.Reverts))
	return hex.EncodeToString(h[:])[:7]
}
