//go:build darwin

// tray.go — macOS menu bar status item for UNCWORKS.
// Keeps the app alive when the main window is closed.
package main

import (
	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// initTray registers the system tray icon using RunWithExternalLoop so that
// energye/systray does not attempt to start its own Cocoa run loop (which
// Wails already owns on the main thread).  nativeStart() creates an
// NSStatusBarWindow which must be called on the main thread, so we schedule it
// via GCD dispatch_async on the main queue (see tray_dispatch.go).
func (a *App) initTray() {
	start, _ := systray.RunWithExternalLoop(a.onTrayReady, a.onTrayExit)
	trayDispatchMain(start)
}

func (a *App) onTrayReady() {
	// Use the app icon as the menu bar icon.
	// macOS renders template images correctly in both light and dark menu bars.
	systray.SetTemplateIcon(trayIconBytes(), trayIconBytes())
	systray.SetTooltip("UNCWORKS")

	mShow := systray.AddMenuItem("Show UNCWORKS", "Bring the UNCWORKS window to front")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit UNCWORKS", "Fully quit the application")

	mShow.Click(func() {
		runtime.WindowShow(a.ctx)
	})
	mQuit.Click(func() {
		systray.Quit()
		runtime.Quit(a.ctx)
	})
}

func (a *App) onTrayExit() {}
