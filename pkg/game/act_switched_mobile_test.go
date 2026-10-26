package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestActRoutesSwitchedMobileOutputToItsDescriptor(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Test Room"}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	owner := NewPlayer(1, "Controller", 1001)
	observer := NewPlayer(2, "Observer", 1001)
	for _, player := range []*Player{owner, observer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatal(err)
		}
	}
	mob := NewMob(&parser.Mob{VNum: 9001, Keywords: "watcher", ShortDesc: "a watcher", LongDesc: "A watcher is here."}, 1001)
	w.activeMobs[mob.GetID()] = mob

	var ownerOutput, observerOutput, mobileOutput strings.Builder
	w.MessageSink = func(name string, message []byte) {
		switch name {
		case owner.Name:
			ownerOutput.Write(message)
		case observer.Name:
			observerOutput.Write(message)
		}
	}
	w.MobileMessageSink = func(body *MobInstance, message []byte) {
		if body == mob {
			mobileOutput.Write(message)
		}
	}
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == mob {
			return owner
		}
		return nil
	}

	Act(w, false, mob, nil, nil, nil, "You get $p.", "", ToChar)
	Act(w, false, mob, nil, nil, nil, "$n gets $p.", "", ToRoom)

	if got, want := mobileOutput.String(), "You get something.\r\n"; got != want {
		t.Fatalf("mobile descriptor output = %q, want %q", got, want)
	}
	if got, want := observerOutput.String(), "A watcher gets something.\r\n"; got != want {
		t.Fatalf("room observer output = %q, want %q", got, want)
	}
	if got := ownerOutput.String(); got != "" {
		t.Fatalf("detached original body received duplicate room output %q", got)
	}
}
