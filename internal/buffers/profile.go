package buffers

import (
	"context"
	"log"
	"time"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type ProfileBuffer struct {
	*GenericBuffer
	ch *repository.ClickhouseRepo
}

func NewProfileBuffer(ch *repository.ClickhouseRepo) *ProfileBuffer {
	return &ProfileBuffer{
		GenericBuffer: NewGenericBuffer("ProfileBuffer"),
		ch:            ch,
	}
}

func (b *ProfileBuffer) TryFlush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}
	log.Printf("Flushing %d items from %s", len(b.items), b.name)
	ctx := context.Background()
	batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO profiles")
	if err != nil {
		log.Printf("Failed to prepare batch for profiles: %v", err)
		return err
	}

	for _, item := range b.items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		profile := models.Profile{
			ID:         strOrEmptyObj(m, "profileId"),
			FirstName:  strOrEmptyObj(m, "firstName"),
			LastName:   strOrEmptyObj(m, "lastName"),
			Email:      strOrEmptyObj(m, "email"),
			Avatar:     strOrEmptyObj(m, "avatar"),
			ProjectID:  strOrEmptyObj(m, "projectId"),
			CreatedAt:  time.Now(),
		}

		if props, ok := m["properties"].(map[string]interface{}); ok {
			profile.Properties = props
		} else {
			profile.Properties = make(map[string]interface{})
		}

		if err := batch.AppendStruct(&profile); err != nil {
			log.Printf("Error appending profile to batch: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to flush profiles: %v", err)
		return err
	}

	b.items = make([]interface{}, 0)
	return nil
}

func strOrEmptyObj(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
