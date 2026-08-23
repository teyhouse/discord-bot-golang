package commands

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestBuildSummaryExcludesBotAndEmpty(t *testing.T) {
	msgs := []*discordgo.Message{
		{Author: &discordgo.User{Username: "alice"}, Content: "first message"},
		{Author: &discordgo.User{ID: "bot-id", Username: "DevBot"}, Content: "bot noise"},
		{Author: &discordgo.User{Username: "bob"}, Content: "   "},
		{Author: &discordgo.User{Username: "carol"}, Content: "second message"},
	}
	got := buildSummary(msgs, "bot-id")
	want := "1. alice: first message\n2. carol: second message"
	if got != want {
		t.Errorf("buildSummary = %q, want %q", got, want)
	}
}

func TestBuildSummaryRewritesMentions(t *testing.T) {
	msgs := []*discordgo.Message{
		{
			Author:   &discordgo.User{ID: "u1", Username: "teyhouse"},
			Content:  "<@999> hello",
			Mentions: []*discordgo.User{{ID: "999", Username: "alice"}},
		},
	}
	got := buildSummary(msgs, "bot-id")
	want := "1. teyhouse: @alice hello"
	if got != want {
		t.Errorf("buildSummary = %q, want %q", got, want)
	}
}

func TestBuildSummaryEmpty(t *testing.T) {
	if got, want := buildSummary(nil, "bot"), "No recent messages to summarize"; got != want {
		t.Errorf("buildSummary = %q, want %q", got, want)
	}
}

func TestSnippetTruncatesAtFifteenRunes(t *testing.T) {
	if got := snippet(strings.Repeat("a", 20)); len(got) != 18 { // 15 chars + ...
		t.Errorf("snippet length = %d, want 18: %q", len(got), got)
	}
	if !strings.HasSuffix(snippet(strings.Repeat("a", 20)), "...") {
		t.Error("truncated snippet must end with ellipsis")
	}
	if got, want := snippet("short"), "short"; got != want {
		t.Errorf("snippet = %q, want %q", got, want)
	}
	multibyte := strings.Repeat("é", 20)
	if got := snippet(multibyte); len([]rune(got)) != 18 {
		t.Errorf("multibyte snippet split a rune: %q", got)
	}
}
