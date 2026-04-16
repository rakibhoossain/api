package buffers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"strconv"
	"time"
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
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract TrackPayload
		var p models.TrackPayload
		if eventMap, ok := m["event"].(models.TrackPayload); ok {
			p = eventMap
		} else {
			// Fallback if it's a map (unmarshaled from json elsewhere)
			eventRaw, _ := json.Marshal(m["event"])
			json.Unmarshal(eventRaw, &p)
		}

		properties := p.Properties
		if properties == nil {
			properties = make(map[string]interface{})
		}

		name := p.Name
		
		// Revenue logic: name == "revenue" or __revenue in props
		revenue := float64(0)
		if name == "revenue" {
			revenue = parseRevenue(properties["__revenue"])
		} else if rev, ok := properties["__revenue"]; ok {
			revenue = parseRevenue(rev)
		}

		// Duration logic
		duration := uint64(0)
		if dur, ok := properties["duration"]; ok {
			duration = parseDuration(dur)
		} else if dur, ok := properties["__duration"]; ok {
			duration = parseDuration(dur)
		}

		// Geo mapping
		var geo models.GeoLocation
		if g, ok := m["geo"].(models.GeoLocation); ok {
			geo = g
		} else if g, ok := m["geo"].(map[string]interface{}); ok {
			// fallback if it's a map
			geoRaw, _ := json.Marshal(g)
			json.Unmarshal(geoRaw, &geo)
		}

		var lat, lon *float32
		if geo.Latitude != 0 || geo.Longitude != 0 {
			l1 := float32(geo.Latitude)
			l2 := float32(geo.Longitude)
			lat = &l1
			lon = &l2
		}

		// Headers mapping
		headers := make(map[string]string)
		if h, ok := m["headers"].(map[string]string); ok {
			headers = h
		}

		// Prepare strongly typed struct matching clickhouse schema expectations
		eventModel := models.Event{
			ID:             uuid.New(),
			Name:           name,
			DeviceID:       strOrEmpty(m, "deviceId"),
			ProfileID:      models.ConvertProfileID(p.ProfileID),
			ProjectID:      strOrEmpty(m, "projectId"),
			SessionID:      strOrEmpty(m, "sessionId"),
			Path:           strOrEmpty(properties, "__path"),
			Origin:         strOrEmpty(properties, "__origin"),
			Referrer:       strOrEmpty(properties, "__referrer"),
			ReferrerName:   strOrEmpty(properties, "__referrer_name"),
			ReferrerType:   strOrEmpty(properties, "__referrer_type"),
			Revenue:        revenue,
			Duration:       duration,
			Properties:     properties,
			CreatedAt:      time.Now(), // Default, check for timestamp in payload
			Country:        geo.Country,
			City:           geo.City,
			Region:         geo.Region,
			Latitude:       lat,
			Longitude:      lon,
			OS:             headers["os"], // Placeholder if not parsed yet
			Browser:        headers["browser"],
			BrowserVersion: headers["browser_version"],
			OSVersion:      headers["os_version"],
			NetworkOrg:     strOrEmpty(m, "network_org"),
		}

		// Check for timestamp in payload
		if p.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, p.Timestamp); err == nil {
				eventModel.CreatedAt = t
			}
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

func parseRevenue(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}

func parseDuration(v interface{}) uint64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return uint64(val)
	case int:
		return uint64(val)
	case int64:
		return uint64(val)
	case string:
		if i, err := strconv.ParseUint(val, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func strOrEmpty(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
