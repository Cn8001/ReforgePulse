package app

import (
	"bufio"
	"bytes"
	"strings"
)

// About is what the Settings page shows about this build.
type About struct {
	Version string `json:"version"`
	DataDir string `json:"dataDir"` // the one folder ReforgePulse keeps everything in
	Terms   string `json:"terms"`
}

type AboutService struct {
	about About
}

func NewAboutService(a About) *AboutService {
	return &AboutService{about: a}
}

func (s *AboutService) Info() About { return s.about }

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
