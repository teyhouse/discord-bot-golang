// Command bot wires configuration, logging and Discord together.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teyhouse/discord-bot-golang/internal/config"
	"github.com/teyhouse/discord-bot-golang/internal/discord"
	"github.com/teyhouse/discord-bot-golang/internal/discord/handlers"
	"github.com/teyhouse/discord-bot-golang/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	client, err := discord.NewClient(cfg, log)
	if err != nil {
		log.Error("client init failed", "err", err)
		os.Exit(1)
	}

	for _, id := range cfg.WhitelistedUsers {
		if !isSnowflake(id) {
			log.Warn("config: whitelist entry does not look like a Discord user ID (snowflakes are numeric, no leading zero)", "entry", id)
		}
	}
	log.Debug("config: effective whitelist", "public_commands", []string{"ping"}, "whitelisted_users", cfg.WhitelistedUsers)

	router := discord.NewRouter(client, cfg, log, discord.PermissionCheckerFor(cfg))
	router.RegisterPublicCommand("ping")
	if err := router.RegisterCommands(); err != nil {
		log.Error("command registration failed", "err", err)
		os.Exit(1)
	}
	router.RegisterHandlers()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	seen, err := handlers.NewSeenStore(cfg.DataDir)
	if err != nil {
		log.Error("state store init failed", "err", err)
		os.Exit(1)
	}

	watcher := handlers.NewReactionWatcher(client.Session(), cfg.ChannelID, 30*time.Second, seen, log)
	go watcher.Start(ctx)

	// The blank net/http/pprof import above registers its handlers on
	// http.DefaultServeMux; nothing else ever registers there.
	pprofSrv := &http.Server{
		Addr:              "127.0.0.1:6060",
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		// pprof sidecar, internal only — never expose this port.
		if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("pprof listener exited", "err", err)
		}
	}()
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
			log.Warn("pprof shutdown failed", "err", err)
		}
	}()

	log.Info("bot starting", "guild_id", cfg.GuildID, "channel_id", cfg.ChannelID, "whitelist_enabled", len(cfg.WhitelistedUsers) > 0)
	if err := client.Start(ctx); err != nil {
		log.Error("bot stopped", "err", err)
		os.Exit(1)
	}
	log.Info("bot stopped cleanly")
}

// isSnowflake reports whether id looks like a Discord snowflake ID:
// digits only, no leading zero.
func isSnowflake(id string) bool {
	if id == "" || (len(id) > 1 && id[0] == '0') {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
