package reporter

import (
	"context"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// Handler delivers classified insights to an external destination.
type Handler interface {
	// Deliver sends insights to the configured destination.
	Deliver(ctx context.Context, reporter *models.Reporter, insights []models.Insight) error
	// Type returns the reporter type this handler handles.
	Type() v1.ReporterType
}
