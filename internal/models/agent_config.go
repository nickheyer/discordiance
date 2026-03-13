package models

import (
	"time"
)

type Agent struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Name         string    `gorm:"not null" json:"name"`
	BaseURL      string    `json:"base_url" mapstructure:"base_url"`
	APIKey       string    `json:"-" mapstructure:"api_key"`
	OrgID        string    `json:"org_id" mapstructure:"org_id"`
	Model        string    `gorm:"not null" json:"model" mapstructure:"model"`
	SystemPrompt string    `gorm:"type:text" json:"system_prompt" mapstructure:"system_prompt"`
	BatchSize    int       `gorm:"default:10" json:"batch_size" mapstructure:"batch_size"`
	BatchTimeout int       `gorm:"default:30" json:"batch_timeout" mapstructure:"batch_timeout_seconds"`
}
