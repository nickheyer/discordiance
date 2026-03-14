package platform

import (
	"context"
	"time"

	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// RawContent is the normalized output of a platform adapter.
type RawContent struct {
	PlatformID     string
	SourceID       string
	ConversationID string
	Content        string
	Author         string
	SourceURL      string
	Medium         v1.InsightMedium
	Timestamp      time.Time
}

// Adapter ingests content from an external platform.
type Adapter interface {
	// Start begins ingestion. Content is pushed to out until ctx is cancelled.
	Start(ctx context.Context, out chan<- RawContent) error
	// Type returns the platform type this adapter handles.
	Type() v1.PlatformType
}
