// kube.go — kubeconfig context enumeration and cluster resource checks.
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

// KubeContext represents a kubeconfig context with its server URL.
type KubeContext struct {
	Name      string
	ServerURL string
	Active    bool
}

// ListContexts parses ~/.kube/config and returns all contexts with server URLs.
func ListContexts() ([]KubeContext, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	var contexts []KubeContext
	for name, ctx := range cfg.Contexts {
		serverURL := ""
		if cluster, ok := cfg.Clusters[ctx.Cluster]; ok {
			serverURL = cluster.Server
		}
		contexts = append(contexts, KubeContext{
			Name:      name,
			ServerURL: serverURL,
			Active:    name == cfg.CurrentContext,
		})
	}
	return contexts, nil
}

// ActiveContext returns the active kubeconfig context, or an error if none.
func ActiveContext() (KubeContext, error) {
	contexts, err := ListContexts()
	if err != nil {
		return KubeContext{}, err
	}
	for _, ctx := range contexts {
		if ctx.Active {
			return ctx, nil
		}
	}
	return KubeContext{}, fmt.Errorf("no active kubeconfig context found")
}

// clusterResources holds allocatable CPU (millicores) and memory (bytes) for a cluster.
type clusterResources struct {
	CPUMillicores int64
	MemoryBytes   int64
}

// nodeAllocatable fetches the total allocatable CPU and memory across all nodes.
func nodeAllocatable(kubeContext string) (clusterResources, error) {
	args := []string{"get", "nodes", "-o", "json"}
	if kubeContext != "" {
		args = append([]string{"--context", kubeContext}, args...)
	}
	out, err := exec.Command("kubectl", args...).Output()
	if err != nil {
		return clusterResources{}, fmt.Errorf("kubectl get nodes: %w", err)
	}

	var result struct {
		Items []struct {
			Status struct {
				Allocatable map[string]string `json:"allocatable"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return clusterResources{}, fmt.Errorf("parse nodes: %w", err)
	}

	var res clusterResources
	for _, node := range result.Items {
		if cpu, ok := node.Status.Allocatable["cpu"]; ok {
			res.CPUMillicores += parseCPUMillicores(cpu)
		}
		if mem, ok := node.Status.Allocatable["memory"]; ok {
			res.MemoryBytes += parseMemoryBytes(mem)
		}
	}
	return res, nil
}

// parseCPUMillicores parses k8s CPU quantity strings like "4", "2000m".
func parseCPUMillicores(s string) int64 {
	if strings.HasSuffix(s, "m") {
		v, _ := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		return v
	}
	v, _ := strconv.ParseFloat(s, 64)
	return int64(v * 1000)
}

// parseMemoryBytes parses k8s memory quantity strings like "8Gi", "4096Mi", "4096000Ki".
func parseMemoryBytes(s string) int64 {
	units := []struct {
		suffix     string
		multiplier int64
	}{
		{"Ki", 1024},
		{"Mi", 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"K", 1000},
		{"M", 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
	}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			v, _ := strconv.ParseInt(strings.TrimSuffix(s, u.suffix), 10, 64)
			return v * u.multiplier
		}
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
