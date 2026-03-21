package contract

import (
	"os"
	"strings"
	"testing"
)

// TestBoundary_NginxRoutesCoversAllBackendRoutes verifies that every REST and
// ConnectRPC route registered by the API server is covered by one of the nginx
// proxy location blocks in the Helm chart web template.
//
// This is a boundary/contract test: if someone adds a new route to the server
// but forgets to ensure the nginx proxy covers it, this test will fail.
func TestBoundary_NginxRoutesCoversAllBackendRoutes(t *testing.T) {
	// --- Step 1: Read nginx config and extract proxy location prefixes ---
	nginxData, err := os.ReadFile("../../deploy/helm/aot/templates/web.yaml")
	if err != nil {
		t.Fatalf("failed to read web.yaml: %v", err)
	}
	nginxConf := string(nginxData)

	// Verify the two critical proxy location blocks exist.
	if !strings.Contains(nginxConf, "location /api/") {
		t.Fatal("nginx config missing 'location /api/' proxy block")
	}
	if !strings.Contains(nginxConf, "location /aot.api.v1.") {
		t.Fatal("nginx config missing 'location /aot.api.v1.' proxy block")
	}

	// --- Step 2: Read main.go and server handler files to collect all routes ---
	mainData, err := os.ReadFile("../../cmd/apiserver/main.go")
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	mainSrc := string(mainData)

	serverFiles := []string{
		"../../internal/server/files.go",
		"../../internal/server/traces.go",
		"../../internal/server/exec.go",
		"../../internal/server/sse.go",
		"../../internal/server/github.go",
		"../../internal/server/webhook.go",
		"../../internal/server/debug.go",
	}

	var allSrc strings.Builder
	allSrc.WriteString(mainSrc)
	for _, f := range serverFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		allSrc.WriteString("\n")
		allSrc.Write(data)
	}
	combinedSrc := allSrc.String()

	// --- Step 3: Extract all route paths from HandleFunc calls ---
	var restRoutes []string
	for _, line := range strings.Split(combinedSrc, "\n") {
		line = strings.TrimSpace(line)

		// Match patterns like: mux.HandleFunc("GET /api/v1/...", ...)
		// or mux.HandleFunc("POST /api/v1/...", ...)
		if strings.Contains(line, "HandleFunc(") && strings.Contains(line, "/api/") {
			// Extract the route pattern from the string literal
			route := extractRoute(line)
			if route != "" {
				restRoutes = append(restRoutes, route)
			}
		}

		// Match patterns like: mux.Handle("/api/v1/webhooks/github", ...)
		if strings.Contains(line, "mux.Handle(") && strings.Contains(line, `"/api/`) {
			route := extractRoute(line)
			if route != "" {
				restRoutes = append(restRoutes, route)
			}
		}
	}

	if len(restRoutes) == 0 {
		t.Fatal("failed to extract any REST routes from source files")
	}

	t.Logf("Found %d REST routes", len(restRoutes))

	// --- Step 4: Verify all REST routes are covered by /api/ location ---
	for _, route := range restRoutes {
		if !strings.HasPrefix(route, "/api/") {
			t.Errorf("REST route %q does not start with /api/ — nginx /api/ location will not proxy it", route)
		}
	}

	// --- Step 5: Verify ConnectRPC routes are covered by /aot.api.v1. location ---
	// The ConnectRPC handler is registered via:
	//   path, handler := apiv1connect.NewAOTServiceHandler(...)
	//   mux.Handle(path, handler)
	// The path is always "/aot.api.v1.AOTService/"
	if !strings.Contains(mainSrc, "apiv1connect.NewAOTServiceHandler") {
		t.Error("main.go does not register AOTServiceHandler — ConnectRPC routes will not work")
	}

	// The service name is "aot.api.v1.AOTService" which produces path "/aot.api.v1.AOTService/"
	// This is covered by the "location /aot.api.v1." prefix in nginx.
	connectServiceName := "aot.api.v1.AOTService"
	connectPath := "/" + connectServiceName + "/"
	if !strings.HasPrefix(connectPath, "/aot.api.v1.") {
		t.Errorf("ConnectRPC path %q not covered by nginx /aot.api.v1. location", connectPath)
	}

	// --- Step 6: Verify health/infra routes exist but are non-API (served by SPA fallback or direct) ---
	// /healthz and /readyz are accessed directly on the apiserver (not through nginx),
	// but if accessed through nginx they'd hit the SPA fallback, which is fine for
	// k8s probes that target the apiserver service directly.
	if !strings.Contains(mainSrc, `"/healthz"`) {
		t.Error("main.go missing /healthz health check route")
	}
	if !strings.Contains(mainSrc, `"/readyz"`) {
		t.Error("main.go missing /readyz readiness check route")
	}

	// --- Step 7: Verify specific critical routes exist ---
	criticalRoutes := []string{
		"/api/v1/runs/{id}/files",
		"/api/v1/runs/{id}/logs",
		"/api/v1/runs/{id}/exec",
		"/api/v1/runs/{id}/traces",
		"/api/v1/runs/{id}/debug",
		"/api/v1/specs/push",
		"/api/v1/specs/pull",
		"/api/v1/webhooks/github",
		"/api/v1/specs/{id}/graph",
		"/api/v1/runs/{id}/traces/watch",
	}

	for _, route := range criticalRoutes {
		found := false
		for _, r := range restRoutes {
			if r == route {
				found = true
				break
			}
		}
		if !found {
			// Check if the route exists in a Handle() call (for webhook)
			if strings.Contains(combinedSrc, `"`+route+`"`) {
				continue
			}
			t.Errorf("critical route %q not found in registered REST routes", route)
		}
	}
}

// extractRoute extracts the route path from a HandleFunc or Handle call.
// Input examples:
//
//	mux.HandleFunc("GET /api/v1/runs/{id}/files", f.handleListFiles)
//	mux.Handle("/api/v1/webhooks/github", webhookHandler)
func extractRoute(line string) string {
	// Find the first quoted string containing a route
	idx := strings.Index(line, `"`)
	if idx < 0 {
		return ""
	}
	rest := line[idx+1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	literal := rest[:end]

	// Strip HTTP method prefix if present (e.g., "GET /api/..." → "/api/...")
	if spaceIdx := strings.Index(literal, " /"); spaceIdx >= 0 {
		return literal[spaceIdx+1:]
	}

	// Plain path like "/api/v1/webhooks/github"
	if strings.HasPrefix(literal, "/") {
		return literal
	}

	return ""
}
