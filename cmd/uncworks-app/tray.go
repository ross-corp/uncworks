//go:build darwin

// tray.go — macOS menu bar status item for UNCWORKS.
// Keeps the app alive when the main window is closed.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── Dot icon helpers ─────────────────────────────────────────────────────────

// dotColor names the palette of macOS system status colors.
type dotColor int

const (
	dotGreen  dotColor = iota // macOS systemGreen  — success / ok
	dotRed                    // macOS systemRed    — error / missing
	dotOrange                 // macOS systemOrange — degraded / warning
	dotBlue                   // macOS systemBlue   — active / running
	dotGray                   // macOS systemGray   — neutral / unknown
)

var dotPalette = map[dotColor][3]uint8{
	dotGreen:  {52, 199, 89},
	dotRed:    {255, 59, 48},
	dotOrange: {255, 149, 0},
	dotBlue:   {0, 122, 255},
	dotGray:   {142, 142, 147},
}

// dotIconPNG returns a 16×16 RGBA PNG with an antialiased filled circle in the
// given macOS system color. The background is fully transparent so the dot
// sits cleanly against the menu item background in both light and dark mode.
func dotIconPNG(c dotColor) []byte {
	const size = 16
	const cx, cy = float64(size)/2 - 0.5, float64(size)/2 - 0.5
	const radius = float64(size)/2 - 1.5 // leave 1.5 px margin for AA fringe

	rgb := dotPalette[c]
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			// Smooth step for antialiasing over a 1 px fringe.
			alpha := 1.0 - math.Max(0, math.Min(1, dist-radius))
			if alpha > 0 {
				img.Set(x, y, color.RGBA{
					R: rgb[0],
					G: rgb[1],
					B: rgb[2],
					A: uint8(alpha * 255),
				})
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// pre-rendered dot icon bytes (generated once at init time).
var (
	iconGreen  = dotIconPNG(dotGreen)
	iconRed    = dotIconPNG(dotRed)
	iconOrange = dotIconPNG(dotOrange)
	iconBlue   = dotIconPNG(dotBlue)
	iconGray   = dotIconPNG(dotGray)
)

// ── trayStatusItem ───────────────────────────────────────────────────────────

// trayStatusItem wraps a systray.MenuItem used as a read-only status display.
// Safe to call before the item is created (nil guard).
type trayStatusItem struct {
	item *systray.MenuItem
}

func (t *trayStatusItem) setDot(icon []byte, title string) {
	if t.item == nil {
		return
	}
	t.item.SetIcon(icon)
	t.item.SetTitle(title)
}

// set updates the cluster status menu item.
func (t *trayStatusItem) set(status string) {
	switch status {
	case "running":
		t.setDot(iconGreen, "Cluster: running")
	case "degraded":
		t.setDot(iconOrange, "Cluster: degraded")
	case "stopped":
		t.setDot(iconRed, "Cluster: stopped")
	default:
		t.setDot(iconGray, "Cluster: "+status)
	}
}

// setConfig updates the configuration status menu item based on current settings.
func (t *trayStatusItem) setConfig() {
	s, _ := loadAppSettings()
	hasLLM := s.LLMAPIKey != "" ||
		s.LiteLLMURL == "" ||
		s.LiteLLMURL == "http://litellm:4000" ||
		strings.HasPrefix(s.LiteLLMURL, "http://localhost:")
	if hasLLM {
		t.setDot(iconGreen, "Config: OK")
	} else {
		t.setDot(iconRed, "Config: no LLM key")
	}
}

// setJobs updates the active jobs menu item. count=-1 means API unreachable.
func (t *trayStatusItem) setJobs(count int) {
	switch {
	case count < 0:
		t.setDot(iconGray, "Jobs: API unreachable")
	case count == 0:
		t.setDot(iconGray, "Jobs: none active")
	case count == 1:
		t.setDot(iconBlue, "Jobs: 1 running")
	default:
		t.setDot(iconBlue, fmt.Sprintf("Jobs: %d running", count))
	}
}

// ── Tray lifecycle ───────────────────────────────────────────────────────────

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
		trayDispatchMain(a.onTrayReady)
	}, a.onTrayExit)
	trayDispatchMain(start)
}

func (a *App) onTrayReady() {
	systray.SetTemplateIcon(trayIconBytes(), trayIconBytes())
	systray.SetTooltip("UNCWORKS")

	// ── Status section ──────────────────────────────────────────────────────
	mConfig := systray.AddMenuItem("Config: checking…", "")
	mConfig.Disable()
	mConfig.SetIcon(iconGray)
	a.trayConfig.item = mConfig

	mJobs := systray.AddMenuItem("Jobs: checking…", "")
	mJobs.Disable()
	mJobs.SetIcon(iconGray)
	a.trayJobs.item = mJobs

	mCluster := systray.AddMenuItem("Cluster: checking…", "")
	mCluster.Disable()
	mCluster.SetIcon(iconGray)
	a.trayCluster.item = mCluster

	systray.AddSeparator()

	// ── Actions ─────────────────────────────────────────────────────────────
	mShow := systray.AddMenuItem("Show UNCWORKS", "")
	mPrefs := systray.AddMenuItem("Settings…", "")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit UNCWORKS", "")

	// ── Handlers ────────────────────────────────────────────────────────────
	// Wails runtime calls must NOT happen on the main thread (Obj-C callback).
	mShow.Click(func() {
		go runtime.WindowShow(a.ctx)
	})
	mPrefs.Click(func() {
		go func() {
			runtime.WindowShow(a.ctx)
			runtime.EventsEmit(a.ctx, "app:open-settings")
		}()
	})
	mQuit.Click(func() {
		systray.Quit()
		go runtime.Quit(a.ctx)
	})

	// Attach NSMenu to the status item so it appears on click.
	systray.CreateMenu()

	// Seed all status items immediately so the menu isn't blank on first open.
	go func() {
		a.trayConfig.setConfig()
		a.trayJobs.setJobs(a.countRunningJobs())
		status := a.ClusterStatus()
		a.trayCluster.set(status)
	}()
}

func (a *App) onTrayExit() {}
