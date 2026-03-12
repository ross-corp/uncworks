package eventbus

import (
	"testing"
	"time"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeEvent(runID string, eventType apiv1.AgentRunEventType) *apiv1.AgentRunEvent {
	return &apiv1.AgentRunEvent{
		AgentRunId: runID,
		Type:       eventType,
		Payload:    "test",
		Timestamp:  timestamppb.Now(),
	}
}

func TestSingleSubscriber(t *testing.T) {
	bus := NewChannelBus()
	ch, id := bus.Subscribe("run-1")
	defer bus.Unsubscribe("run-1", id)

	event := makeEvent("run-1", apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED)
	bus.Publish("run-1", event)

	select {
	case got := <-ch:
		if got.AgentRunId != "run-1" {
			t.Errorf("expected run-1, got %s", got.AgentRunId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewChannelBus()
	ch1, id1 := bus.Subscribe("run-1")
	defer bus.Unsubscribe("run-1", id1)
	ch2, id2 := bus.Subscribe("run-1")
	defer bus.Unsubscribe("run-1", id2)

	event := makeEvent("run-1", apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG)
	bus.Publish("run-1", event)

	for i, ch := range []<-chan *apiv1.AgentRunEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.AgentRunId != "run-1" {
				t.Errorf("subscriber %d: expected run-1, got %s", i, got.AgentRunId)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestCrossRunIsolation(t *testing.T) {
	bus := NewChannelBus()
	ch1, id1 := bus.Subscribe("run-1")
	defer bus.Unsubscribe("run-1", id1)
	ch2, id2 := bus.Subscribe("run-2")
	defer bus.Unsubscribe("run-2", id2)

	bus.Publish("run-1", makeEvent("run-1", apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_PHASE_CHANGED))

	// run-1 subscriber should get event
	select {
	case <-ch1:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("run-1 subscriber did not receive event")
	}

	// run-2 subscriber should NOT get event
	select {
	case <-ch2:
		t.Fatal("run-2 subscriber should not receive run-1 event")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSlowClientDrop(t *testing.T) {
	bus := NewChannelBus()
	ch, id := bus.Subscribe("run-1")
	defer bus.Unsubscribe("run-1", id)

	// Fill the buffer (capacity 64)
	for i := 0; i < subscriberBufferSize+10; i++ {
		bus.Publish("run-1", makeEvent("run-1", apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG))
	}

	// Should have exactly subscriberBufferSize events
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != subscriberBufferSize {
		t.Errorf("expected %d events (buffer size), got %d", subscriberBufferSize, count)
	}
}

func TestUnsubscribeCleanup(t *testing.T) {
	bus := NewChannelBus()
	ch, id := bus.Subscribe("run-1")

	bus.Unsubscribe("run-1", id)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestEmptyTopicRemoval(t *testing.T) {
	bus := NewChannelBus()
	_, id := bus.Subscribe("run-1")
	bus.Unsubscribe("run-1", id)

	bus.mu.RLock()
	_, exists := bus.topics["run-1"]
	bus.mu.RUnlock()

	if exists {
		t.Error("expected topic to be removed when last subscriber unsubscribes")
	}
}

func TestNoOpEventBus(t *testing.T) {
	bus := &NoOpEventBus{}

	// Should not panic
	bus.Publish("run-1", makeEvent("run-1", apiv1.AgentRunEventType_AGENT_RUN_EVENT_TYPE_LOG))

	ch, id := bus.Subscribe("run-1")
	if ch == nil {
		t.Error("expected non-nil channel from NoOpEventBus")
	}

	bus.Unsubscribe("run-1", id)
}
