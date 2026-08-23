package handlers

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestSummarizeTruncatesAndPrefixes(t *testing.T) {
	got := Summarize(strings.Repeat("x", 500))
	if !strings.HasPrefix(got, "Summary: ") {
		t.Errorf("missing prefix: %q", got[:20])
	}
	if len([]rune(got)) > 201 { // 200 chars + ellipsis
		t.Errorf("summary too long: %d runes", len([]rune(got)))
	}
}

func TestSummarizeStripsMarkdown(t *testing.T) {
	got := Summarize("**bold** `code` ~~gone~~ # header\n> quote _it_")
	want := "Summary: bold code gone header quote it"
	if got != want {
		t.Errorf("Summarize = %q, want %q", got, want)
	}
}

func TestReplaceMentions(t *testing.T) {
	mentions := []*discordgo.User{
		{ID: "123", Username: "alice"},
	}
	got := ReplaceMentions("<@123> and <@!123> ping <@999>", mentions)
	want := "@alice and @alice ping <@999>"
	if got != want {
		t.Errorf("ReplaceMentions = %q, want %q", got, want)
	}
}

func TestSummarizeWithMentionThenStrip(t *testing.T) {
	content := ReplaceMentions("<@123> **Interaction Test**", []*discordgo.User{{ID: "123", Username: "alice"}})
	if got, want := Summarize(content), "Summary: @alice Interaction Test"; got != want {
		t.Errorf("Summarize = %q, want %q", got, want)
	}
}

func TestStripBotMention(t *testing.T) {
	got := stripBotMention("<@123> hello there", "123")
	if want := "hello there"; got != want {
		t.Errorf("stripBotMention = %q, want %q", got, want)
	}
	got = stripBotMention("<@!123>hello<@123>", "123")
	if want := "hello"; got != want {
		t.Errorf("stripBotMention = %q, want %q", got, want)
	}
}

func TestSummarizeSkipsBotTag(t *testing.T) {
	content := stripBotMention("<@123> **Interaction Test**", "123")
	if got, want := Summarize(content), "Summary: Interaction Test"; got != want {
		t.Errorf("Summarize = %q, want %q", got, want)
	}
}

func TestTruncateRunesDoesNotSplitRune(t *testing.T) {
	s := strings.Repeat("é", 10)
	got := truncateRunes(s, 5)
	if len(got) != 5*2 || got != strings.Repeat("é", 5) {
		t.Errorf("truncateRunes split a rune: %q", got)
	}
}

func FuzzSummarize(f *testing.F) {
	f.Add("**hello** world")
	f.Add(strings.Repeat("a", 300))
	f.Add("")
	f.Fuzz(func(t *testing.T, content string) {
		got := Summarize(content)
		if !strings.HasPrefix(got, "Summary: ") {
			t.Fatalf("output lost prefix for input %q", content)
		}
	})
}
