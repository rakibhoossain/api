package buffers

import (
	"github.com/redis/go-redis/v9"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

// Global buffers registry for manual and scheduled flushes.
type Buffers struct {
	EventBuffer           *EventBuffer
	ProfileBuffer         *ProfileBuffer
	SessionBuffer         *SessionBuffer
	ProfileBackfillBuffer *ProfileBackfillBuffer
	ReplayBuffer          *ReplayBuffer
	rdb                   *redis.Client
}

func InitBuffers(ch *repository.ClickhouseRepo, rdb *redis.Client) *Buffers {
	b := &Buffers{rdb: rdb}
	b.EventBuffer = NewEventBuffer(ch, b)
	b.ProfileBuffer = NewProfileBuffer(ch, b)
	b.SessionBuffer = NewSessionBuffer(ch, b)
	b.ProfileBackfillBuffer = NewProfileBackfillBuffer(ch, b)
	b.ReplayBuffer = NewReplayBuffer(ch, b)
	return b
}
