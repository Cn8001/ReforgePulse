package app

import "log/slog"

// LogService lets the window write its own errors into the app log, next to
// the backend's, since the webview's console is not reachable in a release.
type LogService struct {
	log *slog.Logger
}

func NewLogService(l *slog.Logger) *LogService { return &LogService{log: l} }

// Frontend records an error the window caught.
func (s *LogService) Frontend(message, where string) {
	s.log.Error("window error", "where", where, "message", message)
}
