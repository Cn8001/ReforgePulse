package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
)

// About is what the Settings page shows about this build.
type About struct {
	Version string `json:"version"`
	DataDir string `json:"dataDir"` // the one folder ReforgePulse keeps everything in
	Terms   string `json:"terms"`
	License string `json:"license"`
	Source  string `json:"source"`  // the repository's web page
	Wording string `json:"wording"` // names this text of the terms and licence, for consent
}

// ConsentStore remembers which wording of the terms the user accepted.
type ConsentStore interface {
	Accepted(ctx context.Context, wording string) (bool, error)
	Accept(ctx context.Context, wording string) error
}

type AboutService struct {
	about   About
	consent ConsentStore
}

func NewAboutService(a About, c ConsentStore) *AboutService {
	return &AboutService{about: a, consent: c}
}

func (s *AboutService) Info() About { return s.about }

// Accepted reports whether the user accepted these terms and this licence.
// The window shows them before anything else until they do.
func (s *AboutService) Accepted(ctx context.Context) (bool, error) {
	ok, err := s.consent.Accepted(ctx, s.about.Wording)
	if err != nil {
		return false, fmt.Errorf("read consent: %w", err)
	}
	return ok, nil
}

func (s *AboutService) Accept(ctx context.Context) error {
	if err := s.consent.Accept(ctx, s.about.Wording); err != nil {
		return fmt.Errorf("accept terms: %w", err)
	}
	return nil
}

// VersionFrom reads info.version from build/config.yml, the one place the
// version lives. A full YAML parser would be a dependency for one line.
func VersionFrom(config []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(config))
	inInfo := false
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inInfo = strings.HasPrefix(line, "info:")
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if inInfo && ok && key == "version" {
			value, _, _ = strings.Cut(value, "#")
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return "dev"
}
