package buffers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type SessionBuffer struct {
	*RedisBuffer
	ch  *repository.ClickhouseRepo
	rdb *redis.Client
}

func NewSessionBuffer(ch *repository.ClickhouseRepo, b *Buffers) *SessionBuffer {
	return &SessionBuffer{
		RedisBuffer: NewRedisBuffer("Session", b.rdb),
		ch:          ch,
		rdb:         b.rdb,
	}
}

// AddEvent applies the VersionedCollapsingMergeTree logic:
// it fetches the existing session, emits sign=-1 for the old version,
// and sign=1 for the new version, incrementing version numbers.
func (b *SessionBuffer) AddEvent(ctx context.Context, event models.Event) {
	if event.SessionID == "" || event.Name == "session_start" || event.Name == "session_end" {
		return // Handled externally or irrelevant to session aggregation
	}

	sessionKey := "session:" + event.SessionID
	val, err := b.rdb.Get(ctx, sessionKey).Result()

	var newSession models.Session
	var oldSession models.Session
	var hasOld bool

	if err == nil && val != "" {
		if err := json.Unmarshal([]byte(val), &newSession); err == nil {
			oldSession = newSession
			oldSession.Sign = -1
			hasOld = true

			// Update new session
			newSession.Sign = 1
			newSession.EndedAt = event.CreatedAt
			newSession.Version = newSession.Version + 1

			if newSession.EntryPath == "" && event.Path != "" {
				newSession.EntryPath = event.Path
			}
			if newSession.EntryOrigin == "" && event.Origin != "" {
				newSession.EntryOrigin = event.Origin
			}
			if event.Path != "" {
				newSession.ExitPath = event.Path
			}
			if event.Origin != "" {
				newSession.ExitOrigin = event.Origin
			}

			dur := newSession.EndedAt.UnixMilli() - newSession.CreatedAt.UnixMilli()
			if dur > 0 {
				newSession.Duration = uint32(dur)
			}

			addedRev := float64(0)
			if event.Name == "revenue" {
				addedRev = event.Revenue
			}
			newSession.Revenue += addedRev

			if event.Name == "screen_view" && event.Path != "" {
				newSession.ScreenViewCount++
			} else {
				newSession.EventCount++
			}

			if newSession.ScreenViewCount > 1 {
				newSession.IsBounce = false
			}

			if event.ProfileID != "" && event.ProfileID != event.DeviceID {
				newSession.ProfileID = event.ProfileID
			}
		}
	}

	if !hasOld {
		// Create new
		newSession = models.Session{
			ID:              event.SessionID,
			IsBounce:        true,
			ProfileID:       event.ProfileID,
			ProjectID:       event.ProjectID,
			DeviceID:        event.DeviceID,
			CreatedAt:       event.CreatedAt,
			EndedAt:         event.CreatedAt,
			EventCount:      0,
			ScreenViewCount: 0,
			EntryPath:       event.Path,
			EntryOrigin:     event.Origin,
			ExitPath:        event.Path,
			ExitOrigin:      event.Origin,
			Revenue:         event.Revenue,
			Referrer:        event.Referrer,
			ReferrerName:    event.ReferrerName,
			ReferrerType:    event.ReferrerType,
			OS:              event.OS,
			OSVersion:       event.OSVersion,
			Browser:         event.Browser,
			BrowserVersion:  event.BrowserVersion,
			Device:          event.Device,
			Brand:           event.Brand,
			Model:           event.Model,
			Country:         event.Country,
			Region:          event.Region,
			City:            event.City,
			Longitude:       event.Longitude,
			Latitude:        event.Latitude,
			Duration:        0,
			Sign:            1,
			Version:         1,
			NetworkOrg:      event.NetworkOrg,
		}
		if event.Name == "screen_view" {
			newSession.ScreenViewCount = 1
		} else {
			newSession.EventCount = 1
		}
		if event.Name != "revenue" {
			newSession.Revenue = 0
		}
	}

	// Cache in Redis
	newSessionBytes, _ := json.Marshal(newSession)
	
	pipeline := b.rdb.TxPipeline()
	pipeline.Set(ctx, sessionKey, string(newSessionBytes), 60*time.Minute)
	
	if newSession.ProfileID != "" {
		pipeline.Set(ctx, "session:"+newSession.ProjectID+":"+newSession.ProfileID, newSession.ID, 60*time.Minute)
	}
	
	// Add to generic RedisBuffer
	if hasOld {
		oldBytes, _ := json.Marshal(oldSession)
		pipeline.RPush(ctx, b.redisKey, string(oldBytes))
	}
	pipeline.RPush(ctx, b.redisKey, string(newSessionBytes))
	
	inc := int64(1)
	if hasOld {
		inc = 2
	}
	pipeline.IncrBy(ctx, b.bufferCountKey, inc)
	_, err = pipeline.Exec(ctx)
	if err != nil {
		log.Printf("[SessionBuffer] Failed to add session: %v", err)
	}
}

func (b *SessionBuffer) TryFlush() error {
	return b.RedisBuffer.TryFlush(func(ctx context.Context, items []string) error {
		batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO sessions")
		if err != nil {
			log.Printf("Failed to prepare batch for sessions: %v", err)
			return err
		}

		for _, item := range items {
			var session models.Session
			if err := json.Unmarshal([]byte(item), &session); err == nil {
				if session.ID == "" {
					session.ID = uuid.New().String()
				}
				if err := batch.AppendStruct(&session); err != nil {
					log.Printf("Error appending session to batch: %v", err)
				}
			}
		}

		if err := batch.Send(); err != nil {
			log.Printf("Failed to flush sessions: %v", err)
			return err
		}
		return nil
	})
}
