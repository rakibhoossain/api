package buffers

import "log"

type SessionBuffer struct {
	*GenericBuffer
}

func NewSessionBuffer() *SessionBuffer {
	return &SessionBuffer{GenericBuffer: NewGenericBuffer("SessionBuffer")}
}

func (b *SessionBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)

	b.items = make([]interface{}, 0)
	return nil
}
