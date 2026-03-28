//go:build darwin

// cmd/uncworks-app — UNCWORKS native macOS application (Wails v2).
// Embeds the React web frontend in a native WKWebView window.
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
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	appMenu := menu.NewMenu()

	// Application menu (UNCWORKS)
	appMenuItem := appMenu.AddSubmenu("UNCWORKS")
	appMenuItem.AddText("About UNCWORKS", nil, func(_ *menu.CallbackData) {
		runtime.WindowShow(app.ctx)
	})
	appMenuItem.AddSeparator()
	appMenuItem.AddText("Preferences…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, "app:open-settings")
	})
	appMenuItem.AddSeparator()
	appMenuItem.AddText("Quit UNCWORKS", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(app.ctx)
	})

	// Edit menu — gives us system cut/copy/paste/select-all
	editMenu := appMenu.AddSubmenu("Edit")
	editMenu.AddText("Undo", keys.CmdOrCtrl("z"), nil)
	editMenu.AddText("Redo", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey), nil)
	editMenu.AddSeparator()
	editMenu.AddText("Cut", keys.CmdOrCtrl("x"), nil)
	editMenu.AddText("Copy", keys.CmdOrCtrl("c"), nil)
	editMenu.AddText("Paste", keys.CmdOrCtrl("v"), nil)
	editMenu.AddText("Select All", keys.CmdOrCtrl("a"), nil)

	err := wails.Run(&options.App{
		Title:     "UNCWORKS",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		Menu:      appMenu,
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
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
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
