//go:build darwin

// services.go — Service discovery and management for the desktop app.
// Wraps kubectl to list, restart, and port-forward in-cluster services.
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// ServiceInfo describes a manageable UNCWORKS in-cluster service.
type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	ClusterPort int    `json:"clusterPort"`
	LocalPort   int    `json:"localPort"`   // 0 = not forwarded
	Ready       bool   `json:"ready"`
	Forwarding  bool   `json:"forwarding"`
}

// knownServices lists the services we expose in the UI.
// ClusterPort is the container port exposed by the k8s Service.
var knownServices = []ServiceInfo{
	{Name: "apiserver", DisplayName: "API Server", ClusterPort: 50055},
	{Name: "web", DisplayName: "Web UI", ClusterPort: 3000},
	{Name: "worker", DisplayName: "Worker", ClusterPort: 0},
	{Name: "controller", DisplayName: "Controller", ClusterPort: 0},
}

// portForwardProcs tracks running kubectl port-forward subprocesses.
type portForwardManager struct {
	mu    sync.Mutex
	procs map[string]*exec.Cmd // keyed by service name
	ports map[string]int       // local port per service
}

func newPortForwardManager() *portForwardManager {
	return &portForwardManager{
		procs: make(map[string]*exec.Cmd),
		ports: make(map[string]int),
	}
}

func (m *portForwardManager) isForwarding(name string) (bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.procs[name]
	if !ok || cmd.ProcessState != nil {
		return false, 0
	}
	return true, m.ports[name]
}

func (m *portForwardManager) start(name, namespace string, localPort, clusterPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cmd, ok := m.procs[name]; ok && cmd.Process != nil && cmd.ProcessState == nil {
		return fmt.Errorf("already forwarding %s", name)
	}

	svcName := fmt.Sprintf("svc/uncworks-aot-%s", name)
	portArg := fmt.Sprintf("%d:%d", localPort, clusterPort)
	cmd := exec.Command("kubectl", "port-forward", "-n", namespace, svcName, portArg)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start port-forward %s: %w", name, err)
	}
	m.procs[name] = cmd
	m.ports[name] = localPort
	return nil
}

func (m *portForwardManager) stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.procs[name]
	if !ok {
		return nil
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	delete(m.procs, name)
	delete(m.ports, name)
	return nil
}

func (m *portForwardManager) stopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, cmd := range m.procs {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		delete(m.procs, name)
		delete(m.ports, name)
	}
}

// queryPodReady returns whether the named deployment has Ready replicas.
func queryPodReady(namespace, name string) bool {
	// kubectl get deploy uncworks-aot-<name> -n <ns> -o json
	deployName := fmt.Sprintf("uncworks-aot-%s", name)
	out, err := exec.Command("kubectl", "get", "deploy", deployName,
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
	out, err := exec.Command("kubectl", "rollout", "restart", "deploy", deployName,
		"-n", namespace).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart %s: %s", name, strings.TrimSpace(string(out)))
	}
	return nil
}
