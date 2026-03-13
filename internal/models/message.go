package models

import (
	"time"
)

type Message struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	ProductID    uint      `gorm:"not null;index" json:"product_id"`
	PlatformType string    `gorm:"not null" json:"platform_type"`
	ExternalID   string    `gorm:"index" json:"external_id"`
	ChannelID    string    `json:"channel_id"`
	AuthorID     string    `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	Timestamp    time.Time `gorm:"not null;index" json:"timestamp"`
	Processed    bool      `gorm:"default:false;index" json:"processed"`
}
