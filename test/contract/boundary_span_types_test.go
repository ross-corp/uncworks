package contract

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// validSpanTypes is the canonical set of span types that the frontend's
// TraceSpan["type"] union accepts (from web/src/types/agent-run.ts):
//
//	type: "llm" | "tool" | "thought" | "input" | "delegate" | "lifecycle"
var validSpanTypes = map[string]bool{
	"llm":       true,
	"tool":      true,
	"thought":   true,
	"input":     true,
	"delegate":  true,
	"lifecycle": true,
}

// TestBoundary_SpanTypes_GatewayUsesValidTypes reads the sidecar gateway.go
// source and verifies that every Type: "..." assignment in TraceSpan creation
// uses a type from the frontend's valid type union.
func TestBoundary_SpanTypes_GatewayUsesValidTypes(t *testing.T) {
	gatewayPath := findProjectRoot(t, "internal/sidecar/gateway.go")

	data, err := os.ReadFile(gatewayPath)
	if err != nil {
		t.Fatalf("failed to read gateway.go: %v", err)
	}

	// Match Type: "..." patterns in Go source.
	// This captures the type string from TraceSpan struct literal assignments.
	re := regexp.MustCompile(`Type:\s*"([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		t.Fatal("no Type: assignments found in gateway.go — regex may be wrong or source changed")
	}

	seen := make(map[string]bool)
	for _, m := range matches {
		typeStr := m[1]
		seen[typeStr] = true
		if !validSpanTypes[typeStr] {
			t.Errorf("gateway.go uses span type %q which is NOT in the frontend's valid type union %v",
				typeStr, sortedKeys(validSpanTypes))
		}
	}

	t.Logf("Found %d Type: assignments using %d distinct types: %v", len(matches), len(seen), sortedKeys(seen))

	// Also verify we found all the types the gateway actually uses.
	// Known types from source inspection: "llm", "tool", "thought", "input"
	expectedGatewayTypes := []string{"llm", "tool", "thought", "input"}
	for _, exp := range expectedGatewayTypes {
		if !seen[exp] {
			t.Errorf("expected gateway.go to use span type %q but it was not found", exp)
		}
	}
}

// TestBoundary_SpanTypes_FrontendTypesExhaustive verifies the frontend type
// union file contains all the types we expect.
func TestBoundary_SpanTypes_FrontendTypesExhaustive(t *testing.T) {
	tsPath := findProjectRoot(t, "web/src/types/agent-run.ts")

	data, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("failed to read agent-run.ts: %v", err)
	}

	content := string(data)

	// Look for the type union in the TraceSpan interface.
	// The line looks like: type: "llm" | "tool" | "thought" | "input" | "delegate" | "lifecycle";
	// We need to find the line with quoted strings separated by |, NOT the one that says `type: string;`.
	typeLineRe := regexp.MustCompile(`type:\s*("[^"]+"\s*\|[^;]+);`)
	typeLineMatch := typeLineRe.FindStringSubmatch(content)
	if len(typeLineMatch) < 2 {
		t.Fatal("could not find type union line (type: \"...\" | \"...\";) in agent-run.ts")
	}

	typeLine := typeLineMatch[1]
	typeValueRe := regexp.MustCompile(`"([^"]+)"`)
	typeValues := typeValueRe.FindAllStringSubmatch(typeLine, -1)

	frontendTypes := make(map[string]bool)
	for _, tv := range typeValues {
		frontendTypes[tv[1]] = true
	}

	// Verify our canonical set matches what the frontend defines
	for k := range validSpanTypes {
		if !frontendTypes[k] {
			t.Errorf("canonical validSpanTypes includes %q but frontend does not define it", k)
		}
	}
	for k := range frontendTypes {
		if !validSpanTypes[k] {
			t.Errorf("frontend defines span type %q but it is missing from validSpanTypes", k)
		}
	}
}

// findProjectRoot locates a file relative to the project root by walking up
// from the test file's location.
func findProjectRoot(t *testing.T, relPath string) string {
	t.Helper()

	// Get the directory of the current test file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file location")
	}

	dir := filepath.Dir(filename)
	// Walk up to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.mod)")
		}
		dir = parent
	}

	fullPath := filepath.Join(dir, relPath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("file not found: %s", fullPath)
	}
	return fullPath
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if strings.Compare(keys[i], keys[j]) > 0 {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
