//go:build darwin

// tray.go — macOS menu bar status item for UNCWORKS.
// Keeps the app alive when the main window is closed.
package main

import (
	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// initTray starts the system tray icon in a goroutine.
// Must be called after the Wails app context is available.
func (a *App) initTray() {
	go systray.Run(a.onTrayReady, a.onTrayExit)
}

func (a *App) onTrayReady() {
	// Use the app icon as the menu bar icon.
	// macOS renders template images correctly in both light and dark menu bars.
	systray.SetTemplateIcon(trayIconBytes(), trayIconBytes())
	systray.SetTooltip("UNCWORKS")

	mShow := systray.AddMenuItem("Show UNCWORKS", "Bring the UNCWORKS window to front")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit UNCWORKS", "Fully quit the application")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				runtime.WindowShow(a.ctx)
			case <-mQuit.ClickedCh:
				systray.Quit()
				runtime.Quit(a.ctx)
				return
			}
		}
	}()
}

func (a *App) onTrayExit() {}
