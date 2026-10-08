package game

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type diagnosticSessions struct {
	observer *Player
	messages []string
	probe    func()
}

func (s *diagnosticSessions) EachSession(fn func(interface{}, func(string))) {
	fn(s.observer, func(msg string) {
		if s.probe != nil {
			s.probe()
		}
		s.messages = append(s.messages, msg)
	})
}

func diagnosticCapture(t *testing.T, typ int) (*diagnosticSessions, *bytes.Buffer) {
	t.Helper()
	old := getImmortalSessionProvider()
	writer := getLogWriter()
	t.Cleanup(func() { SetImmortalSessionProvider(old); SetLogWriter(writer) })
	p := NewPlayer(99, "Observer", 1001)
	p.SetLevel(LVL_IMMORT)
	p.SetPlrFlag(PrfLog1, typ&1 != 0)
	p.SetPlrFlag(PrfLog2, typ&2 != 0)
	s := &diagnosticSessions{observer: p}
	SetImmortalSessionProvider(s)
	file := &bytes.Buffer{}
	SetLogWriter(file)
	return s, file
}

func requireDiagnostic(t *testing.T, s *diagnosticSessions, file *bytes.Buffer, payload string, toFile bool) {
	t.Helper()
	want := "[ " + payload + " ]\r\n"
	if len(s.messages) != 1 || s.messages[0] != want {
		t.Fatalf("diagnostic bytes=%q want %q", s.messages, want)
	}
	if strings.Contains(file.String(), payload) != toFile {
		t.Fatalf("file flag=%t bytes=%q", toFile, file.String())
	}
}

// src/fight.c:1300-1309: heal/bless first; NRM/31/file FALSE last.
func TestGameDiagnosticKillMilestone(t *testing.T) {
	w := newCounterProcsTestWorld(t)
	ch := NewPlayer(1, "Victor", 1001)
	ch.SetHP(1)
	ch.Kills = 5000
	other := NewPlayer(2, "Friend", 1001)
	other.SetHP(1)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatal(err)
	}
	if err := w.AddPlayer(other); err != nil {
		t.Fatal(err)
	}
	var events []string
	w.MessageSink = func(_ string, msg []byte) { events = append(events, string(msg)) }
	s, file := diagnosticCapture(t, MudlogNormal)
	s.probe = func() {
		if ch.GetHP() != ch.GetMaxHP() || other.GetHP() != other.GetMaxHP() || len(events) != 2 {
			t.Fatal("milestone logged before reward/blessing")
		}
		if !w.mu.TryLock() {
			t.Fatal("world lock held at producer")
		}
		w.mu.Unlock()
		if !ch.mu.TryLock() {
			t.Fatal("player lock held at producer")
		}
		ch.mu.Unlock()
	}
	w.counter_procs(ch, 5000)
	requireDiagnostic(t, s, file, "Victor hit 5000 kills.", false)
	s.messages = nil
	w.counter_procs(ch, 4999)
	if len(s.messages) != 0 {
		t.Fatal("non-milestone log")
	}
	for _, state := range []struct{ level, typ int }{{30, 2}, {31, 1}, {31, 0}} {
		s.observer.SetLevel(state.level)
		s.observer.SetPlrFlag(PrfLog1, state.typ&1 != 0)
		s.observer.SetPlrFlag(PrfLog2, state.typ&2 != 0)
		s.probe = nil
		s.messages = nil
		w.counter_procs(ch, 5000)
		if len(s.messages) != 0 {
			t.Fatalf("wrong NRM/31 gate %+v", state)
		}
	}
}

// src/fight.c:1318-1333: corpse short-circuits; only a mortal cross-room attacker logs.
func TestGameDiagnosticCrossRoomDamage(t *testing.T) {
	w, players := newMessageTestWorld(t)
	ch, victim := players[0], players[2]
	ch.SetLevel(20)
	victim.SetLevel(20)
	victim.SetPosition(combat.PosStanding)
	s, file := diagnosticCapture(t, MudlogNormal)
	s.probe = func() {
		if !w.mu.TryLock() {
			t.Fatal("world locked at cross-room log")
		}
		w.mu.Unlock()
	}
	hp := victim.GetHP()
	if !w.DamageRefused(ch, victim) || victim.GetHP() != hp {
		t.Fatal("cross-room damage not refused before mutation")
	}
	requireDiagnostic(t, s, file, "Attempt to assign damage when ch and vict are in different rooms.", false)
	s.messages = nil
	ch.SetLevel(LVL_IMMORT)
	w.DamageRefused(ch, victim)
	if len(s.messages) != 0 {
		t.Fatal("immortal cross-room diagnostic")
	}
	ch.SetLevel(20)
	victim.SetPosition(combat.PosDead)
	w.DamageRefused(ch, victim)
	if len(s.messages) != 0 {
		t.Fatal("corpse guard must precede cross-room log")
	}
	victim.SetPosition(combat.PosStanding)
	for _, typ := range []int{0, 1} {
		s.observer.SetPlrFlag(PrfLog1, typ == 1)
		s.observer.SetPlrFlag(PrfLog2, false)
		w.DamageRefused(ch, victim)
		if len(s.messages) != 0 {
			t.Fatal("cross-room diagnostic wrong type")
		}
	}
}

// src/act.movement.c:261-292; src/utils.c:145-148: destination look,
// then BRF/31/file TRUE before the death cry and deferred extraction.
func TestGameDiagnosticDeathTrapMovement(t *testing.T) {
	w, players := newMessageTestWorld(t)
	ch := players[0]
	ch.SetLevel(10)
	ch.SetMove(100)
	w.CreateRoomExit(1001, "north", 1002)
	w.SetRoomFlagBit(1002, 1)
	s, file := diagnosticCapture(t, MudlogBrief)
	var events []string
	w.MovementLook = func(p *Player) {
		if p == ch {
			events = append(events, "look")
		}
	}
	w.MessageSink = func(_ string, msg []byte) { events = append(events, string(msg)) }
	s.probe = func() {
		if ch.GetRoom() != 1002 || ch.HasPLRFlag(PlrExtract) || len(events) == 0 || events[len(events)-1] != "look" {
			t.Fatalf("death-trap log boundary: room=%d extract=%t events=%q", ch.GetRoom(), ch.HasPLRFlag(PlrExtract), events)
		}
		if !w.mu.TryLock() {
			t.Fatal("world lock held at death-trap diagnostic")
		}
		w.mu.Unlock()
	}
	w.DoMove(ch, "north")
	requireDiagnostic(t, s, file, "Player1 hit death trap #1002 (Room B)", true)
	if !ch.HasPLRFlag(PlrExtract) || len(events) < 2 {
		t.Fatal("death trap did not cry then queue extraction")
	}
	s.messages = nil
	s.probe = nil
	file.Reset()
	players[1].SetLevel(LVL_IMMORT)
	w.DoMove(players[1], "north")
	if len(s.messages) != 0 || file.Len() != 0 {
		t.Fatal("immortal death-trap diagnostic")
	}
}
