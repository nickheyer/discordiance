package models

import (
	"time"
)

type Platform struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"not null" json:"name"`
	Type      string    `gorm:"not null" json:"type" mapstructure:"type"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	// Discord fields
	Token      string `json:"token,omitempty" mapstructure:"token"`
	ChannelIds string `json:"channel_ids,omitempty" mapstructure:"channel_ids"`
}
