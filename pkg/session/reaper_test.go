package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeTestManagerWithVoidRooms builds a Manager whose world contains the
// void room (vnum 1), disconnect room (vnum 3), and a mortal room, so idle
// lifecycle tests can drive the C thresholds deterministically.
func makeTestManagerWithVoidRooms(t *testing.T) *Manager {
	return makeTestManagerWithVoidRoomsAndDB(t, nil)
}

func makeTestManagerWithVoidRoomsAndDB(t *testing.T, database db.GameStore) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1, Name: "Limbo", Zone: 0},
			{VNum: 3, Name: "A Totally Empty Room", Zone: 0},
			{VNum: 1001, Name: "Room A", Zone: 1},
			{VNum: game.MortalStartRoom, Name: "The Adventurers Guild", Zone: 80},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return newTestManager(t, w, database)
}

// registerTestSession adds a test session to the manager and its player to the
// world, marking the IP connection count as already decremented so cleanup
// paths do not touch the empty IP counter.
func registerTestSession(t *testing.T, m *Manager, s *Session, key string) {
	t.Helper()
	s.connCountDecremented = true
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	m.mu.Lock()
	m.sessions[key] = s
	m.mu.Unlock()
}

// wsCommand builds the MsgCommand frame a WebSocket client sends, exactly as
// the browser/direct client sends it and as readPump hands to handleMessage.
func wsCommand(t *testing.T, command string) []byte {
	t.Helper()
	data, err := json.Marshal(CommandData{Command: command})
	if err != nil {
		t.Fatalf("marshal CommandData: %v", err)
	}
	msg, err := json.Marshal(ClientMessage{Type: MsgCommand, Data: data})
	if err != nil {
		t.Fatalf("marshal ClientMessage: %v", err)
	}
	return msg
}

// TestConnectedPlayerWithAncientLastActiveUntouched: the only cleanup sweep
// retained after DP-1311 (Manager.ExtractPendingChars, which runs deferred
// world extractions) must leave an authenticated, connected player alone no
// matter how old their last inbound activity is. Wall-clock silence is not a
// lifecycle signal (C comm.c: read failure closes a socket; silence does not).
func TestConnectedPlayerWithAncientLastActiveUntouched(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Ancient", 1001, true)
	s.lastActive.Store(time.Now().Add(-24 * time.Hour).UnixNano())
	registerTestSession(t, m, s, "Ancient")

	m.ExtractPendingChars()

	if _, ok := m.GetSession("Ancient"); !ok {
		t.Error("connected session with ancient lastActive must stay registered")
	}
	if _, ok := m.world.GetPlayer("Ancient"); !ok {
		t.Error("connected player with ancient lastActive must stay in the world")
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Errorf("player room = %d, want 1001 (no move without idle ticks)", got)
	}
	if got := s.player.GetWasInRoom(); got != 0 {
		t.Errorf("WasInRoom = %d, want 0 (never voided)", got)
	}
}

// TestCommandResetsIdleTimerWhilePlaying: a command accepted on the WebSocket
// path (handleMessage MsgCommand) resets a nonzero idle timer while the
// player is in their room — comm.c:600-601.
func TestCommandResetsIdleTimerWhilePlaying(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Typer", 1001, true)
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	registerTestSession(t, m, s, "Typer")
	s.player.SetIdleTimer(5)

	if err := s.handleMessage(wsCommand(t, "look")); err != nil {
		t.Fatalf("handleMessage(MsgCommand): %v", err)
	}

	if got := s.player.GetIdleTimer(); got != 0 {
		t.Errorf("IdleTimer = %d, want 0 after a command", got)
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Errorf("room = %d, want 1001 (not in void, no move)", got)
	}
	if got := s.player.GetWasInRoom(); got != 0 {
		t.Errorf("WasInRoom = %d, want 0", got)
	}
}

// TestTelnetCommandResetsIdleTimerWhilePlaying: the telnet path (TerminalLine)
// reaches the same handleCommand seam, so the reset applies there too (R5e —
// both transports converge before dispatch).
func TestTelnetCommandResetsIdleTimerWhilePlaying(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "TelTyper", 1001, true)
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	s.terminalNamed = true // name accepted: lines route to the interpreter
	registerTestSession(t, m, s, "TelTyper")
	s.player.SetIdleTimer(5)

	if !s.TerminalLine("look") {
		t.Fatal("TerminalLine(look) closed the connection")
	}

	if got := s.player.GetIdleTimer(); got != 0 {
		t.Errorf("IdleTimer = %d, want 0 after a telnet command", got)
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Errorf("room = %d, want 1001", got)
	}
}

// TestCommandFromVoidReturnsPlayer: a command from a voided player resets the
// timer and returns them to WasInRoom before dispatch — comm.c:602-608.
// Covers both transport seams.
func TestCommandFromVoidReturnsPlayer(t *testing.T) {
	for _, tc := range []struct {
		name string
		send func(t *testing.T, s *Session)
	}{
		{"websocket", func(t *testing.T, s *Session) {
			if err := s.handleMessage(wsCommand(t, "look")); err != nil {
				t.Fatalf("handleMessage(MsgCommand): %v", err)
			}
		}},
		{"telnet", func(t *testing.T, s *Session) {
			if !s.TerminalLine("look") {
				t.Fatal("TerminalLine(look) closed the connection")
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManagerWithVoidRooms(t)
			s := makeTestSession(t, m, "Returner", 1, true)
			s.limiter = rate.NewLimiter(rate.Inf, 1000)
			s.terminalNamed = true
			registerTestSession(t, m, s, "Returner")
			s.player.SetWasInRoom(1001)
			s.player.SetIdleTimer(9)

			tc.send(t, s)

			if got := s.player.GetIdleTimer(); got != 0 {
				t.Errorf("IdleTimer = %d, want 0", got)
			}
			if got := s.player.GetRoom(); got != 1001 {
				t.Errorf("room = %d, want 1001 (returned from void)", got)
			}
			if got := s.player.GetWasInRoom(); got != 0 {
				t.Errorf("WasInRoom = %d, want 0 after return", got)
			}
		})
	}
}

// TestDeferredDrainCommandResetsIdleTimer: a command that arrives while
// wait>0 is deferred to the per-pulse drain queue (DP-1201, comm.c:603) and
// must be reset when dequeued for execution, not only at intake.
func TestDeferredDrainCommandResetsIdleTimer(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Drainer", 1001, true)
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	registerTestSession(t, m, s, "Drainer")
	s.player.SetWaitStatePulses(2) // wait>0: the next command defers to the drain queue

	if err := s.handleMessage(wsCommand(t, "look")); err != nil {
		t.Fatalf("handleMessage(MsgCommand): %v", err)
	}
	if got := s.queueLen(); got != 1 {
		t.Fatalf("queueLen = %d, want 1 (command deferred)", got)
	}

	// Simulate the silence that accumulates while the command waits: only the
	// drain-time reset can clear this.
	s.player.SetIdleTimer(7)
	// One command drains per pulse (comm.c:603 decrements wait first); step
	// the heartbeat until the deferred command executes.
	for i := 0; i < 4 && s.queueLen() > 0; i++ {
		m.DrainInputQueues()
	}

	if got := s.queueLen(); got != 0 {
		t.Errorf("queueLen = %d, want 0 after drain", got)
	}
	if got := s.player.GetIdleTimer(); got != 0 {
		t.Errorf("IdleTimer = %d, want 0 after drain-time reset", got)
	}
}

// TestNinthTickVoidsMortal: C's check_idling (limits.c:419-437) increments
// the timer once per point_update tick and voids only when timer > 8
// (IDLE_TO_VOID, src/utils.h:131): eight ticks in place, room 1 on tick nine.
func TestNinthTickVoidsMortal(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Silent", 1001, true)
	registerTestSession(t, m, s, "Silent")

	for tick := 1; tick <= 8; tick++ {
		m.world.CheckIdling(s.player)
		if got := s.player.GetRoom(); got != 1001 {
			t.Fatalf("tick %d: room = %d, want 1001 (void before tick nine)", tick, got)
		}
		if got := s.player.GetWasInRoom(); got != 0 {
			t.Fatalf("tick %d: WasInRoom = %d, want 0 before void", tick, got)
		}
	}

	m.world.CheckIdling(s.player) // tick nine

	if got := s.player.GetRoom(); got != 1 {
		t.Errorf("tick nine: room = %d, want 1 (void)", got)
	}
	if got := s.player.GetWasInRoom(); got != 1001 {
		t.Errorf("tick nine: WasInRoom = %d, want 1001", got)
	}
}

func TestNinthTickVoidsMortalAndStopsCombat(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	m.world.SetCombatEngine(m.combatEngine)
	idler := makeTestSession(t, m, "Idler", 1001, true)
	opponent := makeTestSession(t, m, "Opponent", 1001, true)
	registerTestSession(t, m, idler, "Idler")
	registerTestSession(t, m, opponent, "Opponent")

	if err := m.combatEngine.StartCombat(idler.player, opponent.player); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	if !m.combatEngine.IsFighting("Idler") {
		t.Fatal("precondition: combat pair was not enrolled")
	}

	for tick := 1; tick <= 9; tick++ {
		m.world.CheckIdling(idler.player)
	}

	if got := idler.player.GetFighting(); got != "" {
		t.Errorf("idler fighting = %q, want empty", got)
	}
	if got := opponent.player.GetFighting(); got != "" {
		t.Errorf("opponent fighting = %q, want empty", got)
	}
	if m.combatEngine.IsFighting("Idler") || m.combatEngine.IsFighting("Opponent") {
		t.Error("combat-engine pair remains after void transition")
	}
	if got := idler.player.GetPosition(); got != combat.PosStanding {
		t.Errorf("idler position = %d, want standing", got)
	}
	if got := opponent.player.GetPosition(); got != combat.PosStanding {
		t.Errorf("opponent position = %d, want standing", got)
	}

	idler.resetIdleOnCommand()
	if got := idler.player.GetRoom(); got != 1001 {
		t.Errorf("room after return = %d, want 1001", got)
	}
	if got := idler.player.GetPosition(); got != combat.PosStanding {
		t.Errorf("position after return = %d, want standing", got)
	}
}

// TestImmortalIdleImmunity: check_idling only voids/disconnects mortals
// (limits.c:424-425 gates the thresholds on GET_LEVEL < LVL_IMMORT), but the
// timer increment itself is unconditional — the users idle column ticks for
// immortals too. DP-1311 review F5.
func TestImmortalIdleImmunity(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Wizzen", 1001, true)
	s.player.Level = game.LVL_IMMORT
	registerTestSession(t, m, s, "Wizzen")

	for tick := 0; tick < 12; tick++ {
		m.world.CheckIdling(s.player)
	}

	if got := s.player.GetRoom(); got != 1001 {
		t.Errorf("immortal room = %d, want 1001 after 12 ticks", got)
	}
	if got := s.player.GetWasInRoom(); got != 0 {
		t.Errorf("immortal WasInRoom = %d, want 0", got)
	}
	if got := s.player.GetIdleTimer(); got != 12 {
		t.Errorf("immortal IdleTimer = %d, want 12 (C increments unconditionally)", got)
	}
}

// TestConnectedIdleDisconnectClosesSession: review finding F1 — a
// still-connected player past IDLE_DISCONNECT is force-rented and
// DISCONNECTED (limits.c:438-451: close_socket, desc = NULL, extract_char).
// Go must retire the session and close the transport, never show the main
// menu the way a death or legal quit does (extract_char_final only sends
// CON_MENU when desc is still set, handler.c:1172-1175).
func TestConnectedIdleDisconnectClosesSession(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "IdleLink", 1001, true)
	// transportDone nil ⇒ hasTransport() true: this player is CONNECTED.
	registerTestSession(t, m, s, "IdleLink")

	for tick := 0; tick < 31; tick++ {
		m.world.CheckIdling(s.player)
	}
	m.ExtractPendingChars()

	if _, ok := m.GetSession("IdleLink"); ok {
		t.Error("connected idle-disconnect must unregister the session (no menu)")
	}
	if _, ok := m.world.GetPlayer("IdleLink"); ok {
		t.Error("connected idle-disconnect must extract the character")
	}
	if !s.SendClosed() {
		t.Error("connected idle-disconnect must close the session's send channel")
	}
}

// TestIdleDisconnectKeepsObjectsRentStyle: free_rent is YES (src/config.c:106),
// so C's idle disconnect runs Crash_rentsave(ch, 0) (limits.c:445-446): norent
// objects are destroyed, the rest leave with the character — nothing is
// dropped in room 3. Extraction must not scatter an idler's inventory on the
// disconnect-room floor.
func TestIdleDisconnectKeepsObjectsRentStyle(t *testing.T) {
	database := &captureSaveDB{}
	m := makeTestManagerWithVoidRoomsAndDB(t, database)
	s := makeTestSession(t, m, "PackMule", 1001, true)
	registerTestSession(t, m, s, "PackMule")

	keepsake := game.NewObjectInstance(&parser.Obj{VNum: 4299, ShortDesc: "a worn locket"}, 0)
	keepsake.Location = game.LocInventoryPlayer(s.player.Name)
	s.player.Inventory.Items = append(s.player.Inventory.Items, keepsake)
	norent := game.NewObjectInstance(&parser.Obj{
		VNum: 4300, ShortDesc: "a melting token", ExtraFlags: [4]int{game.FlagNoRent},
	}, 0)
	norent.Location = game.LocInventoryPlayer(s.player.Name)
	s.player.Inventory.Items = append(s.player.Inventory.Items, norent)

	for tick := 0; tick < 31; tick++ {
		m.world.CheckIdling(s.player)
	}
	m.ExtractPendingChars()

	if got := len(m.world.GetItemsInRoom(3)); got != 0 {
		t.Errorf("room 3 has %d items after idle disconnect, want 0 (rent file keeps them)", got)
	}
	if !s.player.RentedOut {
		t.Error("idle-disconnected character must be marked RentedOut (Crash_rentsave)")
	}
	if len(s.player.Inventory.Items) != 1 {
		t.Errorf("idle-disconnected inventory = %d items, want 1 (leaves with the character)", len(s.player.Inventory.Items))
	}
	if len(database.saved) != 1 {
		t.Fatalf("SavePlayer called %d times, want 1", len(database.saved))
	}
	inventory, equipment := savedItemCounts(t, database.saved[0])
	if inventory != 1 || equipment != 0 {
		t.Errorf("saved inventory=%d equipment=%d, want 1/0 (rent keeps rentable, excludes NORENT)", inventory, equipment)
	}
}

// TestLinkdeadPlayerFollowsTickDrivenLifecycle: a genuinely dead transport
// leaves the character linkdead in the world (DP-1323); the ordinary
// tick-driven CheckIdling path then voids it on the ninth tick and extracts
// it when the timer passes IDLE_DISCONNECT (30) — and the tick-driven
// extraction retires the dead session too, leaving neither a player nor a
// stale registration (DP-1311 requirement D).
func TestLinkdeadPlayerFollowsTickDrivenLifecycle(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Ghost", 1001, true)
	s.transportDone = make(chan struct{})
	registerTestSession(t, m, s, "Ghost")

	// Dead transport: retained linkdead, as C's close_socket leaves the char.
	if !m.HandleTransportDisconnect(s) {
		t.Fatal("HandleTransportDisconnect did not retain the playing character")
	}
	if !s.player.IsLinkless() {
		t.Fatal("character not marked linkless after transport death")
	}

	// Before the C tick threshold the character stays in place, linkdead.
	for tick := 1; tick <= 8; tick++ {
		m.world.CheckIdling(s.player)
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Fatalf("linkdead tick 8: room = %d, want 1001", got)
	}
	if _, ok := m.GetSession("Ghost"); !ok {
		t.Fatal("linkdead session must stay registered before the tick threshold")
	}

	// Tick nine voids it, same as a connected player.
	m.world.CheckIdling(s.player)
	if got := s.player.GetRoom(); got != 1 {
		t.Fatalf("linkdead tick nine: room = %d, want 1", got)
	}

	// ticks 10..31: timer passes IDLE_DISCONNECT (30) on the next tick.
	for tick := 10; tick <= 31; tick++ {
		m.world.CheckIdling(s.player)
	}

	// Extraction is deferred to the world's extract pass, which the manager
	// drains each heartbeat; that drain must retire the dead session.
	m.ExtractPendingChars()

	if _, ok := m.world.GetPlayer("Ghost"); ok {
		t.Error("extracted linkdead player must be gone from the world")
	}
	if _, ok := m.GetSession("Ghost"); ok {
		t.Error("extracted linkdead session must be unregistered (no stale registration)")
	}
	if !s.SendClosed() {
		t.Error("retired linkdead session must have its send channel closed (no channel leak)")
	}
}

// TestAbruptCloseLeavesCharacterLinkdead: closing the client socket of a
// playing character without quitting leaves the character in the world,
// linkless and still registered, as C's close_socket does and as telnet does
// (DP-1323). The tick-driven lifecycle — not any wall-clock sweep — later
// extracts it and cleans the session up.
func TestAbruptCloseLeavesCharacterLinkdead(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)

	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{}
	headers.Set("Origin", "https://darkpawns.org")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Create a new character through the WebSocket so the session is registered.
	wsWrite(t, client, MsgLogin, map[string]interface{}{
		"player_name": "WritePumpGhost",
		"password":    "hunter2",
		"new_char":    true,
	})
	charStages := []string{"Y", "hunter2", "hunter2", "N", "M", "H", "W", "K"}
	for _, choice := range charStages {
		wsReadUntilType(t, client, MsgCharCreate)
		wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": choice})
	}
	wsReadUntilType(t, client, MsgCharCreate)
	wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": "Y"})
	wsReadUntilType(t, client, MsgCharCreate)
	wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": ""})
	wsReadUntilType(t, client, MsgCharCreate)
	wsWrite(t, client, MsgCharInput, map[string]interface{}{"choice": "1"})
	wsReadUntilType(t, client, MsgState)

	if _, ok := m.GetSession("WritePumpGhost"); !ok {
		t.Fatal("session should be registered after char creation")
	}

	// Abruptly close the client socket without sending quit.
	_ = client.Close()

	// Wait for the pumps to notice.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s, ok := m.GetSession("WritePumpGhost")
		if !ok {
			t.Fatal("a dropped connection removed the character; C leaves it linkdead")
		}
		if s.player != nil && s.player.IsLinkless() {
			// Give the writer time to exit; the session must stay registered.
			time.Sleep(200 * time.Millisecond)
			if _, ok := m.GetSession("WritePumpGhost"); !ok {
				t.Fatal("the linkdead session was unregistered after the writer exited")
			}
			// Drive the C lifecycle: 9 ticks to the void, then past
			// IDLE_DISCONNECT; the extraction drain retires the session.
			for i := 0; i < 31; i++ {
				m.world.CheckIdling(s.player)
			}
			m.ExtractPendingChars()
			if _, ok := m.GetSession("WritePumpGhost"); ok {
				t.Fatal("tick-driven extraction left a linkdead WebSocket session registered")
			}
			if _, ok := m.world.GetPlayer("WritePumpGhost"); ok {
				t.Fatal("tick-driven extraction left the character in the world")
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("character never went linkless after the client disconnected")
}

// A failed writer ping can exit before readPump observes the closed socket.
// Both pump defers must converge on linkdead retention regardless of order.
func TestWriterFirstDisconnectLeavesCharacterLinkdead(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "WriterFirst", 1001, true)
	s.transportDone = make(chan struct{})
	registerTestSession(t, m, s, s.playerName)

	s.finishWebSocketTransport() // writer exits first
	s.finishWebSocketTransport() // reader exits later

	if _, ok := m.GetSession(s.playerName); !ok {
		t.Fatal("writer-first disconnect unregistered the playing character")
	}
	if !s.player.IsLinkless() {
		t.Fatal("writer-first disconnect did not mark the character linkless")
	}
}
