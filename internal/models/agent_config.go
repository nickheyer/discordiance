package models

import (
	"time"
)

type AgentConfig struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ProductID    uint      `gorm:"uniqueIndex;not null" json:"product_id"`
	BaseURL      string    `json:"base_url"`
	APIKey       string    `json:"-"` // hidden from JSON
	OrgID        string    `json:"org_id"`
	Model        string    `gorm:"not null" json:"model"`
	SystemPrompt string    `gorm:"type:text" json:"system_prompt"`
	BatchSize    int       `gorm:"default:10" json:"batch_size"`
	BatchTimeout int       `gorm:"default:30" json:"batch_timeout"`
}
