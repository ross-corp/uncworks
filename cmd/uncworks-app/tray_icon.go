//go:build darwin

// tray_icon.go — embeds the app icon PNG for use as the menu bar template icon.
package main

import _ "embed"

//go:embed build/appicon.png
var trayIconPNG []byte

// trayIconBytes returns the PNG bytes used as the menu bar status item icon.
func trayIconBytes() []byte { return trayIconPNG }
