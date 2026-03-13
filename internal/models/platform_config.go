package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type PlatformConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ProductID uint      `gorm:"not null;index" json:"product_id"`
	Type      string    `gorm:"not null" json:"type"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Settings  JSONMap   `gorm:"type:text" json:"settings"`
}

// JSONMap is a map[string]string stored as JSON text in the database.
type JSONMap map[string]string

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONMap: %T", value)
	}
	return json.Unmarshal(bytes, j)
}
