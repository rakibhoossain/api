package buffers

import "github.com/openpanel-dev/openpanel-api/internal/repository"

// Global buffers registry for manual and scheduled flushes.
type Buffers struct {
	EventBuffer           *EventBuffer
	ProfileBuffer         *ProfileBuffer
	SessionBuffer         *SessionBuffer
	ProfileBackfillBuffer *ProfileBackfillBuffer
	ReplayBuffer          *ReplayBuffer
}

func InitBuffers(ch *repository.ClickhouseRepo) *Buffers {
	return &Buffers{
		EventBuffer:           NewEventBuffer(ch),
		ProfileBuffer:         NewProfileBuffer(ch),
		SessionBuffer:         NewSessionBuffer(ch),
		ProfileBackfillBuffer: NewProfileBackfillBuffer(ch),
		ReplayBuffer:          NewReplayBuffer(ch),
	}
}
