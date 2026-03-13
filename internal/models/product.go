package models

import (
	"time"
)

type Product struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	Description string    `gorm:"type:text" json:"description"`
	RepoURL     string    `json:"repo_url"`
	GitHubToken string    `json:"github_token"`

	Insights []Insight `gorm:"foreignKey:ProductID" json:"insights,omitempty"`
	Messages []Message `gorm:"foreignKey:ProductID" json:"messages,omitempty"`
}
