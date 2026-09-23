package game

import (
	"slices"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestNightGateCreatesAndRemovesOnlyMoonPhasePortal(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 4004}, {VNum: 4005}},
		Objs:  []parser.Obj{{VNum: BluePortalVNum, Keywords: "portal", ShortDesc: "a blue portal"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) {
		messages[name] = append(messages[name], string(msg))
	}
	sleeping := NewPlayer(3, "GateSleeping", 4004)
	sleeping.SetPosition(combat.PosSleeping)
	for _, p := range []*Player{NewPlayer(1, "GateThree", 4004), NewPlayer(2, "GateFull", 4005), sleeping} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s): %v", p.Name, err)
		}
	}

	loadNightGateForWorld(w, MoonThreeFull)
	if got := len(w.GetItemsInRoom(4004)); got != 1 || w.GetItemsInRoom(4004)[0].GetVNum() != BluePortalVNum {
		t.Fatalf("phase-matched room objects = %#v, want one blue portal", w.GetItemsInRoom(4004))
	}
	if got := len(w.GetItemsInRoom(4005)); got != 0 {
		t.Fatalf("nonmatching phase room has %d objects, want none", got)
	}
	if !slices.Equal(messages["GateThree"], []string{"A shimmering portal of blue light suddenly appears in the darkness!\r\n"}) {
		t.Fatalf("gate room message = %q", messages["GateThree"])
	}
	if len(messages["GateFull"]) != 0 {
		t.Fatalf("unmatched room received gate message: %q", messages["GateFull"])
	}
	if len(messages["GateSleeping"]) != 0 {
		t.Fatalf("sleeping room occupant received gate message: %q", messages["GateSleeping"])
	}

	removeNightGateForWorld(w)
	if got := len(w.GetItemsInRoom(4004)); got != 0 {
		t.Fatalf("phase-matched room still has %d objects after removal", got)
	}
	if !slices.Equal(messages["GateThree"], []string{
		"A shimmering portal of blue light suddenly appears in the darkness!\r\n",
		"The shimmering blue portal of light fades out of existence.\r\n",
	}) {
		t.Fatalf("gate room messages after removal = %q", messages["GateThree"])
	}
	if len(messages["GateSleeping"]) != 0 {
		t.Fatalf("sleeping room occupant received gate removal: %q", messages["GateSleeping"])
	}
}

func TestGhostShipOpensAndClosesMatchingRuntimeExits(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 19100}, {VNum: 19173}, {VNum: 19174}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string][]string)
	w.MessageSink = func(name string, msg []byte) {
		messages[name] = append(messages[name], string(msg))
	}
	for _, p := range []*Player{
		NewPlayer(1, "Ship", 19100),
		NewPlayer(2, "DockOne", 19173),
		NewPlayer(3, "DockTwo", 19174),
	} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s): %v", p.Name, err)
		}
	}

	dprng.ResetStream(1)
	t.Cleanup(func() { dprng.ResetStream(1) })
	ghostShipAppearForWorld(w)

	ship := w.GetRoomInWorld(19100)
	if exit, ok := ship.Exits["south"]; !ok || (exit.ToRoom != 19173 && exit.ToRoom != 19174) {
		t.Fatalf("ship south exit = %+v, present=%t", exit, ok)
	}
	dock := ship.Exits["south"].ToRoom
	if exit, ok := w.GetRoomInWorld(dock).Exits["north"]; !ok || exit.ToRoom != 19100 {
		t.Fatalf("selected dock north exit = %+v, present=%t", exit, ok)
	}
	otherDock := 19173
	if dock == 19173 {
		otherDock = 19174
	}
	if _, ok := w.GetRoomInWorld(otherDock).Exits["north"]; ok {
		t.Fatalf("unselected dock %d received a north exit", otherDock)
	}
	if got := len(messages["Ship"]); got != 1 || messages["Ship"][0] != "Suddenly a dock appears to the south!\r\n" {
		t.Fatalf("ship room messages = %q", messages["Ship"])
	}
	dockName := "DockOne"
	otherDockName := "DockTwo"
	if dock == 19174 {
		dockName, otherDockName = otherDockName, dockName
	}
	if got := messages[dockName]; !slices.Equal(got, []string{"Suddenly a ghostly ship appears to the north!\r\n"}) {
		t.Fatalf("selected dock messages = %q", got)
	}
	if len(messages[otherDockName]) != 0 {
		t.Fatalf("unselected dock received appearance message: %q", messages[otherDockName])
	}

	ghostShipDisappearForWorld(w)
	if _, ok := w.GetRoomInWorld(dock).Exits["north"]; ok {
		t.Fatal("selected dock retained its north exit after sunrise")
	}
	if _, ok := w.GetRoomInWorld(19100).Exits["south"]; ok {
		t.Fatal("ship retained its south exit after sunrise")
	}
	if _, ok := w.GetRoomInWorld(otherDock).Exits["north"]; ok {
		t.Fatalf("unselected dock %d retained a north exit after sunrise", otherDock)
	}
	if got := messages[dockName]; !slices.Equal(got, []string{
		"Suddenly a ghostly ship appears to the north!\r\n",
		"Suddenly the ghostly ship to the north disappears!\r\n",
	}) {
		t.Fatalf("selected dock messages after disappearance = %q", got)
	}
	if got := messages["Ship"]; !slices.Equal(got, []string{
		"Suddenly a dock appears to the south!\r\n",
		"Suddenly the dock to the south disappears!\r\n",
		"The ghost ship has set sail!\r\n",
	}) {
		t.Fatalf("ship messages after disappearance = %q", got)
	}
}
