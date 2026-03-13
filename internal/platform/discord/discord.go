package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
)

func init() {
	platform.Register("discord", func() platform.Platform {
		return &Discord{}
	})
}

type Discord struct {
	session    *discordgo.Session
	handler    platform.MessageHandler
	channelIDs []string
	mu         sync.RWMutex
	healthy    bool
}

func (d *Discord) Type() string { return "discord" }

func (d *Discord) Connect(ctx context.Context, cfg models.Platform) error {
	slog.Info("discord: connecting")

	if cfg.Token == "" {
		return fmt.Errorf("discord: token is required")
	}
	slog.Info("discord: token present", "length", len(cfg.Token))

	if cfg.ChannelIds != "" {
		d.channelIDs = strings.Split(cfg.ChannelIds, ",")
		for i := range d.channelIDs {
			d.channelIDs[i] = strings.TrimSpace(d.channelIDs[i])
		}
		slog.Info("discord: configured channels", "channel_ids", d.channelIDs)
	} else {
		slog.Info("discord: no channels configured, will auto-discover")
	}

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return fmt.Errorf("discord: creating session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds
	slog.Info("discord: session created", "intents", session.Identify.Intents)

	d.session = session
	slog.Info("discord: connection configured")
	return nil
}

func (d *Discord) Start(ctx context.Context, handler platform.MessageHandler) error {
	slog.Info("discord: starting real-time listener")
	d.handler = handler

	d.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore bot's own messages
		if m.Author.ID == s.State.User.ID {
			return
		}

		// Filter by channel if configured
		if len(d.channelIDs) > 0 && !d.isWatchedChannel(m.ChannelID) {
			slog.Debug("discord: ignoring message from unwatched channel", "channel_id", m.ChannelID, "author", m.Author.Username)
			return
		}

		slog.Info("discord: message received", "channel_id", m.ChannelID, "author", m.Author.Username, "message_id", m.ID, "length", len(m.Content))

		msg := models.Message{
			PlatformType: "discord",
			ExternalID:   m.ID,
			ChannelID:    m.ChannelID,
			AuthorID:     m.Author.ID,
			AuthorName:   m.Author.Username,
			Content:      m.Content,
			Timestamp:    m.Timestamp,
		}

		handler(msg)
	})

	d.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		guilds := make([]string, len(r.Guilds))
		for i, g := range r.Guilds {
			guilds[i] = g.ID
		}
		slog.Info("discord: bot ready", "user", r.User.Username, "user_id", r.User.ID, "guilds", guilds, "guild_count", len(guilds))
	})

	slog.Info("discord: opening websocket connection")
	if err := d.session.Open(); err != nil {
		slog.Error("discord: failed to open connection", "error", err)
		return fmt.Errorf("discord: opening connection: %w", err)
	}

	d.mu.Lock()
	d.healthy = true
	d.mu.Unlock()

	slog.Info("discord: platform connected and listening", "watched_channels", d.channelIDs)
	return nil
}

func (d *Discord) Backfill(ctx context.Context, channelID string, cursor string, handler platform.MessageHandler) error {
	if cursor == "" {
		cursor = "0"
	}

	slog.Info("discord: backfill starting", "channel_id", channelID, "cursor", cursor)

	afterID := cursor
	totalMessages := 0
	pageCount := 0

	for {
		select {
		case <-ctx.Done():
			slog.Info("discord: backfill cancelled", "channel_id", channelID, "messages_fetched", totalMessages, "pages", pageCount)
			return ctx.Err()
		default:
		}

		pageCount++
		slog.Info("discord: backfill fetching page", "channel_id", channelID, "page", pageCount, "after_id", afterID)

		messages, err := d.session.ChannelMessages(channelID, 100, "", afterID, "")
		if err != nil {
			slog.Error("discord: backfill page fetch failed", "channel_id", channelID, "page", pageCount, "error", err)
			return fmt.Errorf("discord: fetching messages for channel %s: %w", channelID, err)
		}

		slog.Info("discord: backfill page received", "channel_id", channelID, "page", pageCount, "messages_in_page", len(messages))

		if len(messages) == 0 {
			slog.Info("discord: backfill empty page, channel complete", "channel_id", channelID)
			break
		}

		// ChannelMessages returns newest-first; reverse to process oldest-first
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		pageUserMessages := 0
		for _, m := range messages {
			if m.Author.Bot {
				continue
			}

			handler(models.Message{
				PlatformType: "discord",
				ExternalID:   m.ID,
				ChannelID:    m.ChannelID,
				AuthorID:     m.Author.ID,
				AuthorName:   m.Author.Username,
				Content:      m.Content,
				Timestamp:    m.Timestamp,
			})
			pageUserMessages++
			totalMessages++
		}

		slog.Info("discord: backfill page processed", "channel_id", channelID, "page", pageCount,
			"user_messages", pageUserMessages, "total_messages", totalMessages)

		afterID = messages[len(messages)-1].ID

		if len(messages) < 100 {
			slog.Info("discord: backfill last page (< 100 messages)", "channel_id", channelID)
			break
		}

		slog.Debug("discord: backfill rate limit sleep", "channel_id", channelID, "duration", "250ms")
		time.Sleep(250 * time.Millisecond)
	}

	slog.Info("discord: backfill complete", "channel_id", channelID, "total_messages", totalMessages, "pages", pageCount)
	return nil
}

func (d *Discord) DiscoverChannels(ctx context.Context) ([]string, error) {
	if len(d.channelIDs) > 0 {
		slog.Info("discord: using configured channels", "count", len(d.channelIDs), "channels", d.channelIDs)
		return d.channelIDs, nil
	}

	slog.Info("discord: auto-discovering channels from guilds", "guild_count", len(d.session.State.Guilds))

	var channels []string
	for _, g := range d.session.State.Guilds {
		slog.Info("discord: listing channels for guild", "guild_id", g.ID)
		guildChannels, err := d.session.GuildChannels(g.ID)
		if err != nil {
			slog.Warn("discord: failed to list channels for guild", "guild_id", g.ID, "error", err)
			continue
		}

		guildTextCount := 0
		for _, ch := range guildChannels {
			if ch.Type == discordgo.ChannelTypeGuildText {
				channels = append(channels, ch.ID)
				guildTextCount++
				slog.Debug("discord: discovered text channel", "guild_id", g.ID, "channel_id", ch.ID, "channel_name", ch.Name)
			}
		}
		slog.Info("discord: guild channels discovered", "guild_id", g.ID, "total_channels", len(guildChannels), "text_channels", guildTextCount)
	}

	if len(channels) == 0 {
		slog.Error("discord: no text channels found in any guild")
		return nil, fmt.Errorf("discord: no text channels found")
	}

	slog.Info("discord: auto-discovered channels", "count", len(channels), "channels", channels)
	return channels, nil
}

func (d *Discord) Stop(ctx context.Context) error {
	slog.Info("discord: stopping")

	d.mu.Lock()
	d.healthy = false
	d.mu.Unlock()

	if d.session != nil {
		if err := d.session.Close(); err != nil {
			slog.Error("discord: error closing session", "error", err)
			return err
		}
	}

	slog.Info("discord: stopped")
	return nil
}

func (d *Discord) Healthy(ctx context.Context) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.healthy
}

func (d *Discord) isWatchedChannel(channelID string) bool {
	for _, id := range d.channelIDs {
		if id == channelID {
			return true
		}
	}
	return false
}
