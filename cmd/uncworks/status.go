// status.go — uncworks status: show health of the UNCWORKS stack.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"
)

// podStatus holds the summarised status of a single pod for output purposes.
type podStatus struct {
	Name   string `json:"name"`
	Phase  string `json:"phase"`
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	namespace := fs.String("namespace", defaultNamespace, "Kubernetes namespace")
	context := fs.String("context", "", "Kubeconfig context to use")
	output := fs.String("output", "", `Output format. One of: json`)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks status [flags]\n\nShow health of the UNCWORKS stack.\nExits non-zero if any pod is not ready.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *output != "" && *output != "json" {
		return fmt.Errorf("unsupported output format %q: use 'json'", *output)
	}
	if err := checkPrereqs(); err != nil {
		return err
	}

	kubectlArgs := []string{"get", "pods", "--namespace", *namespace, "-o", "json"}
	if *context != "" {
		kubectlArgs = append([]string{"--context", *context}, kubectlArgs...)
	}
	out, err := exec.Command("kubectl", kubectlArgs...).Output()
	if err != nil {
		return fmt.Errorf("kubectl get pods: %w", err)
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
					State struct {
						Waiting struct {
							Reason string `json:"reason"`
						} `json:"waiting"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &podList); err != nil {
		return fmt.Errorf("parse pods: %w", err)
	}

	if len(podList.Items) == 0 {
		msg := fmt.Sprintf("no pods found in namespace %q — is UNCWORKS installed? Run 'uncworks setup'", *namespace)
		if *output == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]string{"status": "not_installed", "message": msg})
		} else {
			fmt.Printf("No pods found in namespace %q. Is UNCWORKS installed? Run 'uncworks setup'.\n", *namespace)
		}
		return fmt.Errorf("%s", msg)
	}

	// Collect summarised statuses.
	var statuses []podStatus
	allReady := true
	for _, pod := range podList.Items {
		ps := podStatus{
			Name:  pod.Metadata.Name,
			Phase: pod.Status.Phase,
			Ready: true,
		}
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				ps.Ready = false
				allReady = false
				if cs.State.Waiting.Reason != "" {
					ps.Reason = cs.State.Waiting.Reason
				}
			}
		}
		statuses = append(statuses, ps)
	}

	if *output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(statuses)
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "POD\tPHASE\tREADY")
		for _, ps := range statuses {
			readyStr := "Yes"
			if !ps.Ready {
				readyStr = "No"
				if ps.Reason != "" {
					readyStr = "No (" + ps.Reason + ")"
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", ps.Name, ps.Phase, readyStr)
		}
		w.Flush()
	}

	if !allReady {
		return fmt.Errorf("one or more pods are not ready")
	}
	return nil
}
