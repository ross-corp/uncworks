// Package eventbus provides an in-process pub/sub event bus for AgentRun events.
// Events are ephemeral and not persisted — subscribers only receive events
// published while they are actively subscribed.
package eventbus

import (
	"sync"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

const subscriberBufferSize = 64

// EventBus defines the interface for publishing and subscribing to AgentRun events.
type EventBus interface {
	// Publish sends an event to all subscribers for the given agent run ID.
	// Non-blocking: if a subscriber's buffer is full, the event is dropped.
	Publish(agentRunID string, event *apiv1.AgentRunEvent)

	// Subscribe registers a new subscriber for the given agent run ID.
	// Returns a channel that receives events and a subscription ID for unsubscribing.
	Subscribe(agentRunID string) (ch <-chan *apiv1.AgentRunEvent, id int)

	// Unsubscribe removes a subscriber. The returned channel is closed.
	Unsubscribe(agentRunID string, id int)
}

// NoOpEventBus is an EventBus that discards all events. Useful for tests
// that don't need event delivery.
type NoOpEventBus struct{}

func (n *NoOpEventBus) Publish(string, *apiv1.AgentRunEvent) {}

func (n *NoOpEventBus) Subscribe(string) (<-chan *apiv1.AgentRunEvent, int) {
	return make(chan *apiv1.AgentRunEvent, 1), 0
}

func (n *NoOpEventBus) Unsubscribe(string, int) {}

type subscriber struct {
	id int
	ch chan *apiv1.AgentRunEvent
}

// ChannelBus is a channel-based EventBus implementation.
type ChannelBus struct {
	mu     sync.RWMutex
	topics map[string][]subscriber
	nextID int
}

// NewChannelBus creates a new channel-based event bus.
func NewChannelBus() *ChannelBus {
	return &ChannelBus{
		topics: make(map[string][]subscriber),
	}
}

// Publish sends an event to all subscribers for the given agent run ID.
func (b *ChannelBus) Publish(agentRunID string, event *apiv1.AgentRunEvent) {
	b.mu.RLock()
	subs := b.topics[agentRunID]
	b.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.ch <- event:
		default:
			// Drop event for slow subscriber
		}
	}
}

// Subscribe registers a new subscriber for the given agent run ID.
func (b *ChannelBus) Subscribe(agentRunID string) (<-chan *apiv1.AgentRunEvent, int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *apiv1.AgentRunEvent, subscriberBufferSize)
	id := b.nextID
	b.nextID++

	b.topics[agentRunID] = append(b.topics[agentRunID], subscriber{id: id, ch: ch})
	return ch, id
}

// Unsubscribe removes a subscriber and closes its channel.
func (b *ChannelBus) Unsubscribe(agentRunID string, id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.topics[agentRunID]
	for i, sub := range subs {
		if sub.id == id {
			close(sub.ch)
			b.topics[agentRunID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// Remove empty topic
	if len(b.topics[agentRunID]) == 0 {
		delete(b.topics, agentRunID)
	}
}
