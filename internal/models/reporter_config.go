package models

import (
	"time"
)

type Reporter struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"not null" json:"name"`
	Type      string    `gorm:"not null" json:"type" mapstructure:"type"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	// GitHub fields
	Token    string `json:"token,omitempty" mapstructure:"token"`
	Repo     string `json:"repo,omitempty" mapstructure:"repo"`
	Labels   string `json:"labels,omitempty" mapstructure:"labels"`
	AutoFile bool   `gorm:"default:true" json:"auto_file" mapstructure:"auto_file"`
}
