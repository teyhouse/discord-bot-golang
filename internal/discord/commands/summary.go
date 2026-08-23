package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/teyhouse/discord-bot-golang/internal/discord/handlers"
	"github.com/teyhouse/discord-bot-golang/internal/discord/middleware"
)

// summaryMessageCount is how many recent messages the command inspects.
const summaryMessageCount = 20

// summarySnippetLen bounds each message to a short overview snippet.
const summarySnippetLen = 15

// SummaryHandler replies ephemerally with an overview of the last 20
// messages in the channel, excluding the bot's own and content-less ones.
func SummaryHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	msgs, err := s.ChannelMessages(i.ChannelID, summaryMessageCount, "", "", "")
	if err != nil {
		middleware.Respond(s, i, "Could not fetch messages", true)
		return
	}
	middleware.Respond(s, i, buildSummary(msgs, s.State.User.ID), true)
}

// buildSummary renders a numbered list of author and a short content
// snippet for every qualifying message, oldest first.
func buildSummary(msgs []*discordgo.Message, botID string) string {
	var lines []string
	n := 0
	for _, m := range msgs {
		if m.Author == nil || m.Author.ID == botID {
			continue // exclude the bot's own messages
		}
		content := strings.TrimSpace(handlers.ReplaceMentions(m.Content, m.Mentions))
		if content == "" {
			continue // skip embeds, images, etc.
		}
		n++
		lines = append(lines, fmt.Sprintf("%d. %s: %s", n, m.Author.Username, snippet(content)))
	}
	if len(lines) == 0 {
		return "No recent messages to summarize"
	}
	return strings.Join(lines, "\n")
}

// snippet truncates content to summarySnippetLen runes, appending an
// ellipsis when something was cut.
func snippet(content string) string {
	runes := []rune(strings.Join(strings.Fields(content), " "))
	if len(runes) <= summarySnippetLen {
		return string(runes)
	}
	return string(runes[:summarySnippetLen]) + "..."
}
