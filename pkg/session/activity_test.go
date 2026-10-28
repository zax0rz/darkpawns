package session

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

func TestActivityLoginAndAcceptedCreationOnly(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Returner")
	m := entryWorldManager(t, database)
	h := telemetry.New(time.Now())
	failed := makeCharSession(t, m)
	if err := failed.handleLogin(loginMsg("Returner", "wrong")); err != nil {
		t.Fatal(err)
	}
	h.Sample(time.Now(), 0)
	if len(h.Snapshot().Activity) != 0 {
		t.Fatal("failed authentication recorded")
	}
	s := failed
	if err := s.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if !s.authenticated {
		t.Fatalf("returning authentication did not succeed: stage=%s output=%s", s.charStage, renderedOutput(s))
	}
	created := makeCharSession(t, m)
	created.charName = "Mira"
	created.charPassword = "hash"
	created.charClass = 0
	created.charRace = 0
	if err := created.persistAcceptedCharacter(); err != nil {
		t.Fatal(err)
	}
	if err := created.persistAcceptedCharacter(); err != nil {
		t.Fatal(err)
	}
	bad := makeCharSession(t, m)
	bad.charName = ""
	if err := bad.persistAcceptedCharacter(); err == nil {
		t.Fatal("invalid character saved")
	}
	h.Sample(time.Now(), 0)
	events := h.Snapshot().Activity
	if len(events) != 2 || events[0].Player != "Returner" || events[0].Kind != "login" || events[0].Class == "" || events[1].Player != "Mira" || events[1].Class != "Mage" || events[1].Kind != "character" {
		t.Fatalf("incorrect activity: %+v", events)
	}
}
