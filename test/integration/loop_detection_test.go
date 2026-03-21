package integration

import (
	"fmt"
	"testing"

	"github.com/uncworks/aot/internal/sidecar"
)

// makeToolCallEvent builds a message_end JSONL line containing a tool_use block.
func makeToolCallEvent(toolName, inputJSON string) string {
	return fmt.Sprintf(
		`{"type":"message_end","message":{"content":[{"type":"tool_use","name":%q,"input":%s}]}}`,
		toolName, inputJSON,
	)
}

func TestExtractToolCallSignature_IdenticalEvents(t *testing.T) {
	// Five identical tool_use events should all produce the same signature.
	input := `{"command":"echo hello"}`
	line := makeToolCallEvent("Bash", input)

	var sigs []string
	for i := 0; i < 5; i++ {
		sig := sidecar.ExtractToolCallSignature(line)
		if sig == "" {
			t.Fatalf("iteration %d: expected non-empty signature", i)
		}
		sigs = append(sigs, sig)
	}

	for i := 1; i < len(sigs); i++ {
		if sigs[i] != sigs[0] {
			t.Errorf("signature mismatch: sigs[0]=%q, sigs[%d]=%q", sigs[0], i, sigs[i])
		}
	}
}

func TestExtractToolCallSignature_DifferentEvents(t *testing.T) {
	line1 := makeToolCallEvent("Bash", `{"command":"echo hello"}`)
	line2 := makeToolCallEvent("Read", `{"file_path":"/foo/bar.go"}`)
	line3 := makeToolCallEvent("Bash", `{"command":"ls -la"}`)

	sig1 := sidecar.ExtractToolCallSignature(line1)
	sig2 := sidecar.ExtractToolCallSignature(line2)
	sig3 := sidecar.ExtractToolCallSignature(line3)

	if sig1 == "" || sig2 == "" || sig3 == "" {
		t.Fatalf("expected non-empty signatures: sig1=%q sig2=%q sig3=%q", sig1, sig2, sig3)
	}

	// Different tool names produce different signatures.
	if sig1 == sig2 {
		t.Errorf("Bash and Read should have different signatures: %q == %q", sig1, sig2)
	}

	// Same tool but different input lengths produce different signatures.
	if sig1 == sig3 {
		t.Errorf("Bash with different inputs should differ: %q == %q", sig1, sig3)
	}
}

func TestExtractToolCallSignature_NonToolEvents(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"message_start", `{"type":"message_start"}`},
		{"message_update", `{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"hi"}}`},
		{"session", `{"type":"session","timestamp":"2025-01-01T00:00:00Z"}`},
		{"empty", ``},
		{"invalid json", `not json at all`},
		// message_end without tool_use content
		{"message_end_text_only", `{"type":"message_end","message":{"content":[{"type":"text","text":"hello"}]}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := sidecar.ExtractToolCallSignature(tt.line)
			if sig != "" {
				t.Errorf("expected empty signature for %q, got %q", tt.name, sig)
			}
		})
	}
}

func TestMaxRepeatedToolCalls(t *testing.T) {
	if sidecar.MaxRepeatedToolCalls != 5 {
		t.Errorf("MaxRepeatedToolCalls = %d, want 5", sidecar.MaxRepeatedToolCalls)
	}
}
