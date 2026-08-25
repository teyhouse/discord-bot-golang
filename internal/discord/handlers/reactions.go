// Package handlers implements Discord event handlers: the :eyes: reaction
// watcher and the @bot mention responder.
package handlers

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// eyesEmoji is the Unicode :eyes: emoji the watcher looks for.
const eyesEmoji = "\U0001F440"

// ReactionWatcher periodically scans a channel for recent messages with an
// :eyes: reaction and replies "I saw that!" once per message.
type ReactionWatcher struct {
	session   *discordgo.Session
	channelID string
	interval  time.Duration
	window    time.Duration
	seen      *SeenStore
	logger    *slog.Logger
}

// NewReactionWatcher creates a watcher polling every interval for messages
// from the last window (default 5 minutes) that carry :eyes:.
func NewReactionWatcher(s *discordgo.Session, channelID string, interval time.Duration, seen *SeenStore, logger *slog.Logger) *ReactionWatcher {
	return &ReactionWatcher{
		session:   s,
		channelID: channelID,
		interval:  interval,
		window:    5 * time.Minute,
		seen:      seen,
		logger:    logger,
	}
}

// Start runs the watch loop until ctx is cancelled. Single worker by design:
// scans never overlap, which keeps us well inside Discord rate limits.
func (w *ReactionWatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.checkOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkOnce(ctx)
		}
	}
}

// checkOnce fetches the most recent messages and replies to any unprocessed
// message in the window carrying an :eyes: reaction.
func (w *ReactionWatcher) checkOnce(ctx context.Context) {
	if w.session == nil {
		return
	}
	cutoff := time.Now().Add(-w.window)
	msgs, err := w.session.ChannelMessages(w.channelID, 100, "", "", "")
	if err != nil {
		w.logger.Error("handlers: fetching messages failed", "err", err)
		return
	}
	for _, m := range msgs {
		if ctx.Err() != nil {
			return
		}
		if m.Timestamp.Before(cutoff) {
			continue
		}
		if !hasEyes(m.Reactions) || w.seen.Seen(m.ID) {
			continue
		}
		content := strings.ReplaceAll(m.Content, "\n", "\n> ")
		quote := "> **" + m.Author.Username + "**: " + content + "\n\nI saw that!"
		if _, err := w.session.ChannelMessageSendReply(w.channelID, quote, m.Reference()); err != nil {
			w.logger.Error("handlers: replying to watched message failed", "message_id", m.ID, "err", err)
			return
		}
		if err := w.seen.Mark(m.ID); err != nil {
			w.logger.Error("handlers: persisting seen state failed", "message_id", m.ID, "err", err)
		} else {
			w.logger.Debug("handlers: acknowledged eyes reaction", "message_id", m.ID)
		}
	}
}

func hasEyes(reactions []*discordgo.MessageReactions) bool {
	for _, r := range reactions {
		if r.Emoji.Name == eyesEmoji && r.Count > 0 && !r.Me {
			return true
		}
	}
	return false
}
