package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

func isolateOLCSaveList(t *testing.T) {
	t.Helper()
	original := olcSaveList
	olcSaveList = olc.NewSaveList()
	t.Cleanup(func() { olcSaveList = original })
}

func TestOLCDispatchAndSaveInfo(t *testing.T) {
	isolateOLCSaveList(t)
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Savebuilder", 31, 3000)
	s.olcZone = 99 // C saveinfo does not require authorization for the dirty zones.
	entry, ok := cmdRegistry.Lookup("olc")
	if !ok {
		t.Fatal("olc is not registered")
	}
	if entry.MinLevel != 31 || entry.MinPosition != combat.PosDead {
		t.Fatalf("olc gates = %d/%d", entry.MinLevel, entry.MinPosition)
	}
	file := captureMudlogFile(t)
	for _, args := range [][]string{nil, {"unrelated"}, {"sa"}} {
		if err := ExecuteCommand(s, "olc", args); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(drainSessionText(t, s), ""); got != "The database is up to date.\r\n" {
			t.Fatalf("empty info = %q", got)
		}
	}
	olcSaveList.Mark(olc.KindRoom, 30)
	olcSaveList.Mark(olc.KindMob, 31)
	olcSaveList.Mark(olc.KindObject, 30)
	olcSaveList.Mark(olc.KindZone, 32)
	olcSaveList.Mark(olc.KindShop, 31)
	olcSaveList.Mark(olc.KindRoom, 30) // must not move to the head
	want := "The following OLC components need saving:-\r\n - Shops for zone 31.\r\n - Zone info for zone 32.\r\n - Objects for zone 30.\r\n - Mobiles for zone 31.\r\n - Rooms for zone 30.\r\n"
	if err := ExecuteCommand(s, "olc", []string{"sa"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(drainSessionText(t, s), ""); got != want {
		t.Fatalf("dirty info = %q, want %q", got, want)
	}
	if len(olcSaveList.Ordered()) != 5 || file.Len() != 0 {
		t.Fatal("save-info wrote or logged")
	}
	// do_olc returns before parsing for an NPC, including a switched descriptor.
	s.isSwitched = true
	s.switchedMob = &game.MobInstance{}
	if err := cmdOlc(s, []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if len(s.send) != 0 || len(olcSaveList.Ordered()) != 5 {
		t.Fatal("NPC olc produced output or saved")
	}
}
