package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// Adapter connects to Discord via a bot and streams messages.
type Adapter struct {
	platformID string
	botToken   string
	guildIDs   map[string]bool
	channelIDs map[string]bool
	session    *discordgo.Session
}

// NewAdapter creates a Discord adapter from a platform model.
func NewAdapter(p *models.Platform) (platform.Adapter, error) {
	if p.DiscordConfig == nil {
		return nil, fmt.Errorf("discord config is nil for platform %s", p.ID)
	}

	cfg := p.DiscordConfig

	guildIDs := toSet(cfg.GuildIDs)
	channelIDs := toSet(cfg.ChannelIDs)

	return &Adapter{
		platformID: p.ID,
		botToken:   cfg.BotToken,
		guildIDs:   guildIDs,
		channelIDs: channelIDs,
	}, nil
}

func (a *Adapter) Type() v1.PlatformType {
	return v1.PlatformType_PLATFORM_TYPE_DISCORD
}

func (a *Adapter) Start(ctx context.Context, out chan<- platform.RawContent) error {
	session, err := discordgo.New("Bot " + a.botToken)
	if err != nil {
		return fmt.Errorf("create discord session: %w", err)
	}
	a.session = session

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore bot's own messages.
		if m.Author.Bot {
			return
		}

		// Filter by guild.
		if len(a.guildIDs) > 0 && !a.guildIDs[m.GuildID] {
			return
		}

		// Filter by channel.
		if len(a.channelIDs) > 0 && !a.channelIDs[m.ChannelID] {
			return
		}

		ts := time.Time(m.Timestamp)

		raw := platform.RawContent{
			PlatformID:     a.platformID,
			SourceID:       m.ID,
			ConversationID: m.ChannelID,
			Content:        m.Content,
			Author:         m.Author.Username,
			SourceURL:      fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID),
			Medium:         v1.InsightMedium_INSIGHT_MEDIUM_MESSAGE,
			Timestamp:      ts,
		}

		select {
		case out <- raw:
		case <-ctx.Done():
			return
		}
	})

	if err := session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	slog.Info("discord: connected",
		"platform_id", a.platformID,
		"guilds", len(a.guildIDs),
		"channels", len(a.channelIDs))

	// Block until context is cancelled, then close.
	<-ctx.Done()
	slog.Info("discord: disconnecting", "platform_id", a.platformID)
	session.Close()
	return nil
}

// Factory returns a platform.Factory for Discord adapters.
func Factory(p *models.Platform) (platform.Adapter, error) {
	// Load the Discord config's JSON arrays into the model.
	if p.DiscordConfig != nil {
		return NewAdapter(p)
	}
	return nil, fmt.Errorf("platform %s has no discord config", p.ID)
}

func toSet(jsonArray string) map[string]bool {
	var items []string
	_ = json.Unmarshal([]byte(jsonArray), &items)
	set := make(map[string]bool, len(items))
	for _, item := range items {
		if item != "" {
			set[item] = true
		}
	}
	return set
}

// TestConnection verifies the bot token is valid by opening and immediately closing a session.
func TestConnection(botToken string) error {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return err
	}
	session.Identify.Intents = discordgo.IntentsGuilds

	errCh := make(chan error, 1)
	session.AddHandler(func(_ *discordgo.Session, _ *discordgo.Ready) {
		errCh <- nil
	})

	if err := session.Open(); err != nil {
		return err
	}
	defer session.Close()

	select {
	case err := <-errCh:
		return err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("connection test timed out")
	}
}
