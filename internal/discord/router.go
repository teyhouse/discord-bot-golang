package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/teyhouse/discord-bot-golang/internal/config"
	"github.com/teyhouse/discord-bot-golang/internal/discord/commands"
	"github.com/teyhouse/discord-bot-golang/internal/discord/handlers"
	"github.com/teyhouse/discord-bot-golang/internal/discord/middleware"
)

// Router is the central registration point for slash commands and event
// handlers.
type Router struct {
	client      *Client
	config      *config.Config
	logger      *slog.Logger
	checker     middleware.PermissionChecker
	cmdHandlers map[string]middleware.HandlerFunc
	public      map[string]struct{}
}

// NewRouter builds a router over an unstarted client. checker may be nil,
// in which case a whitelist checker over the configured user IDs is used
// (an empty WHITELISTED_USER_IDS yields public mode).
func NewRouter(client *Client, cfg *config.Config, logger *slog.Logger, checker middleware.PermissionChecker) *Router {
	if checker == nil {
		checker = PermissionCheckerFor(cfg)
	}
	cmds := commands.New(logger)
	return &Router{
		client:  client,
		config:  cfg,
		logger:  logger,
		checker: checker,
		cmdHandlers: map[string]middleware.HandlerFunc{
			"ping":    cmds.PingHandler,
			"whisper": cmds.WhisperHandler,
			"summary": cmds.SummaryHandler,
			"pn":      cmds.PNHandler,
		},
		public: make(map[string]struct{}),
	}
}

// PermissionCheckerFor selects the permission model for a configuration:
// an exact whitelist of the configured user IDs. An empty whitelist
// produces a checker that allows everyone (public bot).
func PermissionCheckerFor(cfg *config.Config) middleware.PermissionChecker {
	return middleware.NewWhitelistChecker(cfg.WhitelistedUsers)
}

// RegisterCommands creates all guild-scoped slash commands so they appear
// immediately after startup.
func (r *Router) RegisterCommands() error {
	appID, err := r.client.ApplicationID()
	if err != nil {
		return err
	}
	for _, cmd := range commands.Commands {
		created, err := r.client.Session().ApplicationCommandCreate(appID, r.config.GuildID, cmd)
		if err != nil {
			return fmt.Errorf("discord: registering command %q: %w", cmd.Name, err)
		}
		r.logger.Info("discord: registered command", "name", created.Name, "id", created.ID)
	}
	return nil
}

// RegisterHandlers wires command interactions and message events.
func (r *Router) RegisterHandlers() {
	session := r.client.Session()
	session.AddHandler(r.onInteraction)
	session.AddHandler(handlers.MentionHandler)
}

// onInteraction dispatches a command interaction through its handler,
// wrapped with the permission check unless the command is registered as
// public.
func (r *Router) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	handler, ok := r.cmdHandlers[name]
	if !ok {
		return
	}
	if _, isPublic := r.public[name]; isPublic {
		handler(s, i)
		return
	}
	middleware.WithPermissionCheck(r.checker, r.logger, name, handler)(s, i)
}

// RegisterPublicCommand opts a registered command out of the permission check.
func (r *Router) RegisterPublicCommand(name string) {
	if _, ok := r.cmdHandlers[name]; !ok {
		r.logger.Warn("discord: public override for unknown command", "name", name)
		return
	}
	r.public[name] = struct{}{}
}
