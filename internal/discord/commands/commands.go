// Package commands implements the bot's slash commands.
package commands

import (
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
}

// PingHandler responds publicly with "Pong".
func PingHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	middleware.Respond(s, i, "Pong", false)
}

// WhisperHandler responds ephemerally so only the invoker sees it.
func WhisperHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	middleware.Respond(s, i, "pssst, only visible for you", true)
}
