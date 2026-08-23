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

// PingHandler responds publicly with "Pong".
func PingHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	middleware.Respond(s, i, "Pong", false)
}

// WhisperHandler responds ephemerally so only the invoker sees it.
func WhisperHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	middleware.Respond(s, i, "pssst, only visible for you", true)
}

// pnGreeting is the direct message sent by the /pn command.
const pnGreeting = "Hello there, nice to meet you! :wave:"

// PNHandler sends the invoker a private DM and confirms ephemerally.
func PNHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channel, err := s.UserChannelCreate(i.Member.User.ID)
	if err != nil {
		slog.Error("commands: opening DM channel failed", "user_id", i.Member.User.ID, "err", err)
		middleware.Respond(s, i, "Could not send you a message (are DMs allowed?)", true)
		return
	}
	if _, err := s.ChannelMessageSend(channel.ID, pnGreeting); err != nil {
		slog.Error("commands: sending DM failed", "user_id", i.Member.User.ID, "err", err)
		middleware.Respond(s, i, "Could not send you a message (are DMs allowed?)", true)
		return
	}
	middleware.Respond(s, i, "Sent you a message :envelope_with_arrow:", true)
}
