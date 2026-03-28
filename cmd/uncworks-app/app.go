//go:build darwin

// app.go — Wails application backend: cluster lifecycle and menu bar status.
package main

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds application state and exposes methods to the Wails frontend.
type App struct {
	ctx            context.Context
	statusPollStop context.CancelFunc
}

func NewApp() *App { return &App{} }

// startup is called when the app starts. Begins polling cluster status.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	pollCtx, cancel := context.WithCancel(ctx)
	a.statusPollStop = cancel
	go a.pollStatus(pollCtx)
}

// shutdown is called before the app exits.
func (a *App) shutdown(_ context.Context) {
	if a.statusPollStop != nil {
		a.statusPollStop()
	}
}

// ClusterStatus returns the current health of the UNCWORKS stack.
// Called by the frontend to update the menu bar icon and status display.
func (a *App) ClusterStatus() string {
	out, err := exec.CommandContext(a.ctx, "uncworks", "status", "--namespace", "uncworks").Output()
	if err != nil {
		return "stopped"
	}
	if strings.Contains(string(out), "No pods found") {
		return "stopped"
	}
	// If all pods show "Yes" in the READY column, we're running.
	if strings.Contains(string(out), "No (") || strings.Contains(string(out), "No\n") {
		return "degraded"
	}
	return "running"
}

// StartCluster invokes `uncworks setup` and streams output to the frontend.
func (a *App) StartCluster() {
	cmd := exec.CommandContext(a.ctx, "uncworks", "setup", "--non-interactive")
	cmd.Stdout = &frontendWriter{ctx: a.ctx, event: "setup:output"}
	cmd.Stderr = &frontendWriter{ctx: a.ctx, event: "setup:output"}
	_ = cmd.Run()
	runtime.EventsEmit(a.ctx, "setup:done")
}

// StopCluster invokes `uncworks teardown`.
func (a *App) StopCluster() {
	cmd := exec.CommandContext(a.ctx, "uncworks", "teardown")
	cmd.Stdout = &frontendWriter{ctx: a.ctx, event: "teardown:output"}
	cmd.Stderr = &frontendWriter{ctx: a.ctx, event: "teardown:output"}
	_ = cmd.Run()
	runtime.EventsEmit(a.ctx, "teardown:done")
}

// pollStatus periodically emits cluster status events to the frontend.
func (a *App) pollStatus(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status := a.ClusterStatus()
			runtime.EventsEmit(ctx, "cluster:status", status)
		}
	}
}

// frontendWriter streams command output lines as Wails events.
type frontendWriter struct {
	ctx   context.Context
	event string
	buf   strings.Builder
}

func (w *frontendWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			if line := w.buf.String(); line != "" {
				runtime.EventsEmit(w.ctx, w.event, line)
			}
			w.buf.Reset()
		} else {
			w.buf.WriteByte(b)
		}
	}
	return len(p), nil
}
