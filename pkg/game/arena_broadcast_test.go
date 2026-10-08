package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func arenaBroadcastWorld(t *testing.T) (*World, *Player, *Player, *Player, map[string]*strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Arena", Flags: []string{"arena"}}, {VNum: 1002, Name: "Remote"}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	actor := NewPlayer(1, "Actor", 1001)
	local := NewPlayer(2, "Local", 1001)
	remote := NewPlayer(3, "Remote", 1002)
	for _, p := range []*Player{actor, local, remote} {
		p.SetLevel(15)
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	return w, actor, local, remote, captureOutput(w)
}

func TestArenaActRemoteAndLocal(t *testing.T) {
	for _, kind := range []int{ToRoom, ToNotVict} {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			w, actor, local, remote, out := arenaBroadcastWorld(t)
			Act(w, false, actor, local, nil, nil, "$n greets $N.", "", kind)
			want := "&RBroadcast: Actor greets Local.&n\r\n"
			if got := outputOf(out, remote.Name); got != want {
				t.Fatalf("remote=%q want %q", got, want)
			}
			localWant := want
			if kind == ToRoom {
				localWant += "Actor greets Local.\r\n"
			}
			if got := outputOf(out, local.Name); got != localWant {
				t.Fatalf("local=%q want %q", got, localWant)
			}
			if got := outputOf(out, actor.Name); got != "" {
				t.Fatalf("actor=%q", got)
			}
		})
	}
}

func TestArenaActNoBroadcastToggle(t *testing.T) {
	w, actor, local, remote, out := arenaBroadcastWorld(t)
	w.ExecGenTog(remote, "nobroadcast")
	w.ExecGenTog(local, "nobroadcast")
	out[local.Name].Reset()
	if remote.GetFlags()&(1<<uint(PrfNoBroad)) == 0 || remote.GetNoBroadcast() {
		t.Fatal("toggle representation changed")
	}
	out[remote.Name].Reset()
	Act(w, false, actor, nil, nil, nil, "$n waves.", "", ToRoom)
	if got := outputOf(out, remote.Name); got != "" {
		t.Fatalf("NOBROAD received %q", got)
	}
	if got := outputOf(out, local.Name); got != "Actor waves.\r\n" {
		t.Fatalf("local NOBROAD normal act=%q", got)
	}
	w.ExecGenTog(remote, "nobroadcast")
	out[remote.Name].Reset()
	remote.SetNoBroadcast(true) // Legacy bool is not what the live toggle writes.
	Act(w, false, actor, nil, nil, nil, "$n waves.", "", ToRoom)
	if got := outputOf(out, remote.Name); got != "&RBroadcast: Actor waves.&n\r\n" {
		t.Fatalf("toggle-on control=%q", got)
	}
}

func TestArenaActInvisibleActor(t *testing.T) {
	w, actor, _, remote, out := arenaBroadcastWorld(t)
	actor.SetAffect(affInvisible, true)
	Act(w, true, actor, nil, nil, nil, "$n waves.", "", ToRoom)
	if got := outputOf(out, remote.Name); got != "" {
		t.Fatalf("invisible actor leaked %q", got)
	}
	remote.SetAffect(affDetectInvisible, true)
	Act(w, true, actor, nil, nil, nil, "$n waves.", "", ToRoom)
	if got := outputOf(out, remote.Name); got != "&RBroadcast: Actor waves.&n\r\n" {
		t.Fatalf("detect-invisible control=%q", got)
	}
}

func TestArenaActNonArenaGate(t *testing.T) {
	w, actor, local, remote, out := arenaBroadcastWorld(t)
	w.GetRoomInWorld(1001).Flags = nil
	Act(w, false, actor, nil, nil, nil, "$n waves.", "", ToRoom)
	if got := outputOf(out, remote.Name); got != "" {
		t.Fatalf("non-arena broadcast=%q", got)
	}
	if got := outputOf(out, local.Name); got != "Actor waves.\r\n" {
		t.Fatalf("non-arena local=%q", got)
	}
	w.GetRoomInWorld(1001).Flags = []string{"arena"}
	Act(w, false, actor, nil, nil, nil, "$n waves.", "", ToRoom)
	if got := outputOf(out, remote.Name); got != "&RBroadcast: Actor waves.&n\r\n" {
		t.Fatalf("arena control=%q", got)
	}
}

func TestArenaActSleepWritingAndFullText(t *testing.T) {
	w, actor, local, remote, out := arenaBroadcastWorld(t)
	local.SetPosition(combat.PosSleeping)
	remote.SetPlrFlag(PlrWriting, true)
	format := "$n " + strings.Repeat("full text ", 30) + "$N."
	Act(w, false, actor, local, nil, nil, format, "", ToNotVict)
	want := "&RBroadcast: Actor " + strings.Repeat("full text ", 30) + "Local.&n\r\n"
	if got := outputOf(out, local.Name); got != want {
		t.Fatalf("sleep/full text=%q", got)
	}
	if got := outputOf(out, remote.Name); got != "" {
		t.Fatalf("writing recipient=%q", got)
	}
	// TO_CHAR/TO_VICT never invoke the arena hook, even with ToSleep.
	Act(w, false, actor, remote, nil, nil, "$n waves.", "", ToChar|ToSleep)
	if got := outputOf(out, local.Name); got != want {
		t.Fatal("TO_CHAR broadcast")
	}
	Act(w, false, actor, actor, nil, nil, "$n waves.", "", ToVict|ToSleep)
	if got := outputOf(out, local.Name); got != want {
		t.Fatal("TO_VICT broadcast")
	}
}
