package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"

	"github.com/cn8001/reforgepulse/internal/app"
	"github.com/cn8001/reforgepulse/internal/applog"
	"github.com/cn8001/reforgepulse/internal/mcp"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/platform/windows"
	"github.com/cn8001/reforgepulse/internal/probe"
	"github.com/cn8001/reforgepulse/internal/store"
	"github.com/cn8001/reforgepulse/internal/tasks"
	"github.com/cn8001/reforgepulse/internal/updates"
)

var (
	//go:embed all:frontend/dist
	assets embed.FS
	//go:embed build/config.yml
	buildConfig []byte // the version lives here
	//go:embed TERMS.md
	terms string
	//go:embed LICENSE
	license string
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// An update restarts this exe as a helper that only swaps the new file
	// in; it must not ask for UAC or open the data folder first.
	updater.HandleHelperMode()
	host := windows.System{}
	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode == "probe" {
		return runProbe(host) // started by the window, already elevated
	}
	mcpMode := mode == "mcp"
	if !mcpMode && elevate(host) {
		return nil
	}
	dir, err := dataDir()
	if err != nil {
		return err
	}
	var logger *slog.Logger
	var hw hardwareReads
	if mcpMode {
		hw = localHardware(host)
	} else {
		f, err := openLog(dir)
		if err != nil {
			return err
		}
		defer f.Close()
		logger = slog.New(slog.NewTextHandler(f, nil))
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("find executable: %w", err)
		}
		hw = probeHardware(exe, f, logger)
	}
	// The window exists only after the backend, so task updates go through
	// emit, which runWindow points at the window's event bus.
	var emit func(tasks.Status)
	b, err := newBackend(host, dir, func(s tasks.Status) {
		if emit != nil {
			emit(s)
		}
	}, hw)
	if err != nil {
		return err
	}
	defer b.Close()
	if mcpMode {
		// stdout carries the protocol, so the log goes to stderr only.
		log.SetOutput(os.Stderr)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return mcp.Serve(ctx, os.Stdin, os.Stdout, app.VersionFrom(buildConfig), b.tools())
	}
	return runWindow(b, dir, logger, &emit)
}

// runProbe serves the hardware reads to the window over stdin and stdout,
// until the window closes them.
func runProbe(host windows.System) error {
	hw := localHardware(host)
	defer hw.Close()
	return probe.Serve(context.Background(), os.Stdin, os.Stdout, probe.Handlers(probe.Local{
		Cores: hw.Cores, Loads: hw.Loads, Sensors: hw.Sensors, Tester: hw.Tester,
	}))
}

// maxLog is where the log rotates; two files of this size at most.
const maxLog = 1 << 20

// openLog sends everything the window process could lose to one file: the
// standard logger, and the runtime's own report of a fatal crash, which
// otherwise goes to a console the elevated window does not have.
func openLog(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data folder: %w", err)
	}
	f, err := applog.Open(dir, maxLog)
	if err != nil {
		return nil, err
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		f.Close()
		return nil, fmt.Errorf("send crash reports to the log: %w", err)
	}
	log.SetOutput(f)
	return f, nil
}

// taskEvent carries task updates to the status bar.
const taskEvent = "task"

func runWindow(b *backend, dir string, logger *slog.Logger, emit *func(tasks.Status)) error {
	application.RegisterEvent[tasks.Status](taskEvent)
	a := application.New(application.Options{
		Name:        "ReforgePulse",
		Description: "Finds what slows down or crashes a Windows PC and cleans up what is not needed",
		Services:    b.services(),
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows:     application.WindowsOptions{WebviewUserDataPath: filepath.Join(dir, "webview")},
		Logger:      logger,
		// Wails quits on a panic in a call by default; one broken call must
		// not close the app, and the stack goes to the log to be fixed.
		PanicHandler: func(p *application.PanicDetails) {
			logger.Error("panic", "error", p.Error, "stack", p.FullStackTrace)
		},
	})
	*emit = func(s tasks.Status) { a.Event.Emit(taskEvent, s) }

	// The accent comes from a.Env and updates from a.Updater, which exist
	// only once the app does.
	a.RegisterService(application.NewService(app.NewThemeService(a.Env)))
	a.RegisterService(application.NewService(app.NewLogService(logger)))
	a.RegisterService(application.NewService(app.NewUpdateService(app.UpdateDeps{
		Source: updateSource(a.Updater), Runner: b.runner, Journal: b.journal, Settings: store.NewSettings(b.db),
		Version: app.VersionFrom(buildConfig),
	})))

	// Mica gives the window the Windows 11 material. It starts dark, the
	// default theme; light mode paints its own opaque background over it.
	// Frameless, as in bmeditor: the frontend draws its own top bar with the
	// tabs, settings and window buttons.
	a.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "ReforgePulse",
		Frameless:      true,
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

// updateSource is GitHub Releases, except under the dev server, where the
// exe is a throwaway build that must not replace itself.
func updateSource(u *updater.Updater) app.UpdateSource {
	if os.Getenv("FRONTEND_DEVSERVER_URL") != "" {
		return app.UpdatesUnavailable(errors.New("updates are off while running from the dev server"))
	}
	src, err := updates.New(u, app.VersionFrom(buildConfig))
	if err != nil {
		return app.UpdatesUnavailable(err)
	}
	return src
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
