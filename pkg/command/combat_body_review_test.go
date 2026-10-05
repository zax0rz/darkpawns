package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCombatBodySkillAudienceCollision(t *testing.T) {
	for _, path := range []string{"ordinary", "after-damage", "retaliation"} {
		t.Run(path, func(t *testing.T) {
			s := newRescueCommandSession(t, nil)
			t.Cleanup(s.world.StopAITicker)
			listener := game.NewPlayer(3, "Collision", 1001)
			if err := s.world.AddPlayer(listener); err != nil {
				t.Fatal(err)
			}
			target := game.NewMob(&parser.Mob{VNum: 300, ShortDesc: "Collision", Keywords: "collision", Position: 8}, 1001)
			var out strings.Builder
			s.world.MessageSink = func(name string, b []byte) {
				if name == listener.Name {
					out.Write(b)
				}
			}
			r := game.SkillResult{MessageToRoom: "first line", MessageToRoomSecond: "second line"}
			if path == "after-damage" {
				r.SkillMsgAfterDamage = true
			}
			if path == "retaliation" {
				r.RetaliateHitAfterMessages = true
			}
			if err := sendSkillResult(s, s.player, target, r); err != nil {
				t.Fatal(err)
			}
			if out.String() != "First line\r\nSecond line\r\n" {
				t.Fatalf("wrong body excluded: %q", out.String())
			}
		})
	}
}
