package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/mcp"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
	"github.com/cn8001/reforgepulse/internal/tasks"
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
	mcpMode := len(os.Args) > 1 && os.Args[1] == "mcp"
	if !mcpMode && elevate(host) {
		return nil
	}
	dir, err := dataDir()
	if err != nil {
		return err
	}
	// The window exists only after the backend, so task updates go through
	// emit, which runWindow points at the window's event bus.
	var emit func(tasks.Status)
	b, err := newBackend(host, dir, func(s tasks.Status) {
		if emit != nil {
			emit(s)
		}
	})
	if err != nil {
		return err
	}
	defer b.Close()
	if mcpMode {
		// stdout carries the protocol, so the log goes to stderr only.
		log.SetOutput(os.Stderr)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return mcp.Serve(ctx, os.Stdin, os.Stdout, b.tools())
	}
	return runWindow(b, dir, &emit)
}

// taskEvent carries task updates to the status bar.
const taskEvent = "task"

func runWindow(b *backend, dir string, emit *func(tasks.Status)) error {
	application.RegisterEvent[tasks.Status](taskEvent)
	a := application.New(application.Options{
		Name:        "ReforgePulse",
		Description: "Finds what slows down or crashes a Windows PC and cleans up what is not needed",
		Services:    b.services(),
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows:     application.WindowsOptions{WebviewUserDataPath: filepath.Join(dir, "webview")},
	})
	*emit = func(s tasks.Status) { a.Event.Emit(taskEvent, s) }

	// The accent comes from a.Env, which exists only once the app does.
	a.RegisterService(application.NewService(app.NewThemeService(a.Env)))

	// Mica gives the window the Windows 11 material. It starts dark, the
	// default theme; light mode paints its own opaque background over it.
	a.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "ReforgePulse",
		Width:          1100,
		Height:         700,
		MinWidth:       760,
		MinHeight:      520,
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows:        application.WindowsWindow{BackdropType: application.Mica, Theme: application.Dark},
		URL:            "/",
	})
	return a.Run()
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
