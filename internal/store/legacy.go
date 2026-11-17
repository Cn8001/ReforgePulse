package store

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// legacyChange is a line of the old journal.jsonl: the first line of an ID
// is the change, later lines only update its outcome.
type legacyChange struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Action     string    `json:"action"`
	Summary    string    `json:"summary"`
	Target     string    `json:"target"`
	Before     string    `json:"before"`
	After      string    `json:"after"`
	Reversible bool      `json:"reversible"`
	Reverts    string    `json:"reverts"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
}

type legacyResolution struct {
	Key   string    `json:"key"`
	Title string    `json:"title"`
	At    time.Time `json:"at"`
}

// ImportFiles moves what earlier versions kept in files under dir into db,
// once, in one transaction, then removes the files. Terms acceptance is not
// carried over: the terms are shown again.
func ImportFiles(ctx context.Context, db *sql.DB, dir string) error {
	journalPath := filepath.Join(dir, "journal.jsonl")
	reportsDir := filepath.Join(dir, "reports")
	resolvedPath := filepath.Join(dir, "resolved.json")
	if !exists(journalPath) && !exists(reportsDir) && !exists(resolvedPath) {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback()
	if err := importJournal(ctx, tx, journalPath); err != nil {
		return err
	}
	if err := importReports(ctx, tx, reportsDir); err != nil {
		return err
	}
	if err := importResolved(ctx, tx, resolvedPath); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	for _, p := range []string{journalPath, reportsDir, resolvedPath, filepath.Join(dir, "accepted.json")} {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("remove imported %s: %w", filepath.Base(p), err)
		}
	}
	return nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func importJournal(ctx context.Context, tx *sql.Tx, path string) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read old change log: %w", err)
	}
	seen := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var c legacyChange
		if json.Unmarshal(sc.Bytes(), &c) != nil || c.ID == "" {
			continue // a torn line from a crash mid-write
		}
		if seen[c.ID] {
			_, err = tx.ExecContext(ctx, `UPDATE changes SET status = ?, error = ?,
				after = CASE WHEN ? = '' THEN after ELSE ? END WHERE id = ?`, c.Status, c.Error, c.After, c.After, c.ID)
		} else {
			seen[c.ID] = true
			_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO changes
				(id, at, action, summary, target, before, after, reversible, reverts, status, error)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				c.ID, c.At.UTC().Format(Stamp), c.Action, c.Summary, c.Target, c.Before, c.After, c.Reversible, c.Reverts, c.Status, c.Error)
		}
		if err != nil {
			return fmt.Errorf("import change %s: %w", c.ID, err)
		}
	}
	return nil
}

func importReports(ctx context.Context, tx *sql.Tx, dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("list old reports: %w", err)
	}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil || !json.Valid(body) {
			continue // a report torn by a crash is not worth stopping for
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO reports (id, at, body) VALUES (?, ?, ?)`, id, id, string(body)); err != nil {
			return fmt.Errorf("import report %s: %w", id, err)
		}
	}
	return nil
}

func importResolved(ctx context.Context, tx *sql.Tx, path string) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read old resolutions: %w", err)
	}
	var list []legacyResolution
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	for _, r := range list {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO resolutions (key, title, at) VALUES (?, ?, ?)`,
			r.Key, r.Title, r.At.UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("import resolution %s: %w", r.Key, err)
		}
	}
	return nil
}
