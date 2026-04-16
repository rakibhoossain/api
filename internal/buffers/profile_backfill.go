package buffers

import "log"

type ProfileBackfillBuffer struct {
	*GenericBuffer
}

func NewProfileBackfillBuffer() *ProfileBackfillBuffer {
	return &ProfileBackfillBuffer{GenericBuffer: NewGenericBuffer("ProfileBackfillBuffer")}
}

func (b *ProfileBackfillBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)

	b.items = make([]interface{}, 0)
	return nil
}
