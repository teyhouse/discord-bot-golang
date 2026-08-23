// Package middleware provides composable permission checks that wrap
// slash-command handlers.
package middleware

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// PermissionChecker reports whether a user may invoke a guarded command.
type PermissionChecker interface {
	Allowed(userID string) bool
}

// WhitelistChecker grants access to an explicit set of user IDs.
// An empty whitelist means every user is allowed (public bot mode).
type WhitelistChecker struct {
	allowed map[string]struct{}
}

// NewWhitelistChecker builds a checker from trimmed user IDs.
func NewWhitelistChecker(userIDs []string) *WhitelistChecker {
	m := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		m[id] = struct{}{}
	}
	return &WhitelistChecker{allowed: m}
}

// Allowed reports whether userID is whitelisted. With no whitelist entries
// it always returns true, disabling permission checks entirely.
func (w *WhitelistChecker) Allowed(userID string) bool {
	if len(w.allowed) == 0 {
		return true
	}
	_, ok := w.allowed[userID]
	return ok
}

// HandlerFunc is the signature of a slash-command interaction handler.
type HandlerFunc func(s *discordgo.Session, i *discordgo.InteractionCreate)

// WithPermissionCheck wraps next so it only runs when checker allows the
// invoking member. Denied users get an ephemeral notice. Every decision is
// logged at Debug level with the command name for traceability.
func WithPermissionCheck(checker PermissionChecker, command string, next HandlerFunc) HandlerFunc {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		userID := ""
		if i.Member != nil && i.Member.User != nil {
			userID = i.Member.User.ID
		}
		if !checker.Allowed(userID) {
			slog.Debug("middleware: access denied", "command", command, "user_id", userID)
			Respond(s, i, "Not authorized", true)
			return
		}
		slog.Debug("middleware: access granted", "command", command, "user_id", userID)
		next(s, i)
	}
}

// Respond replies to an interaction with content; ephemeral makes it
// visible only to the invoking user.
func Respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	if s == nil || i == nil {
		slog.Error("middleware: cannot respond without session or interaction")
		return
	}
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	}); err != nil {
		slog.Error("middleware: interaction respond failed", "err", err)
	}
}
