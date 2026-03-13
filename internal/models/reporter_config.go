package models

import (
	"time"
)

type ReporterConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ProductID uint      `gorm:"not null;index" json:"product_id"`
	Type      string    `gorm:"not null" json:"type"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Settings  JSONMap   `gorm:"type:text" json:"settings"`
}
