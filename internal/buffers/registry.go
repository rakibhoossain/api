package buffers

// Global buffers registry for manual and scheduled flushes.
type Buffers struct {
	EventBuffer           *EventBuffer
	ProfileBuffer         *ProfileBuffer
	SessionBuffer         *SessionBuffer
	ProfileBackfillBuffer *ProfileBackfillBuffer
	ReplayBuffer          *ReplayBuffer
}

func InitBuffers() *Buffers {
	return &Buffers{
		EventBuffer:           NewEventBuffer(),
		ProfileBuffer:         NewProfileBuffer(),
		SessionBuffer:         NewSessionBuffer(),
		ProfileBackfillBuffer: NewProfileBackfillBuffer(),
		ReplayBuffer:          NewReplayBuffer(),
	}
}
