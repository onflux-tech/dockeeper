package events

import (
	"log/slog"
	"sync"
)

const bufferSize = 1000

// Handler processes events dispatched by the Bus.
type Handler interface {
	Handle(event Event)
}

// HandlerFunc adapts a plain function into a Handler.
type HandlerFunc func(Event)

func (f HandlerFunc) Handle(e Event) { f(e) }

// Bus is an in-process event bus with buffered async dispatch.
type Bus struct {
	ch       chan Event
	handlers []Handler
	mu       sync.RWMutex
	done     chan struct{}
	once     sync.Once
}

// NewBus creates and starts a new EventBus.
func NewBus() *Bus {
	b := &Bus{
		ch:   make(chan Event, bufferSize),
		done: make(chan struct{}),
	}
	go b.loop()
	return b
}

// Subscribe registers a handler to receive events.
// Must be called before events are published to avoid missing early events.
func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Publish sends an event to all handlers asynchronously.
// It never blocks the caller — if the buffer is full, the oldest event is discarded.
func (b *Bus) Publish(e Event) {
	select {
	case b.ch <- e:
	default:
		// Buffer full — discard oldest to make room.
		select {
		case <-b.ch:
			slog.Warn("eventbus buffer full, dropped oldest event")
		default:
		}
		// Try again.
		select {
		case b.ch <- e:
		default:
			slog.Warn("eventbus: failed to enqueue event after drop", "type", string(e.Type))
		}
	}
}

// Close drains remaining events and stops the dispatch loop.
func (b *Bus) Close() {
	b.once.Do(func() {
		close(b.ch)
		<-b.done // wait for loop to finish
	})
}

// loop consumes events from the channel and dispatches to all handlers.
func (b *Bus) loop() {
	defer close(b.done)
	for e := range b.ch {
		b.mu.RLock()
		handlers := b.handlers
		b.mu.RUnlock()

		for _, h := range handlers {
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("eventbus handler panicked", "error", r, "event_type", string(e.Type))
					}
				}()
				h.Handle(e)
			}()
		}
	}
}
