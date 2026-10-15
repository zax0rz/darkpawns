package spells_test

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func newGroupRecallWorld(t *testing.T, bfr bool) (*game.World, map[string]*strings.Builder) {
	t.Helper()
	flags := []string{}
	if bfr {
		flags = []string{"131072"} // ROOM_BFR
	}
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Recall origin", Zone: 1, Sector: game.SECT_INSIDE, Flags: flags},
		{VNum: 8004, Name: "Mortal start", Zone: 1, Sector: game.SECT_INSIDE},
		{VNum: 18201, Name: "Kiroshi start", Zone: 1, Sector: game.SECT_INSIDE},
		{VNum: 21258, Name: "Alaozar start", Zone: 1, Sector: game.SECT_INSIDE},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	outputs := make(map[string]*strings.Builder)
	w.MessageSink = func(name string, msg []byte) {
		if b := outputs[name]; b != nil {
			b.Write(msg)
		}
	}
	return w, outputs
}

func addGroupRecallPlayer(t *testing.T, w *game.World, outputs map[string]*strings.Builder, id int, name string, room int) *game.Player {
	t.Helper()
	p := game.NewPlayer(id, name, room)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	outputs[name] = &strings.Builder{}
	return p
}

func TestCastWordOfRecallGateBytes(t *testing.T) {
	for _, tt := range []struct {
		name        string
		bfr         bool
		fighting    bool
		wantName    string
		wantMessage string
	}{
		{
			name:        "BFR room sends spell failure to victim",
			bfr:         true,
			wantName:    "Victim",
			wantMessage: "Your magic ebbs and dissolves as you lose your concentration.\r\n",
		},
		{
			name:        "fighting caster receives concentration failure",
			fighting:    true,
			wantName:    "Caster",
			wantMessage: "Your concentration is broken by your fighting!\r\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, outputs := newGroupRecallWorld(t, tt.bfr)
			caster := addGroupRecallPlayer(t, w, outputs, 1, "Caster", 1001)
			victim := addGroupRecallPlayer(t, w, outputs, 2, "Victim", 1001)
			if tt.fighting {
				caster.SetFightingBody(victim)
			}

			spells.ExecuteManualSpell(spells.SpellWordOfRecall, 30, caster, victim, nil, "", w)

			if got := outputs[tt.wantName].String(); got != tt.wantMessage {
				t.Fatalf("%s output = %q, want %q", tt.wantName, got, tt.wantMessage)
			}
			if caster.GetRoomVNum() != 1001 || victim.GetRoomVNum() != 1001 {
				t.Fatalf("failed recall moved players: caster=%d victim=%d", caster.GetRoomVNum(), victim.GetRoomVNum())
			}
		})
	}
}

func TestMagGroupsGroupRecallDispatchesSleepingMember(t *testing.T) {
	w, outputs := newGroupRecallWorld(t, false)
	leader := addGroupRecallPlayer(t, w, outputs, 1, "Leader", 1001)
	member := addGroupRecallPlayer(t, w, outputs, 2, "Sleeper", 1001)
	leader.SetInGroup(true)
	member.SetInGroup(true)
	w.AddFollowerQuiet(member, leader)
	member.SetPosition(game.PosSleeping)

	spells.MagGroups(30, leader, spells.SpellGroupRecall, int(spells.SaveSpell), w)

	if got := member.GetRoomVNum(); got != 8004 {
		t.Fatalf("sleeping group member room = %d, want mortal start 8004", got)
	}
	if got, want := outputs[member.Name].String(), "You have a strange dream about falling..\r\n"; got != want {
		t.Fatalf("sleeping member output = %q, want %q", got, want)
	}
	if got := leader.GetRoomVNum(); got != 8004 {
		t.Fatalf("group leader room = %d, want mortal start 8004", got)
	}
}
