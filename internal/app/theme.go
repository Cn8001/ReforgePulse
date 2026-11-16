package app

// AccentSource is satisfied by Wails' environment manager (app.Env).
type AccentSource interface {
	GetAccentColor() string
}

// ThemeService hands the frontend the Windows accent colour, which CSS
// cannot read on its own.
type ThemeService struct {
	src AccentSource
}

func NewThemeService(src AccentSource) *ThemeService {
	return &ThemeService{src: src}
}

func (s *ThemeService) Accent() string {
	return s.src.GetAccentColor()
}
