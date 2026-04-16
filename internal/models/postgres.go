package models

import (
	"time"

	"github.com/google/uuid"
)

type Salt struct {
	Salt      string    `json:"salt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Chat struct {
	ID        string                 `json:"id"`
	Messages  map[string]interface{} `json:"messages"`
	ProjectID string                 `json:"projectId"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

type Dashboard struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ProjectID string    `json:"projectId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Report struct {
	ID           uuid.UUID              `json:"id"`
	Interval     string                 `json:"interval"`
	ChartType    string                 `json:"chartType"`
	Breakdowns   map[string]interface{} `json:"breakdowns"`
	Events       map[string]interface{} `json:"events"`
	ProjectID    string                 `json:"projectId"`
	DashboardID  string                 `json:"dashboardId"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Name         string                 `json:"name"`
	Range        string                 `json:"range"`
	LineType     string                 `json:"lineType"`
	Previous     bool                   `json:"previous"`
	Formula      *string                `json:"formula"`
	Metric       string                 `json:"metric"`
	Unit         *string                `json:"unit"`
	Criteria     *string                `json:"criteria"`
	FunnelGroup  *string                `json:"funnelGroup"`
	FunnelWindow *float64               `json:"funnelWindow"`
	Options      map[string]interface{} `json:"options"`
}

type ReportLayout struct {
	ID        uuid.UUID `json:"id"`
	ReportID  uuid.UUID `json:"reportId"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	W         int       `json:"w"`
	H         int       `json:"h"`
	MinW      *int      `json:"minW"`
	MinH      *int      `json:"minH"`
	MaxW      *int      `json:"maxW"`
	MaxH      *int      `json:"maxH"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EventMeta struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Conversion *bool     `json:"conversion"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	ProjectID  string    `json:"projectId"`
	Color      *string   `json:"color"`
	Icon       *string   `json:"icon"`
}
