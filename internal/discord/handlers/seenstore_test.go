package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSeenStoreMarkAndSeen(t *testing.T) {
	dir := t.TempDir()
	s, err := NewSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Seen("m1") {
		t.Error("unknown message must not be marked seen")
	}
	if err := s.Mark("m1"); err != nil {
		t.Fatal(err)
	}
	if !s.Seen("m1") {
		t.Error("marked message should be seen")
	}

	reloaded, err := NewSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Seen("m1") {
		t.Error("state must survive restart")
	}
}

func TestSeenStorePrunesOldFiles(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	stale := map[string][]string{old: {"ancient"}}
	data, _ := json.Marshal(stale[old])
	if err := os.WriteFile(filepath.Join(dir, old+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewSeenStore(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, old+".json")); !os.IsNotExist(err) {
		t.Error("state file older than two days must be removed")
	}
}

func TestSeenStoreKeepsYesterday(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	data, _ := json.Marshal([]string{"old-msg"})
	if err := os.WriteFile(filepath.Join(dir, yesterday+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := NewSeenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Seen("old-msg") {
		t.Error("yesterday's IDs must remain visible")
	}
	if _, err := os.Stat(filepath.Join(dir, yesterday+".json")); err != nil {
		t.Errorf("yesterday's file must be kept: %v", err)
	}
}

func TestSeenStoreIgnoresForeignFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := NewSeenStore(dir)
	if err != nil {
		t.Fatalf("non-state files must not break loading: %v", err)
	}
	if err := s.Mark("x"); err != nil {
		t.Fatal(err)
	}
}
