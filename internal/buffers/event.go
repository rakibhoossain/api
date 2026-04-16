package buffers

import "log"

type EventBuffer struct {
	*GenericBuffer
}

func NewEventBuffer() *EventBuffer {
	return &EventBuffer{GenericBuffer: NewGenericBuffer("EventBuffer")}
}

func (b *EventBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)
	// Clickhouse bulk insert goes here logic

	b.items = make([]interface{}, 0)
	return nil
}
