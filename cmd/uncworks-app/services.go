//go:build darwin

// services.go — Service discovery and management for the desktop app.
// Wraps kubectl to list, restart, and port-forward in-cluster services.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ServiceInfo describes a manageable UNCWORKS in-cluster service.
type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	ClusterPort int    `json:"clusterPort"`
	LocalPort   int    `json:"localPort"`  // 0 = not forwarded
	Ready       bool   `json:"ready"`
	Forwarding  bool   `json:"forwarding"`
}

// knownServices lists the services we expose in the UI.
// ClusterPort is the container port exposed by the k8s Service.
var knownServices = []ServiceInfo{
	{Name: "apiserver", DisplayName: "API Server", ClusterPort: 50055},
	{Name: "litellm", DisplayName: "LiteLLM", ClusterPort: 4000},
	{Name: "web", DisplayName: "Web UI", ClusterPort: 3000},
	{Name: "worker", DisplayName: "Worker", ClusterPort: 0},
	{Name: "controller", DisplayName: "Controller", ClusterPort: 0},
}

// pfEntry holds the running process and the cancel func for its watcher goroutine.
type pfEntry struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

// portForwardManager tracks running kubectl port-forward subprocesses and
// automatically restarts them when they die unexpectedly.
type portForwardManager struct {
	mu      sync.Mutex
	entries map[string]*pfEntry // keyed by service name
	ports   map[string]int      // local port per service
	stopped bool                // set true by stopAll; prevents watcher restarts

	// wailsCtx is set by setWailsCtx after the Wails runtime is ready.
	// Used to emit reconnect events to the frontend. May be nil if not yet set.
	wailsCtx context.Context
}

func newPortForwardManager() *portForwardManager {
	return &portForwardManager{
		entries: make(map[string]*pfEntry),
		ports:   make(map[string]int),
	}
}

// setWailsCtx wires the Wails runtime context so the manager can emit events.
func (m *portForwardManager) setWailsCtx(ctx context.Context) {
	m.mu.Lock()
	m.wailsCtx = ctx
	m.mu.Unlock()
}

// emitEvent emits a Wails event if the runtime context is available.
// Safe to call without holding mu.
func (m *portForwardManager) emitEvent(event string, data ...interface{}) {
	m.mu.Lock()
	ctx := m.wailsCtx
	m.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, event, data...)
	}
}

// isForwarding returns (true, localPort) when the named service has an active
// kubectl port-forward process. ProcessState is set by cmd.Wait() after exit,
// so a non-nil ProcessState reliably indicates the process has exited.
func (m *portForwardManager) isForwarding(name string) (bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[name]
	if !ok {
		return false, 0
	}
	// ProcessState is populated only after Wait() returns (watcher calls Wait).
	// A running process has a non-nil Process and nil ProcessState.
	if e.cmd.Process == nil || e.cmd.ProcessState != nil {
		return false, 0
	}
	return true, m.ports[name]
}

// pfBackoff returns successive backoff durations: 1s, 2s, 4s, …, capped at 30s.
func pfBackoff(attempt int) time.Duration {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// spawnOne starts a single kubectl port-forward process and returns it.
// Caller must not hold mu.
func spawnOne(name, namespace string, localPort, clusterPort int) (*exec.Cmd, error) {
	svcName := fmt.Sprintf("svc/uncworks-aot-%s", name)
	portArg := fmt.Sprintf("%d:%d", localPort, clusterPort)
	cmd := exec.Command("kubectl", "port-forward", "-n", namespace, svcName, portArg)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start port-forward %s: %w", name, err)
	}
	return cmd, nil
}

// watchAndRestart runs in its own goroutine. It waits for the kubectl process
// to exit, then restarts it with exponential backoff until ctx is cancelled.
func (m *portForwardManager) watchAndRestart(
	ctx context.Context,
	name, namespace string,
	localPort, clusterPort int,
) {
	attempt := 0
	for {
		// Grab the current cmd under the lock.
		m.mu.Lock()
		e, ok := m.entries[name]
		m.mu.Unlock()

		if !ok {
			return // entry was removed by stop()
		}

		// Wait for the process to exit. Wait() sets ProcessState on the cmd.
		if err := e.cmd.Wait(); err != nil {
			log.Printf("port-forward %s exited: %v", name, err)
		} else {
			log.Printf("port-forward %s exited cleanly", name)
		}

		// Check whether we should stop retrying.
		select {
		case <-ctx.Done():
			return
		default:
		}

		m.mu.Lock()
		if m.stopped {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		// Emit reconnecting event so the UI can show a spinner.
		m.emitEvent("portforward:reconnecting", name)

		backoff := pfBackoff(attempt)
		log.Printf("port-forward %s: restarting in %s (attempt %d)", name, backoff, attempt+1)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		attempt++

		// Spawn a fresh process.
		newCmd, err := spawnOne(name, namespace, localPort, clusterPort)
		if err != nil {
			log.Printf("port-forward %s: restart failed: %v", name, err)
			// Keep looping; backoff continues on the next iteration.
			// Swap in a sentinel so Wait() on the next loop iteration returns
			// immediately without hanging on a nil Process.
			continue
		}

		m.mu.Lock()
		if m.stopped {
			// Shutdown raced with restart; kill the just-started process.
			_ = newCmd.Process.Kill()
			m.mu.Unlock()
			return
		}
		// Update the entry's cmd in place. The cancel/context stay the same.
		e.cmd = newCmd
		m.mu.Unlock()

		log.Printf("port-forward %s: restarted on :%d", name, localPort)
		m.emitEvent("portforward:connected", name, localPort)

		// Reset backoff on a successful spawn.
		attempt = 0
	}
}

// start begins kubectl port-forward for the named service and launches a
// watcher goroutine that restarts it on unexpected exit.
func (m *portForwardManager) start(name, namespace string, localPort, clusterPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return fmt.Errorf("port-forward manager is stopped")
	}

	// If a healthy entry already exists, do nothing.
	if e, ok := m.entries[name]; ok {
		if e.cmd.Process != nil && e.cmd.ProcessState == nil {
			return fmt.Errorf("already forwarding %s", name)
		}
		// Stale entry from a previous stop; cancel its watcher before replacing.
		e.cancel()
		delete(m.entries, name)
		delete(m.ports, name)
	}

	cmd, err := spawnOne(name, namespace, localPort, clusterPort)
	if err != nil {
		return err
	}

	watchCtx, cancel := context.WithCancel(context.Background())
	e := &pfEntry{cmd: cmd, cancel: cancel}
	m.entries[name] = e
	m.ports[name] = localPort

	go m.watchAndRestart(watchCtx, name, namespace, localPort, clusterPort)
	return nil
}

// stop kills the port-forward for the named service and cancels its watcher.
func (m *portForwardManager) stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[name]
	if !ok {
		return nil
	}
	// Cancel the watcher goroutine first so it does not restart after the kill.
	e.cancel()
	if e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
	delete(m.entries, name)
	delete(m.ports, name)
	return nil
}

// stopAll kills every active port-forward and prevents any further restarts.
// Safe to call concurrently. Iterates over a snapshot to avoid holding the
// lock while killing processes, eliminating the risk of deadlock if a watcher
// tries to acquire the lock during shutdown.
func (m *portForwardManager) stopAll() {
	m.mu.Lock()
	m.stopped = true
	// Collect entries under the lock, then release before killing.
	snapshot := make(map[string]*pfEntry, len(m.entries))
	for k, v := range m.entries {
		snapshot[k] = v
	}
	m.entries = make(map[string]*pfEntry)
	m.ports = make(map[string]int)
	m.mu.Unlock()

	for name, e := range snapshot {
		e.cancel()
		if e.cmd.Process != nil {
			_ = e.cmd.Process.Kill()
		}
		log.Printf("port-forward %s: stopped", name)
	}
}

// kubectlTimeout is the default timeout for kubectl read queries.
const kubectlTimeout = 10 * time.Second

// queryPodReady returns whether the named deployment has Ready replicas.
func queryPodReady(namespace, name string) bool {
	// kubectl get deploy uncworks-aot-<name> -n <ns> -o json
	deployName := fmt.Sprintf("uncworks-aot-%s", name)
	ctx, cancel := context.WithTimeout(context.Background(), kubectlTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "get", "deploy", deployName,
		"-n", namespace, "-o", "json").Output()
	if err != nil {
		return false
	}
	var obj struct {
		Status struct {
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	}
	if err := json.Unmarshal(out, &obj); err != nil {
		return false
	}
	return obj.Status.ReadyReplicas > 0
}

// restartService runs kubectl rollout restart on the named deployment.
func restartService(namespace, name string) error {
	deployName := fmt.Sprintf("uncworks-aot-%s", name)
	ctx, cancel := context.WithTimeout(context.Background(), kubectlTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "rollout", "restart", "deploy", deployName,
		"-n", namespace).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart %s: %s", name, strings.TrimSpace(string(out)))
	}
	return nil
}
