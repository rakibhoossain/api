package models

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

type GroupPayload struct {
	ID         string                 `json:"id" validate:"required"`
	Type       string                 `json:"type" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type AssignGroupPayload struct {
	GroupIDs  []string `json:"groupIds" validate:"required,min=1"`
	ProfileID interface{} `json:"profileId,omitempty"` // string or float64 per Zod union
}

type TrackPayload struct {
	Name       string                 `json:"name" validate:"required"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	ProfileID  interface{}            `json:"profileId,omitempty"`
	Groups     []string               `json:"groups,omitempty"`
}

type IdentifyPayload struct {
	ProfileID  interface{}            `json:"profileId" validate:"required"`
	FirstName  string                 `json:"firstName,omitempty"`
	LastName   string                 `json:"lastName,omitempty"`
	Email      string                 `json:"email,omitempty" validate:"omitempty,email"`
	Avatar     string                 `json:"avatar,omitempty" validate:"omitempty,url"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type IncrementPayload struct {
	ProfileID interface{} `json:"profileId" validate:"required"`
	Property  string      `json:"property" validate:"required"`
	Value     float64     `json:"value,omitempty" validate:"omitempty,gt=0"`
}

type DecrementPayload struct {
	ProfileID interface{} `json:"profileId" validate:"required"`
	Property  string      `json:"property" validate:"required"`
	Value     float64     `json:"value,omitempty" validate:"omitempty,gt=0"`
}

type AliasPayload struct {
	ProfileID interface{} `json:"profileId" validate:"required"`
	Alias     string      `json:"alias" validate:"required"`
}

type ReplayPayload struct {
	ChunkIndex     int    `json:"chunk_index" validate:"min=0,max=65535"`
	EventsCount    int    `json:"events_count" validate:"min=1"`
	IsFullSnapshot bool   `json:"is_full_snapshot"`
	StartedAt      string `json:"started_at" validate:"required"`
	EndedAt        string `json:"ended_at" validate:"required"`
	Payload        string `json:"payload" validate:"required,max=2097152"` // 2MB max
}

type TrackHandlerPayload struct {
	Type    string          `json:"type" validate:"required"`
	Payload json.RawMessage `json:"payload" validate:"required"`
}

// Deprecated Payload Types for legacy endpoints
type DeprecatedOpenpanelEventOptions struct {
	ProfileID string `json:"profileId,omitempty"`
}

type DeprecatedPostEventPayload struct {
	Name       string                 `json:"name" validate:"required"`
	Timestamp  string                 `json:"timestamp"`
	ProfileID  string                 `json:"profileId,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type DeprecatedUpdateProfilePayload struct {
	ProfileID  string                 `json:"profileId" validate:"required"`
	FirstName  string                 `json:"firstName,omitempty"`
	LastName   string                 `json:"lastName,omitempty"`
	Email      string                 `json:"email,omitempty" validate:"omitempty,email"`
	Avatar     string                 `json:"avatar,omitempty" validate:"omitempty,url"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type DeprecatedIncrementProfilePayload struct {
	ProfileID string  `json:"profileId,omitempty"`
	Property  string  `json:"property" validate:"required"`
	Value     float64 `json:"value" validate:"required"`
}

func ConvertProfileID(id interface{}) string {
	if id == nil {
		return ""
	}
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	}
	// best effort
	str, _ := id.(string)
	return str
}
