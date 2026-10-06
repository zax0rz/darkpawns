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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, actor, _, watch := wizardLogFixture(t)
			file := captureMudlogFile(t)
			// Raise invis above watcher for non-MAX producers: it must not affect load,
			// purge or newbie's distinct C thresholds.
			if tc.command == "load" || tc.command == "purge" || tc.command == "wnewbie" {
				actor.player.SetInvisLevel(41)
			}
			if err := executeCommand(actor, tc.command, tc.args, false); err != nil {
				t.Fatal(err)
			}
			got := strings.Join(drainSessionText(t, watch), "")
			want := "[ " + tc.payload + " ]\r\n"
			if !strings.Contains(got, want) {
				t.Fatalf("observer = %q, missing %q", got, want)
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
