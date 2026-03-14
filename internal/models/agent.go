package models

import (
	"time"

	"gorm.io/gorm"
)

type Agent struct {
	ID          string         `gorm:"primaryKey;size:36"`
	Name        string         `gorm:"not null"`
	BaseURL     string         `gorm:"not null"`
	Model       string         `gorm:"not null"`
	APIKey      string         `gorm:"not null"`
	MaxTokens   int32
	Temperature float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
