package models

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID          string         `gorm:"primaryKey;size:36"`
	Name        string         `gorm:"not null"`
	Description string         `gorm:"type:text"`
	Contexts    []ProductContext
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type ProductContext struct {
	ID        string `gorm:"primaryKey;size:36"`
	ProductID string `gorm:"size:36;not null;index"`
	Type      int32  `gorm:"not null"`
	Value     string `gorm:"type:text;not null"`
	Label     string
}
