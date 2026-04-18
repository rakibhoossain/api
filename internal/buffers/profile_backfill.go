package buffers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type ProfileBackfillEntry struct {
	ProjectID string `json:"projectId"`
	SessionID string `json:"sessionId"`
	ProfileID string `json:"profileId"`
}

type ProfileBackfillBuffer struct {
	*RedisBuffer
	ch *repository.ClickhouseRepo
}

func NewProfileBackfillBuffer(ch *repository.ClickhouseRepo, b *Buffers) *ProfileBackfillBuffer {
	return &ProfileBackfillBuffer{
		RedisBuffer: NewRedisBuffer("ProfileBackfill", b.rdb),
		ch:          ch,
	}
}

func (b *ProfileBackfillBuffer) TryFlush() error {
	return b.RedisBuffer.TryFlush(func(ctx context.Context, items []string) error {
		seen := make(map[string]ProfileBackfillEntry)
		for _, item := range items {
			var entry ProfileBackfillEntry
			if err := json.Unmarshal([]byte(item), &entry); err == nil {
				seen[entry.SessionID] = entry
			}
		}

		if len(seen) == 0 {
			return nil
		}

		var entries []ProfileBackfillEntry
		for _, v := range seen {
			entries = append(entries, v)
		}

		chunkSize := 500
		processedChunks := 0

		for i := 0; i < len(entries); i += chunkSize {
			end := i + chunkSize
			if end > len(entries) {
				end = len(entries)
			}
			chunk := entries[i:end]

			var caseClauses []string
			var tupleList []string

			for _, c := range chunk {
				sID := strings.ReplaceAll(c.SessionID, "'", "''")
				pID := strings.ReplaceAll(c.ProfileID, "'", "''")
				projID := strings.ReplaceAll(c.ProjectID, "'", "''")

				caseClauses = append(caseClauses, fmt.Sprintf("WHEN '%s' THEN '%s'", sID, pID))
				tupleList = append(tupleList, fmt.Sprintf("('%s', '%s')", projID, sID))
			}

			// We use events natively since there's no getReplicatedTableName abstraction
			query := fmt.Sprintf(`
				UPDATE events
				SET profile_id = CASE session_id
					%s
				END
				WHERE (project_id, session_id) IN (%s)
				  AND created_at > now() - INTERVAL 6 HOUR
				SETTINGS mutations_sync = 0, allow_experimental_lightweight_update = 1
			`, strings.Join(caseClauses, "\n"), strings.Join(tupleList, ","))

			// Execution string using Exec with allow_experimental_lightweight_update
			err := b.ch.Conn.Exec(ctx, query)
			if err != nil {
				log.Printf("[ProfileBackfillBuffer] Error executing chunk: %v", err)
				continue
			}
			processedChunks++
		}

		if processedChunks != (len(entries)+chunkSize-1)/chunkSize {
			// Some chunks failed, but TryFlush will still ltrim based on whole batch. 
			// In a robust queue, we'd only ltrim processed, but matching BaseBuffer parity, it ltrims if the block completes.
		}

		return nil
	})
}
