package reporter

import (
	"context"

	"github.com/nickheyer/discordiance/internal/models"
)

// Reporter defines the interface for any issue destination (GitHub, Jira, etc.).
type Reporter interface {
	// Type returns the reporter identifier (e.g. "github").
	Type() string

	// Init initializes the reporter with the given settings.
	Init(ctx context.Context, settings models.JSONMap) error

	// FileReport creates a new issue/ticket in the external system.
	FileReport(ctx context.Context, issue models.Issue) (*models.Report, error)

	// UpdateReport updates an existing issue/ticket.
	UpdateReport(ctx context.Context, externalID string, issue models.Issue) error

	// FindDuplicate checks if a similar issue already exists.
	FindDuplicate(ctx context.Context, issue models.Issue) (string, error)

	// Close gracefully shuts down the reporter.
	Close(ctx context.Context) error

	// Healthy reports whether the reporter is operational.
	Healthy(ctx context.Context) bool
}
