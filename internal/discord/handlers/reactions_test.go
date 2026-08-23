package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// TestReactionWatcherStartStopsOnCancel verifies the watch loop exits
// promptly on context cancellation without leaking goroutines (goleak in
// TestMain enforces this package-wide).
func TestReactionWatcherStartStopsOnCancel(t *testing.T) {
	w := &ReactionWatcher{
		session:   nil, // Start only selects on ctx/ticker until a tick fires
		channelID: "1",
		interval:  time.Hour,
		window:    5 * time.Minute,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not stop after context cancellation")
	}
}

func TestHasEyes(t *testing.T) {
	cases := []struct {
		name string
		rs   []*discordgo.MessageReactions
		want bool
	}{
		{"none", nil, false},
		{"other emoji", []*discordgo.MessageReactions{{Emoji: &discordgo.Emoji{Name: "👍"}, Count: 3}}, false},
		{"eyes", []*discordgo.MessageReactions{{Emoji: &discordgo.Emoji{Name: eyesEmoji}, Count: 2}}, true},
		{"eyes already reacted by bot", []*discordgo.MessageReactions{{Emoji: &discordgo.Emoji{Name: eyesEmoji}, Count: 2, Me: true}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasEyes(tc.rs); got != tc.want {
				t.Errorf("hasEyes = %v, want %v", got, tc.want)
			}
		})
	}
}
