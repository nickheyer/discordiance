package models

import (
	"time"
)

type Report struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	InsightID    uint      `gorm:"column:issue_id;not null;index" json:"insight_id"`
	ReporterType string    `gorm:"not null" json:"reporter_type"`
	ExternalID   string    `json:"external_id"`
	ExternalURL  string    `json:"external_url"`
	Status       string    `gorm:"default:filed" json:"status"`
}
