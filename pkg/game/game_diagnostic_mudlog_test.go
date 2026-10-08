package game

import (
	"bytes"
	"strings"
	"testing"
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
