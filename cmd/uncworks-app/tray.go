//go:build darwin

// tray.go — macOS menu bar status item for UNCWORKS.
// Keeps the app alive when the main window is closed.
package main

import (
	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// trayStatusItem wraps a systray.MenuItem used as a read-only status display.
// safe to call before the item is created (nil guard).
type trayStatusItem struct {
	item *systray.MenuItem
}

func (t *trayStatusItem) set(status string) {
	if t.item == nil {
		return
	}
	label := map[string]string{
		"running":  "Cluster  running",
		"degraded": "Cluster  degraded",
		"stopped":  "Cluster  stopped",
	}[status]
	if label == "" {
		label = "Cluster  " + status
	}
	t.item.SetTitle(label)
}

// initTray registers the system tray icon using RunWithExternalLoop so that
// energye/systray does not attempt to start its own Cocoa run loop (Wails
// already owns the main thread). nativeStart() and all NSMenu/NSMenuItem calls
// must happen on the main thread, so we double-dispatch via GCD:
//
//  1. trayDispatchMain(start) → nativeStart() runs on main thread
//  2. inside onReady wrapper → trayDispatchMain(onTrayReady) → AddMenuItem
//     runs on main thread (fixes blank menu on first click)
func (a *App) initTray() {
	start, _ := systray.RunWithExternalLoop(func() {
		// RunWithExternalLoop calls onReady in a goroutine. Dispatch back to
		// the main thread so that AddMenuItem (NSMenuItem) calls are safe.
		trayDispatchMain(a.onTrayReady)
	}, a.onTrayExit)
	trayDispatchMain(start)
}

func (a *App) onTrayReady() {
	systray.SetTemplateIcon(trayIconBytes(), trayIconBytes())
	systray.SetTooltip("UNCWORKS")

	// ── Status section ──────────────────────────────────────────────────────
	mStatus := systray.AddMenuItem("Cluster  checking…", "")
	mStatus.Disable()
	a.trayCluster.item = mStatus

	systray.AddSeparator()

	// ── Actions ─────────────────────────────────────────────────────────────
	mShow := systray.AddMenuItem("Show UNCWORKS", "")
	mPrefs := systray.AddMenuItem("Preferences…", "")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit UNCWORKS", "")

	// ── Handlers ────────────────────────────────────────────────────────────
	mShow.Click(func() {
		runtime.WindowShow(a.ctx)
	})
	mPrefs.Click(func() {
		runtime.WindowShow(a.ctx)
		runtime.EventsEmit(a.ctx, "app:open-settings")
	})
	mQuit.Click(func() {
		systray.Quit()
		runtime.Quit(a.ctx)
	})

	// Attach the NSMenu to the status item so it appears on click.
	// energye/systray on macOS does not do this automatically — without
	// CreateMenu() the items exist but are never shown.
	systray.CreateMenu()

	// Seed with current status immediately.
	go func() {
		status := a.ClusterStatus()
		a.trayCluster.set(status)
	}()
}

func (a *App) onTrayExit() {}
