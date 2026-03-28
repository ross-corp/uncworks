// status.go — uncworks status: show health of the UNCWORKS stack.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"text/tabwriter"
	"os"
)

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	namespace := fs.String("namespace", defaultNamespace, "Kubernetes namespace")
	context := fs.String("context", "", "Kubeconfig context to use")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks status [flags]\n\nShow health of the UNCWORKS stack.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
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
					Ready   bool   `json:"ready"`
					State   struct {
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
		fmt.Printf("No pods found in namespace %q. Is UNCWORKS installed? Run 'uncworks setup'.\n", *namespace)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "POD\tPHASE\tREADY")
	for _, pod := range podList.Items {
		ready := true
		reason := ""
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				ready = false
				if cs.State.Waiting.Reason != "" {
					reason = cs.State.Waiting.Reason
				}
			}
		}
		readyStr := "Yes"
		if !ready {
			readyStr = "No"
			if reason != "" {
				readyStr = "No (" + reason + ")"
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", pod.Metadata.Name, pod.Status.Phase, readyStr)
	}
	w.Flush()
	return nil
}
