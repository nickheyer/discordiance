package models

import (
	"time"
)

type Insight struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ProductID    uint      `gorm:"not null;index" json:"product_id"`
	Title        string    `gorm:"not null" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	Severity     string    `json:"severity"`
	Category     string    `json:"category"`
	Fingerprint  string    `gorm:"index" json:"fingerprint"`
	Status       string    `gorm:"default:open;index" json:"status"`
	SourceMsgIDs string    `gorm:"type:text" json:"source_msg_ids"` // comma-separated message IDs

	Product Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Reports []Report `gorm:"foreignKey:InsightID" json:"reports,omitempty"`
}

// TableName keeps the existing "issues" table for backward compatibility.
func (Insight) TableName() string { return "issues" }
