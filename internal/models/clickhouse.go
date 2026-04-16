package models

import (
	"time"

	"github.com/google/uuid"
)

type SessionReplayChunk struct {
	ProjectID      string    `json:"project_id"`
	SessionID      string    `json:"session_id"`
	ChunkIndex     uint16    `json:"chunk_index"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at"`
	EventsCount    uint16    `json:"events_count"`
	IsFullSnapshot bool      `json:"is_full_snapshot"`
	Payload        string    `json:"payload"`
}

type Profile struct {
	ID         string                 `json:"id"`
	IsExternal bool                   `json:"is_external"`
	FirstName  string                 `json:"first_name"`
	LastName   string                 `json:"last_name"`
	Email      string                 `json:"email"`
	Avatar     string                 `json:"avatar"`
	Properties map[string]interface{} `json:"properties"`
	ProjectID  string                 `json:"project_id"`
	CreatedAt  time.Time              `json:"created_at"`
}

type Session struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	ProfileID       string    `json:"profile_id"`
	DeviceID        string    `json:"device_id"`
	CreatedAt       time.Time `json:"created_at"`
	EndedAt         time.Time `json:"ended_at"`
	IsBounce        bool      `json:"is_bounce"`
	EntryOrigin     string    `json:"entry_origin"`
	EntryPath       string    `json:"entry_path"`
	ExitOrigin      string    `json:"exit_origin"`
	ExitPath        string    `json:"exit_path"`
	ScreenViewCount int32     `json:"screen_view_count"`
	Revenue         float64   `json:"revenue"`
	EventCount      int32     `json:"event_count"`
	Duration        uint32    `json:"duration"`
	Country         string    `json:"country"`
	Region          string    `json:"region"`
	City            string    `json:"city"`
	Longitude       *float32  `json:"longitude"`
	Latitude        *float32  `json:"latitude"`
	Device          string    `json:"device"`
	Brand           string    `json:"brand"`
	Model           string    `json:"model"`
	Browser         string    `json:"browser"`
	BrowserVersion  string    `json:"browser_version"`
	OS              string    `json:"os"`
	OSVersion       string    `json:"os_version"`
	UtmMedium       string    `json:"utm_medium"`
	UtmSource       string    `json:"utm_source"`
	UtmCampaign     string    `json:"utm_campaign"`
	UtmContent      string    `json:"utm_content"`
	UtmTerm         string    `json:"utm_term"`
	Referrer        string    `json:"referrer"`
	ReferrerName    string    `json:"referrer_name"`
	ReferrerType    string    `json:"referrer_type"`
	Sign            int8      `json:"sign"`
	Version         uint64    `json:"version"`
	NetworkOrg      string    `json:"network_org"`
}

type Event struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	DeviceID       string                 `json:"device_id"`
	ProfileID      string                 `json:"profile_id"`
	ProjectID      string                 `json:"project_id"`
	SessionID      string                 `json:"session_id"`
	Path           string                 `json:"path"`
	Origin         string                 `json:"origin"`
	Referrer       string                 `json:"referrer"`
	ReferrerName   string                 `json:"referrer_name"`
	ReferrerType   string                 `json:"referrer_type"`
	Revenue        float64                `json:"revenue"`
	Duration       uint64                 `json:"duration"`
	Properties     map[string]interface{} `json:"properties"`
	CreatedAt      time.Time              `json:"created_at"`
	Country        string                 `json:"country"`
	City           string                 `json:"city"`
	Region         string                 `json:"region"`
	Longitude      *float32               `json:"longitude"`
	Latitude       *float32               `json:"latitude"`
	OS             string                 `json:"os"`
	OSVersion      string                 `json:"os_version"`
	Browser        string                 `json:"browser"`
	BrowserVersion string                 `json:"browser_version"`
	Device         string                 `json:"device"`
	Brand          string                 `json:"brand"`
	Model          string                 `json:"model"`
	NetworkOrg     string                 `json:"network_org"`
}
