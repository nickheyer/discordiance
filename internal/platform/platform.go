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

	// Connect initializes the platform connection with the given settings.
	Connect(ctx context.Context, settings models.JSONMap) error

	// Start begins listening for real-time messages, delivering them to the handler.
	Start(ctx context.Context, handler MessageHandler) error

	// Backfill scans historical messages from a channel starting after the given cursor.
	// Cursor is an opaque platform-specific string (e.g. Discord snowflake). Empty means start from beginning.
	Backfill(ctx context.Context, channelID string, cursor string, handler MessageHandler) error

	// DiscoverChannels returns the channels to backfill.
	// Returns configured channels if any, otherwise auto-discovers text channels.
	DiscoverChannels(ctx context.Context) ([]string, error)

	// Stop gracefully shuts down the platform connection.
	Stop(ctx context.Context) error

	// Healthy reports whether the platform connection is alive.
	Healthy(ctx context.Context) bool
}
