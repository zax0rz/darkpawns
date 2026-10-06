package session

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// These are producer proofs for D7's first mudlog family: each exercises the
// real registered command handler on real *Session objects and asserts that
// the produced mudlog reaches the real consumer (the manager registers itself
// as game's immortal session provider, manager.go:433) at the exact C
// position, type, level and file flag. Removing any inserted game.MudLog call
// makes the matching assertion fail; a consumer-only test cannot.

// drainSessionText returns the player-facing text of every queued frame, in
// order. Both the direct command output (MsgText) and the manager's
// MessageSink path used by game.Player.SendMessage (MsgEvent type "text")
// are represented, matching readSessionText.
func drainSessionText(t *testing.T, s *Session) []string {
	t.Helper()
	var out []string
	for {
		select {
		case msg := <-s.send:
			var sm ServerMessage
			if err := json.Unmarshal(msg, &sm); err != nil {
				t.Fatalf("session message: %v", err)
			}
			data, _ := json.Marshal(sm.Data)
			switch sm.Type {
			case MsgText:
				var td TextData
				_ = json.Unmarshal(data, &td)
				out = append(out, td.Text)
			case MsgEvent:
				var ed EventData
				_ = json.Unmarshal(data, &ed)
				if ed.Type == "text" {
					out = append(out, ed.Text)
				}
			}
		default:
			return out
		}
	}
}

// captureMudlogFile redirects the alog/basic_mud_log file side to a buffer.
func captureMudlogFile(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	game.SetLogWriter(buf)
	t.Cleanup(func() { game.SetLogWriter(os.Stderr) })
	return buf
}

// mudlogFixture is a level-38 actor (satisfies LVL_GOD and LVL_GRGOD) with
// syslog normal, plus a mortal victim in the same room. Both are registered
// sessions, so the real provider sees them.
func mudlogFixture(t *testing.T) (m *Manager, actor, victim *Session, file *bytes.Buffer) {
	t.Helper()
	file = captureMudlogFile(t)
	m = makeTestManager(t)
	actor = makeTestSession(t, m, "Godactor", 1001, true)
	actor.player.SetLevel(38)
	actor.player.SetPlrFlag(game.PrfLog2, true) // syslog normal (level 2)
	registerTestSession(t, m, actor, "Godactor")
	victim = makeTestSession(t, m, "Victim", 1001, true)
	registerTestSession(t, m, victim, "Victim")
	return m, actor, victim, file
}

// TestWizutilMudlogProducerBytesAndOrder asserts the exact consumer line and
// its position relative to the command's other messages, per case.
func TestWizutilMudlogProducerBytesAndOrder(t *testing.T) {
	t.Run("pardon", func(t *testing.T) {
		_, actor, victim, file := mudlogFixture(t)
		victim.player.SetPlrFlag(game.PlrOutlaw, true)
		if err := cmdPardon(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		// act.wizard.c:2113-2121: clear flag, actor ack, victim line, then the
		// BRF log whose string has no final period.
		want := []string{
			"Pardoned.\r\n",
			"[ (GC) Victim pardoned by Godactor ]\r\n",
		}
		if got := drainSessionText(t, actor); !equalStrings(got, want) {
			t.Fatalf("actor lines = %q, want %q", got, want)
		}
		if got := drainSessionText(t, victim); !equalStrings(got, []string{"You have been pardoned by the Gods!\r\n"}) {
			t.Fatalf("victim lines = %q", got)
		}
		if !strings.Contains(file.String(), "(GC) Victim pardoned by Godactor\n") {
			t.Fatalf("file = %q", file.String())
		}
	})

	t.Run("notitle both directions log before the ack", func(t *testing.T) {
		_, actor, _, file := mudlogFixture(t)
		for _, want := range []string{"ON", "OFF"} {
			if err := cmdNotitle(actor, []string{"Victim"}); err != nil {
				t.Fatal(err)
			}
			wantLines := []string{
				"[ (GC) Notitle " + want + " for Victim by Godactor. ]\r\n",
				"(GC) Notitle " + want + " for Victim by Godactor.\r\n",
			}
			if got := drainSessionText(t, actor); !equalStrings(got, wantLines) {
				t.Fatalf("notitle %s lines = %q, want %q", want, got, wantLines)
			}
		}
		// NRM (type 2) is the only normal-type producer in this family.
		if !strings.Contains(file.String(), "(GC) Notitle ON for Victim by Godactor.\n") ||
			!strings.Contains(file.String(), "(GC) Notitle OFF for Victim by Godactor.\n") {
			t.Fatalf("file = %q", file.String())
		}
	})

	t.Run("mute both directions log before the ack", func(t *testing.T) {
		_, actor, _, file := mudlogFixture(t)
		for _, want := range []string{"ON", "OFF"} {
			if err := cmdMute(actor, []string{"Victim"}); err != nil {
				t.Fatal(err)
			}
			wantLines := []string{
				"[ (GC) Squelch " + want + " for Victim by Godactor. ]\r\n",
				"(GC) Squelch " + want + " for Victim by Godactor.\r\n",
			}
			if got := drainSessionText(t, actor); !equalStrings(got, wantLines) {
				t.Fatalf("mute %s lines = %q, want %q", want, got, wantLines)
			}
		}
		if !strings.Contains(file.String(), "(GC) Squelch ON for Victim by Godactor.\n") {
			t.Fatalf("file = %q", file.String())
		}
	})
}

// TestWizutilMudlogFreezeThawOrder covers the two lifecycle cases whose
// ordering is load-bearing: freeze logs after the room act, thaw logs before
// anything.
func TestWizutilMudlogFreezeThawOrder(t *testing.T) {
	t.Run("freeze room act precedes the log", func(t *testing.T) {
		_, actor, victim, file := mudlogFixture(t)
		if err := cmdFreeze(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		want := []string{
			"Frozen.\r\n",
			"A sudden cold wind conjured from nowhere freezes Victim!\r\n",
			"[ (GC) Victim frozen by Godactor. ]\r\n",
		}
		if got := drainSessionText(t, actor); !equalStrings(got, want) {
			t.Fatalf("actor lines = %q, want %q", got, want)
		}
		if victim.player.GetFlags()&(1<<game.PlrFrozen) == 0 {
			t.Fatal("victim not frozen")
		}
		if !strings.Contains(file.String(), "(GC) Victim frozen by Godactor.\n") {
			t.Fatalf("file = %q", file.String())
		}
	})

	t.Run("thaw logs before every message and before flag removal", func(t *testing.T) {
		_, actor, victim, file := mudlogFixture(t)
		victim.player.SetPlrFlag(game.PlrFrozen, true)
		victim.player.FreezeLevel = 34
		// Record the flag state at the instant the file side is written: C logs
		// before REMOVE_BIT_AR (act.wizard.c:2168-2169).
		frozenAtLog := true
		probe := &flagProbe{buf: file, when: func() {
			frozenAtLog = victim.player.GetFlags()&(1<<game.PlrFrozen) != 0
		}}
		game.SetLogWriter(probe)
		if err := cmdThaw(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if !probe.saw {
			t.Fatalf("thaw produced no file log: %q", file.String())
		}
		if !frozenAtLog {
			t.Fatal("thaw logged after clearing PLR_FROZEN; C logs first")
		}
		want := []string{
			"[ (GC) Victim un-frozen by Godactor. ]\r\n",
			"Thawed.\r\n",
			"A sudden fireball conjured from nowhere thaws Victim!\r\n",
		}
		if got := drainSessionText(t, actor); !equalStrings(got, want) {
			t.Fatalf("actor lines = %q, want %q", got, want)
		}
		if victim.player.GetFlags()&(1<<game.PlrFrozen) != 0 {
			t.Fatal("victim still frozen")
		}
		if got := drainSessionText(t, victim); !equalStrings(got, []string{
			"A fireball suddenly explodes in front of you, melting the ice!\r\nYou feel thawed.\r\n",
		}) {
			t.Fatalf("victim lines = %q", got)
		}
		if !strings.Contains(file.String(), "(GC) Victim un-frozen by Godactor.\n") {
			t.Fatalf("file = %q", file.String())
		}
	})
}

// flagProbe is a log writer that records game state at the moment the file
// side of a mudlog is written.
type flagProbe struct {
	buf  *bytes.Buffer
	when func()
	saw  bool
}

func (p *flagProbe) Write(b []byte) (int, error) {
	p.saw = true
	p.when()
	return p.buf.Write(b)
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// addObserver registers an extra immortal session so a test can vary the
// recipient level and syslog bits independently of the actor.
func addObserver(t *testing.T, m *Manager, name string, level int, log1, log2 bool) *Session {
	t.Helper()
	s := makeTestSession(t, m, name, 1001, true)
	s.player.SetLevel(level)
	s.player.SetPlrFlag(game.PrfLog1, log1)
	s.player.SetPlrFlag(game.PrfLog2, log2)
	registerTestSession(t, m, s, name)
	return s
}

// TestWizutilMudlogRecipientFilter proves the producer's level threshold and
// type, and that rejected command branches log nothing.
func TestWizutilMudlogRecipientFilter(t *testing.T) {
	t.Run("minimum level is LVL_GOD", func(t *testing.T) {
		m, actor, victim, _ := mudlogFixture(t)
		victim.player.SetPlrFlag(game.PlrOutlaw, true)
		below := addObserver(t, m, "BelowGod", 33, false, true)
		atGod := addObserver(t, m, "AtGod", 34, false, true)
		if err := cmdPardon(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if got := drainSessionText(t, below); len(got) != 0 {
			t.Fatalf("level-33 observer received %q", got)
		}
		if got := drainSessionText(t, atGod); !equalStrings(got, []string{"[ (GC) Victim pardoned by Godactor ]\r\n"}) {
			t.Fatalf("level-34 observer = %q", got)
		}
	})

	t.Run("actor invisibility raises the threshold", func(t *testing.T) {
		m, actor, victim, _ := mudlogFixture(t)
		victim.player.SetPlrFlag(game.PlrOutlaw, true)
		actor.player.SetInvisLevel(35)
		atGod := addObserver(t, m, "AtGodInvis", 34, false, true)
		atInvis := addObserver(t, m, "AtInvis", 35, false, true)
		if err := cmdPardon(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if got := drainSessionText(t, atGod); len(got) != 0 {
			t.Fatalf("level-34 observer received an invis-35 line: %q", got)
		}
		if got := drainSessionText(t, atInvis); len(got) != 1 {
			t.Fatalf("level-35 observer = %q", got)
		}
	})

	t.Run("NRM needs normal and BRF reaches brief", func(t *testing.T) {
		m, actor, victim, _ := mudlogFixture(t)
		brief := addObserver(t, m, "BriefOnly", 34, true, false)
		normal := addObserver(t, m, "NormalOnly", 34, false, true)
		if err := cmdNotitle(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if got := drainSessionText(t, brief); len(got) != 0 {
			t.Fatalf("brief observer received NRM: %q", got)
		}
		if got := drainSessionText(t, normal); len(got) != 1 {
			t.Fatalf("normal observer = %q", got)
		}
		// BRF (type 1) reaches a brief-only observer.
		victim.player.SetPlrFlag(game.PlrOutlaw, true)
		if err := cmdPardon(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if got := drainSessionText(t, brief); len(got) != 1 {
			t.Fatalf("brief observer missed BRF: %q", got)
		}
	})

	t.Run("rejected branches emit no producer log", func(t *testing.T) {
		_, actor, victim, file := mudlogFixture(t)
		// pardon non-outlaw, thaw not-frozen, freeze self, freeze already frozen.
		if err := cmdPardon(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if err := cmdThaw(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if err := cmdFreeze(actor, []string{"Godactor"}); err != nil {
			t.Fatal(err)
		}
		victim.player.SetPlrFlag(game.PlrFrozen, true)
		if err := cmdFreeze(actor, []string{"Victim"}); err != nil {
			t.Fatal(err)
		}
		if file.Len() != 0 {
			t.Fatalf("a rejected branch logged: %q", file.String())
		}
		if refusals := drainSessionText(t, actor); len(refusals) != 4 {
			t.Fatalf("expected four refusals, got %q", refusals)
		}
	})
}

// TestSkillsetMudlogIsFileOnly proves C's file-only diagnostic: BRF with level
// -1 and file=TRUE, emitted before SET_SKILL and never broadcast, even to a
// fully qualified observer.
func TestSkillsetMudlogIsFileOnly(t *testing.T) {
	m, actor, _, file := mudlogFixture(t)
	qualified := addObserver(t, m, "QualifiedGod", 60, true, true)
	if err := cmdSkillset(actor, []string{"Victim", "'kick'", "50"}); err != nil {
		t.Fatal(err)
	}
	name := game.SkillCatalogName(game.FindSkillNum("kick"))
	if !strings.Contains(file.String(), "Godactor changed Victim's "+name+" to 50.\n") {
		t.Fatalf("file = %q", file.String())
	}
	if got := drainSessionText(t, qualified); len(got) != 0 {
		t.Fatalf("skillset broadcast to a qualified observer: %q", got)
	}
	if got := drainSessionText(t, actor); !equalStrings(got, []string{"You change Victim's " + name + " to 50.\n\r"}) {
		t.Fatalf("actor ack = %q", got)
	}
}
