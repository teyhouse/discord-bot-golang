package middleware

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestWhitelistCheckerAllowed(t *testing.T) {
	w := NewWhitelistChecker([]string{"a", "b"})
	if !w.Allowed("a") || !w.Allowed("b") {
		t.Error("listed users should be allowed")
	}
	if w.Allowed("c") {
		t.Error("unlisted user should be denied")
	}
}

func TestWhitelistCheckerEmptyIsPublic(t *testing.T) {
	w := NewWhitelistChecker(nil)
	if !w.Allowed("anyone") || !w.Allowed("") {
		t.Error("empty whitelist must allow everyone")
	}
}

func TestWithPermissionCheckDeniesEphemeral(t *testing.T) {
	checker := NewWhitelistChecker([]string{"allowed"})
	var called bool
	wrapped := WithPermissionCheck(checker, nil, "test", func(*discordgo.Session, *discordgo.InteractionCreate) { called = true })

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{User: &discordgo.User{ID: "denied"}},
		},
	}
	wrapped(nil, i)
	if called {
		t.Error("handler must not run for denied user")
	}
}

func TestWithPermissionCheckAllowsMember(t *testing.T) {
	checker := NewWhitelistChecker([]string{"allowed"})
	var called bool
	wrapped := WithPermissionCheck(checker, nil, "test", func(*discordgo.Session, *discordgo.InteractionCreate) { called = true })

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{User: &discordgo.User{ID: "allowed"}},
		},
	}
	wrapped(nil, i)
	if !called {
		t.Error("handler must run for allowed user")
	}
}

func TestWithPermissionCheckNilMemberDenied(t *testing.T) {
	checker := NewWhitelistChecker([]string{"allowed"})
	var called bool
	wrapped := WithPermissionCheck(checker, nil, "test", func(*discordgo.Session, *discordgo.InteractionCreate) { called = true })

	wrapped(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{}})
	if called {
		t.Error("interaction without member must not reach the handler")
	}
}
