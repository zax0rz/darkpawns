package session

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

// DP-1371 batch 8 — the five interpreter.c rows in the reconnect / connect /
// character-creation family.
//
// C (src/interpreter.c) → Go producer, all file TRUE:
//
//	:1641  "%s [%s] has reconnected."                        NRM / MAX(LVL_IMMORT, invis)
//	       -> reconnect.go:178, finishDupeCheck's default arm (linkdead resume)
//	:1650  "%s has re-logged in ... disconnecting old socket." NRM / MAX(LVL_IMMORT, invis)
//	       -> reconnect.go:171, the usurp arm (the old body is taken over)
//	:1655  "%s [%s] has reconnected."                        NRM / MAX(LVL_IMMORT, invis)
//	       -> reconnect.go:165, the unswitch arm
//	:1927  "%s [%s] has connected."                          BRF / MAX(LVL_IMMORT, invis)
//	       -> session_login.go:312, handleLogin
//	:2155  "%s [%s] new player."                             NRM / LVL_IMMORT (no invis term)
//	       -> char_creation.go:363, handleCharInput at the stats_roll "Y" arm
//
// Arm→C-line attribution is read from the switch itself
// (pkg/session/reconnect.go:158-178): `case unswitch` logs the reconnect line at
// :165, `case usurp` logs the re-logged-in line at :171, and the default arm —
// the linkdead resume — logs the reconnect line after its Act at :178. The :1650
// payload is unique; of the two identical reconnect texts, C's :1641 is the
// linkdead resume and C's :1655 the unswitch arm.
//
// Every assertion names that producer's exact bracketed bytes, so a sibling line
// in the same flow cannot satisfy it.

// i8Rig is the Batch 8 observer set on a manager: log levels 1 and 2, no flags,
// the LVL_IMMORT floor pair and the invisibility pair.
type i8Rig struct {
	log1, log2, none, floor, under, at36, under36 *Session
}

func newI8Rig(t *testing.T, m *Manager) (i8Rig, *bytes.Buffer) {
	t.Helper()
	file := captureMudlogFile(t)
	return i8Rig{
		log1:    addObserver(t, m, "I8log1", 40, true, false),
		log2:    addObserver(t, m, "I8log2", 40, false, true),
		none:    addObserver(t, m, "I8none", 40, false, false),
		floor:   addObserver(t, m, "I8floor", game.LVL_IMMORT, true, true),
		under:   addObserver(t, m, "I8under", game.LVL_IMMORT-1, true, true),
		at36:    addObserver(t, m, "I8at36", 36, true, true),
		under36: addObserver(t, m, "I8under36", 35, true, true),
	}, file
}

// i8Saw reports whether a session received this producer's exact bracketed line.
func i8Saw(t *testing.T, s *Session, payload string) bool {
	t.Helper()
	return strings.Contains(strings.Join(drainSessionText(t, s), ""), "[ "+payload+" ]\r\n")
}

func reconnectedPayload(name, host string) string { return name + " [" + host + "] has reconnected." }

func reloggedPayload(name string) string {
	return name + " has re-logged in ... disconnecting old socket."
}
func connectedPayload(name, host string) string { return name + " [" + host + "] has connected." }
func newPlayerPayload(name, host string) string { return name + " [" + host + "] new player." }

// TestReconnectMudlogProducers drives the three finishDupeCheck arms and pins
// payload, file and the negatives for each producer line.
func TestReconnectMudlogProducers(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(m *Manager, old, fresh *Session)
		payload func(host string) string
	}{
		{
			"linkdead_resume_1641",
			func(m *Manager, old, fresh *Session) {
				old.player.SetLinkless(true)
				old.DetachTransport()
				old.player.SetPosition(game.PosResting)
			},
			func(host string) string { return reconnectedPayload("Returner", host) },
		},
		{
			"usurp_1650",
			func(m *Manager, old, fresh *Session) {},
			func(host string) string { return reloggedPayload("Returner") },
		},
		{
			"unswitch_1655",
			func(m *Manager, old, fresh *Session) { old.isSwitched = true },
			func(host string) string { return reconnectedPayload("Returner", host) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, old, _, fresh := dupeCheckFixture(t)
			rig, file := newI8Rig(t, m)
			tc.prepare(m, old, fresh)
			if !fresh.performDupeCheck() {
				t.Fatal("performDupeCheck did not reconnect")
			}
			payload := tc.payload(fresh.MudHost())
			if got := strings.Count(file.String(), payload); got != 1 {
				t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
			}
			if !i8Saw(t, rig.log2, payload) {
				t.Fatalf("LOG2-only observer did not receive %q", bracketOf(payload))
			}
			if i8Saw(t, rig.log1, payload) {
				t.Fatalf("LOG1-only observer received %q: type <= BRF", bracketOf(payload))
			}
			if i8Saw(t, rig.none, payload) {
				t.Fatalf("no-flag observer received %q: type would be OFF", bracketOf(payload))
			}
			if i8Saw(t, rig.under, payload) {
				t.Fatalf("level-%d observer received %q", game.LVL_IMMORT-1, bracketOf(payload))
			}
		})
	}
}

// bracketOf is the consumer byte shape used by the session-side assertions.
func bracketOf(payload string) string { return "[ " + payload + " ]\r\n" }

// TestReconnectMudlogLevelDiscriminators: MAX(LVL_IMMORT, invis) on each arm —
// the LVL_IMMORT floor and an invisibility above it.
func TestReconnectMudlogLevelDiscriminators(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(old *Session)
		payload func(host string) string
	}{
		{"linkdead_1641", func(old *Session) {
			old.player.SetLinkless(true)
			old.DetachTransport()
		}, func(host string) string { return reconnectedPayload("Returner", host) }},
		{"usurp_1650", func(old *Session) {}, func(host string) string { return reloggedPayload("Returner") }},
		{"unswitch_1655", func(old *Session) { old.isSwitched = true }, func(host string) string { return reconnectedPayload("Returner", host) }},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/lv_immort_floor", func(t *testing.T) {
			m, old, _, fresh := dupeCheckFixture(t)
			rig, _ := newI8Rig(t, m)
			tc.prepare(old)
			if !fresh.performDupeCheck() {
				t.Fatal("performDupeCheck did not reconnect")
			}
			payload := tc.payload(fresh.MudHost())
			if i8Saw(t, rig.under, payload) {
				t.Fatalf("level-%d observer received %q", game.LVL_IMMORT-1, bracketOf(payload))
			}
			if !i8Saw(t, rig.floor, payload) {
				t.Fatalf("level-%d observer did not receive %q", game.LVL_IMMORT, bracketOf(payload))
			}
		})
		t.Run(tc.name+"/invis_raises_threshold", func(t *testing.T) {
			m, old, _, fresh := dupeCheckFixture(t)
			rig, _ := newI8Rig(t, m)
			tc.prepare(old)
			old.player.SetInvisLevel(36)
			if !fresh.performDupeCheck() {
				t.Fatal("performDupeCheck did not reconnect")
			}
			payload := tc.payload(fresh.MudHost())
			if i8Saw(t, rig.under36, payload) {
				t.Fatalf("level-35 observer received %q", bracketOf(payload))
			}
			if !i8Saw(t, rig.at36, payload) {
				t.Fatalf("level-36 observer did not receive %q at invisibility 36", bracketOf(payload))
			}
		})
	}
}

// connectFixture seeds a database-backed player with a bcrypt hash so the login
// flow reaches the connect producer (session_login.go:312).
func connectFixture(t *testing.T) (*Manager, string) {
	t.Helper()
	database := testutil.NewMockDatabase()
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	rec := &db.PlayerRecord{
		Name: "Connected", Password: string(hash), RoomVNum: game.MortalStartRoom,
		Level: 1, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassWarrior, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(rec); err != nil {
		t.Fatal(err)
	}
	return m, "Connected"
}

// TestLoginConnectMudlog pins the :1927 producer (payload, file, type, floor).
func TestLoginConnectMudlog(t *testing.T) {
	m, name := connectFixture(t)
	rig, file := newI8Rig(t, m)
	s := makeCharSession(t, m)
	if err, panicked := callHandleLogin(s, loginMsg(name, "secret123")); panicked || err != nil {
		t.Fatalf("callHandleLogin = (%v, panicked=%v)", err, panicked)
	}
	payload := connectedPayload(name, s.MudHost())
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
	}
	// BRF: reaches a LOG1-only observer, never a no-flag one.
	if !i8Saw(t, rig.log1, payload) {
		t.Fatalf("LOG1-only observer did not receive %q", bracketOf(payload))
	}
	if i8Saw(t, rig.none, payload) {
		t.Fatalf("no-flag observer received %q: type would be OFF", bracketOf(payload))
	}
	if i8Saw(t, rig.under, payload) {
		t.Fatalf("level-%d observer received %q", game.LVL_IMMORT-1, bracketOf(payload))
	}
	if !i8Saw(t, rig.floor, payload) {
		t.Fatalf("level-%d observer did not receive %q", game.LVL_IMMORT, bracketOf(payload))
	}
}

// TestNewPlayerMudlog pins the :2155 producer, whose level is a fixed
// LVL_IMMORT (C has no invisibility term here, and a creation session has no
// player to carry one).
func TestNewPlayerMudlog(t *testing.T) {
	m := makeTestManager(t)
	rig, file := newI8Rig(t, m)
	s := makeCharSession(t, m)
	s.charCreating = true
	s.charStage = "stats_roll"
	s.charName = "Newbie"
	sendCharInput(t, s, "Y")

	payload := newPlayerPayload("Newbie", s.MudHost())
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
	}
	if !i8Saw(t, rig.log2, payload) {
		t.Fatalf("LOG2-only observer did not receive %q", bracketOf(payload))
	}
	if i8Saw(t, rig.log1, payload) {
		t.Fatalf("LOG1-only observer received %q: type <= BRF", bracketOf(payload))
	}
	if i8Saw(t, rig.under, payload) {
		t.Fatalf("level-%d observer received %q", game.LVL_IMMORT-1, bracketOf(payload))
	}
	if !i8Saw(t, rig.floor, payload) {
		t.Fatalf("level-%d observer did not receive %q", game.LVL_IMMORT, bracketOf(payload))
	}
}
