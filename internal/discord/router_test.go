package discord

import (
	"testing"

	"github.com/teyhouse/discord-bot-golang/internal/config"
)

// Regression test: the checker built from configuration must reflect the
// configured whitelist. A previous version of NewRouter silently fell back
// to an empty whitelist (public mode) when no explicit checker was passed,
// making WHITELISTED_USER_IDS a no-op.
func TestPermissionCheckerFor(t *testing.T) {
	cfg := &config.Config{WhitelistedUsers: []string{"138372729660243968"}}

	checker := PermissionCheckerFor(cfg)
	if !checker.Allowed("138372729660243968") {
		t.Error("configured user must be allowed")
	}
	if checker.Allowed("0138372729660243968") {
		t.Error("modified ID (leading zero) must be denied")
	}
	if checker.Allowed("999999999999999999") {
		t.Error("unknown user must be denied")
	}

	public := PermissionCheckerFor(&config.Config{})
	if !public.Allowed("anyone") {
		t.Error("empty whitelist must allow everyone")
	}
}
