package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/openpanel-dev/openpanel-api/internal/buffers"
	"github.com/openpanel-dev/openpanel-api/internal/models"
)

func HandleSessionEndTask(b *buffers.Buffers, rdb *redis.Client) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var p SessionEndPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
		}

		sessionKey := fmt.Sprintf("session:active:%s:%s", p.ProjectID, p.DeviceID)
		val, err := rdb.Get(ctx, sessionKey).Result()
		if err != nil {
			return fmt.Errorf("could not find session in redis: %w", err)
		}

		var sessionData struct {
			SessionID  string `json:"sid"`
			StartTime  int64  `json:"st"`
			LastActive int64  `json:"la"`
		}
		json.Unmarshal([]byte(val), &sessionData)

		// Check if this is still the same session being ended
		if sessionData.SessionID != p.SessionID {
			log.Printf("Session end task for %s ignored as current session is %s", p.SessionID, sessionData.SessionID)
			return nil
		}

		duration := (sessionData.LastActive - sessionData.StartTime) / 1000 // seconds
		isBounce := false // Simplified bounce logic (could count events if we had them)

		// Create session_end event
		b.EventBuffer.Add(map[string]interface{}{
			"projectId": p.ProjectID,
			"deviceId":  p.DeviceID,
			"sessionId": p.SessionID,
			"event": models.TrackPayload{
				Name:      "session_end",
				Timestamp: time.UnixMilli(sessionData.LastActive).Add(1 * time.Second).Format(time.RFC3339),
				Properties: map[string]interface{}{
					"duration": duration,
					"__bounce": isBounce,
				},
			},
		})

		// Flush session info to ClickHouse
		// We need more info for a full session row (UA, Geo, etc.), 
		// but for now let's use what we have.
		b.SessionBuffer.Add(map[string]interface{}{
			"projectId": p.ProjectID,
			"deviceId":  p.DeviceID,
			"sessionId": p.SessionID,
			"profileId": "", // if we had it
			"duration":  uint32(duration),
		})

		// Cleanup Redis
		rdb.Del(ctx, sessionKey)

		log.Printf("Session %s ended. Duration: %d seconds.", p.SessionID, duration)
		return nil
	}
}
