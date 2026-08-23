package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// SeenStore tracks processed message IDs across restarts using one JSON
// file per day. It keeps the current day and the previous day in memory
// and deletes files older than two days.
type SeenStore struct {
	dir string

	mu  sync.Mutex
	day string          // YYYY-MM-DD of ids
	ids map[string]struct{} // message IDs seen today or yesterday
}

// NewSeenStore loads state for today and yesterday from dir, creating it if
// needed, and prunes files older than two days.
func NewSeenStore(dir string) (*SeenStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("handlers: creating state dir %s: %w", dir, err)
	}
	s := &SeenStore{dir: dir, ids: make(map[string]struct{})}
	if err := s.rotateLocked(time.Now()); err != nil {
		return nil, err
	}
	return s, nil
}

// Seen reports whether the message has already been processed.
func (s *SeenStore) Seen(messageID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.rotateLocked(time.Now()); err != nil {
		return false
	}
	_, ok := s.ids[messageID]
	return ok
}

// Mark records a message ID and persists today's state to disk.
func (s *SeenStore) Mark(messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if err := s.rotateLocked(now); err != nil {
		return err
	}
	if _, ok := s.ids[messageID]; ok {
		return nil
	}
	s.ids[messageID] = struct{}{}
	return s.persistLocked(now)
}

// rotateLocked switches the in-memory set to the new day when midnight
// passes: it keeps yesterday's IDs, drops anything older, and prunes stale
// files. The caller must hold s.mu.
func (s *SeenStore) rotateLocked(now time.Time) error {
	today := now.Format("2006-01-02")
	if s.day == today {
		return nil
	}

	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	kept := make(map[string]struct{})
	for _, day := range []string{yesterday, today} {
		ids, err := readFile(filepath.Join(s.dir, day+".json"))
		if err != nil {
			return err
		}
		for id := range ids {
			kept[id] = struct{}{}
		}
	}
	s.day = today
	s.ids = kept

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("handlers: reading state dir: %w", err)
	}
	cutoff := now.AddDate(0, 0, -2).Format("2006-01-02")
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		day := strings.TrimSuffix(name, ".json")
		if day >= cutoff {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, name)); err != nil {
			return fmt.Errorf("handlers: removing stale state file %s: %w", name, err)
		}
	}
	return nil
}

// persistLocked atomically writes today's state file. Caller holds s.mu.
func (s *SeenStore) persistLocked(now time.Time) error {
	path := filepath.Join(s.dir, s.day+".json")
	f, err := os.CreateTemp(s.dir, ".state-*")
	if err != nil {
		return fmt.Errorf("handlers: creating temp state file: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sortedKeys(s.ids)); err != nil {
		f.Close()
		return fmt.Errorf("handlers: encoding state: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("handlers: closing state file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("handlers: replacing state file: %w", err)
	}
	return nil
}

func readFile(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("handlers: reading state file %s: %w", path, err)
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("handlers: parsing state file %s: %w", path, err)
	}
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m, nil
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
