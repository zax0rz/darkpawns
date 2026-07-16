package session

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// DP-1115: quit / reallyquit split — C do_quit (src/act.other.c:72-181)
//
// C has two subcommands around one do_quit: quit (SCMD_QUIT) is safe only in
// temple/home/owned rooms and refuses elsewhere, directing to reallyquit
// (SCMD_REALLY_QUIT), which logs out anywhere but loses your equipment (no
// Crash_rentsave). The Go port had collapsed both into one unrestricted
// handler with reallyquit as a bare alias.
// ---------------------------------------------------------------------------

// captureSaveDB records every SavePlayer call so tests can inspect what the
// quit teardown actually persisted (equipment kept vs lost).
type captureSaveDB struct {
	mockAgentKeyDB
	saved []*db.PlayerRecord
}

func (m *captureSaveDB) SavePlayer(p *db.PlayerRecord) error {
	m.saved = append(m.saved, p)
	return nil
}

// makeQuitTestManager builds a Manager over the rooms C's do_quit cares about:
// safe temples 8004/8008, the unsafe newbie infirmary 8162, and the hometown
// home rooms 18201 (hometown 2) and 21202/21258 (hometown 3).
func makeQuitTestManager(t *testing.T, database db.Database) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 8004, Name: "The Temple", Zone: 1},
			{VNum: 8008, Name: "The Second Temple", Zone: 1},
			{VNum: 8162, Name: "Temple Infirmary", Zone: 1},
			{VNum: 18201, Name: "Kir-Oshi Home", Zone: 1},
			{VNum: 21202, Name: "Alaozar Home", Zone: 1},
			{VNum: 21258, Name: "Alaozar Estate", Zone: 1},
			{VNum: 9000, Name: "A Player House", Zone: 1},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return newTestManager(t, w, database)
}

// makeQuitSession registers an authenticated session whose player is live in
// the world (needed for the equipment extraction paths).
func makeQuitSession(t *testing.T, m *Manager, id int, name string, level, roomVNum int) *Session {
	t.Helper()
	s := m.NewSession()
	s.player = game.NewPlayer(id, name, roomVNum)
	s.player.Level = level
	s.playerName = name
	s.authenticated = true
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	m.mu.Lock()
	m.sessions[s.player.Name] = s
	m.mu.Unlock()
	return s
}

// equipQuitTestSword wedges a wielded sword into the player's equipment.
func equipQuitTestSword(t *testing.T, s *Session) *game.ObjectInstance {
	t.Helper()
	sword := game.NewObjectInstance(&parser.Obj{
		VNum: 9001, ShortDesc: "a steel sword", Keywords: "sword", Weight: 3,
	}, -1)
	sword.Location = game.LocEquippedPlayer(s.player.Name, game.SlotWield)
	if err := s.player.Equipment.SetSlot(game.SlotWield, sword); err != nil {
		t.Fatalf("SetSlot: %v", err)
	}
	return sword
}

// carryQuitTestBag puts a carried bag into the player's inventory.
func carryQuitTestBag(t *testing.T, s *Session) *game.ObjectInstance {
	t.Helper()
	bag := game.NewObjectInstance(&parser.Obj{
		VNum: 9002, ShortDesc: "a cloth bag", Keywords: "bag", Weight: 1,
	}, -1)
	if err := s.player.Inventory.AddItem(bag); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	bag.Location = game.LocInventoryPlayer(s.player.Name)
	return bag
}

// savedItemCounts decodes the saved record's inventory/equipment payloads.
func savedItemCounts(t *testing.T, rec *db.PlayerRecord) (inventory, equipment int) {
	t.Helper()
	var invItems, eqItems []game.SaveItemData
	if len(rec.Inventory) > 0 {
		if err := json.Unmarshal(rec.Inventory, &invItems); err != nil {
			t.Fatalf("unmarshal inventory: %v", err)
		}
	}
	if len(rec.Equipment) > 0 {
		if err := json.Unmarshal(rec.Equipment, &eqItems); err != nil {
			t.Fatalf("unmarshal equipment: %v", err)
		}
	}
	return len(invItems), len(eqItems)
}

// C: quit from a non-safe room refuses with the REALLYQUIT block (+ RECALL
// hint at level <= 5) and does NOT log out.
func TestQuitFromUnsafeRoomRefusesAndKeepsSession(t *testing.T) {
	m := makeQuitTestManager(t, nil)
	s := makeQuitSession(t, m, 1, "Quitter", 3, 8162)
	equipQuitTestSword(t, s)

	if err := ExecuteCommand(s, "quit", nil); err != nil {
		t.Fatalf("ExecuteCommand quit: %v", err)
	}

	want := "Type REALLYQUIT to quit the game and lose your eq.\r\n" +
		"Return to the temple and QUIT to leave the game and keep your equipment.\r\n" +
		"You can type RECALL to return to your temple.\r\n"
	if got := readMsgText(t, s); got != want {
		t.Fatalf("refuse message = %q, want %q", got, want)
	}
	if _, ok := m.GetSession("Quitter"); !ok {
		t.Fatal("session was torn down; C refuses the quit and keeps the session")
	}
	if _, ok := m.world.GetPlayer("Quitter"); !ok {
		t.Fatal("player was removed from the world on a refused quit")
	}
	if got := len(s.player.Equipment.GetEquippedItems()); got != 1 {
		t.Fatalf("equipment changed on refused quit: %d items, want 1", got)
	}
}

// C: quit from either temple logs out with the goodbye line, and the rent save
// keeps equipment.
func TestQuitFromSafeRoomLogsOutKeepingEquipment(t *testing.T) {
	for _, roomVNum := range []int{8004, 8008} {
		t.Run(fmt.Sprintf("room-%d", roomVNum), func(t *testing.T) {
			database := &captureSaveDB{}
			m := makeQuitTestManager(t, database)
			s := makeQuitSession(t, m, roomVNum, "Keeper", 10, roomVNum)
			equipQuitTestSword(t, s)

			if err := ExecuteCommand(s, "quit", nil); err != nil {
				t.Fatalf("ExecuteCommand quit: %v", err)
			}

			if got := readMsgText(t, s); got != "Goodbye, friend.. Come back soon!\r\n" {
				t.Fatalf("goodbye = %q, want %q", got, "Goodbye, friend.. Come back soon!\r\n")
			}
			if _, ok := m.GetSession("Keeper"); ok {
				t.Fatal("session still registered after a successful quit")
			}
			if len(database.saved) != 1 {
				t.Fatalf("SavePlayer called %d times, want 1", len(database.saved))
			}
			inventory, equipment := savedItemCounts(t, database.saved[0])
			if equipment != 1 || inventory != 0 {
				t.Fatalf("safe quit saved inventory=%d equipment=%d, want 0/1 (rent keeps eq)", inventory, equipment)
			}
		})
	}
}

// C's home rooms are safe only for their matching hometown.
func TestQuitHomeRoomGatesByHometown(t *testing.T) {
	tests := []struct {
		name     string
		room     int
		hometown int
		safe     bool
	}{
		{name: "Kir-Oshi match", room: 18201, hometown: 2, safe: true},
		{name: "Kir-Oshi mismatch", room: 18201, hometown: 3},
		{name: "Alaozar home match", room: 21202, hometown: 3, safe: true},
		{name: "Alaozar home mismatch", room: 21202, hometown: 2},
		{name: "Alaozar estate match", room: 21258, hometown: 3, safe: true},
		{name: "Alaozar estate mismatch", room: 21258, hometown: 2},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeQuitTestManager(t, nil)
			s := makeQuitSession(t, m, 100+i, "Homesteader", 10, tt.room)
			s.player.Hometown = tt.hometown

			if err := ExecuteCommand(s, "quit", nil); err != nil {
				t.Fatalf("ExecuteCommand quit: %v", err)
			}
			if tt.safe {
				if got := readMsgText(t, s); got != "Goodbye, friend.. Come back soon!\r\n" {
					t.Fatalf("safe hometown message = %q", got)
				}
				if _, ok := m.GetSession("Homesteader"); ok {
					t.Fatal("matching hometown did not permit quit")
				}
				return
			}

			want := "Type REALLYQUIT to quit the game and lose your eq.\r\n" +
				"Return to the temple and QUIT to leave the game and keep your equipment.\r\n"
			if got := readMsgText(t, s); got != want {
				t.Fatalf("mismatched hometown message = %q, want %q", got, want)
			}
			if _, ok := m.GetSession("Homesteader"); !ok {
				t.Fatal("mismatched hometown incorrectly permitted quit")
			}
		})
	}
}

func TestQuitFromOwnedHouseIsSafe(t *testing.T) {
	database := &captureSaveDB{}
	m := makeQuitTestManager(t, database)
	m.world.HouseControl = append(m.world.HouseControl, game.HouseControl{VNum: 9000, Owner: 77})
	s := makeQuitSession(t, m, 77, "Homeowner", 10, 9000)
	equipQuitTestSword(t, s)

	if err := ExecuteCommand(s, "quit", nil); err != nil {
		t.Fatalf("ExecuteCommand quit: %v", err)
	}
	if len(database.saved) != 1 {
		t.Fatalf("SavePlayer called %d times, want 1", len(database.saved))
	}
	_, equipment := savedItemCounts(t, database.saved[0])
	if equipment != 1 {
		t.Fatalf("owned-house quit saved %d equipment items, want 1", equipment)
	}
}

// C: reallyquit from a non-safe room logs out, but without Crash_rentsave the
// equipment is not persisted — the saved char has no inventory/equipment.
func TestReallyQuitFromUnsafeRoomLosesEquipment(t *testing.T) {
	database := &captureSaveDB{}
	m := makeQuitTestManager(t, database)
	s := makeQuitSession(t, m, 1, "Risker", 10, 8162)
	equipQuitTestSword(t, s)
	carryQuitTestBag(t, s)
	// Fill the inventory so the old unequip-then-extract approach could not
	// move the sword into it. Destructive logout must still remove both.
	s.player.Inventory.Capacity = 1

	if err := ExecuteCommand(s, "reallyquit", nil); err != nil {
		t.Fatalf("ExecuteCommand reallyquit: %v", err)
	}

	if got := readMsgText(t, s); got != "Goodbye, friend.. Come back soon!\r\n" {
		t.Fatalf("goodbye = %q, want %q", got, "Goodbye, friend.. Come back soon!\r\n")
	}
	if _, ok := m.GetSession("Risker"); ok {
		t.Fatal("session still registered after reallyquit")
	}
	if len(database.saved) != 1 {
		t.Fatalf("SavePlayer called %d times, want 1", len(database.saved))
	}
	inventory, equipment := savedItemCounts(t, database.saved[0])
	if equipment != 0 || inventory != 0 {
		t.Fatalf("unsafe reallyquit saved inventory=%d equipment=%d, want 0/0 (LOSTEQ)", inventory, equipment)
	}
}

func TestReallyQuitClosesDuplicateIDBeforeFinalDestructiveSave(t *testing.T) {
	database := &captureSaveDB{}
	m := makeQuitTestManager(t, database)
	quitter := makeQuitSession(t, m, 42, "Risker", 10, 8162)
	equipQuitTestSword(t, quitter)

	duplicate := makeQuitSession(t, m, 42, "Shadow", 10, 8162)
	equipQuitTestSword(t, duplicate)
	duplicateClosed := false
	duplicate.SetCloseFunc(func() { duplicateClosed = true })

	if err := ExecuteCommand(quitter, "reallyquit", nil); err != nil {
		t.Fatalf("ExecuteCommand reallyquit: %v", err)
	}
	if _, ok := m.GetSession("Shadow"); ok {
		t.Fatal("duplicate-ID session remains registered")
	}
	if !duplicateClosed {
		t.Fatal("duplicate-ID transport was not closed")
	}
	if len(database.saved) != 2 {
		t.Fatalf("SavePlayer called %d times, want duplicate + quitter", len(database.saved))
	}
	inventory, equipment := savedItemCounts(t, database.saved[len(database.saved)-1])
	if inventory != 0 || equipment != 0 {
		t.Fatalf("final save inventory=%d equipment=%d, want destructive 0/0", inventory, equipment)
	}
}

// C: reallyquit from a SAFE room still rent-saves — isokquit keeps equipment
// regardless of subcmd.
func TestReallyQuitFromSafeRoomKeepsEquipment(t *testing.T) {
	database := &captureSaveDB{}
	m := makeQuitTestManager(t, database)
	s := makeQuitSession(t, m, 1, "Cautious", 10, 8004)
	equipQuitTestSword(t, s)

	if err := ExecuteCommand(s, "reallyquit", nil); err != nil {
		t.Fatalf("ExecuteCommand reallyquit: %v", err)
	}

	if len(database.saved) != 1 {
		t.Fatalf("SavePlayer called %d times, want 1", len(database.saved))
	}
	inventory, equipment := savedItemCounts(t, database.saved[0])
	if equipment != 1 || inventory != 0 {
		t.Fatalf("safe reallyquit saved inventory=%d equipment=%d, want 0/1", inventory, equipment)
	}
}

// C: POS_FIGHTING gate message is byte-exact and neither subcommand logs out.
func TestQuitFightingRefusesWithCMessage(t *testing.T) {
	for _, cmd := range []string{"quit", "reallyquit"} {
		t.Run(cmd, func(t *testing.T) {
			m := makeQuitTestManager(t, nil)
			s := makeQuitSession(t, m, 1, "Brawler", 10, 8004)
			s.player.Fighting = "Target"
			s.player.SetPosition(combat.PosFighting)

			if err := ExecuteCommand(s, cmd, nil); err != nil {
				t.Fatalf("ExecuteCommand %s: %v", cmd, err)
			}
			want := "No way!  You're fighting for your life!\r\n"
			if got := readMsgText(t, s); got != want {
				t.Fatalf("fighting message = %q, want %q", got, want)
			}
			if _, ok := m.GetSession("Brawler"); !ok {
				t.Fatal("session torn down while fighting")
			}
		})
	}
}

func TestQuitWhileIncapacitatedDiesWithoutLoggingOut(t *testing.T) {
	m := makeQuitTestManager(t, nil)
	s := makeQuitSession(t, m, 1, "Doomed", 10, 8004)
	s.player.SetPosition(combat.PosIncap)

	if err := ExecuteCommand(s, "quit", nil); err != nil {
		t.Fatalf("ExecuteCommand quit: %v", err)
	}
	if got := readMsgText(t, s); got != "You die before your time...\r\n" {
		t.Fatalf("incap message = %q", got)
	}
	if s.player.Deaths != 1 {
		t.Fatalf("death count = %d, want 1", s.player.Deaths)
	}
	if _, ok := m.GetSession("Doomed"); !ok {
		t.Fatal("incap death incorrectly tore down the session")
	}
}

func TestQuitBroadcastsOneLeaveAndTurnsOffInfobar(t *testing.T) {
	m := makeQuitTestManager(t, nil)
	quitter := makeQuitSession(t, m, 1, "Leaver", 10, 8004)
	observer := makeQuitSession(t, m, 2, "Observer", 10, 8004)
	quitter.infobarMode = InfobarOn

	if err := ExecuteCommand(quitter, "quit", nil); err != nil {
		t.Fatalf("ExecuteCommand quit: %v", err)
	}
	if got := readMsgText(t, observer); got != "Leaver has left the game." {
		t.Fatalf("observer message = %q", got)
	}
	if got := len(observer.send); got != 0 {
		t.Fatalf("observer received %d duplicate leave messages", got)
	}
	if quitter.infobarMode != InfobarOff {
		t.Fatalf("infobar mode = %d, want off", quitter.infobarMode)
	}
}

// C registers reallyquit as its own do_quit subcmd (src/interpreter.c:657);
// it must not collapse into a bare alias of quit.
func TestReallyQuitIsNotABareAlias(t *testing.T) {
	quitEntry, ok := cmdRegistry.Lookup("quit")
	if !ok {
		t.Fatal("'quit' command not found in registry")
	}
	reallyEntry, ok := cmdRegistry.Lookup("reallyquit")
	if !ok {
		t.Fatal("'reallyquit' command not found in registry")
	}
	if quitEntry == reallyEntry {
		t.Fatal("reallyquit is a bare alias of quit; C keeps SCMD_QUIT and SCMD_REALLY_QUIT distinct")
	}
	if reallyEntry.Name != "reallyquit" {
		t.Fatalf("reallyquit resolved to primary %q, want 'reallyquit'", reallyEntry.Name)
	}
}
