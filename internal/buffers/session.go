package buffers

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type SessionBuffer struct {
	*GenericBuffer
	ch *repository.ClickhouseRepo
}

func NewSessionBuffer(ch *repository.ClickhouseRepo) *SessionBuffer {
	return &SessionBuffer{
		GenericBuffer: NewGenericBuffer("SessionBuffer"),
		ch:            ch,
	}
}

func (b *SessionBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)

	ctx := context.Background()
	batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO sessions")
	if err != nil {
		log.Printf("Failed to prepare batch for sessions: %v", err)
		return err
	}

	for _, item := range b.items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		session := models.Session{
			ID:        strOrEmptySess(m, "sessionId"),
			ProjectID: strOrEmptySess(m, "projectId"),
			ProfileID: strOrEmptySess(m, "profileId"),
			DeviceID:  strOrEmptySess(m, "deviceId"),
			CreatedAt: time.Now(), 
		}
		if session.ID == "" {
			session.ID = uuid.New().String()
		}

		if err := batch.AppendStruct(&session); err != nil {
			log.Printf("Error appending session to batch: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to flush sessions: %v", err)
		return err
	}

	b.items = make([]interface{}, 0)
	return nil
}

func strOrEmptySess(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
