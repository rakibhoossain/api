package buffers

import (
	"log"

	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type ProfileBackfillBuffer struct {
	*GenericBuffer
	ch *repository.ClickhouseRepo
}

func NewProfileBackfillBuffer(ch *repository.ClickhouseRepo) *ProfileBackfillBuffer {
	return &ProfileBackfillBuffer{
		GenericBuffer: NewGenericBuffer("ProfileBackfillBuffer"),
		ch:            ch,
	}
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
