//go:build darwin

// app.go — Wails application backend: cluster lifecycle, settings, and service management.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds application state and exposes methods to the Wails frontend.
type App struct {
	ctx            context.Context
	statusPollStop context.CancelFunc
	pf             *portForwardManager
}

func NewApp() *App {
	return &App{
		pf: newPortForwardManager(),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	bootstrapPATH()
	_ = bootstrapConfig()
	pollCtx, cancel := context.WithCancel(ctx)
	a.statusPollStop = cancel
	go a.pollStatus(pollCtx)
	a.initTray()
}

// bootstrapPATH extends the process PATH to include common tool locations that
// macOS GUI apps don't inherit from the user's shell environment.
func bootstrapPATH() {
	extra := []string{
		"/opt/homebrew/bin",           // Homebrew (Apple Silicon)
		"/usr/local/bin",              // Homebrew (Intel) + Docker Desktop
		"/opt/homebrew/sbin",
		"/usr/local/sbin",
		"/nix/var/nix/profiles/default/bin", // Nix
	}
	// Also include nix per-user profile and macOS user profile (nix-darwin)
	if home, err := os.UserHomeDir(); err == nil {
		extra = append(extra,
			home+"/.nix-profile/bin",
			"/etc/profiles/per-user/"+os.Getenv("USER")+"/bin",
		)
	}
	current := os.Getenv("PATH")
	for _, p := range extra {
		if !strings.Contains(current, p) {
			current = p + ":" + current
		}
	}
	_ = os.Setenv("PATH", current)
}

// shutdown is called before the app exits.
func (a *App) shutdown(_ context.Context) {
	if a.statusPollStop != nil {
		a.statusPollStop()
	}
	a.pf.stopAll()
	systray.Quit()
}

// ── Settings ──────────────────────────────────────────────────────────────────

// GetSettings returns the persisted app settings.
// GitHubAuthed is set dynamically from Keychain presence, not from the YAML file.
func (a *App) GetSettings() (AppSettings, error) {
	s, err := loadAppSettings()
	if err != nil {
		return s, err
	}
	s.GitHubAuthed = isGitHubAuthed()
	return s, nil
}

// SaveSettings persists the app settings to disk.
func (a *App) SaveSettings(s AppSettings) error {
	return saveAppSettings(s)
}

// knownEnvVars is the curated list we expose in the Settings UI.
var knownEnvVars = []struct {
	Key  string
	Desc string
}{
	{"EDITOR",           "Preferred text editor (e.g. nvim, vim, nano)"},
	{"VISUAL",           "Preferred visual editor (falls back to EDITOR)"},
	{"PAGER",            "Preferred pager for long output (e.g. less, more)"},
	{"SHELL",            "Preferred shell for subprocesses"},
	{"XDG_CONFIG_HOME",  "User config directory (default: ~/.config)"},
	{"XDG_DATA_HOME",    "User data directory (default: ~/.local/share)"},
	{"XDG_STATE_HOME",   "User state directory (default: ~/.local/state)"},
	{"XDG_CACHE_HOME",   "User cache directory (default: ~/.cache)"},
	{"XDG_RUNTIME_DIR",  "Runtime files directory (sockets, PIDs)"},
	{"KUBECONFIG",       "Path to kubeconfig file (default: ~/.kube/config)"},
}

// GetEnvVars returns the curated environment variables with their system
// values and any user overrides stored in settings.
func (a *App) GetEnvVars() ([]EnvVarInfo, error) {
	s, err := loadAppSettings()
	if err != nil {
		return nil, err
	}
	overrides := s.EnvOverrides
	if overrides == nil {
		overrides = map[string]string{}
	}
	result := make([]EnvVarInfo, len(knownEnvVars))
	for i, v := range knownEnvVars {
		result[i] = EnvVarInfo{
			Key:      v.Key,
			System:   os.Getenv(v.Key),
			Override: overrides[v.Key],
			Desc:     v.Desc,
		}
	}
	return result, nil
}

// ── Kube helpers ──────────────────────────────────────────────────────────────

// GetKubeContexts returns all kubeconfig context names.
func (a *App) GetKubeContexts() ([]string, error) {
	out, err := exec.Command("kubectl", "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		return nil, err
	}
	var ctxs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if t := strings.TrimSpace(line); t != "" {
			ctxs = append(ctxs, t)
		}
	}
	return ctxs, nil
}

// AutodetectNamespace tries to find the uncworks namespace in the given kubecontext.
// Returns the detected namespace name, or empty string if not found.
func (a *App) AutodetectNamespace(kubeContext string) string {
	out, err := exec.Command("kubectl", "--context="+kubeContext, "get", "ns", "-o", "jsonpath={.items[*].metadata.name}").Output()
	if err != nil {
		return ""
	}
	for _, ns := range strings.Fields(string(out)) {
		if ns == "uncworks" {
			return ns
		}
	}
	return ""
}

// ── Cluster lifecycle ─────────────────────────────────────────────────────────

// ClusterStatus returns "running", "degraded", or "stopped".
func (a *App) ClusterStatus() string {
	s, _ := loadAppSettings()
	ns := s.Namespace
	if ns == "" {
		ns = "uncworks"
	}
	out, err := exec.CommandContext(a.ctx, "kubectl", "get", "pods",
		"-n", ns, "--no-headers").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return "stopped"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "Running") {
			return "degraded"
		}
	}
	return "running"
}

// applyEnvOverrides adds any user-configured env overrides to a command.
func applyEnvOverrides(cmd *exec.Cmd) {
	s, err := loadAppSettings()
	if err != nil || len(s.EnvOverrides) == 0 {
		return
	}
	env := os.Environ()
	for k, v := range s.EnvOverrides {
		if v != "" {
			env = append(env, k+"="+v)
		}
	}
	cmd.Env = env
}

// StartCluster invokes `uncworks setup` and streams output to the frontend.
func (a *App) StartCluster() {
	cmd := exec.CommandContext(a.ctx, "uncworks", "setup", "--non-interactive")
	applyEnvOverrides(cmd)
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

// ── Service management ────────────────────────────────────────────────────────

// ListServices returns live status for all known UNCWORKS services.
func (a *App) ListServices() []ServiceInfo {
	s, _ := loadAppSettings()
	ns := s.Namespace
	if ns == "" {
		ns = "uncworks"
	}
	result := make([]ServiceInfo, len(knownServices))
	for i, svc := range knownServices {
		svc.Ready = queryPodReady(ns, svc.Name)
		fwd, port := a.pf.isForwarding(svc.Name)
		svc.Forwarding = fwd
		svc.LocalPort = port
		result[i] = svc
	}
	return result
}

// RestartService runs kubectl rollout restart on the named deployment.
func (a *App) RestartService(name string) error {
	s, _ := loadAppSettings()
	ns := s.Namespace
	if ns == "" {
		ns = "uncworks"
	}
	return restartService(ns, name)
}

// StartPortForward begins kubectl port-forward for the named service.
// localPort=0 means auto-assign from the configured port range.
func (a *App) StartPortForward(name string, localPort int) error {
	s, _ := loadAppSettings()
	ns := s.Namespace
	if ns == "" {
		ns = "uncworks"
	}

	var clusterPort int
	for _, svc := range knownServices {
		if svc.Name == name {
			clusterPort = svc.ClusterPort
			break
		}
	}
	if clusterPort == 0 {
		return fmt.Errorf("service %q has no cluster port to forward", name)
	}

	if localPort == 0 {
		localPort = a.nextFreePort(s)
	}
	return a.pf.start(name, ns, localPort, clusterPort)
}

// StopPortForward stops the port-forward for the named service.
func (a *App) StopPortForward(name string) error {
	return a.pf.stop(name)
}

// OpenService opens the port-forwarded URL for a service in the default browser.
func (a *App) OpenService(name string) error {
	fwd, port := a.pf.isForwarding(name)
	if !fwd {
		return fmt.Errorf("service %q is not currently port-forwarded", name)
	}
	url := fmt.Sprintf("http://localhost:%d", port)
	return exec.Command("open", url).Run()
}

// nextFreePort returns the first port in the configured range not already in use.
func (a *App) nextFreePort(s AppSettings) int {
	start := s.PortRangeStart
	end := s.PortRangeEnd
	if start == 0 {
		start = 50100
	}
	if end == 0 {
		end = 50120
	}
	a.pf.mu.Lock()
	defer a.pf.mu.Unlock()
	used := make(map[int]bool)
	for _, p := range a.pf.ports {
		used[p] = true
	}
	for p := start; p <= end; p++ {
		if !used[p] {
			return p
		}
	}
	return start // fallback
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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

// ── API proxy middleware ───────────────────────────────────────────────────────

// APIProxyMiddleware is a Wails AssetServer middleware that transparently proxies
// ConnectRPC (/aot.api.v1.*) and REST API (/api/*) requests to the configured
// UNCWORKS API server. This lets the embedded frontend reach the in-cluster
// apiserver without needing VITE_API_URL set at build time.
func (a *App) APIProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/aot.api.v1.") && !strings.HasPrefix(p, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		s, _ := loadAppSettings()
		target := s.APIServerURL
		if target == "" {
			target = "http://localhost:50055"
		}
		u, err := url.Parse(target)
		if err != nil {
			http.Error(w, "invalid apiserver URL", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(u)
		r.Host = u.Host
		proxy.ServeHTTP(w, r)
	})
}
