package buffers

import "log"

type ProfileBuffer struct {
	*GenericBuffer
}

func NewProfileBuffer() *ProfileBuffer {
	return &ProfileBuffer{GenericBuffer: NewGenericBuffer("ProfileBuffer")}
}

func (b *ProfileBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)

	b.items = make([]interface{}, 0)
	return nil
}
