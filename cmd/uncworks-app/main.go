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
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	// keys is used by Preferences… Cmd+, and Quit Cmd+Q above
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	_, closeLog := setupLogging()
	defer closeLog()
	withCrashReporting(run)
}

func run() {
	app := NewApp()

	// Resolve title bar style from persisted settings before the window opens.
	titleBar := mac.TitleBarHiddenInset()
	if s, err := loadAppSettings(); err == nil && s.ShowTrafficLights {
		titleBar = mac.TitleBarHidden()
	}

	appMenu := menu.NewMenu()

	// Application menu (UNCWORKS)
	appMenuItem := appMenu.AddSubmenu("UNCWORKS")
	appMenuItem.AddText("About UNCWORKS", nil, func(_ *menu.CallbackData) {
		runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{ //nolint:errcheck
			Type:    runtime.InfoDialog,
			Title:   "UNCWORKS",
			Message: "Agentic development environment\n\nVersion: dev\n\n© ROSS CORP",
		})
	})
	appMenuItem.AddSeparator()
	appMenuItem.AddText("Preferences…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, "app:open-settings")
	})
	appMenuItem.AddSeparator()
	appMenuItem.AddText("Quit UNCWORKS", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		btn, _ := runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         "Quit UNCWORKS?",
			Message:       "Are you sure you want to quit UNCWORKS?",
			Buttons:       []string{"Quit", "Cancel"},
			DefaultButton: "Cancel",
			CancelButton:  "Cancel",
		})
		if btn == "Quit" {
			runtime.Quit(app.ctx)
		}
	})

	// Edit menu — Wails v2 / WKWebView does not participate in the macOS
	// responder chain for clipboard operations, so we intercept the standard
	// shortcuts in Go and dispatch them via JavaScript + pbpaste/pbcopy.
	editMenu := appMenu.AddSubmenu("Edit")
	editMenu.AddText("Undo", nil, nil)
	editMenu.AddText("Redo", nil, nil)
	editMenu.AddSeparator()
	editMenu.AddText("Cut", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {
		runtime.WindowExecJS(app.ctx, `(function(){
			const el = document.activeElement;
			if (!el || (el.tagName !== 'INPUT' && el.tagName !== 'TEXTAREA')) return;
			const s = el.selectionStart, e = el.selectionEnd;
			if (s === e) return;
			const text = el.value.slice(s, e);
			navigator.clipboard.writeText(text).catch(()=>{});
			const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
			setter.call(el, el.value.slice(0, s) + el.value.slice(e));
			el.dispatchEvent(new Event('input', {bubbles: true}));
			el.selectionStart = el.selectionEnd = s;
		})()`)
	})
	editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {
		runtime.WindowExecJS(app.ctx, `(function(){
			const el = document.activeElement;
			let text = '';
			if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
				text = el.value.slice(el.selectionStart, el.selectionEnd);
			} else {
				const sel = window.getSelection();
				if (sel) text = sel.toString();
			}
			if (text) navigator.clipboard.writeText(text).catch(()=>{});
		})()`)
	})
	editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {
		app.pasteFromClipboard()
	})
	editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {
		runtime.WindowExecJS(app.ctx, `(function(){
			const el = document.activeElement;
			if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
				el.select();
			} else {
				document.execCommand('selectAll');
			}
		})()`)
	})

	// Window menu — standard macOS window management
	windowMenu := appMenu.AddSubmenu("Window")
	windowMenu.AddText("Close Window", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
		runtime.WindowHide(app.ctx)
	})

	err := wails.Run(&options.App{
		Title:     "UNCWORKS",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		Menu:      appMenu,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: app.APIProxyMiddleware,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			// Hide window instead of quitting; app lives in menu bar.
			runtime.WindowHide(ctx)
			return true // returning true prevents the default close/quit
		},
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             titleBar,
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
