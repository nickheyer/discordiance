package models

import "time"

type Report struct {
	ID         string `gorm:"primaryKey;size:36"`
	PipelineID string `gorm:"size:36;not null;index"`
	ProductID  string `gorm:"size:36;not null;index"`
	TotalInsights int32
	HotCount      int32
	BurnCount     int32
	ColdCount     int32
	GeneratedAt   time.Time
	Entries       []ReportEntry
}

type ReportEntry struct {
	ID                   string `gorm:"primaryKey;size:36"`
	ReportID             string `gorm:"size:36;not null;index"`
	InsightID            string `gorm:"size:36;not null"`
	State                int32  `gorm:"not null"`
	ContentPreview       string `gorm:"type:text"`
	ClassificationReason string `gorm:"type:text"`
	PlatformID           string `gorm:"size:36"`
	SourceURL            string
}
