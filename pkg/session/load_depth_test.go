package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestLoadRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["load"]
	if !ok {
		t.Fatal("load command has no C gate")
	}
	if entry.MinLevel != LVL_IMMORT || entry.MinPosition != combat.PosDead {
		t.Fatalf("load gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_IMMORT, combat.PosDead)
	}
}

func TestLoadCArgumentHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "mob abbreviation", got: loadIsAbbrev("m", "mob"), want: true},
		{name: "object abbreviation", got: loadIsAbbrev("ob", "object"), want: true},
		{name: "object is not obj", got: loadIsAbbrev("object", "obj"), want: false},
		{name: "keyword prefix", got: loadNameMatches("bre", "loaf bread"), want: true},
		{name: "keyword substring is not enough", got: loadNameMatches("rea", "bread"), want: false},
		{name: "atoi prefix", got: loadAtoi("25abc"), want: 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestCmdLoadObjectNameUsesCPlacementAndAudiences(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Load room", Zone: 1}},
		Objs: []parser.Obj{{
			VNum: 3001, Keywords: "loaf bread", ShortDesc: "a loaf of bread", Cost: 3,
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	m := newTestManager(t, w, nil)
	actor := makeCommandTestSession(t, m, "Loader", LVL_IMPL, 1001)
	observer := makeCommandTestSession(t, m, "Observer", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, observer)

	dprng.ResetStream(1)
	if err := cmdLoad(actor, []string{"obj", "bread"}); err != nil {
		t.Fatalf("cmdLoad: %v", err)
	}

	actorText := drainSendChannel(t, actor)
	if !strings.Contains(actorText, "You create a loaf of bread.") {
		t.Fatalf("actor output = %q, want creation line", actorText)
	}
	if strings.Contains(actorText, "appears") || !strings.Contains(actorText, "walls run red") && !strings.Contains(actorText, "flames of Hell") && !strings.Contains(actorText, "shriek of a dying dragon") {
		t.Fatalf("actor output = %q, want exactly one C load narration", actorText)
	}
	observerText := drainSendChannel(t, observer)
	if !strings.Contains(observerText, "Loader makes a strange magickal gesture.") ||
		!strings.Contains(observerText, "Loader has created a loaf of bread!") {
		t.Fatalf("observer output = %q, want room creation narration", observerText)
	}
	if strings.Contains(observerText, "appears") {
		t.Fatalf("observer output = %q, must not contain ordinary spawn announcement", observerText)
	}
	if got := len(actor.player.Inventory.Items); got != 1 {
		t.Fatalf("actor inventory length = %d, want one loaded object", got)
	}
}

func TestLoadCostAndObjectLimits(t *testing.T) {
	tests := []struct {
		level int
		vnum  int
		cost  int
		want  string
	}{
		{level: 34, vnum: 9001, cost: 6001, want: "That is beyond your godly powers..."},
		{level: 34, vnum: 81, cost: 1, want: "You're not godly enough to load that!"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if max, ok := loadCostLimit(tt.level); tt.cost > max && !ok {
				t.Fatalf("level %d cost limit = (%d, %v), want a limit", tt.level, max, ok)
			}
			if tt.vnum == 81 {
				if got := loadObjectLimits[tt.vnum]; got != LVL_GRGOD {
					t.Fatalf("object %d minimum level = %d, want %d", tt.vnum, got, LVL_GRGOD)
				}
				return
			}
			max, ok := loadCostLimit(tt.level)
			if !ok || tt.cost <= max {
				t.Fatalf("cost %d is not above level %d cap %d", tt.cost, tt.level, max)
			}
		})
	}
}
