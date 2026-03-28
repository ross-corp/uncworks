// setup.go — uncworks setup: deploy UNCWORKS into a local Kubernetes cluster.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

// setupConfig holds all values needed to install UNCWORKS.
type setupConfig struct {
	KubeContext   string
	Namespace     string
	LLMKey        string
	GitHubToken   string
	TemporalHost  string
	ChartRef      string
	ChartVersion  string
	ValuesFile    string
}

func runSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	kubeCtx := fs.String("context", "", "Kubeconfig context to use")
	namespace := fs.String("namespace", defaultNamespace, "Kubernetes namespace to install into")
	llmKey := fs.String("llm-key", "", "LLM API key (OpenRouter or OpenAI)")
	githubToken := fs.String("github-token", "", "GitHub personal access token")
	temporalHost := fs.String("temporal-host", "", "Temporal gRPC address (e.g. temporal:7233)")
	chartVersion := fs.String("version", "", "Chart version to install (default: latest)")
	valuesFile := fs.String("values", "", "Additional Helm values file")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks setup [flags]\n\nDeploy UNCWORKS into a local Kubernetes cluster.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := checkPrereqs(); err != nil {
		return err
	}

	cfg := &setupConfig{
		Namespace:    *namespace,
		ChartRef:     "oci://ghcr.io/uncworks/charts/aot",
		ChartVersion: *chartVersion,
		ValuesFile:   *valuesFile,
	}

	// Determine kubeconfig context.
	if *kubeCtx != "" {
		cfg.KubeContext = *kubeCtx
	} else {
		ctx, err := selectContext()
		if err != nil {
			return err
		}
		cfg.KubeContext = ctx
	}

	// Collect required values (interactive if not provided via flags).
	if err := collectValues(cfg, *llmKey, *githubToken, *temporalHost); err != nil {
		return err
	}

	// Preflight: check cluster resources.
	if err := resourcePreflight(cfg.KubeContext); err != nil {
		return err
	}

	// Deploy via Helm.
	if err := helmInstall(cfg); err != nil {
		return err
	}

	// Post-install output.
	printPostInstall(cfg)
	return nil
}

// selectContext picks the active context or prompts the user to choose.
func selectContext() (string, error) {
	contexts, err := ListContexts()
	if err != nil || len(contexts) == 0 {
		printNoClusterInstructions()
		return "", fmt.Errorf("no Kubernetes contexts found")
	}

	// Sort: active first, then alphabetical.
	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].Active != contexts[j].Active {
			return contexts[i].Active
		}
		return contexts[i].Name < contexts[j].Name
	})

	if len(contexts) == 1 {
		fmt.Printf("Using Kubernetes context: %s (%s)\n", contexts[0].Name, contexts[0].ServerURL)
		return contexts[0].Name, nil
	}

	fmt.Println("Available Kubernetes contexts:")
	for i, ctx := range contexts {
		active := "  "
		if ctx.Active {
			active = "* "
		}
		fmt.Printf("  %s[%d] %s  (%s)\n", active, i+1, ctx.Name, ctx.ServerURL)
	}
	fmt.Print("Select context [1]: ")
	line := readLine()
	if line == "" {
		// Default to the active context (index 0 after sort).
		return contexts[0].Name, nil
	}
	idx := 0
	_, err = fmt.Sscanf(line, "%d", &idx)
	if err != nil || idx < 1 || idx > len(contexts) {
		return "", fmt.Errorf("invalid selection %q", line)
	}
	return contexts[idx-1].Name, nil
}

// collectValues prompts for any required values not supplied via flags.
func collectValues(cfg *setupConfig, llmKey, githubToken, temporalHost string) error {
	cfg.LLMKey = llmKey
	cfg.GitHubToken = githubToken
	cfg.TemporalHost = temporalHost

	if cfg.LLMKey == "" {
		cfg.LLMKey = readSecret("LLM API key (OpenRouter/OpenAI): ")
	}
	if cfg.GitHubToken == "" {
		cfg.GitHubToken = readSecret("GitHub personal access token: ")
	}
	if cfg.TemporalHost == "" {
		cfg.TemporalHost = readPrompt("Temporal gRPC address", "temporal:7233")
	}
	return nil
}

// resourcePreflight checks the cluster has enough allocatable resources.
func resourcePreflight(kubeCtx string) error {
	fmt.Print("Checking cluster resources... ")
	res, err := nodeAllocatable(kubeCtx)
	if err != nil {
		fmt.Println("skipped (could not query nodes)")
		return nil
	}

	minCPU := int64(2000)   // 2 vCPUs
	minMem := int64(2 * 1024 * 1024 * 1024) // 2Gi
	recCPU := int64(4000)   // 4 vCPUs
	recMem := int64(4 * 1024 * 1024 * 1024) // 4Gi

	if res.CPUMillicores < minCPU || res.MemoryBytes < minMem {
		return fmt.Errorf("insufficient cluster resources: have %dm CPU / %dMi memory, need at least 2 CPU / 2Gi",
			res.CPUMillicores, res.MemoryBytes/1024/1024)
	}

	if res.CPUMillicores < recCPU || res.MemoryBytes < recMem {
		fmt.Printf("warning: below recommended resources (%dm CPU / %dMi memory). Install may be slow.\n",
			res.CPUMillicores, res.MemoryBytes/1024/1024)
		fmt.Print("Continue anyway? [Y/n]: ")
		if line := readLine(); strings.ToLower(line) == "n" {
			return fmt.Errorf("setup cancelled")
		}
	} else {
		fmt.Printf("OK (%dm CPU / %dMi)\n", res.CPUMillicores, res.MemoryBytes/1024/1024)
	}
	return nil
}

// helmInstall runs helm upgrade --install with collected values.
func helmInstall(cfg *setupConfig) error {
	helmArgs := []string{
		"upgrade", "--install", defaultReleaseName, cfg.ChartRef,
		"--namespace", cfg.Namespace,
		"--create-namespace",
		"--set", fmt.Sprintf("temporal.host=%s", cfg.TemporalHost),
	}

	if cfg.ChartVersion != "" {
		helmArgs = append(helmArgs, "--version", cfg.ChartVersion)
	}
	if cfg.KubeContext != "" {
		helmArgs = append(helmArgs, "--kube-context", cfg.KubeContext)
	}
	if cfg.ValuesFile != "" {
		helmArgs = append(helmArgs, "-f", cfg.ValuesFile)
	}

	// Write secrets to a temp file to avoid them appearing in process list.
	tmpValues, err := writeTempValues(cfg)
	if err != nil {
		return err
	}
	defer os.Remove(tmpValues)
	helmArgs = append(helmArgs, "-f", tmpValues)

	fmt.Printf("Installing UNCWORKS (release: %s, namespace: %s)...\n", defaultReleaseName, cfg.Namespace)
	cmd := exec.Command("helm", helmArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm upgrade --install: %w", err)
	}
	return nil
}

// writeTempValues writes secrets to a temp values YAML file.
func writeTempValues(cfg *setupConfig) (string, error) {
	f, err := os.CreateTemp("", "uncworks-values-*.yaml")
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	if cfg.LLMKey != "" {
		sb.WriteString(fmt.Sprintf("llm:\n  apiKey: %q\n", cfg.LLMKey))
	}
	if cfg.GitHubToken != "" {
		// GitHub token goes into a k8s Secret; we pass it as a values override
		// that the helm chart uses to create the secret.
		sb.WriteString(fmt.Sprintf("github:\n  token: %q\n", cfg.GitHubToken))
	}
	if _, err := f.WriteString(sb.String()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// printPostInstall shows the web UI URL and next steps.
func printPostInstall(cfg *setupConfig) {
	fmt.Printf("\nUNCWORKS installed successfully!\n\n")
	fmt.Printf("  Web UI:  http://localhost:%d  (run 'uncworks open')\n", defaultWebLocalPort)
	fmt.Printf("  TUI:     uncworks tui\n")
	fmt.Printf("  Status:  uncworks status\n\n")
	fmt.Printf("Run 'uncworks open' to start port-forward and open the web UI.\n")
}

// printNoClusterInstructions prints platform-specific cluster install recommendations.
func printNoClusterInstructions() {
	fmt.Println("No Kubernetes contexts found. You need a local Kubernetes cluster to use UNCWORKS.\n")
	if runtime.GOOS == "darwin" {
		fmt.Println("Recommended options for macOS:")
		fmt.Println("  Docker Desktop  → Enable Kubernetes in Preferences > Kubernetes")
		fmt.Println("  OrbStack        → brew install orbstack  (fastest option)")
		fmt.Println("  Rancher Desktop → brew install --cask rancher\n")
	} else {
		fmt.Println("Recommended options for Linux:")
		fmt.Println("  k3d  → curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash")
		fmt.Println("  kind → go install sigs.k8s.io/kind@latest\n")
	}
	fmt.Println("After installing, run 'uncworks setup' again.")
}

// readLine reads a trimmed line from stdin.
func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

// readPrompt reads a value with a default.
func readPrompt(label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	line := readLine()
	if line == "" {
		return defaultVal
	}
	return line
}

// readSecret reads a value with masked input (no terminal echo).
// Falls back to visible input if terminal manipulation fails.
func readSecret(label string) string {
	fmt.Print(label)
	// Try to disable echo via stty.
	sttyOff := exec.Command("stty", "-echo")
	sttyOff.Stdin = os.Stdin
	_ = sttyOff.Run()

	val := readLine()

	sttyOn := exec.Command("stty", "echo")
	sttyOn.Stdin = os.Stdin
	_ = sttyOn.Run()
	fmt.Println() // newline after masked input
	return val
}
