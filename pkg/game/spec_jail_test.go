package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

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
			wantInmate := "The guard says, 'Time's up, scum!'\r\n" +
				"The guard throws you out of the cell!\r\n\r\n" +
				"Jail entrance\r\nOutside the cell.\r\n[ Exits: None! ]\r\n"
			if got := out[inmate.Name]; got != wantInmate {
				t.Fatalf("inmate output = %q, want %q", got, wantInmate)
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
			inmate.SetGold(100)
			inmate.SetMove(100)
			cmd := ""
			arg := ""
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
				cmd = "say"
				arg = "release"
			}
			if specJail(w, inmate, nil, cmd, arg) || inmate.GetRoomVNum() != 8118 || inmate.GetGold() != 100 || inmate.GetMove() != 100 || len(out) != 0 {
				t.Fatalf("gate failed: room=%d gold=%d move=%d output=%+v", inmate.GetRoomVNum(), inmate.GetGold(), inmate.GetMove(), out)
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
