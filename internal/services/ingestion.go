package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"strconv"
	"github.com/google/uuid"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/openpanel-dev/openpanel-api/internal/buffers"
	"github.com/openpanel-dev/openpanel-api/internal/enrich"
	"github.com/openpanel-dev/openpanel-api/internal/models"
	"github.com/openpanel-dev/openpanel-api/internal/repository"
	"github.com/openpanel-dev/openpanel-api/internal/tasks"
)

type IngestionService struct {
	pg      *repository.PostgresRepo
	asynq   *asynq.Client
	redis   *redis.Client
	buffers *buffers.Buffers
}

func NewIngestionService(pg *repository.PostgresRepo, asynqClient *asynq.Client, rdb *redis.Client, b *buffers.Buffers) *IngestionService {
	return &IngestionService{
		pg:      pg,
		asynq:   asynqClient,
		redis:   rdb,
		buffers: b,
	}
}

type ProcessResult struct {
	DeviceID  string
	SessionID string
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

func (s *IngestionService) ProcessEvent(ctx context.Context, projectId string, ip string, ua string, body models.TrackPayload, headers map[string]string) (*ProcessResult, error) {
	currentSalt, _, err := s.pg.GetSalts(ctx)
	if err != nil {
		log.Printf("Error fetching salts: %v", err)
		currentSalt = "fallback" // Fallback if DB fails
	}

	deviceId := enrich.GenerateDeviceID(currentSalt, projectId, ip, ua)
	if override, ok := body.Properties["__deviceId"].(string); ok && override != "" {
		deviceId = override
	}

	now := time.Now().UnixMilli()
	sessionId := enrich.GetSessionID(projectId, deviceId, now, 1000*60*30, 1000*60)

	// Build strongly-typed eventModel immediately
	geo := enrich.GetGeoLocation(ip)
	uaInfo := enrich.GetUAInfo(ua)

	properties := body.Properties
	if properties == nil {
		properties = make(map[string]interface{})
	}

	revenue := float64(0)
	if body.Name == "revenue" {
		revenue = parseRevenue(properties["__revenue"])
	} else if rev, ok := properties["__revenue"]; ok {
		revenue = parseRevenue(rev)
	}

	duration := uint64(0)
	if dur, ok := properties["duration"]; ok {
		duration = parseDuration(dur)
	} else if dur, ok := properties["__duration"]; ok {
		duration = parseDuration(dur)
	}

	var lat, lon *float32
	if geo.Latitude != 0 || geo.Longitude != 0 {
		l1 := float32(geo.Latitude)
		l2 := float32(geo.Longitude)
		lat = &l1
		lon = &l2
	}

	createdAt := time.Now()
	if body.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, body.Timestamp); err == nil {
			createdAt = t
		}
	}

	eventModel := models.Event{
		ID:             uuid.New(),
		Name:           body.Name,
		DeviceID:       deviceId,
		ProfileID:      models.ConvertProfileID(body.ProfileID),
		ProjectID:      projectId,
		SessionID:      sessionId,
		Path:           strOrEmpty(properties, "__path"),
		Origin:         strOrEmpty(properties, "__origin"),
		Referrer:       strOrEmpty(properties, "__referrer"),
		ReferrerName:   strOrEmpty(properties, "__referrer_name"),
		ReferrerType:   strOrEmpty(properties, "__referrer_type"),
		Revenue:        revenue,
		Duration:       duration,
		Properties:     properties,
		CreatedAt:      createdAt,
		Country:        geo.Country,
		City:           geo.City,
		Region:         geo.Region,
		Latitude:       lat,
		Longitude:      lon,
		OS:             uaInfo.OS,
		Browser:        uaInfo.Browser,
		BrowserVersion: uaInfo.BrowserVersion,
		OSVersion:      uaInfo.OSVersion,
		Device:         uaInfo.Device,
		Brand:          uaInfo.Brand,
		Model:          uaInfo.Model,
		NetworkOrg:     geo.NetworkOrg,
	}

	// Fetch active session from Redis strictly to schedule the session_end correctly
	sessionKey := fmt.Sprintf("session:active:%s:%s", projectId, deviceId)
	val, err := s.redis.Get(ctx, sessionKey).Result()
	
	isNewSession := false
	var sessionData struct {
		SessionID  string `json:"sid"`
		StartTime  int64  `json:"st"`
		LastActive int64  `json:"la"`
	}

	if err == redis.Nil {
		isNewSession = true
	} else if err == nil {
		json.Unmarshal([]byte(val), &sessionData)
		if sessionData.SessionID != sessionId {
			isNewSession = true
		}
	}

	if isNewSession {
		sessionData.SessionID = sessionId
		sessionData.StartTime = now
		sessionData.LastActive = now

		// Create session_start event explicitly
		sessionStartEvent := eventModel
		sessionStartEvent.ID = uuid.New()
		sessionStartEvent.Name = "session_start"
		sessionStartEvent.CreatedAt = eventModel.CreatedAt.Add(-100 * time.Millisecond)
		s.buffers.EventBuffer.AddEvent(ctx, sessionStartEvent)

		// Schedule session:end task

		payload, _ := json.Marshal(tasks.SessionEndPayload{
			ProjectID: projectId,
			DeviceID:  deviceId,
			SessionID: sessionId,
		})
		task := asynq.NewTask(tasks.TypeSessionEnd, payload)
		_, err := s.asynq.Enqueue(task, asynq.ProcessIn(30*time.Minute), asynq.TaskID("session:end:"+sessionId))
		if err != nil {
			log.Printf("Error scheduling session end task: %v", err)
		}
	} else {
		sessionData.LastActive = now
		// Extend session:end task by re-enqueueing
		payload, _ := json.Marshal(tasks.SessionEndPayload{
			ProjectID: projectId,
			DeviceID:  deviceId,
			SessionID: sessionId,
		})
		task := asynq.NewTask(tasks.TypeSessionEnd, payload)
		s.asynq.Enqueue(task, asynq.ProcessIn(30*time.Minute), asynq.TaskID("session:end:"+sessionId))
	}

	dataBytes, _ := json.Marshal(sessionData)
	s.redis.Set(ctx, sessionKey, dataBytes, 40*time.Minute)

	// Add event directly to logically governed hold structures
	s.buffers.EventBuffer.AddEvent(ctx, eventModel)

	// Maintain TS-like SessionLifecycle Logic in buffers
	s.buffers.SessionBuffer.AddEvent(ctx, eventModel)

	return &ProcessResult{
		DeviceID:  deviceId,
		SessionID: sessionId,
	}, nil
}
