// Package config loads and validates bot configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the bot.
type Config struct {
	BotToken  string
	ChannelID string
	GuildID   string
	LogLevel  string
	DataDir   string
	// WhitelistedUsers is empty for a public bot (no permission checks).
	WhitelistedUsers []string
}

// Load reads .env (optional) and the process environment, validates required
// fields, and returns a Config. It fails fast on missing required values.
func Load() (*Config, error) {
	_ = godotenv.Load() // missing .env is fine; real env takes precedence

	cfg := &Config{
		BotToken:  os.Getenv("BOT_TOKEN"),
		ChannelID: os.Getenv("CHANNEL_ID"),
		GuildID:   os.Getenv("GUILD_ID"),
		LogLevel:  os.Getenv("LOG_LEVEL"),
		DataDir:   os.Getenv("DATA_DIR"),
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}

	for id := range strings.SplitSeq(os.Getenv("WHITELISTED_USER_IDS"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			cfg.WhitelistedUsers = append(cfg.WhitelistedUsers, id)
		}
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("config: BOT_TOKEN is required")
	}
	if cfg.ChannelID == "" {
		return nil, fmt.Errorf("config: CHANNEL_ID is required")
	}
	if cfg.GuildID == "" {
		return nil, fmt.Errorf("config: GUILD_ID is required")
	}

	switch strings.ToLower(cfg.LogLevel) {
	case "", "info":
		cfg.LogLevel = "info"
	case "debug", "warn", "error":
	default:
		return nil, fmt.Errorf("config: invalid LOG_LEVEL %q (want debug, info, warn or error)", cfg.LogLevel)
	}

	return cfg, nil
}
