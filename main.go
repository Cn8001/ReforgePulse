package main

import (
	"embed"
	"errors"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	host := windows.System{}
	if elevate(host) {
		return nil
	}
	app := application.New(application.Options{
		Name:        "ReforgePulse",
		Description: "Finds what slows down or crashes a Windows PC and cleans up what is not needed",
		Services:    services(),
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
	})
	// Translucent background plus Mica lets the window take the Windows 11
	// material and theme instead of a colour of our own.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "ReforgePulse",
		Width:          1100,
		Height:         700,
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows:        application.WindowsWindow{BackdropType: application.Mica},
		URL:            "/",
	})
	return app.Run()
}

// elevate restarts the app through UAC unless it already runs elevated, and
// reports whether the elevated copy took over. Dev mode is skipped because
// the elevated copy would lose the dev server. When elevation is refused or
// fails, the app keeps running with fewer checks rather than not at all.
func elevate(p platform.Privileges) bool {
	if p.Elevated() || os.Getenv("FRONTEND_DEVSERVER_URL") != "" {
		return false
	}
	err := p.Relaunch(os.Args[1:])
	if err != nil && !errors.Is(err, platform.ErrDenied) {
		log.Printf("running without administrator rights: %v", err)
	}
	return err == nil
}
