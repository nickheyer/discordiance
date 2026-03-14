package models

import (
	"time"

	"gorm.io/gorm"
)

type Pipeline struct {
	ID                    string         `gorm:"primaryKey;size:36"`
	Name                  string         `gorm:"not null"`
	Description           string         `gorm:"type:text"`
	ProductID             string         `gorm:"size:36;not null;index"`
	Status                int32          `gorm:"not null;default:1"`
	BatchMode             int32          `gorm:"not null;default:1"`
	BatchSize             int32
	TimeWindowSeconds     int32
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             gorm.DeletedAt `gorm:"index"`

	Product   Product
	Agents    []PipelineAgent
	Platforms []PipelinePlatform
	Reporters []PipelineReporter
}

type PipelineAgent struct {
	PipelineID string `gorm:"primaryKey;size:36"`
	AgentID    string `gorm:"primaryKey;size:36"`
}

type PipelinePlatform struct {
	PipelineID string `gorm:"primaryKey;size:36"`
	PlatformID string `gorm:"primaryKey;size:36"`
}

type PipelineReporter struct {
	PipelineID string `gorm:"primaryKey;size:36"`
	ReporterID string `gorm:"primaryKey;size:36"`
}
