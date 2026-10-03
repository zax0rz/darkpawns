package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestStatPlayer_DefaultPositionStanding is the DP-1332 regression: C's
// do_stat_character prints "Default position: Standing" for all characters
// (using mob_specials.default_pos, which clear_char initializes to POS_STANDING
// even for players). The port was using the player's current position instead.
func TestStatPlayer_DefaultPositionStanding(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.SetPosition(combat.PosResting)

	s.sendStatPlayerReport(s.player, 1001, true, 0)

	var texts []string
	for {
		select {
		case msg := <-s.send:
			if IsInputMarkFrame(msg) {
				continue
			}
			texts = append(texts, string(msg))
		default:
			goto drain
		}
	}
drain:

	found := false
	for _, raw := range texts {
		if strings.Contains(raw, "Default position:") {
			found = true
			if strings.Contains(raw, "Default position: Resting") {
				t.Errorf("stat used current position for Default position; want Standing (C's clear_char default).\n  raw=%s", raw)
			}
			if !strings.Contains(raw, "Default position: Standing") {
				t.Errorf("Default position line missing 'Standing':\n  raw=%s", raw)
			}
			break
		}
	}
	if !found {
		t.Fatal("stat output did not contain a 'Default position:' line")
	}
}

func TestStatMobileEquipmentPoints(t *testing.T) {
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 300, ShortDesc: "a guard", Keywords: "guard", AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11, Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 7}}}, Objs: []parser.Obj{{VNum: 200, TypeFlag: game.ITEM_ARMOR, Values: [4]int{10}, Affects: []parser.ObjAffect{{Location: 18, Modifier: 4}, {Location: 19, Modifier: 3}, {Location: 20, Modifier: -2}}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)
	mob, err := w.SpawnMob(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := w.SpawnObject(200, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipMobileObject(mob, obj, 5); err != nil {
		t.Fatal(err)
	}
	s.sendStatMob(mob)
	var text strings.Builder
	for len(s.send) > 0 {
		text.WriteString(readSendText(t, s))
	}
	if !strings.Contains(text.String(), "AC: [70/10], Hitroll: [ 4], Damroll: [10], Saving throws: [-2/0/0/0/0]") {
		t.Fatalf("stat did not read effective mobile equipment points: %s", text.String())
	}
}
