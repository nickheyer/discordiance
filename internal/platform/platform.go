package platform

import (
	"context"

	"github.com/nickheyer/discordiance/internal/models"
)

// MessageHandler is called by platforms when new messages arrive.
type MessageHandler func(msg models.Message)

// Platform defines the interface for any message source (Discord, Slack, etc.).
type Platform interface {
	// Type returns the platform identifier (e.g. "discord").
	Type() string

	// Connect initializes the platform connection with the given config.
	Connect(ctx context.Context, cfg models.Platform) error

	// Start begins listening for real-time messages, delivering them to the handler.
	Start(ctx context.Context, handler MessageHandler) error

	// Backfill scans historical messages from a channel starting after the given cursor.
	Backfill(ctx context.Context, channelID string, cursor string, handler MessageHandler) error

	// DiscoverChannels returns the channels to backfill.
	DiscoverChannels(ctx context.Context) ([]string, error)

	// Stop gracefully shuts down the platform connection.
	Stop(ctx context.Context) error

	// Healthy reports whether the platform connection is alive.
	Healthy(ctx context.Context) bool
}
