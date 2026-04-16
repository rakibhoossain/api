package buffers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type EventBuffer struct {
	*GenericBuffer
	ch *repository.ClickhouseRepo
}

func NewEventBuffer(ch *repository.ClickhouseRepo) *EventBuffer {
	return &EventBuffer{
		GenericBuffer: NewGenericBuffer("EventBuffer"),
		ch:            ch,
	}
}

func (b *EventBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)
	ctx := context.Background()
	batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		log.Printf("Failed to prepare batch for events: %v", err)
		return err
	}

	for _, item := range b.items {
		// Event mappings from internal map shape (from track handlers)
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Ensure we convert correctly
		eventRaw, _ := json.Marshal(m["event"])
		var evProps map[string]interface{}
		json.Unmarshal(eventRaw, &evProps) // or read native props

		// Prepare strongly typed struct matching clickhouse schema expectations
		eventModel := models.Event{
			ID:           uuid.New(),
			Name:         strOrEmpty(evProps, "name"),
			DeviceID:     strOrEmpty(m, "deviceId"),
			ProfileID:    strOrEmpty(evProps, "profileId"),
			ProjectID:    strOrEmpty(m, "projectId"),
			SessionID:    strOrEmpty(m, "sessionId"),
			Path:         strOrEmpty(evProps, "path"),
			Origin:       strOrEmpty(evProps, "origin"),
			Referrer:     strOrEmpty(evProps, "referrer"),
			ReferrerName: strOrEmpty(evProps, "referrerName"),
			ReferrerType: strOrEmpty(evProps, "referrerType"),
			Revenue:      0,
			Duration:     0,
			Properties:   evProps,
		}

		err = batch.AppendStruct(&eventModel)
		if err != nil {
			log.Printf("Error appending event to batch: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to flush events: %v", err)
		return err
	}

	b.items = make([]interface{}, 0)
	return nil
}

func strOrEmpty(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
