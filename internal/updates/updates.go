// Package updates checks GitHub Releases for a newer ReforgePulse and puts
// it in place through the Wails updater. Nothing runs on its own: the user
// asks from Settings, and every download is checked against the release's
// SHA256SUMS before it is used.
package updates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// Repository is where releases are published; `make release` names the
// assets the way the provider's default matcher expects.
const Repository = "Cn8001/ReforgePulse"

// Source drives the updater the Wails app owns.
type Source struct {
	u    *updater.Updater
	prov *watchedProvider
}

// watchedProvider passes the provider's byte counts on to whoever started
// the current download, so the window can show how far it is. The updater
// refuses a second download while one runs, so one listener is enough.
type watchedProvider struct {
	updater.Provider
	mu     sync.Mutex
	listen func(written, total int64)
}

func (p *watchedProvider) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	return p.Provider.Download(ctx, r, dst, func(written, total int64) {
		onProgress(written, total)
		p.mu.Lock()
		listen := p.listen
		p.mu.Unlock()
		if listen != nil {
			listen(written, total)
		}
	})
}

func (p *watchedProvider) watch(listen func(written, total int64)) {
	p.mu.Lock()
	p.listen = listen
	p.mu.Unlock()
}

// New configures u for this version. A dev build has no version to compare,
// so it cannot update.
func New(u *updater.Updater, version string) (*Source, error) {
	if version == "" || version == "dev" {
		return nil, errors.New("a development build cannot update itself")
	}
	gh, err := github.New(github.Config{Repository: Repository, ChecksumAsset: "SHA256SUMS"})
	if err != nil {
		return nil, fmt.Errorf("set up github releases: %w", err)
	}
	prov := &watchedProvider{Provider: gh}
	err = u.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{prov}, Window: updater.WindowNone})
	if err != nil {
		return nil, fmt.Errorf("set up updater: %w", err)
	}
	return &Source{u: u, prov: prov}, nil
}

// Check returns the newer release, or nil when this one is the latest.
func (s *Source) Check(ctx context.Context) (*domain.Release, error) {
	rel, err := s.u.Check(ctx)
	if err != nil {
		return nil, fmt.Errorf("check github releases: %w", err)
	}
	if rel == nil {
		return nil, nil
	}
	return &domain.Release{
		Version: rel.Version, Name: rel.Name, Notes: rel.Notes, PublishedAt: rel.PublishedAt,
		Size: rel.Artifact.Size, URL: "https://github.com/" + Repository + "/releases/tag/v" + rel.Version,
	}, nil
}

// Download fetches and verifies the release Check found, ready to swap in,
// telling progress the bytes written so far and the size when known.
func (s *Source) Download(ctx context.Context, progress func(written, total int64)) error {
	s.prov.watch(progress)
	defer s.prov.watch(nil)
	if err := s.u.DownloadAndInstall(ctx); err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	return nil
}

// Restart closes the app; a helper swaps the new file in and starts it.
func (s *Source) Restart(ctx context.Context) error {
	if err := s.u.Restart(ctx); err != nil {
		return fmt.Errorf("restart into update: %w", err)
	}
	return nil
}
