package models

import (
	"time"
)

const (
	MaxFileSize       = 1 << 20      // 1MB per file
	MaxProductFileSum = 10 * (1 << 20) // 10MB total per product
)

type ProductFile struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ProductID uint      `gorm:"not null;index" json:"product_id"`
	Filename  string    `gorm:"not null" json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	Content   []byte    `gorm:"type:blob" json:"-"`
}
