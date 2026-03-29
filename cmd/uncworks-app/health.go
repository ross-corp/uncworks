//go:build darwin

// health.go — Structured health checks for UNCWORKS desktop app dependencies.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// HealthStatus represents the status of a single dependency.
type HealthStatus string

const (
	HealthOK       HealthStatus = "ok"
	HealthDegraded HealthStatus = "degraded"
	HealthDown     HealthStatus = "down"
	HealthUnknown  HealthStatus = "unknown"
)

// HealthComponent is one checked dependency.
type HealthComponent struct {
	Name    string       `json:"name"`
	Label   string       `json:"label"`
	Status  HealthStatus `json:"status"`
	Message string       `json:"message"`
}

// HealthReport is the full result returned to the frontend.
type HealthReport struct {
	Overall    HealthStatus      `json:"overall"`
	Components []HealthComponent `json:"components"`
}

// HealthCheck runs all dependency checks and returns a structured report.
func (a *App) HealthCheck() HealthReport {
	s, _ := loadAppSettings()
	ns := s.Namespace
	if ns == "" {
		ns = "uncworks"
	}

	litellmURL := s.LiteLLMURL
	if litellmURL == "" {
		litellmURL = "http://litellm:4000"
	}

	components := []HealthComponent{
		checkKubernetes(ns),
		checkDeploy(ns, "apiserver", "API Server"),
		checkDeploy(ns, "worker", "Worker"),
		checkDeploy(ns, "controller", "Controller"),
		checkDeploy(ns, "web", "Web UI"),
		checkAPIHTTP(a),
		checkLiteLLM(a, litellmURL),
	}

	overall := HealthOK
	for _, c := range components {
		if c.Status == HealthDown {
			overall = HealthDown
			break
		}
		if c.Status == HealthDegraded || c.Status == HealthUnknown {
			overall = HealthDegraded
		}
	}

	return HealthReport{
		Overall:    overall,
		Components: components,
	}
}

// checkKubernetes verifies the cluster is reachable and has Ready nodes.
func checkKubernetes(namespace string) HealthComponent {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "get", "nodes", "--no-headers").Output()
	if err != nil {
		return HealthComponent{
			Name:    "kubernetes",
			Label:   "Kubernetes",
			Status:  HealthDown,
			Message: "cluster unreachable — check kubeconfig",
		}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	ready := 0
	for _, l := range lines {
		if strings.Contains(l, " Ready") {
			ready++
		}
	}
	if ready == 0 {
		return HealthComponent{
			Name:    "kubernetes",
			Label:   "Kubernetes",
			Status:  HealthDegraded,
			Message: fmt.Sprintf("%d node(s) found, none Ready", len(lines)),
		}
	}
	return HealthComponent{
		Name:    "kubernetes",
		Label:   "Kubernetes",
		Status:  HealthOK,
		Message: fmt.Sprintf("%d node(s) Ready", ready),
	}
}

// checkDeploy checks that a named deployment has at least one Ready replica.
func checkDeploy(namespace, name, label string) HealthComponent {
	deployName := fmt.Sprintf("uncworks-aot-%s", name)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "kubectl", "get", "deploy", deployName,
		"-n", namespace, "-o", "json").Output()
	if err != nil {
		return HealthComponent{
			Name:    name,
			Label:   label,
			Status:  HealthDown,
			Message: "deployment not found",
		}
	}

	var obj struct {
		Status struct {
			Replicas      int `json:"replicas"`
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	}
	if err := json.Unmarshal(out, &obj); err != nil {
		return HealthComponent{Name: name, Label: label, Status: HealthUnknown, Message: "could not parse deployment"}
	}

	if obj.Status.ReadyReplicas == 0 {
		return HealthComponent{
			Name:    name,
			Label:   label,
			Status:  HealthDown,
			Message: fmt.Sprintf("0/%d replicas Ready", obj.Status.Replicas),
		}
	}
	if obj.Status.ReadyReplicas < obj.Status.Replicas {
		return HealthComponent{
			Name:    name,
			Label:   label,
			Status:  HealthDegraded,
			Message: fmt.Sprintf("%d/%d replicas Ready", obj.Status.ReadyReplicas, obj.Status.Replicas),
		}
	}
	return HealthComponent{
		Name:    name,
		Label:   label,
		Status:  HealthOK,
		Message: fmt.Sprintf("%d/%d replicas Ready", obj.Status.ReadyReplicas, obj.Status.Replicas),
	}
}

// checkLiteLLM probes the LiteLLM proxy's /v1/models endpoint.
func checkLiteLLM(a *App, url string) HealthComponent {
	result := a.CheckLiteLLM(url)
	if !result.OK {
		msg := result.Error
		if msg == "" {
			msg = "unreachable"
		}
		return HealthComponent{
			Name:    "litellm",
			Label:   "LiteLLM",
			Status:  HealthDown,
			Message: msg,
		}
	}
	return HealthComponent{
		Name:    "litellm",
		Label:   "LiteLLM",
		Status:  HealthOK,
		Message: fmt.Sprintf("%d model(s) available", len(result.Models)),
	}
}

// checkAPIHTTP probes the API server over any active port-forward.
func checkAPIHTTP(a *App) HealthComponent {
	fwd, port := a.pf.isForwarding("apiserver")
	if !fwd {
		return HealthComponent{
			Name:    "api-http",
			Label:   "API (HTTP)",
			Status:  HealthUnknown,
			Message: "not port-forwarded — forward apiserver to probe",
		}
	}

	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/api/v1/health", port)
	resp, err := client.Get(url)
	if err != nil {
		return HealthComponent{
			Name:    "api-http",
			Label:   "API (HTTP)",
			Status:  HealthDown,
			Message: fmt.Sprintf("unreachable at :%d", port),
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return HealthComponent{
			Name:    "api-http",
			Label:   "API (HTTP)",
			Status:  HealthOK,
			Message: fmt.Sprintf(":%d → %d", port, resp.StatusCode),
		}
	}
	return HealthComponent{
		Name:    "api-http",
		Label:   "API (HTTP)",
		Status:  HealthDegraded,
		Message: fmt.Sprintf(":%d → HTTP %d", port, resp.StatusCode),
	}
}
