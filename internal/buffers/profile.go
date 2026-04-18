package buffers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
)

type ProfileBuffer struct {
	*RedisBuffer
	ch *repository.ClickhouseRepo
}

func NewProfileBuffer(ch *repository.ClickhouseRepo, b *Buffers) *ProfileBuffer {
	return &ProfileBuffer{
		RedisBuffer: NewRedisBuffer("Profile", b.rdb),
		ch:          ch,
	}
}

func mergeProperties(existing, incoming map[string]interface{}) map[string]interface{} {
	if existing == nil {
		return incoming
	}
	merged := make(map[string]interface{})
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range incoming {
		merged[k] = v
	}
	return merged
}

func mergeProfiles(existing *models.Profile, incoming models.Profile) models.Profile {
	if existing == nil {
		return incoming
	}

	merged := incoming

	// Omit geo locations if moving to server explicitly
	if existingDevice, _ := existing.Properties["device"].(string); existingDevice != "server" {
		if incomingDevice, _ := incoming.Properties["device"].(string); incomingDevice == "server" {
			delete(merged.Properties, "city")
			delete(merged.Properties, "country")
			delete(merged.Properties, "region")
			delete(merged.Properties, "longitude")
			delete(merged.Properties, "latitude")
			delete(merged.Properties, "os")
			delete(merged.Properties, "osVersion")
			delete(merged.Properties, "browser")
			delete(merged.Properties, "device")
			delete(merged.Properties, "isServer")
			delete(merged.Properties, "os_version")
			delete(merged.Properties, "browser_version")
		}
	}

	merged.Properties = mergeProperties(existing.Properties, merged.Properties)
	if merged.FirstName == "" {
		merged.FirstName = existing.FirstName
	}
	if merged.LastName == "" {
		merged.LastName = existing.LastName
	}
	if merged.Email == "" {
		merged.Email = existing.Email
	}
	if merged.Avatar == "" {
		merged.Avatar = existing.Avatar
	}

	return merged
}

func (b *ProfileBuffer) TryFlush() error {
	return b.RedisBuffer.TryFlush(func(ctx context.Context, items []string) error {
		var incomingProfiles []models.Profile
		for _, item := range items {
			var profile models.Profile
			if err := json.Unmarshal([]byte(item), &profile); err == nil {
				if profile.Properties == nil {
					profile.Properties = make(map[string]interface{})
				}
				incomingProfiles = append(incomingProfiles, profile)
			}
		}

		if len(incomingProfiles) == 0 {
			return nil
		}

		// 1. Collapse multiple updates for the same profile into a batch map
		mergedInBatch := make(map[string]models.Profile)
		for _, p := range incomingProfiles {
			key := p.ProjectID + ":" + p.ID
			if existing, ok := mergedInBatch[key]; ok {
				mergedInBatch[key] = mergeProfiles(&existing, p)
			} else {
				mergedInBatch[key] = p
			}
		}

		var uniqueProfiles []models.Profile
		var cacheKeys []string
		for _, p := range mergedInBatch {
			uniqueProfiles = append(uniqueProfiles, p)
			cacheKeys = append(cacheKeys, fmt.Sprintf("profile-cache:%s:%s", p.ProjectID, p.ID))
		}

		// 2. MGET from Redis cache
		existingByKey := make(map[string]models.Profile)
		var cacheMisses []models.Profile

		cacheResults, err := b.rdb.MGet(ctx, cacheKeys...).Result()
		if err == nil {
			for i, uniqueProfile := range uniqueProfiles {
				key := uniqueProfile.ProjectID + ":" + uniqueProfile.ID
				if cacheResults[i] != nil {
					if cstr, ok := cacheResults[i].(string); ok {
						var cached models.Profile
						if err := json.Unmarshal([]byte(cstr), &cached); err == nil {
							existingByKey[key] = cached
							continue
						}
					}
				}
				cacheMisses = append(cacheMisses, uniqueProfile)
			}
		} else {
			cacheMisses = uniqueProfiles
		}

		// 3. Batch fetch misses from ClickHouse
		if len(cacheMisses) > 0 {
			var tuples []string
			for _, m := range cacheMisses {
				tuples = append(tuples, fmt.Sprintf("('%s', '%s')", strings.ReplaceAll(m.ID, "'", "''"), strings.ReplaceAll(m.ProjectID, "'", "''")))
			}
			
			query := fmt.Sprintf(`
				SELECT 
					id, project_id,
					argMax(nullIf(first_name, ''), created_at) as first_name,
					argMax(nullIf(last_name, ''), created_at) as last_name,
					argMax(nullIf(email, ''), created_at) as email,
					argMax(nullIf(avatar, ''), created_at) as avatar,
					argMax(is_external, created_at) as is_external,
					argMax(properties, created_at) as properties
				FROM profiles
				WHERE (id, project_id) IN (%s)
				GROUP BY id, project_id
			`, strings.Join(tuples, ","))

			rows, err := b.ch.Conn.Query(ctx, query)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var p models.Profile
					var props string
					if err := rows.Scan(&p.ID, &p.ProjectID, &p.FirstName, &p.LastName, &p.Email, &p.Avatar, &p.IsExternal, &props); err == nil {
						if p.Properties == nil {
							p.Properties = make(map[string]interface{})
						}
						json.Unmarshal([]byte(props), &p.Properties)
						existingByKey[p.ProjectID+":"+p.ID] = p
					}
				}
			}
		}

		// 4. Final Insert and Multi Set
		batch, err := b.ch.Conn.PrepareBatch(ctx, "INSERT INTO profiles")
		if err != nil {
			return err
		}

		pipe := b.rdb.TxPipeline()
		for _, p := range uniqueProfiles {
			key := p.ProjectID + ":" + p.ID
			var merged models.Profile
			if existing, found := existingByKey[key]; found {
				merged = mergeProfiles(&existing, p)
			} else {
				merged = p
			}

			batch.AppendStruct(&merged)

			mergedBytes, _ := json.Marshal(merged)
			pipe.Set(ctx, fmt.Sprintf("profile-cache:%s:%s", p.ProjectID, p.ID), string(mergedBytes), 60*time.Minute)
		}

		if err := batch.Send(); err != nil {
			return err
		}

		_, err = pipe.Exec(ctx)
		return err
	})
}
