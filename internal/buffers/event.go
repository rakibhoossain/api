package buffers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

const addScreenViewScript = `
local sessionKey = KEYS[1]
local profileKey = KEYS[2]
local queueKey = KEYS[3]
local counterKey = KEYS[4]
local newEventData = ARGV[1]
local ttl = tonumber(ARGV[2])

local previousEventData = redis.call("GETDEL", sessionKey)
redis.call("SET", sessionKey, newEventData, "EX", ttl)

if profileKey and profileKey ~= "" then
  redis.call("SET", profileKey, newEventData, "EX", ttl)
end

if previousEventData then
  local prev = cjson.decode(previousEventData)
  local curr = cjson.decode(newEventData)
  
  if prev.ts and curr.ts then
    prev.event.duration = math.max(0, curr.ts - prev.ts)
  end
  
  redis.call("RPUSH", queueKey, cjson.encode(prev.event))
  redis.call("INCR", counterKey)
  return 1
end

return 0
`

const addSessionEndScript = `
local sessionKey = KEYS[1]
local profileKey = KEYS[2]
local queueKey = KEYS[3]
local counterKey = KEYS[4]
local sessionEndJson = ARGV[1]

local previousEventData = redis.call("GETDEL", sessionKey)
local added = 0

if previousEventData then
  local prev = cjson.decode(previousEventData)
  redis.call("RPUSH", queueKey, cjson.encode(prev.event))
  redis.call("INCR", counterKey)
  added = added + 1
end

redis.call("RPUSH", queueKey, sessionEndJson)
redis.call("INCR", counterKey)
added = added + 1

if profileKey and profileKey ~= "" then
  redis.call("DEL", profileKey)
end

return added
`

type EventBuffer struct {
	*RedisBuffer
	ch            *repository.ClickhouseRepo
	screenViewSha string
	sessionEndSha string
}

func NewEventBuffer(ch *repository.ClickhouseRepo, b *Buffers) *EventBuffer {
	eb := &EventBuffer{
		RedisBuffer: NewRedisBuffer("Event", b.rdb),
		ch:          ch,
	}
	eb.loadScripts()
	return eb
}

func (b *EventBuffer) loadScripts() {
	ctx := context.Background()
	screenViewSha, err := b.rdb.ScriptLoad(ctx, addScreenViewScript).Result()
	if err != nil {
		log.Printf("Failed to load addScreenViewScript: %v", err)
	} else {
		b.screenViewSha = screenViewSha
	}

	sessionEndSha, err := b.rdb.ScriptLoad(ctx, addSessionEndScript).Result()
	if err != nil {
		log.Printf("Failed to load addSessionEndScript: %v", err)
	} else {
		b.sessionEndSha = sessionEndSha
	}
}

// evalScript safely runs the loaded script SHA or falls back to EVAL.
func (b *EventBuffer) evalScript(ctx context.Context, scriptName, scriptContent, sha string, keys []string, args ...interface{}) error {
	if sha != "" {
		res := b.rdb.EvalSha(ctx, sha, keys, args...)
		if err := res.Err(); err != nil && err != redis.Nil {
			// fallback check
			// "NOSCRIPT No matching script. Please use EVAL."
			log.Printf("Script %s failed via SHA: %v, falling back to EVAL", scriptName, err)
			errEval := b.rdb.Eval(ctx, scriptContent, keys, args...).Err()
			b.loadScripts()
			return errEval
		}
		return nil
	}
	
	err := b.rdb.Eval(ctx, scriptContent, keys, args...).Err()
	if err != nil && err != redis.Nil {
		b.loadScripts()
		return err
	}
	return nil
}

type eventWithTs struct {
	Event models.Event `json:"event"`
	Ts    int64        `json:"ts"`
}

func (b *EventBuffer) AddEvent(ctx context.Context, event models.Event) {
	eventJsonBytes, _ := json.Marshal(event)
	eventJson := string(eventJsonBytes)

	if event.SessionID != "" && event.Name == "screen_view" {
		sessionKey := fmt.Sprintf("event_buffer:last_screen_view:session:%s", event.SessionID)
		profileKey := ""
		if event.ProfileID != "" {
			profileKey = fmt.Sprintf("event_buffer:last_screen_view:profile:%s:%s", event.ProjectID, event.ProfileID)
		}

		timestamp := event.CreatedAt.UnixMilli()
		if timestamp <= 0 {
			timestamp = time.Now().UnixMilli()
		}

		eventWithTimestampBytes, _ := json.Marshal(eventWithTs{
			Event: event,
			Ts:    timestamp,
		})

		_ = b.evalScript(
			ctx,
			"addScreenView",
			addScreenViewScript,
			b.screenViewSha,
			[]string{sessionKey, profileKey, b.redisKey, b.bufferCountKey},
			string(eventWithTimestampBytes),
			"3600", // 1 hour TTL
		)

	} else if event.SessionID != "" && event.Name == "session_end" {
		sessionKey := fmt.Sprintf("event_buffer:last_screen_view:session:%s", event.SessionID)
		profileKey := ""
		if event.ProfileID != "" {
			profileKey = fmt.Sprintf("event_buffer:last_screen_view:profile:%s:%s", event.ProjectID, event.ProfileID)
		}

		_ = b.evalScript(
			ctx,
			"addSessionEnd",
			addSessionEndScript,
			b.sessionEndSha,
			[]string{sessionKey, profileKey, b.redisKey, b.bufferCountKey},
			eventJson,
		)
	} else {
		// All other events directly queue
		pipe := b.rdb.TxPipeline()
		pipe.RPush(ctx, b.redisKey, eventJson)
		pipe.Incr(ctx, b.bufferCountKey)
		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Printf("Failed to push normal event: %v", err)
		}
	}
}

func (b *EventBuffer) TryFlush() error {
	return b.RedisBuffer.TryFlush(func(ctx context.Context, items []string) error {
		batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO events")
		if err != nil {
			log.Printf("Failed to prepare batch for events: %v", err)
			return err
		}

		for _, item := range items {
			var eventModel models.Event
			if err := json.Unmarshal([]byte(item), &eventModel); err != nil {
				log.Printf("Warning: Dropping malformed JSON event payload from Redis queue")
				continue
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
		return nil
	})
}
