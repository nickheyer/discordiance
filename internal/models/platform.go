package models

import (
	"time"

	"gorm.io/gorm"
)

type Platform struct {
	ID        string         `gorm:"primaryKey;size:36"`
	Name      string         `gorm:"not null"`
	Type      int32          `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	DiscordConfig  *DiscordPlatformConfig
	RedditConfig   *RedditPlatformConfig
	TwitterConfig  *TwitterPlatformConfig
	LinkedInConfig *LinkedInPlatformConfig
	GitHubConfig   *GitHubPlatformConfig
}

type DiscordPlatformConfig struct {
	PlatformID string `gorm:"primaryKey;size:36"`
	BotToken   string `gorm:"not null"`
	GuildIDs   string `gorm:"type:text"` // JSON array
	ChannelIDs string `gorm:"type:text"` // JSON array
}

type RedditPlatformConfig struct {
	PlatformID   string `gorm:"primaryKey;size:36"`
	ClientID     string `gorm:"not null"`
	ClientSecret string `gorm:"not null"`
	Subreddits   string `gorm:"type:text"` // JSON array
}

type TwitterPlatformConfig struct {
	PlatformID  string `gorm:"primaryKey;size:36"`
	BearerToken string `gorm:"not null"`
	Keywords    string `gorm:"type:text"` // JSON array
	Accounts    string `gorm:"type:text"` // JSON array
}

type LinkedInPlatformConfig struct {
	PlatformID  string `gorm:"primaryKey;size:36"`
	AccessToken string `gorm:"not null"`
	CompanyIDs  string `gorm:"type:text"` // JSON array
}

type GitHubPlatformConfig struct {
	PlatformID         string `gorm:"primaryKey;size:36"`
	Token              string `gorm:"not null"`
	Repositories       string `gorm:"type:text"` // JSON array
	IncludeIssues      bool
	IncludeDiscussions bool
}
