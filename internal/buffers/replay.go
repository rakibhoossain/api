package buffers

import "log"

type ReplayBuffer struct {
	*GenericBuffer
}

func NewReplayBuffer() *ReplayBuffer {
	return &ReplayBuffer{GenericBuffer: NewGenericBuffer("ReplayBuffer")}
}

func (b *ReplayBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)

	b.items = make([]interface{}, 0)
	return nil
}
