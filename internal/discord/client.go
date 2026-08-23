// Package discord contains all Discord-specific logic: session management,
// command routing, permission middleware and event handlers.
package discord

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/teyhouse/discord-bot-golang/internal/config"
)

// Client owns the Discord session lifecycle.
type Client struct {
	session *discordgo.Session
	config  *config.Config
	logger  *slog.Logger
}

// NewClient creates a discordgo session with the intents required by the
// handlers. It does not connect; call Start for that.
func NewClient(cfg *config.Config, logger *slog.Logger) (*Client, error) {
	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("discord: creating session: %w", err)
	}
	session.StateEnabled = true
	session.Identify.Intents = discordgo.MakeIntent(
		discordgo.IntentsGuildMessages |
			discordgo.IntentsGuildMessageReactions |
			discordgo.IntentsDirectMessages |
			discordgo.IntentsMessageContent,
	)
	return &Client{session: session, config: cfg, logger: logger}, nil
}

// Start connects to Discord and blocks until ctx is cancelled or the
// connection fails. Shutdown is graceful: it stops listening and closes
// the websocket and REST connections.
func (c *Client) Start(ctx context.Context) error {
	if err := c.session.Open(); err != nil {
		return fmt.Errorf("discord: opening connection: %w", err)
	}
	c.logger.Info("discord: connected", "user", c.session.State.User.ID)

	<-ctx.Done()
	return c.Stop()
}

// Stop closes the underlying session. It is safe to call multiple times.
func (c *Client) Stop() error {
	return c.session.Close()
}

// Session exposes the underlying session for handler registration.
// Treat it as read-only outside this package.
func (c *Client) Session() *discordgo.Session {
	return c.session
}
