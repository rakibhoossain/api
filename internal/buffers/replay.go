package buffers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type ReplayBuffer struct {
	*RedisBuffer
	ch *repository.ClickhouseRepo
}

func NewReplayBuffer(ch *repository.ClickhouseRepo, b *Buffers) *ReplayBuffer {
	return &ReplayBuffer{
		RedisBuffer: NewRedisBuffer("Replay", b.rdb),
		ch:          ch,
	}
}

func (b *ReplayBuffer) TryFlush() error {
	return b.RedisBuffer.TryFlush(func(ctx context.Context, items []string) error {
		batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO session_replay_chunks")
		if err != nil {
			log.Printf("Failed to prepare batch for replay chunks: %v", err)
			return err
		}

		for _, item := range items {
			var chunk models.SessionReplayChunk
			if err := json.Unmarshal([]byte(item), &chunk); err == nil {
				chunk.StartedAt = time.Now()
				chunk.EndedAt = time.Now()
				if err := batch.AppendStruct(&chunk); err != nil {
					log.Printf("Error appending session replay chunk to batch: %v", err)
				}
			}
		}

		if err := batch.Send(); err != nil {
			log.Printf("Failed to flush session replays: %v", err)
			return err
		}
		return nil
	})
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
