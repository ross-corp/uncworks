//go:build darwin

// logging.go — File-based logging and crash reporting for the UNCWORKS desktop app.
// Redirects all stdout/stderr to ~/Library/Logs/UNCWORKS/uncworks.log and writes
// structured crash reports on panic.
package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"time"
)

const logDir = "UNCWORKS"
const logFile = "uncworks.log"

// appLogDir returns ~/Library/Logs/UNCWORKS, creating it if needed.
func appLogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "Logs", logDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// setupLogging opens ~/Library/Logs/UNCWORKS/uncworks.log (append mode) and
// redirects the standard library logger, slog, and the raw stderr fd so that
// all output from the app (including Wails internal logs) lands in the file.
// Returns the path to the log file and a close function.
func setupLogging() (logPath string, close func()) {
	dir, err := appLogDir()
	if err != nil {
		// Can't set up logging — fall back to default (no-op close).
		return "", func() {}
	}

	logPath = filepath.Join(dir, logFile)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", func() {}
	}

	// Write to both the log file and the original stderr so `wails dev` still works.
	multi := io.MultiWriter(f, os.Stderr)

	// Redirect standard log package.
	log.SetOutput(multi)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	// Redirect slog default logger.
	handler := slog.NewTextHandler(multi, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	// Redirect os.Stderr so that anything written to fd 2 (e.g. fmt.Fprintln(os.Stderr, ...))
	// also goes to the log file. We can't dup2 without cgo, but we can replace the *os.File.
	os.Stderr = f

	log.Printf("[uncworks] log file: %s", logPath)
	return logPath, func() { _ = f.Close() }
}

// withCrashReporting wraps fn with a deferred panic recovery that writes a
// crash report to ~/Library/Logs/UNCWORKS/crash-TIMESTAMP.log before re-panicking.
func withCrashReporting(fn func()) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		writeCrashReport(r)
		panic(r) // re-panic so the OS records the exit
	}()
	fn()
}

func writeCrashReport(r any) {
	dir, err := appLogDir()
	if err != nil {
		return
	}
	ts := time.Now().Format("2006-01-02T15-04-05")
	path := filepath.Join(dir, fmt.Sprintf("crash-%s.log", ts))
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "UNCWORKS crash report — %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "panic: %v\n\n", r)
	fmt.Fprintf(f, "goroutine stack:\n%s\n", debug.Stack())

	// Also append to the main log file.
	log.Printf("[uncworks] CRASH: %v\n%s", r, debug.Stack())
}

// OpenLogInConsole opens the UNCWORKS log file in macOS Console.app.
func (a *App) OpenLogInConsole() error {
	dir, err := appLogDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, logFile)
	// Ensure the file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, cerr := os.Create(path)
		if cerr != nil {
			return cerr
		}
		f.Close()
	}
	return exec.Command("open", "-a", "Console", path).Run()
}

// LogPath returns the path to the current log file (exposed to the frontend).
func (a *App) LogPath() string {
	dir, err := appLogDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, logFile)
}
