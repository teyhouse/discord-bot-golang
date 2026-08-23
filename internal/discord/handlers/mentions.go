package handlers

import (
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// maxSummaryLen bounds how much of a mentioned message is echoed back.
const maxSummaryLen = 200

// MentionHandler replies to messages that directly @mention the bot with a
// short summary of the original message.
func MentionHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return // ignore self and webhooks without authors
	}
	if !mentionsBot(s, m.Message) {
		return
	}
	if _, err := s.ChannelMessageSendReply(m.ChannelID, Summarize(m.Content), m.Reference()); err != nil {
		slog.Error("handlers: replying to mention failed", "err", err)
	}
}

// mentionsBot reports whether one of the message's direct mentions is the bot.
func mentionsBot(s *discordgo.Session, m *discordgo.Message) bool {
	if s.State == nil || s.State.User == nil {
		return false
	}
	for _, u := range m.Mentions {
		if u.ID == s.State.User.ID {
			return true
		}
	}
	return false
}

// Summarize strips markdown, truncates to maxSummaryLen and prefixes the
// result so it reads as a summary of the original message.
func Summarize(content string) string {
	stripped := stripMarkdown(content)
	stripped = strings.Join(strings.Fields(stripped), " ")
	prefix := "Summary: "
	limit := maxSummaryLen - len(prefix)
	if len(stripped) > limit {
		stripped = truncateRunes(stripped, limit) + "…"
	}
	return prefix + stripped
}

// stripMarkdown removes common markdown markers (code fences, inline code,
// bold, italics, strikethrough, headers, quotes).
func stripMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"```", "", "`", "",
		"**", "", "__", "", "*", "", "_", "",
		"~~", "", "||", "",
		"#", "", "> ", "",
	)
	return replacer.Replace(s)
}

// truncateRunes cuts s to at most n runes without splitting a rune.
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
