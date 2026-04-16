package tasks

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
	"github.com/openpanel-dev/openpanel-api/internal/buffers"
)

const (
	TypeSalt                 = "cron:salt"
	TypeFlushEvents          = "cron:flushEvents"
	TypeFlushProfiles        = "cron:flushProfiles"
	TypeFlushSessions        = "cron:flushSessions"
	TypeFlushProfileBackfill = "cron:flushProfileBackfill"
	TypeFlushReplay          = "cron:flushReplay"
)

type FlushPayload struct {
	// Add parameters if required
}

func NewFlushEventsTask() (*asynq.Task, error) {
	payload, err := json.Marshal(FlushPayload{})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeFlushEvents, payload), nil
}

// Keep similarly for others...

func HandleFlushEventsTask(b *buffers.Buffers) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("Handling flushEvents...")
		return b.EventBuffer.TryFlush()
	}
}

func HandleFlushProfilesTask(b *buffers.Buffers) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("Handling flushProfiles...")
		return b.ProfileBuffer.TryFlush()
	}
}

func HandleFlushSessionsTask(b *buffers.Buffers) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("Handling flushSessions...")
		return b.SessionBuffer.TryFlush()
	}
}

func HandleFlushProfileBackfillTask(b *buffers.Buffers) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("Handling flushProfileBackfill...")
		return b.ProfileBackfillBuffer.TryFlush()
	}
}

func HandleFlushReplayTask(b *buffers.Buffers) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("Handling flushReplay...")
		return b.ReplayBuffer.TryFlush()
	}
}

func HandleSaltTask() func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Println("Handling salt rotation...")
		// Logic to rotate salts in pg
		return nil
	}
}
