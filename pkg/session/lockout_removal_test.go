package session

import (
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// VULN-024, Option D: the per-account lockout is removed. Ten-plus wrong
// passwords for a victim from IP A must NOT keep the correct password from
// IP B out (this fails on main, which locks the account); the per-IP
// lockout still holds IP A closed (the control).
func TestPerAccountLockoutRemoved(t *testing.T) {
	database := testutil.NewMockDatabase()
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: game.MortalStartRoom, Name: "Start", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreatePlayer(&db.PlayerRecord{
		Name: "Victim", Password: string(hash), RoomVNum: game.MortalStartRoom,
		Level: 1, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassWarrior, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
	}); err != nil {
		t.Fatal(err)
	}

	// IP A: 12 wrong passwords — past the old 10-failure account threshold.
	// One connection per attempt (the realistic brute-force shape): each
	// failed client disconnects, which releases its entry-name claim.
	for i := 0; i < 12; i++ {
		s := makeCharSession(t, m)
		s.request = &http.Request{RemoteAddr: "10.9.9.1:1000"}
		if err, panicked := callHandleLogin(s, loginMsg("Victim", "wrongpass")); panicked || err != nil {
			t.Fatalf("wrong attempt %d: %v", i, err)
		}
		s.releaseEntryName() // the disconnect cleanup a real client triggers
	}

	// IP B: the correct password must now log in.
	s := makeCharSession(t, m)
	s.request = &http.Request{RemoteAddr: "10.9.9.2:1000"}
	if err, panicked := callHandleLogin(s, loginMsg("Victim", "secret123")); panicked || err != nil {
		t.Fatalf("correct login from a different IP: (%v, %v)", err, panicked)
	}
	if !s.authenticated {
		t.Fatal("correct password rejected after another IP's failures — per-account lockout still active")
	}

	// Control: IP A itself is closed by the per-IP lockout (its own failures).
	sA := makeCharSession(t, m)
	sA.request = &http.Request{RemoteAddr: "10.9.9.1:1000"}
	if err, panicked := callHandleLogin(sA, loginMsg("Victim", "secret123")); panicked || err != nil {
		t.Fatalf("IP-A control: (%v, %v)", err, panicked)
	}
	if sA.authenticated {
		t.Fatal("per-IP lockout did not hold the offending IP closed")
	}
}
