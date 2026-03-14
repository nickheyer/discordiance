package models

import (
	"time"

	"gorm.io/gorm"
)

type Reporter struct {
	ID        string         `gorm:"primaryKey;size:36"`
	Name      string         `gorm:"not null"`
	Type      int32          `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	FilterStates    []ReporterFilterState
	FilterPlatforms []ReporterFilterPlatform

	WebhookConfig     *WebhookReporterConfig
	EmailConfig       *EmailReporterConfig
	DiscordConfig     *DiscordReporterConfig
	GitHubIssueConfig *GitHubIssueReporterConfig
}

type ReporterFilterState struct {
	ReporterID string `gorm:"primaryKey;size:36"`
	State      int32  `gorm:"primaryKey"`
}

type ReporterFilterPlatform struct {
	ReporterID string `gorm:"primaryKey;size:36"`
	PlatformID string `gorm:"primaryKey;size:36"`
}

type WebhookReporterConfig struct {
	ReporterID string `gorm:"primaryKey;size:36"`
	URL        string `gorm:"not null"`
	Secret     string
}

type EmailReporterConfig struct {
	ReporterID  string `gorm:"primaryKey;size:36"`
	SMTPHost    string `gorm:"not null"`
	SMTPPort    int32  `gorm:"not null"`
	FromAddress string `gorm:"not null"`
	ToAddresses string `gorm:"type:text;not null"` // JSON array
	Username    string
	Password    string
}

type DiscordReporterConfig struct {
	ReporterID string `gorm:"primaryKey;size:36"`
	WebhookURL string `gorm:"not null"`
}

type GitHubIssueReporterConfig struct {
	ReporterID string `gorm:"primaryKey;size:36"`
	Token      string `gorm:"not null"`
	Owner      string `gorm:"not null"`
	Repo       string `gorm:"not null"`
	Labels     string `gorm:"type:text"` // JSON array
}
