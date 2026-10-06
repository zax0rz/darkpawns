package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func wizardLogFixture(t *testing.T) (*Manager, *Session, *Session, *Session) {
	t.Helper()
	p := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Producer room", Zone: 80}},
		Zones: []parser.Zone{{Number: 80, Name: "Producer zone", TopRoom: 1099}},
		Mobs:  []parser.Mob{{VNum: 3001, Keywords: "trainee guard", ShortDesc: "a guard trainee", Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 1}}},
	}
	for _, v := range []int{3001, 8019, 8062, 8063, 8023} {
		p.Objs = append(p.Objs, parser.Obj{VNum: v, Keywords: "bread", ShortDesc: fmt.Sprintf("object %d", v), Cost: 1})
	}
	w, err := game.NewWorld(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	actor := makeTestSession(t, m, "Logactor", 1001, true)
	actor.player.SetLevel(38)
	victim := makeTestSession(t, m, "Logvictim", 1001, true)
	watch := makeTestSession(t, m, "Logwatch", 1001, true)
	watch.player.SetLevel(40)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	for _, s := range []*Session{actor, victim, watch} {
		s.player.SetPosition(combat.PosStanding)
		registerTestSession(t, m, s, s.player.Name)
	}
	return m, actor, victim, watch
}

// Each subtest invokes the production handler with the real provider and owns
// one C producer. A removed call must fail its exact payload assertion.
func TestWizardMudlogProducers(t *testing.T) {
	cases := []struct {
		name, command, payload string
		args                   []string
	}{
		{"load-mob", "load", "(GC) Logactor loaded a guard trainee at Producer room.", []string{"mob", "3001"}},
		{"load-object", "load", "(GC) Logactor loaded object 3001 at Producer room", []string{"obj", "3001"}},
		{"purge-player", "purge", "(GC) Logactor has purged Logvictim.", []string{"Logvictim"}},
		{"force-single", "force", "(GC) Logactor forced Logvictim to say forced", []string{"Logvictim", "say", "forced"}},
		{"force-room", "force", "(GC) Logactor forced room 1001 to say forced", []string{"room", "say", "forced"}},
		{"force-all", "force", "(GC) Logactor forced all to say forced", []string{"all", "say", "forced"}},
		{"reset-zone", "zreset", "(GC) Logactor reset zone 0 (Producer zone)", []string{"80"}},
		{"reset-world", "zreset", "(GC) Logactor reset entire world.", []string{"*"}},
		{"newbie", "wnewbie", "(GC) Logactor newbied Logvictim.", []string{"Logvictim"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, actor, _, watch := wizardLogFixture(t)
			file := captureMudlogFile(t)
			minimum := 39 // load, force and newbie: actor level + 1.
			if tc.command == "purge" {
				minimum = 34
			}
			if tc.command == "zreset" {
				minimum = 38
			}
			watch.player.SetLevel(minimum)
			below := addObserver(t, m, "BelowProducer", minimum-1, false, true)
			brief := addObserver(t, m, "BriefProducer", minimum, true, false)
			// Raise invis above watcher for non-MAX producers: it must not affect load,
			// purge or newbie's distinct C thresholds.
			if tc.command == "load" || tc.command == "purge" || tc.command == "wnewbie" {
				actor.player.SetInvisLevel(40)
				watch.player.SetPlrFlag(game.PrfLog2, false)
				watch.player.SetPlrFlag(game.PrfLog1, true)
			}
			if err := executeCommand(actor, tc.command, tc.args, false); err != nil {
				t.Fatal(err)
			}
			got := strings.Join(drainSessionText(t, watch), "")
			want := "[ " + tc.payload + " ]\r\n"
			if !strings.Contains(got, want) {
				t.Fatalf("observer = %q, missing %q", got, want)
			}
			if low := strings.Join(drainSessionText(t, below), ""); strings.Contains(low, want) {
				t.Fatalf("below-minimum observer received producer: %q", low)
			}
			briefText := strings.Join(drainSessionText(t, brief), "")
			wantBrief := tc.command == "load" || tc.command == "purge" || tc.command == "wnewbie"
			if strings.Contains(briefText, want) != wantBrief {
				t.Fatalf("brief observer = %q, eligible=%v", briefText, wantBrief)
			}
			if !strings.Contains(file.String(), tc.payload+"\n") {
				t.Fatalf("file = %q", file.String())
			}
			switch tc.command {
			case "force":
				if i := strings.Index(got, want); strings.Contains(got, "forced'") && strings.Index(got, "forced'") < i {
					t.Fatalf("force execution preceded log: %q", got)
				}
			case "purge":
				// The observer sees disintegration, then purge log, then close log.
				if !strings.Contains(got, "Logactor disintegrates Logvictim.\r\n") && actor.player.GetInvisLevel() == 0 {
					t.Fatalf("missing purge room act: %q", got)
				}
			}
		})
	}
}

func TestWizardMudlogThresholdAndOrder(t *testing.T) {
	t.Run("load exact threshold and before narration", func(t *testing.T) {
		_, a, _, w := wizardLogFixture(t)
		w.player.SetLevel(39)
		w.player.SetPlrFlag(game.PrfLog2, false)
		w.player.SetPlrFlag(game.PrfLog1, true)
		if err := cmdLoad(a, []string{"obj", "3001"}); err != nil {
			t.Fatal(err)
		}
		got := strings.Join(drainSessionText(t, w), "")
		if !strings.HasPrefix(got, "[ (GC) Logactor loaded object 3001 at Producer room ]\r\n") {
			t.Fatalf("load log must precede gesture: %q", got)
		}
		w.player.SetLevel(38)
		if err := cmdLoad(a, []string{"obj", "3001"}); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(drainSessionText(t, w), ""); strings.Contains(got, "[ (GC)") {
			t.Fatalf("actor-level observer received load log: %q", got)
		}
	})
	t.Run("force invis threshold and normal type", func(t *testing.T) {
		_, a, _, w := wizardLogFixture(t)
		a.player.SetInvisLevel(40)
		w.player.SetLevel(39)
		if err := cmdForce(a, []string{"room", "say", "one"}); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(drainSessionText(t, w), ""); strings.Contains(got, "[ (GC)") {
			t.Fatalf("invis threshold ignored: %q", got)
		}
		a.player.SetInvisLevel(0)
		w.player.SetPlrFlag(game.PrfLog2, false)
		w.player.SetPlrFlag(game.PrfLog1, true)
		if err := cmdForce(a, []string{"all", "say", "two"}); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(drainSessionText(t, w), ""); strings.Contains(got, "[ (GC)") {
			t.Fatalf("brief observer received normal force log: %q", got)
		}
	})
	t.Run("reset ack precedes log", func(t *testing.T) {
		_, a, _, _ := wizardLogFixture(t)
		a.player.SetPlrFlag(game.PrfLog2, true)
		if err := cmdZreset(a, []string{"80"}); err != nil {
			t.Fatal(err)
		}
		got := strings.Join(drainSessionText(t, a), "")
		if got != "Reset zone 0 (#80): Producer zone.\r\n[ (GC) Logactor reset zone 0 (Producer zone) ]\r\n" {
			t.Fatalf("reset output = %q", got)
		}
	})
	t.Run("newbie after equipment and messages", func(t *testing.T) {
		_, a, v, _ := wizardLogFixture(t)
		file := captureMudlogFile(t)
		ready := false
		game.SetLogWriter(&flagProbe{buf: file, when: func() { ready = len(v.player.Inventory.Items) == 4 && len(a.send) > 0 && len(v.send) > 0 }})
		if err := cmdNewbie(a, []string{"Logvictim"}); err != nil {
			t.Fatal(err)
		}
		if !ready {
			t.Fatal("newbie log preceded equipment or messages")
		}
	})
	t.Run("purge act then log before teardown", func(t *testing.T) {
		m, a, v, w := wizardLogFixture(t)
		file := captureMudlogFile(t)
		live := false
		game.SetLogWriter(&flagProbe{buf: file, when: func() {
			if strings.Contains(file.String(), "has purged") {
				return
			}
			_, live = m.GetSession(v.player.Name)
		}})
		if err := cmdPurge(a, []string{"Logvictim"}); err != nil {
			t.Fatal(err)
		}
		got := strings.Join(drainSessionText(t, w), "")
		if !strings.HasPrefix(got, "Logactor disintegrates Logvictim.\r\n[ (GC) Logactor has purged Logvictim. ]\r\n") || !live {
			t.Fatalf("purge ordering/live=%v: %q", live, got)
		}
	})
}

func TestWizardMudlogForceRawRemainder(t *testing.T) {
	_, a, _, w := wizardLogFixture(t)
	if err := executeCommandRaw(a, "force", []string{"Logvictim", "say", "raw"}, false, "Logvictim   say  raw  "); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(drainSessionText(t, w), "")
	if !strings.Contains(got, "[ (GC) Logactor forced Logvictim to say  raw   ]\r\n") {
		t.Fatalf("raw force log = %q", got)
	}
}

func TestWizardMudlogForceBeforeInterpretation(t *testing.T) {
	for _, kind := range []string{"Logvictim", "room", "all"} {
		t.Run(kind, func(t *testing.T) {
			_, a, v, _ := wizardLogFixture(t)
			file := captureMudlogFile(t)
			atLog := -1
			notification := -1
			game.SetLogWriter(&flagProbe{buf: file, when: func() { atLog = v.player.GetPosition(); notification = len(v.send) }})
			if err := cmdForce(a, []string{kind, "sit"}); err != nil {
				t.Fatal(err)
			}
			wantNotification := 0
			if kind == "Logvictim" {
				wantNotification = 1
			}
			if atLog != combat.PosStanding || notification != wantNotification || v.player.GetPosition() != combat.PosSitting {
				t.Fatalf("position at log=%d, notifications=%d, final=%d", atLog, notification, v.player.GetPosition())
			}
		})
	}
}

func TestWizardMudlogRefusals(t *testing.T) {
	_, a, _, _ := wizardLogFixture(t)
	file := captureMudlogFile(t)
	for _, tc := range []struct {
		command string
		args    []string
	}{
		{"load", []string{"obj", "999999"}},
		{"load", []string{"mob", "999999"}},
		{"force", []string{"Logwatch", "sit"}},
		{"force", []string{"Nobody", "sit"}},
		{"zreset", []string{"999999"}},
		{"wnewbie", []string{"Nobody"}},
		{"purge", []string{"Logwatch"}},
	} {
		if err := executeCommand(a, tc.command, tc.args, false); err != nil {
			t.Fatal(err)
		}
	}
	if file.Len() != 0 {
		t.Fatalf("staged refusal logged: %q", file.String())
	}
}

func TestWizardMudlogNewbieMobileName(t *testing.T) {
	m, a, _, w := wizardLogFixture(t)
	mob, err := m.world.SpawnMobQuiet(3001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	file := captureMudlogFile(t)
	itemsAtLog := 0
	game.SetLogWriter(&flagProbe{buf: file, when: func() { itemsAtLog = len(mob.Inventory) }})
	if err := cmdNewbie(a, []string{"trainee"}); err != nil {
		t.Fatal(err)
	}
	if itemsAtLog != 4 {
		t.Fatalf("mobile gifts at log = %d, want 4", itemsAtLog)
	}
	got := strings.Join(drainSessionText(t, w), "")
	if !strings.Contains(got, "[ (GC) Logactor newbied a guard trainee. ]\r\n") {
		t.Fatalf("mobile target log = %q", got)
	}
}

func TestWizardMudlogNestedForceRemainder(t *testing.T) {
	m, a, v, w := wizardLogFixture(t)
	v.player.SetLevel(36)
	second := makeTestSession(t, m, "Logsecond", 1001, true)
	second.player.SetPosition(combat.PosStanding)
	registerTestSession(t, m, second, second.player.Name)
	if err := cmdForceText(a, "Logvictim force Logsecond say  nested  "); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(drainSessionText(t, w), "")
	if !strings.Contains(got, "[ (GC) Logvictim forced Logsecond to say  nested   ]\r\n") {
		t.Fatalf("nested raw log = %q", got)
	}
}
