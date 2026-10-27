package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/notify"
)

func TestGossipMOTDOverlayIsOptInAndDoesNotEditCText(t *testing.T) {
	w, m, _, _ := shopOutputFixture(t, 1)
	s := makeTestSession(t, m, "NoticeReader", 1001, true)
	w.LibTextDir = t.TempDir()
	for _, name := range []string{"motd", "imotd", "help"} {
		if err := os.WriteFile(filepath.Join(w.LibTextDir, name), []byte("Original C text\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("DP_NOTIFY_GOSSIP_WEBHOOK", "https://discord.com/api/webhooks/123/notice-test-token")
	service, err := notify.New(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.SetNotifications(service)
	original := loginTextForFile(s, "motd")
	if original != "Original C text\r\n" {
		t.Fatalf("disabled bytes changed: %q", original)
	}
	config := service.View().Config
	config.Gossip = true
	if err = service.Save(config); err != nil {
		t.Fatal(err)
	}
	const notice = "\r\nWhat's said on gossip echoes to our Discord.\r\n"
	for _, file := range []string{"motd", "imotd"} {
		if got := loginTextForFile(s, file); got != original+notice {
			t.Fatalf("%s overlay=%q", file, got)
		}
	}
	if got := loginTextForFile(s, "help"); got != original {
		t.Fatal("notice escaped MOTD overlay", got)
	}
	// Returning-menu rendering uses the same overlay, including immortal MOTD.
	s.startReturningMenu("")
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if !strings.Contains(prompt.Prompt, "What's said on gossip echoes to our Discord.") {
		t.Fatal("login omitted gossip notice", prompt.Prompt)
	}
	config = service.View().Config
	config.GossipNotice = false
	if err = service.Save(config); err != nil {
		t.Fatal(err)
	}
	if !service.GossipEnabled() {
		t.Fatal("disabling the notice disabled the mirror")
	}
	for _, file := range []string{"motd", "imotd"} {
		if got := loginTextForFile(s, file); got != original {
			t.Fatalf("%s notice toggle left bytes behind: %q", file, got)
		}
	}
	config = service.View().Config
	config.GossipNotice = true
	config.Gossip = false
	if err = service.Save(config); err != nil {
		t.Fatal(err)
	}
	if got := loginTextForFile(s, "motd"); got != original {
		t.Fatal("disabled overlay left game bytes behind", got)
	}
	raw, err := os.ReadFile(filepath.Join(w.LibTextDir, "motd"))
	if err != nil || string(raw) != "Original C text\n" {
		t.Fatal("changed C text file", err)
	}
}
