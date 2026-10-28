package game

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

func TestActivityDeathAcceptedPathsAndCauses(t *testing.T) {
	for _, kind := range []string{"mob", "player", "trap", "raw", "environment", "raw-killer", "mob-victim", "healthy"} {
		t.Run(kind, func(t *testing.T) {
			w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Arena", Zone: 1}}, Mobs: []parser.Mob{{VNum: 300, ShortDesc: "a guard", Keywords: "guard"}}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			old := combat.GetCallbacks()
			combat.SetCallbacks(w.WireCombatCallbacks())
			t.Cleanup(func() { combat.SetCallbacks(old) })
			p := NewPlayer(1, "Avery", 1001)
			p.SetLevel(10)
			p.InvisLevel = 40
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
			mob, err := w.SpawnMobQuiet(300, 1001)
			if err != nil {
				t.Fatal(err)
			}
			other := NewPlayer(2, "Other", 1001)
			h := telemetry.New(time.Now())
			expected := kind
			switch kind {
			case "mob":
				p.SetHP(-11)
				w.HandleDeath(p, mob, combat.TYPE_HIT)
				w.HandleDeath(p, mob, combat.TYPE_HIT)
			case "player":
				p.SetHP(-11)
				w.HandleDeath(p, other, combat.TYPE_HIT)
			case "trap":
				w.deathTrap(p)
				w.deathTrap(p)
			case "environment":
				p.SetHP(-11)
				w.HandleNonCombatDeath(p)
				w.HandleNonCombatDeath(p)
				expected = "unknown"
			case "raw":
				w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
				w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
				expected = "unknown"
			case "raw-killer":
				w.rawKillCombatant(p, combat.TYPE_HIT, mob)
				expected = "mob"
			case "mob-victim":
				w.HandleDeath(mob, p, combat.TYPE_HIT)
			case "healthy":
				p.SetHP(10)
				w.handlePlayerDeath(p, true, combat.TYPE_HIT, "", mob)
			}
			h.Sample(time.Now(), 0)
			s := h.Snapshot()
			if kind == "mob-victim" || kind == "healthy" {
				if len(s.Activity) != 0 || s.Points[0].PlayerDeaths != 0 {
					t.Fatalf("false player death: %+v", s)
				}
				return
			}
			if len(s.Activity) != 1 || s.Points[0].PlayerDeaths != 1 {
				t.Fatalf("missing/duplicate player death: %+v", s)
			}
			e := s.Activity[0]
			if e.Player != "Avery" || e.RoomName != "Arena" || e.RoomVNum != 1001 || e.Cause != expected || e.MinLevel != 40 {
				t.Fatalf("wrong detail: %+v", e)
			}
			if expected == "mob" && (e.MobVNum != 300 || e.MobName != "a guard") {
				t.Fatalf("missing mob: %+v", e)
			}
		})
	}
}
