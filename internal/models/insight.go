package models

import (
	"time"

	"gorm.io/gorm"
)

type Insight struct {
	ID                   string         `gorm:"primaryKey;size:36"`
	PipelineID           string         `gorm:"size:36;not null;index"`
	PlatformID           string         `gorm:"size:36;not null;index"`
	State                int32          `gorm:"not null;default:1;index"`
	Medium               int32          `gorm:"not null"`
	Content              string         `gorm:"type:text;not null"`
	Author               string
	SourceURL            string
	SourceID             string         `gorm:"index"`
	ConversationID       string         `gorm:"index"`
	ClassificationReason string         `gorm:"type:text"`
	ClassifyingAgentID   string         `gorm:"size:36"`
	SourceTimestamp      time.Time
	IngestedAt           time.Time
	ClassifiedAt         *time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}
