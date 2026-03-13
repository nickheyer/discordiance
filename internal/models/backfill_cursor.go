package models

import (
	"time"
)

type BackfillCursor struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	PlatformConfigID uint      `gorm:"not null;uniqueIndex:idx_platform_channel" json:"platform_config_id"`
	ChannelID        string    `gorm:"not null;uniqueIndex:idx_platform_channel" json:"channel_id"`
	LastMessageID    string    `gorm:"not null" json:"last_message_id"`
}
