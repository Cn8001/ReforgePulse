package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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
