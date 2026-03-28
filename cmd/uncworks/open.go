// open.go — uncworks open: port-forward the web UI and open it in a browser.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const defaultWebLocalPort = 30300
const webServiceName = "svc/uncworks-web"

func runOpen(args []string) error {
	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	namespace := fs.String("namespace", defaultNamespace, "Kubernetes namespace")
	context := fs.String("context", "", "Kubeconfig context to use")
	port := fs.Int("port", defaultWebLocalPort, "Local port to forward to")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks open [flags]\n\nStart port-forward for the web UI and open it in your browser.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := checkPrereqs(); err != nil {
		return err
	}

	// Kill any stale port-forward from a previous run.
	if err := killStalePF(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not kill stale port-forward: %v\n", err)
	}

	pfArgs := []string{
		"port-forward", "--namespace", *namespace,
		webServiceName,
		fmt.Sprintf("%d:3000", *port),
	}
	if *context != "" {
		pfArgs = append([]string{"--context", *context}, pfArgs...)
	}

	cmd := exec.Command("kubectl", pfArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start port-forward: %w", err)
	}

	// Persist PID for cleanup on next run.
	if err := writePIDFile(cmd.Process.Pid); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write PID file: %v\n", err)
	}

	url := fmt.Sprintf("http://localhost:%d", *port)
	fmt.Printf("Web UI available at %s\n", url)
	if err := openBrowser(url); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open browser: %v\n", err)
		fmt.Printf("Open %s in your browser.\n", url)
	}

	// Block until the port-forward exits.
	_ = cmd.Wait()
	return nil
}

func killStalePF() error {
	path, err := pidFilePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	_ = os.Remove(path)
	return nil
}

func writePIDFile(pid int) error {
	path, err := pidFilePath()
	if err != nil {
		return err
	}
	dir := strings.TrimSuffix(path, "/port-forward.pid")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o600)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}
