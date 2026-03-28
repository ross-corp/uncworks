//go:build darwin

// cmd/uncworks-app — UNCWORKS native macOS application (Wails v2).
// Embeds the React web frontend in a native WKWebView window with a menu bar
// status icon for managing the local Kubernetes cluster lifecycle.
//
// Build requirements:
//   - macOS 13+ (Ventura or later)
//   - Wails v2 CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest
//   - Build with: wails build  (from this directory)
//   - Or via task: task app:build
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "UNCWORKS",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			Appearance: mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "UNCWORKS",
				Message: "Agentic development environment",
			},
		},
	})
	if err != nil {
		panic(err)
	}
}
