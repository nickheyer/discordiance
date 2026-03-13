package models

import "time"

type Pipeline struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Name       string    `gorm:"not null" json:"name"`
	Enabled    bool      `gorm:"default:true" json:"enabled"`
	ProductID  uint      `gorm:"not null;index" json:"product_id"`
	PlatformID uint      `gorm:"not null" json:"platform_id"`
	AgentID    uint      `gorm:"not null" json:"agent_id"`

	Product   Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Platform  Platform   `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
	Agent     Agent      `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Reporters []Reporter `gorm:"many2many:pipeline_reporters" json:"reporters,omitempty"`
}
