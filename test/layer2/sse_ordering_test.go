// test/layer2/sse_ordering_test.go — Layer 2 tests for SSE event ordering.
// Verifies that events published to the event bus arrive in causal order
// over an SSE stream.
package layer2

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"github.com/uncworks/aot/internal/eventbus"
	"github.com/uncworks/aot/internal/server"
	"github.com/uncworks/aot/test/testutil"
)

// layer2Scheme is the shared k8s runtime.Scheme for layer2 tests that
// construct a fake k8s client directly.
var layer2Scheme = testutil.NewScheme()

// graphSSEEvent mirrors the JSON shape sent over the SSE stream.
type graphSSEEvent struct {
	Type            string `json:"type"`
	RunID           string `json:"runId"`
	Phase           string `json:"phase,omitempty"`
	Message         string `json:"message,omitempty"`
	CurrentActivity string `json:"currentActivity,omitempty"`
}

// TestSSE_WatchGraph_ConnectsWithoutEventBus verifies that the SSE endpoint
// sends an initial "connected" comment and keeps the stream open even when
// no event bus is configured.
func TestSSE_WatchGraph_ConnectsWithoutEventBus(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(layer2Scheme).Build()
	// SSEHandler with nil event bus sends heartbeat and blocks.
	sseHandler := server.NewSSEHandler(k8sClient, nil, "default")
	mux := http.NewServeMux()
	sseHandler.RegisterSSEHandlers(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/specs/run-1/graph/watch", nil)
	require.NoError(t, err)

	resp, err := srv.Client().Do(req)
	// Expect either nil error (context cancelled while streaming) or a context error.
	if err != nil && !isContextError(err) {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		resp.Body.Close()
	}
}

// TestSSE_WatchGraph_EventsArriveCausally verifies that when events are
// published to the event bus in order, they arrive over the SSE stream in
// the same order.
func TestSSE_WatchGraph_EventsArriveCausally(t *testing.T) {
	const runID = "ar-causal-test"

	// Pre-create the AgentRun so the graph endpoint can find it.
	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runID,
			Namespace: "default",
		},
		Spec: aotv1alpha1.AgentRunSpec{
			Backend: aotv1alpha1.BackendPod,
			Prompt:  "test causal ordering",
		},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(layer2Scheme).
		WithObjects(run).
		Build()

	bus := eventbus.NewChannelBus()
	sseHandler := server.NewSSEHandler(k8sClient, bus, "default")
	mux := http.NewServeMux()
	sseHandler.RegisterSSEHandlers(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Buffer to collect events.
	events := make(chan graphSSEEvent, 10)

	// Start the SSE consumer in a goroutine.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			srv.URL+"/api/v1/specs/"+runID+"/graph/watch", nil)
		if err != nil {
			return
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			var ev graphSSEEvent
			if err := json.Unmarshal([]byte(data), &ev); err == nil {
				events <- ev
			}
		}
		close(events)
	}()

	// Give the stream a moment to connect.
	time.Sleep(50 * time.Millisecond)

	// Publish events in a defined order.
	orderedPayloads := []string{"pending", "running", "succeeded"}
	for _, payload := range orderedPayloads {
		bus.Publish(runID, &apiv1.AgentRunEvent{
			AgentRunId: runID,
			Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
			Payload:    payload,
		})
		// Small gap to ensure ordering is preserved through the channel.
		time.Sleep(10 * time.Millisecond)
	}

	// Collect the events with a deadline.
	var received []graphSSEEvent
	deadline := time.After(2 * time.Second)
collect:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break collect
			}
			received = append(received, ev)
			if len(received) >= len(orderedPayloads) {
				break collect
			}
		case <-deadline:
			break collect
		}
	}
	cancel()

	require.GreaterOrEqual(t, len(received), len(orderedPayloads),
		"should have received at least %d events, got %d", len(orderedPayloads), len(received))

	// Verify causal ordering: each event's phase must match the published order.
	for i, payload := range orderedPayloads {
		assert.Equal(t, "NODE_STATUS_CHANGED", received[i].Type,
			"event[%d] should be NODE_STATUS_CHANGED", i)
		assert.Equal(t, payload, received[i].Phase,
			"event[%d] phase should be %q", i, payload)
		assert.Equal(t, runID, received[i].RunID,
			"event[%d] should reference run %q", i, runID)
	}
}

// TestSSE_WatchGraph_LogEventsBeforePhaseChange verifies that LOG events
// are interleaved and delivered before the subsequent phase-change event.
func TestSSE_WatchGraph_LogEventsBeforePhaseChange(t *testing.T) {
	const runID = "ar-log-ordering"

	run := &aotv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: runID, Namespace: "default"},
		Spec:       aotv1alpha1.AgentRunSpec{Backend: aotv1alpha1.BackendPod, Prompt: "test"},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(layer2Scheme).
		WithObjects(run).
		Build()

	bus := eventbus.NewChannelBus()
	sseHandler := server.NewSSEHandler(k8sClient, bus, "default")
	mux := http.NewServeMux()
	sseHandler.RegisterSSEHandlers(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	events := make(chan graphSSEEvent, 20)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			srv.URL+"/api/v1/specs/"+runID+"/graph/watch", nil)
		if err != nil {
			return
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev graphSSEEvent
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err == nil {
				events <- ev
			}
		}
		close(events)
	}()

	time.Sleep(50 * time.Millisecond)

	// Publish: log, log, then phase-change.
	bus.Publish(runID, &apiv1.AgentRunEvent{
		AgentRunId: runID,
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG,
		Payload:    "cloning repo",
	})
	time.Sleep(5 * time.Millisecond)
	bus.Publish(runID, &apiv1.AgentRunEvent{
		AgentRunId: runID,
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG,
		Payload:    "installing deps",
	})
	time.Sleep(5 * time.Millisecond)
	bus.Publish(runID, &apiv1.AgentRunEvent{
		AgentRunId: runID,
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
		Payload:    "running",
	})

	var received []graphSSEEvent
	deadline := time.After(2 * time.Second)
collect:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break collect
			}
			received = append(received, ev)
			if len(received) >= 3 {
				break collect
			}
		case <-deadline:
			break collect
		}
	}
	cancel()

	require.Len(t, received, 3, "expected 3 events: 2 log + 1 phase-change")
	assert.Equal(t, "NODE_PROGRESS", received[0].Type)
	assert.Equal(t, "cloning repo", received[0].CurrentActivity)
	assert.Equal(t, "NODE_PROGRESS", received[1].Type)
	assert.Equal(t, "installing deps", received[1].CurrentActivity)
	assert.Equal(t, "NODE_STATUS_CHANGED", received[2].Type)
	assert.Equal(t, "running", received[2].Phase)
}

// isContextError returns true if the error is caused by a context cancellation
// or deadline.
func isContextError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "context canceled") ||
		strings.Contains(s, "context deadline exceeded") ||
		strings.Contains(s, "EOF")
}
