package handlers

import (
	"strings"
	"testing"
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
