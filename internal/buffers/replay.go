package buffers

import (
	"context"
	"log"
	"time"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type ReplayBuffer struct {
	*GenericBuffer
	ch *repository.ClickhouseRepo
}

func NewReplayBuffer(ch *repository.ClickhouseRepo) *ReplayBuffer {
	return &ReplayBuffer{
		GenericBuffer: NewGenericBuffer("ReplayBuffer"),
		ch:            ch,
	}
}

func (b *ReplayBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)

	ctx := context.Background()
	batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO session_replay_chunks")
	if err != nil {
		log.Printf("Failed to prepare batch for replay chunks: %v", err)
		return err
	}

	for _, item := range b.items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		chunk := models.SessionReplayChunk{
			ProjectID:      strOrEmptyRep(m, "projectId"),
			SessionID:      strOrEmptyRep(m, "sessionId"),
			ChunkIndex:     uint16(getIntOrZero(m, "chunkIndex")),
			StartedAt:      time.Now(),
			EndedAt:        time.Now(),
			EventsCount:    uint16(getIntOrZero(m, "eventsCount")),
			IsFullSnapshot: getBoolOrFalse(m, "isFullSnapshot"),
			Payload:        strOrEmptyRep(m, "payload"),
		}

		if err := batch.AppendStruct(&chunk); err != nil {
			log.Printf("Error appending session replay chunk to batch: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to flush session replays: %v", err)
		return err
	}

	b.items = make([]interface{}, 0)
	return nil
}

func strOrEmptyRep(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getIntOrZero(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getBoolOrFalse(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}
