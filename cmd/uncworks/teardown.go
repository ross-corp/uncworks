// teardown.go — uncworks teardown: uninstall UNCWORKS from the current cluster.
package main

import (
	"flag"
	"fmt"
	"os/exec"
)

const defaultReleaseName = "uncworks"
const defaultNamespace = "uncworks"

func runTeardown(args []string) error {
	fs := flag.NewFlagSet("teardown", flag.ContinueOnError)
	purge := fs.Bool("purge", false, "Also delete PVCs (destroys all workspace data)")
	namespace := fs.String("namespace", defaultNamespace, "Kubernetes namespace")
	context := fs.String("context", "", "Kubeconfig context to use")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks teardown [flags]\n\nUninstall UNCWORKS from the current cluster.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := checkPrereqs(); err != nil {
		return err
	}

	helmArgs := []string{"uninstall", defaultReleaseName, "--namespace", *namespace}
	if *context != "" {
		helmArgs = append(helmArgs, "--kube-context", *context)
	}
	fmt.Printf("Uninstalling release %q from namespace %q...\n", defaultReleaseName, *namespace)
	cmd := exec.Command("helm", helmArgs...)
	cmd.Stdout = nil
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("helm uninstall: %w\n%s", err, out)
	}
	fmt.Println("UNCWORKS uninstalled.")

	if *purge {
		fmt.Printf("Deleting PVCs in namespace %q...\n", *namespace)
		kubectlArgs := []string{"delete", "pvc", "--all", "--namespace", *namespace}
		if *context != "" {
			kubectlArgs = append([]string{"--context", *context}, kubectlArgs...)
		}
		cmd := exec.Command("kubectl", kubectlArgs...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("kubectl delete pvc: %w\n%s", err, out)
		}
		fmt.Println("PVCs deleted.")
	} else {
		fmt.Println("PVCs retained. Use --purge to delete them.")
	}
	return nil
}
