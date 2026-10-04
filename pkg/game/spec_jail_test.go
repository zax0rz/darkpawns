package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecJail_CommandPathFallsThrough(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	player.SetRoom(8118)
	player.SetGold(100)
	player.SetMove(100)
	player.SetLevel(5)

	if specJail(w, player, nil, "say", "release") {
		t.Fatal("jail room special must reject player commands")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("jail command emitted %q", got)
	}
	if got := player.GetRoomVNum(); got != 8118 {
		t.Fatalf("jail command moved player to room %d", got)
	}
	if got := player.GetGold(); got != 100 {
		t.Fatalf("jail command changed gold to %d", got)
	}
	if got := player.GetMove(); got != 100 {
		t.Fatalf("jail command changed movement to %d", got)
	}
}

func jailTestWorld(t *testing.T) (*World, *Player, *Player, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 8117, Name: "Jail entrance", Description: "Outside the cell.\r\n"},
		{VNum: 8118, Name: "Holding cell"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	inmate := NewPlayer(1, "Inmate", 8118)
	observer := NewPlayer(2, "Observer", 8118)
	observer.SetLevel(LVL_IMMORT)
	for _, p := range []*Player{inmate, observer} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		p.SetCondition(CondFull, -1)
		p.SetCondition(CondThirst, -1)
	}
	return w, inmate, observer, out
}

func TestSpecJailPulseRelease(t *testing.T) {
	for _, pos := range []int{PosSleeping, PosResting, PosStanding} {
		t.Run(string(rune('A'+pos)), func(t *testing.T) {
			w, inmate, observer, out := jailTestWorld(t)
			inmate.SetPosition(pos)
			w.RoomActivity()
			if inmate.GetRoomVNum() != 8117 {
				t.Fatal("pulse failed to release inmate")
			}
			wantPos := pos
			if pos <= PosSleeping {
				wantPos = PosSitting
			}
			if inmate.GetPosition() != wantPos {
				t.Fatal("wrong release position")
			}
			prefix := "The guard says, 'Time's up, scum!'\r\nThe guard throws you out of the cell!\r\n\r\n"
			if !strings.HasPrefix(out[inmate.Name], prefix) || !strings.Contains(out[inmate.Name], "Jail entrance") {
				t.Fatalf("inmate output: %q", out[inmate.Name])
			}
			if want := "The guard says, 'Time's up, scum!'\r\nInmate gets thrown out of the cell!\r\n"; out[observer.Name] != want {
				t.Fatalf("observer: %q want %q", out[observer.Name], want)
			}
		})
	}
}

func TestSpecJailPulseGates(t *testing.T) {
	for _, gate := range []string{"timer", "negative-timer", "immortal", "invisible", "command"} {
		t.Run(gate, func(t *testing.T) {
			w, inmate, _, out := jailTestWorld(t)
			cmd := ""
			switch gate {
			case "timer":
				inmate.JailTimer = 1
			case "negative-timer":
				inmate.JailTimer = -1
			case "immortal":
				inmate.SetLevel(LVL_IMMORT)
			case "invisible":
				inmate.SetInvisLevel(LVL_IMMORT)
			case "command":
				cmd = "look"
			}
			if specJail(w, inmate, nil, cmd, "") || inmate.GetRoomVNum() != 8118 || len(out) != 0 {
				t.Fatalf("gate failed: %+v", out)
			}
		})
	}
}

func TestJailPointUpdateOnlyDecrements(t *testing.T) {
	w, inmate, _, out := jailTestWorld(t)
	inmate.JailTimer = 1
	w.PointUpdate()
	if inmate.JailTimer != 0 || inmate.GetRoomVNum() != 8118 {
		t.Fatal("point update released inmate rather than decrementing")
	}
	if strings.Contains(out[inmate.Name], "sentence") || strings.Contains(out[inmate.Name], "throws") {
		t.Fatalf("premature output: %+v", out)
	}
	w.RoomActivity()
	if inmate.GetRoomVNum() != 8117 {
		t.Fatal("room pulse did not finish release")
	}
}

func TestSpecJailReleaseLeavesMountInCell(t *testing.T) {
	w, inmate, _, _ := jailTestWorld(t)
	mount := newSpecProcTestMob(t, w, 8118, 1)
	inmate.MountName = mount.GetName()
	mount.SetMountRider(inmate.Name)
	if !specJail(w, inmate, nil, "", "") {
		t.Fatal("release did not intercept")
	}
	if mount.GetRoomVNum() != 8118 || inmate.GetRoomVNum() != 8117 {
		t.Fatal("bare C room relocation must leave mount behind")
	}
}
