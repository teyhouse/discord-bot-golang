// Package commands implements the bot's slash commands.
package commands

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/teyhouse/discord-bot-golang/internal/discord/middleware"
)

// Commands lists every slash command registered on startup.
// Keep this in sync with the handlers wired up in the router.
var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "Responds with Pong",
	},
	{
		Name:        "whisper",
		Description: "Sends a private message only you can see",
	},
	{
		Name:        "summary",
		Description: "Short overview of the last 20 messages in this channel",
	},
	{
		Name:        "pn",
		Description: "Sends you a direct message",
	},
}

// Commands carries the dependencies shared by all slash-command handlers.
type Registry struct {
	log *slog.Logger
}

// New builds the command handlers. A nil logger falls back to slog.Default.
func New(log *slog.Logger) *Registry {
	if log == nil {
		log = slog.Default()
	}
	return &Registry{log: log}
}

// PingHandler responds publicly with "Pong".
func (c *Registry) PingHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	middleware.Respond(c.log, s, i, "Pong", false)
}

// WhisperHandler responds ephemerally so only the invoker sees it.
func (c *Registry) WhisperHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	middleware.Respond(c.log, s, i, "pssst, only visible for you", true)
}

// pnGreeting is the direct message sent by the /pn command.
const pnGreeting = "Hello there, nice to meet you! :wave:"

// PNHandler sends the invoker a private DM and confirms ephemerally. It
// defers the interaction first: opening a DM channel and sending are two
// REST round trips, which together can exceed Discord's 3-second window
// for direct responses.
func (c *Registry) PNHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.deferEphemeral(s, i) {
		return
	}
	if i.Member == nil || i.Member.User == nil {
		c.followUp(s, i, "Could not send you a message (are DMs allowed?)")
		return
	}
	channel, err := s.UserChannelCreate(i.Member.User.ID)
	if err != nil {
		c.log.Error("commands: opening DM channel failed", "user_id", i.Member.User.ID, "err", err)
		c.followUp(s, i, "Could not send you a message (are DMs allowed?)")
		return
	}
	if _, err := s.ChannelMessageSend(channel.ID, pnGreeting); err != nil {
		c.log.Error("commands: sending DM failed", "user_id", i.Member.User.ID, "err", err)
		c.followUp(s, i, "Could not send you a message (are DMs allowed?)")
		return
	}
	c.followUp(s, i, "Sent you a message :envelope_with_arrow:")
}

// deferEphemeral acknowledges the interaction with an ephemeral deferred
// response so slow work can continue past the 3-second limit. It reports
// whether the acknowledgement succeeded; on failure it logs and gives up,
// since followups require a valid token from the deferred response.
func (c *Registry) deferEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		c.log.Error("commands: deferring interaction failed", "err", err)
		return false
	}
	return true
}

// followUp posts the final result of a deferred interaction as an
// ephemeral followup message.
func (c *Registry) followUp(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		c.log.Error("commands: interaction followup failed", "err", err)
	}
}
