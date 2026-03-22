package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func TestAgentRunEventToGraphSSE_PhaseChanged(t *testing.T) {
	event := &apiv1.AgentRunEvent{
		AgentRunId: "ar-test-123",
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
		Payload:    "running",
	}

	sse := agentRunEventToGraphSSE(event)

	assert.Equal(t, "NODE_STATUS_CHANGED", sse.Type)
	assert.Equal(t, "ar-test-123", sse.RunID)
	assert.Equal(t, "running", sse.Phase)
	assert.Equal(t, "running", sse.Message)
}

func TestAgentRunEventToGraphSSE_Log(t *testing.T) {
	event := &apiv1.AgentRunEvent{
		AgentRunId: "ar-test-456",
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG,
		Payload:    "Compiling files...",
	}

	sse := agentRunEventToGraphSSE(event)

	assert.Equal(t, "NODE_PROGRESS", sse.Type)
	assert.Equal(t, "ar-test-456", sse.RunID)
	assert.Equal(t, "Compiling files...", sse.CurrentActivity)
}

func TestAgentRunEventToGraphSSE_Completed(t *testing.T) {
	event := &apiv1.AgentRunEvent{
		AgentRunId: "ar-test-789",
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_COMPLETED,
		Payload:    "all tasks done",
	}

	sse := agentRunEventToGraphSSE(event)

	assert.Equal(t, "NODE_STATUS_CHANGED", sse.Type)
	assert.Equal(t, "ar-test-789", sse.RunID)
	assert.Equal(t, "succeeded", sse.Phase)
	assert.Equal(t, "all tasks done", sse.Message)
}

func TestAgentRunEventToTraceSSE_PhaseChanged(t *testing.T) {
	event := &apiv1.AgentRunEvent{
		AgentRunId: "ar-run-1",
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED,
		Payload:    "running",
	}

	span := agentRunEventToTraceSSE(event)

	require.NotNil(t, span, "phase changed events should produce a trace span")
	assert.Equal(t, "ar-run-1-phase-running", span.ID)
	assert.Equal(t, "Phase: running", span.Name)
	assert.Equal(t, "phase", span.Type)
	assert.NotEmpty(t, span.StartTime)
	assert.NotEmpty(t, span.EndTime)
}

func TestAgentRunEventToTraceSSE_Log(t *testing.T) {
	event := &apiv1.AgentRunEvent{
		AgentRunId: "ar-run-2",
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG,
		Payload:    "Building project...",
	}

	span := agentRunEventToTraceSSE(event)

	require.NotNil(t, span, "log events should produce a trace span")
	assert.Equal(t, "Building project...", span.Name)
	assert.Equal(t, "log", span.Type)
}

func TestAgentRunEventToTraceSSE_UnknownReturnsNil(t *testing.T) {
	event := &apiv1.AgentRunEvent{
		AgentRunId: "ar-run-3",
		Type:       apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_UNSPECIFIED,
		Payload:    "something",
	}

	span := agentRunEventToTraceSSE(event)

	assert.Nil(t, span, "unspecified event type should return nil")
}

func TestPhaseToString(t *testing.T) {
	tests := []struct {
		phase apiv1.AgentRunPhase
		want  string
	}{
		{apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING, "pending"},
		{apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING, "running"},
		{apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT, "waiting_for_input"},
		{apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED, "succeeded"},
		{apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED, "failed"},
		{apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED, "cancelled"},
		{apiv1.AgentRunPhase(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := phaseToString(tt.phase)
			assert.Equal(t, tt.want, got)
		})
	}
}
