package models

import (
	"time"
)

type Product struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`

	PlatformConfigs []PlatformConfig `gorm:"foreignKey:ProductID" json:"platform_configs,omitempty"`
	AgentConfig     *AgentConfig     `gorm:"foreignKey:ProductID" json:"agent_config,omitempty"`
	ReporterConfigs []ReporterConfig `gorm:"foreignKey:ProductID" json:"reporter_configs,omitempty"`
	Issues          []Issue          `gorm:"foreignKey:ProductID" json:"issues,omitempty"`
	Messages        []Message        `gorm:"foreignKey:ProductID" json:"messages,omitempty"`
}
