package models

import (
	"time"
)

type ProductContextCache struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ProductID uint      `gorm:"uniqueIndex;not null" json:"product_id"`
	Content   string    `gorm:"type:text" json:"content"`
	FetchedAt time.Time `json:"fetched_at"`
}
