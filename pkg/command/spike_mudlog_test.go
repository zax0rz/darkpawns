package command

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type spikeDiagnosticObserver struct {
	p        *game.Player
	messages []string
	probe    func()
}

func (s *spikeDiagnosticObserver) EachSession(fn func(interface{}, func(string))) {
	fn(s.p, func(msg string) {
		if s.probe != nil {
			s.probe()
		}
		s.messages = append(s.messages, msg)
	})
}

// src/new_cmds.c:1155-1175: all three acts, then BRF/31/file TRUE,
// then PK/death counters and raw kill. Weapon noun is the chosen subcommand.
func TestGameDiagnosticSpikeStake(t *testing.T) {
	for _, verb := range []string{"spike", "stake"} {
		t.Run(verb, func(t *testing.T) {
			s := newSkillCommandSession(t)
			t.Cleanup(s.world.StopAITicker)
			s.player.SetLevel(50)
			s.player.PKs = 7
			victim := game.NewPlayer(2, "Wolf", 1001)
			victim.SetLevel(5)
			victim.Deaths = 3
			bit := game.AffWerewolf
			if verb == "stake" {
				bit = game.AffVampire
			}
			victim.SetAffect(bit, true)
			if err := s.world.AddPlayer(victim); err != nil {
				t.Fatal(err)
			}
			witness := game.NewPlayer(3, "Witness", 1001)
			if err := s.world.AddPlayer(witness); err != nil {
				t.Fatal(err)
			}
			weapon := &game.ObjectInstance{Prototype: &parser.Obj{VNum: 1, Keywords: "spike stake weapon", ShortDesc: "a dual-purpose spike", TypeFlag: 5, WearFlags: [4]int{1 << 13}}}
			s.player.Inventory.Items = append(s.player.Inventory.Items, weapon)
			if err := s.player.Equipment.Equip(weapon, s.player.Inventory); err != nil {
				t.Fatal(err)
			}
			cb := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(cb) })
			combat.SetCallbacks(s.world.WireCombatCallbacks())
			observer := &spikeDiagnosticObserver{p: game.NewPlayer(99, "Observer", 1001)}
			observer.p.SetLevel(game.LVL_IMMORT)
			observer.p.SetPlrFlag(game.PrfLog1, true)
			game.SetImmortalSessionProvider(observer)
			t.Cleanup(func() { game.ClearImmortalSessionProvider(observer) })
			var file bytes.Buffer
			game.SetLogWriter(&file)
			t.Cleanup(func() { game.SetLogWriter(os.Stderr) })
			var audience []string
			s.world.MessageSink = func(_ string, msg []byte) { audience = append(audience, string(msg)) }
			observer.probe = func() {
				if len(s.messages) != 1 || len(audience) != 2 || s.player.PKs != 7 || victim.Deaths != 3 || victim.HasPLRFlag(game.PlrExtract) {
					t.Fatalf("spike log boundary actor=%q audience=%q PK/deaths=%d/%d extract=%t", s.messages, audience, s.player.PKs, victim.Deaths, victim.HasPLRFlag(game.PlrExtract))
				}
			}
			command := CmdSpike
			if verb == "stake" {
				command = CmdStake
			}
			if err := command(s, []string{"Wolf"}); err != nil {
				t.Fatal(err)
			}
			payload := "Tester " + verb + "d Wolf at Test Room."
			if len(observer.messages) != 1 || observer.messages[0] != "[ "+payload+" ]\r\n" || !strings.Contains(file.String(), payload) {
				t.Fatalf("spike diagnostic=%q file=%q", observer.messages, file.String())
			}
			if s.player.PKs != 8 || victim.Deaths != 4 || !victim.HasPLRFlag(game.PlrExtract) {
				t.Fatal("existing raw-kill tail changed")
			}
			observer.messages = nil
			file.Reset()
			observer.probe = nil
			if err := command(s, nil); err != nil {
				t.Fatal(err)
			}
			if len(observer.messages) != 0 || file.Len() != 0 {
				t.Fatal("refused spike logs")
			}
		})
	}
}
