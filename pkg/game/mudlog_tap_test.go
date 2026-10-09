package game

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/mudlog"
)

// Design §9: with the observer queue full, MudLog returns at once and the
// immortal still gets the line. The process feed has no dispatcher during
// package tests, so DefaultQueue taps fill it exactly; the next MudLog call
// exercises the drop path on a full queue.
func TestMudLogTapNeverBlocksTheImmortalLine(t *testing.T) {
	w, _, _, _, output := newChannelWorld(t)
	imm := channelPlayer(t, w, len(w.GetAllPlayers())+50, "Immortal", 1001)
	imm.Level = lvlImmort
	imm.SetPlrFlag(PrfLog1, true)
	imm.SetPlrFlag(PrfLog2, true)

	prev := getImmortalSessionProvider()
	SetImmortalSessionProvider(&testSessions{imm})
	defer SetImmortalSessionProvider(prev)

	for i := 0; i < mudlog.DefaultQueue; i++ {
		mudlog.Tap("", "filler", 1, 31, false)
	}

	done := make(chan struct{})
	go func() {
		MudLog("overflow line", MudlogBrief, lvlImmort, false)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("MudLog blocked on a full observer queue")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if strings.Contains(channelOutput(output, "Immortal"), "[ overflow line ]") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("immortal never got the line; saw %q", channelOutput(output, "Immortal"))
		}
		time.Sleep(time.Millisecond)
	}
}
