package config

import (
	"os"
	"testing"
)

func TestLoadValid(t *testing.T) {
	t.Setenv("BOT_TOKEN", "tok")
	t.Setenv("CHANNEL_ID", "1")
	t.Setenv("GUILD_ID", "2")
	t.Setenv("WHITELISTED_USER_IDS", " a , b ,, c ")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotToken != "tok" || cfg.ChannelID != "1" || cfg.GuildID != "2" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	want := []string{"a", "b", "c"}
	if len(cfg.WhitelistedUsers) != len(want) {
		t.Fatalf("whitelist = %v, want %v", cfg.WhitelistedUsers, want)
	}
	for i, id := range want {
		if cfg.WhitelistedUsers[i] != id {
			t.Errorf("whitelist[%d] = %q, want %q", i, cfg.WhitelistedUsers[i], id)
		}
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	for _, key := range []string{"BOT_TOKEN", "CHANNEL_ID", "GUILD_ID"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("BOT_TOKEN", "tok")
			t.Setenv("CHANNEL_ID", "1")
			t.Setenv("GUILD_ID", "2")
			t.Chdir(t.TempDir())
			os.Unsetenv(key)
			if _, err := Load(); err == nil {
				t.Errorf("expected error when %s is missing", key)
			}
		})
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	t.Setenv("BOT_TOKEN", "tok")
	t.Setenv("CHANNEL_ID", "1")
	t.Setenv("GUILD_ID", "2")
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := Load(); err == nil {
		t.Error("expected error for invalid LOG_LEVEL")
	}
}

func TestLoadEmptyWhitelistIsPublic(t *testing.T) {
	t.Setenv("BOT_TOKEN", "tok")
	t.Setenv("CHANNEL_ID", "1")
	t.Setenv("GUILD_ID", "2")
	t.Setenv("WHITELISTED_USER_IDS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.WhitelistedUsers) != 0 {
		t.Errorf("empty whitelist should yield no users, got %v", cfg.WhitelistedUsers)
	}
}
