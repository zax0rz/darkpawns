package session

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/events"
	"github.com/zax0rz/darkpawns/pkg/notify"
)

func sessionNotificationService(t *testing.T, m *Manager) *notify.Service {
	t.Helper()
	t.Setenv("DP_NOTIFY_PUSHOVER_TOKEN", strings.Repeat("a", 30))
	t.Setenv("DP_NOTIFY_PUSHOVER_USER", strings.Repeat("b", 30))
	service, err := notify.New(filepath.Join(t.TempDir(), "notify.json"))
	if err != nil {
		t.Fatal(err)
	}
	c := service.View().Config
	c.Pushover = notify.Destination{Enabled: true, Events: []string{notify.Created, notify.Login, notify.Logout, notify.Level, notify.Death}}
	if err := service.Save(c); err != nil {
		t.Fatal(err)
	}
	m.SetNotifications(service)
	return service
}

func TestNotificationsFollowWorldEntryAndExit(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Returner")
	m := entryWorldManager(t, database)
	service := sessionNotificationService(t, m)
	s := makeCharSession(t, m)
	if err := s.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	renderedOutput(s)
	sendMenuInput(t, s, "")
	renderedOutput(s)
	if service.View().Status[notify.Pushover].Queued != 0 {
		t.Fatal("menu authentication sent login")
	}
	sendMenuInput(t, s, "1")
	renderedOutput(s)
	if service.View().Status[notify.Pushover].Queued != 1 {
		t.Fatal("world entry did not notify")
	}
	// Leaving through extraction (quit, rent, death) emits one logout, even when
	// transport cleanup follows. Repeated cleanup cannot enqueue another.
	m.world.QueuePlayerExtraction(s.player)
	m.drainPendingExtractions()
	if service.View().Status[notify.Pushover].Queued != 2 {
		t.Fatal("world departure did not notify")
	}
	m.notifyLeftWorld(s, "Returner")
	m.notifyLeftWorld(s, "Returner")
	if service.View().Status[notify.Pushover].Queued != 2 {
		t.Fatal("duplicate logout")
	}
}

func TestNotificationMilestoneObserversAndGuestPrivacy(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Returner")
	m := entryWorldManager(t, database)
	service := sessionNotificationService(t, m)
	s := makeCharSession(t, m)
	if err := s.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	renderedOutput(s)
	sendMenuInput(t, s, "")
	renderedOutput(s)
	sendMenuInput(t, s, "1")
	renderedOutput(s)
	publish := func() {
		t.Helper()
		for _, ev := range []events.BusEvent{events.PlayerLeveledEvent{PlayerID: "Returner", NewLevel: 20}, events.PlayerDiedEvent{PlayerID: "Returner"}} {
			if err := m.world.Events.Publish(context.Background(), ev); err != nil {
				t.Fatal(err)
			}
		}
	}
	before := service.View().Status[notify.Pushover].Queued
	publish()
	if service.View().Status[notify.Pushover].Queued != before+2 {
		t.Fatal("missing level/death observers")
	}
	s.isGuest = true
	publish()
	s.isGuest = false
	s.player.InvisLevel = 32
	publish()
	if service.View().Status[notify.Pushover].Queued != before+2 {
		t.Fatal("guest or invisible staff leaked")
	}
}

func TestCharacterCreationNotificationAfterSaveOnly(t *testing.T) {
	database := entryDatabase(t)
	m := entryWorldManager(t, database)
	service := sessionNotificationService(t, m)
	s := makeCharSession(t, m)
	s.charName = "Avery"
	s.charPassword = "hash"
	s.charClass = 0
	s.charRace = 0
	if err := s.persistAcceptedCharacter(); err != nil {
		t.Fatal(err)
	}
	if err := s.persistAcceptedCharacter(); err != nil {
		t.Fatal(err)
	}
	if service.View().Status[notify.Pushover].Queued != 1 {
		t.Fatal("creation notification missing or duplicated")
	}
	failed := makeCharSession(t, m)
	failed.charName = ""
	if err := failed.persistAcceptedCharacter(); err == nil {
		t.Fatal("invalid creation unexpectedly saved")
	}
	if service.View().Status[notify.Pushover].Queued != 1 {
		t.Fatal("failed creation notified")
	}
}
