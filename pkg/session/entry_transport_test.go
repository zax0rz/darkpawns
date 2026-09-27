package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func entryTransportManager(t *testing.T, database db.GameStore) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: game.MortalStartRoom, Name: "A Burning Hut", Description: "A quiet starting room.", Zone: 80},
			{VNum: game.NewbieStartRoom, Name: "A Burning Hut", Description: "Flames dance along the walls of the ruined hut.", Zone: 80},
			{VNum: game.NewbieHometownRoom(1), Name: "Temple Infirmary", Description: "A quiet infirmary tended by the temple healers.", Zone: 80},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, database)
	return manager
}

// entryWebSocketDial opens a client WebSocket with the browser client's origin.
func entryWebSocketDial(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"https://darkpawns.org"}})
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	return conn
}

// entryWebSocketFreshJourney walks the arriving-player path on a connection the
// caller owns -- password prompt, C's MOTD, the menu, world entry -- and returns the
// state frame. It takes the connection rather than dialing it so a test can hold the
// transport open and observe what dropping it does.
func entryWebSocketFreshJourney(t *testing.T, conn *websocket.Conn, name, password string) StateData {
	t.Helper()
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": name})
	var prompt CharCreateData
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "login_password" || prompt.Prompt != "Password: " || !prompt.Secret {
		t.Fatalf("name %q did not route to password state: %+v", name, prompt)
	}

	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": password})
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "motd" {
		t.Fatalf("password success stage = %q, want motd", prompt.Stage)
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": ""})
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "menu" {
		t.Fatalf("MOTD return stage = %q, want menu", prompt.Stage)
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": "1"})

	stateRaw := wsReadUntilType(t, conn, MsgState)
	stateData, _ := json.Marshal(stateRaw["data"])
	var state StateData
	if err := json.Unmarshal(stateData, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

// entryWebSocketJourneyWithPassword is the fresh journey over a connection this
// helper owns and closes: the shape every arriving player takes.
func entryWebSocketJourneyWithPassword(t *testing.T, serverURL, name, password string) StateData {
	t.Helper()
	conn := entryWebSocketDial(t, serverURL)
	defer func() { _ = conn.Close() }()
	return entryWebSocketFreshJourney(t, conn, name, password)
}

// entryReconnectState is what a reconnect journey observed about the body it took
// over. The live body is handed to the new descriptor, so the client sees the
// world's own account of it (the vars frame) rather than the fresh-entry state
// frame, and the text C sends on the way in.
type entryReconnectState struct {
	Room       int
	RoomName   string
	Level      int
	Health     int
	Transcript []string
}

// entryWebSocketReconnectJourney follows the supersession path: a login for a
// character whose previous session is still linkdead. perform_dupe_check's RECON
// branch (interpreter.c:1528-1659, reconnect.go) hands this descriptor the live
// body, so there is no MOTD and no menu, and the client sends nothing after the
// password: an empty frame whose stage names the path, C's echo_on CR LF,
// "Reconnecting.", and then the world state.
func entryWebSocketReconnectJourney(t *testing.T, serverURL, name, password string) entryReconnectState {
	t.Helper()
	conn := entryWebSocketDial(t, serverURL)
	defer func() { _ = conn.Close() }()

	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": name})
	var prompt CharCreateData
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "login_password" || prompt.Prompt != "Password: " || !prompt.Secret {
		t.Fatalf("name %q did not route to password state: %+v", name, prompt)
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": password})

	// The takeover frame: a stage that says why there is no MOTD, and no prompt.
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "reconnect" || prompt.Prompt != "" {
		t.Fatalf("expected the reconnect takeover frame, got %+v", prompt)
	}

	state := entryReconnectState{}
	for frames := 0; frames < 20; frames++ {
		kind, raw := entryWebSocketReadOne(t, conn)
		if kind == "" {
			break
		}
		data, _ := json.Marshal(raw["data"])
		switch kind {
		case MsgEvent:
			var event struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(data, &event)
			state.Transcript = append(state.Transcript, event.Text)
		case MsgVars:
			var vars struct {
				RoomVNum int    `json:"ROOM_VNUM"`
				RoomName string `json:"ROOM_NAME"`
				Level    int    `json:"LEVEL"`
				Health   int    `json:"HEALTH"`
			}
			if err := json.Unmarshal(data, &vars); err != nil {
				t.Fatalf("unmarshal reconnect vars: %v", err)
			}
			state.Room, state.RoomName = vars.RoomVNum, vars.RoomName
			state.Level, state.Health = vars.Level, vars.Health
			return state
		}
	}
	t.Fatalf("reconnect journey never received the world state; transcript=%q", state.Transcript)
	return state
}

// entryWebSocketReadOne reads exactly one frame, so a journey can assert on the
// frames a narrower helper would skip.
func entryWebSocketReadOne(t *testing.T, conn *websocket.Conn) (string, map[string]interface{}) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}
	kind, _ := msg["type"].(string)
	return kind, msg
}

// entryUnmarshalPrompt decodes a char_create frame into its prompt data.
func entryUnmarshalPrompt(t *testing.T, raw map[string]interface{}, out *CharCreateData) {
	t.Helper()
	data, _ := json.Marshal(raw["data"])
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal char_create frame: %v", err)
	}
}

// requireLinkdeadRetained waits for the session to be registered and linkless --
// the state C leaves a playing character in when its transport drops
// (comm.c:2128-2135; DP-1323) -- and returns it for further assertions.
func requireLinkdeadRetained(t *testing.T, manager *Manager, name string) *Session {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if session, ok := manager.GetSession(name); ok && session != nil && session.player != nil {
			if session.player.IsLinkless() {
				return session
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session %q was not retained linkdead after its transport dropped", name)
	return nil
}

// entryRowDifference names every persisted field that differs between two
// snapshots, so a test can assert that a transport drop rewrote what it should
// have rewritten and nothing else. Comparison is by value; the names are the
// database columns.
func entryRowDifference(before, after *db.PlayerRecord) string {
	var changed []string
	check := func(column string, a, b any) {
		if fmt.Sprint(a) != fmt.Sprint(b) {
			changed = append(changed, column)
		}
	}
	check("id", before.ID, after.ID)
	check("name", before.Name, after.Name)
	check("password_hash", before.Password, after.Password)
	check("description", before.Description, after.Description)
	check("title", before.Title, after.Title)
	check("room_vnum", before.RoomVNum, after.RoomVNum)
	check("level", before.Level, after.Level)
	check("exp", before.Exp, after.Exp)
	check("health", before.Health, after.Health)
	check("max_health", before.MaxHealth, after.MaxHealth)
	check("mana", before.Mana, after.Mana)
	check("max_mana", before.MaxMana, after.MaxMana)
	check("move", before.Move, after.Move)
	check("max_move", before.MaxMove, after.MaxMove)
	check("strength", before.Strength, after.Strength)
	check("class", before.Class, after.Class)
	check("race", before.Race, after.Race)
	check("stat_str", before.StatStr, after.StatStr)
	check("stat_str_add", before.StatStrAdd, after.StatStrAdd)
	check("stat_int", before.StatInt, after.StatInt)
	check("stat_wis", before.StatWis, after.StatWis)
	check("stat_dex", before.StatDex, after.StatDex)
	check("stat_con", before.StatCon, after.StatCon)
	check("stat_cha", before.StatCha, after.StatCha)
	check("hunger", before.Hunger, after.Hunger)
	check("thirst", before.Thirst, after.Thirst)
	check("drunk", before.Drunk, after.Drunk)
	check("hometown", before.Hometown, after.Hometown)
	check("olc_zone", before.OlcZone, after.OlcZone)
	check("inventory", string(before.Inventory), string(after.Inventory))
	check("equipment", string(before.Equipment), string(after.Equipment))
	check("character_data", string(before.CharacterData), string(after.CharacterData))
	check("failed_login_attempts", before.FailedLoginAttempts, after.FailedLoginAttempts)
	check("locked_until", before.LockedUntil, after.LockedUntil)
	return strings.Join(changed, ", ")
}

func entryWebSocketNewCharacterToMenu(t *testing.T, serverURL, name string) (*websocket.Conn, CharStatsDisplay) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	headers := http.Header{"Origin": []string{"https://darkpawns.org"}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": name})
	for _, choice := range []string{"Y", "freshpass", "freshpass", "N", "M", "H", "W", "K"} {
		wsReadUntilType(t, conn, MsgCharCreate)
		wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": choice})
	}
	statsRaw := wsReadUntilType(t, conn, MsgCharCreate)
	statsData, _ := json.Marshal(statsRaw["data"])
	var statsPrompt CharCreateData
	if err := json.Unmarshal(statsData, &statsPrompt); err != nil {
		t.Fatalf("unmarshal stats prompt: %v", err)
	}
	if statsPrompt.Stage != "stats_roll" || statsPrompt.Stats == nil {
		t.Fatalf("new character stats prompt = %+v", statsPrompt)
	}
	stats := *statsPrompt.Stats
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": "Y"})
	if motd := wsReadUntilType(t, conn, MsgCharCreate); motd["data"] == nil {
		t.Fatal("new character did not reach MOTD")
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": ""})
	menu := wsReadUntilType(t, conn, MsgCharCreate)
	menuData, _ := json.Marshal(menu["data"])
	var menuPrompt CharCreateData
	if err := json.Unmarshal(menuData, &menuPrompt); err != nil {
		t.Fatalf("unmarshal menu prompt: %v", err)
	}
	if menuPrompt.Stage != "menu" {
		t.Fatalf("new character menu stage = %q", menuPrompt.Stage)
	}
	return conn, stats
}

// TestEntryWebSocketSavedIdentityAndMenuResume proves the real WebSocket JSON
// boundary and DP-1323's reconnect contract: a case-variant saved name reaches the
// password state and enters through the menu; dropping that transport leaves the
// playing character registered linkdead with its row intact; and a second
// connection for the same name takes over that live body, resuming the same row
// rather than creating a second one.
func TestEntryWebSocketSavedIdentityAndMenuResume(t *testing.T) {
	t.Setenv("JWT_SECRET", "entry-transport-test-jwt-secret-at-least-32")
	database := entryDatabase(t)
	want := entrySeed(t, database, "Aiko")
	manager := entryTransportManager(t, database)
	server := httptest.NewServer(http.HandlerFunc(manager.HandleWebSocket))
	t.Cleanup(server.Close)

	// First journey: MOTD -> menu -> world, over a transport this test keeps so that
	// dropping it is the test's own doing rather than the helper's cleanup.
	conn := entryWebSocketDial(t, server.URL)
	first := entryWebSocketFreshJourney(t, conn, "aiko", "oraclepass")
	if first.Player.Name != "Aiko" || first.Room.VNum != game.MortalStartRoom {
		t.Fatalf("first WebSocket entry: player=%q room=%d", first.Player.Name, first.Room.VNum)
	}
	beforeDrop, err := database.GetPlayer("Aiko")
	if err != nil || beforeDrop == nil {
		t.Fatalf("row before the drop: %v", err)
	}
	// C's entry save stores load_room NOWHERE, so that the next login starts the
	// character in the start rooms for its level (interpreter.c:2186, 2194-2201).
	if beforeDrop.RoomVNum != game.LoadRoomNowhere {
		t.Fatalf("row after world entry stores load room %d, want NOWHERE (%d)",
			beforeDrop.RoomVNum, game.LoadRoomNowhere)
	}

	// The transport drops. C close_socket (comm.c:2128-2135) leaves a playing
	// character in character_list, linkless, for the reaper to extract; DP-1323
	// retains it here the same way.
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	linkdead := requireLinkdeadRetained(t, manager, "Aiko")
	if linkdead.player.GetRoom() != game.MortalStartRoom {
		t.Errorf("linkdead body stands in room %d, want %d", linkdead.player.GetRoom(), game.MortalStartRoom)
	}

	// The drop runs C's lost-link save (comm.c:2130): load_room NOWHERE, the same
	// value entry already stored, which is why no field may move. The save is on this
	// path because the session holds a real stored character (manager.go:1263-1273).
	if !manager.hasDB || linkdead.player.ID != want.ID || linkdead.isGuest {
		t.Errorf("the retained session does not run the lost-link save: hasDB=%v id=%d guest=%v",
			manager.hasDB, linkdead.player.ID, linkdead.isGuest)
	}
	afterDrop, err := database.GetPlayer("aiko")
	if err != nil || afterDrop == nil {
		t.Fatalf("row after the drop: %v", err)
	}
	if diff := entryRowDifference(beforeDrop, afterDrop); diff != "" {
		t.Errorf("the dropped transport rewrote persisted state: changed %q", diff)
	}
	if afterDrop.RoomVNum != game.LoadRoomNowhere {
		t.Errorf("lost-link save stored load room %d, want NOWHERE (%d)", afterDrop.RoomVNum, game.LoadRoomNowhere)
	}

	// Second journey, in case variation (C find_name is case-insensitive): the
	// supersession path takes over the linkdead body instead of the MOTD path.
	reconnected := entryWebSocketReconnectJourney(t, server.URL, "AIKO", "oraclepass")
	if reconnected.Room != game.MortalStartRoom || reconnected.RoomName != "A Burning Hut" {
		t.Errorf("reconnect resumed room %d %q, want %d A Burning Hut",
			reconnected.Room, reconnected.RoomName, game.MortalStartRoom)
	}
	if reconnected.Level != first.Player.Level || reconnected.Health != first.Player.Health {
		t.Errorf("reconnect resumed level %d health %d, want %d/%d",
			reconnected.Level, reconnected.Health, first.Player.Level, first.Player.Health)
	}
	if transcript := strings.Join(reconnected.Transcript, ""); !strings.Contains(transcript, "Reconnecting.") {
		t.Errorf("reconnect transcript = %q, want C's RECON line, which only a linkdead body gets", transcript)
	}
	resumed, ok := manager.GetSession("Aiko")
	if !ok || resumed == nil || resumed.player == nil || resumed.player.IsLinkless() {
		t.Errorf("after the reconnect the name is not held by a live body: ok=%v", ok)
	}

	// Same identity, same row, same persisted state: the character survived the drop
	// and the takeover, and no duplicate row appeared.
	stored, err := database.GetPlayer("aiko")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.ID != want.ID {
		t.Fatalf("case-variant reconnect resolved %+v, want id %d", stored, want.ID)
	}
	if diff := entryRowDifference(beforeDrop, stored); diff != "" {
		t.Errorf("reconnect changed persisted state: %q", diff)
	}
	if count, err := database.CountPlayers(); err != nil || count != 1 {
		t.Fatalf("WebSocket journey player count = %d, err=%v; want one row", count, err)
	}
}

// TestEntryWebSocketNewCharacterPersistsAtMenu proves that accepted stats are
// durable before first world entry, that a disconnect at the menu releases the
// session without touching the row, and that a returning login walks MOTD and menu
// into the world on the same row -- which is then retained linkdead, because by the
// end of that journey the character is playing.
func TestEntryWebSocketNewCharacterPersistsAtMenu(t *testing.T) {
	t.Setenv("JWT_SECRET", "entry-transport-test-jwt-secret-at-least-32")
	database := entryDatabase(t)
	entrySeed(t, database, "Founder")
	manager := entryTransportManager(t, database)
	server := httptest.NewServer(http.HandlerFunc(manager.HandleWebSocket))
	t.Cleanup(server.Close)

	conn, accepted := entryWebSocketNewCharacterToMenu(t, server.URL, "Freshweb")
	beforeDrop, err := database.GetPlayer("Freshweb")
	if err != nil {
		t.Fatal(err)
	}
	if beforeDrop == nil || beforeDrop.ID <= 0 {
		t.Fatalf("accepted character was not persisted: %+v", beforeDrop)
	}
	if beforeDrop.Level != 0 {
		t.Fatalf("accepted character level = %d, want level zero before first entry", beforeDrop.Level)
	}
	if got := (CharStatsDisplay{Str: beforeDrop.StatStr, Int: beforeDrop.StatInt, Wis: beforeDrop.StatWis, Dex: beforeDrop.StatDex, Con: beforeDrop.StatCon, Cha: beforeDrop.StatCha}); got != accepted {
		t.Fatalf("persisted stats = %+v, want accepted %+v", got, accepted)
	}
	if count, err := database.CountPlayers(); err != nil || count != 2 {
		t.Fatalf("accepted character count = %d, err=%v; want founder plus one candidate", count, err)
	}

	// The socket drops while the character is still at the menu. That session is
	// not CON_PLAYING, so cleanup releases it instead of leaving a linkdead body,
	// and nothing saves on the way out: the accepted stats were already durable.
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	waitForSessionGone(t, manager, "Freshweb")
	afterDrop, err := database.GetPlayer("FRESHWEB")
	if err != nil || afterDrop == nil {
		t.Fatalf("row after the menu-stage drop: %v", err)
	}
	if diff := entryRowDifference(beforeDrop, afterDrop); diff != "" {
		t.Errorf("the dropped menu-stage transport rewrote persisted state: changed %q", diff)
	}

	acceptedID := beforeDrop.ID
	state := entryWebSocketJourneyWithPassword(t, server.URL, "FRESHWEB", "freshpass")
	if state.Player.Name != "Freshweb" || state.Player.Level != 1 || state.Room.VNum != game.NewbieStartRoom {
		t.Fatalf("reconnected new character: player=%q level=%d room=%d", state.Player.Name, state.Player.Level, state.Room.VNum)
	}
	// That journey's transport dropped too, and by then the character was playing:
	// it is retained linkdead rather than released.
	requireLinkdeadRetained(t, manager, "Freshweb")
	stored, err := database.GetPlayer("FRESHWEB")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.ID != acceptedID || stored.Level != 1 {
		t.Fatalf("reconnected character persisted row = %+v, want same ID at level one", stored)
	}
	if count, err := database.CountPlayers(); err != nil || count != 2 {
		t.Fatalf("reconnected character count = %d, err=%v; want no duplicate", count, err)
	}
}

// waitForSessionGone waits for a dropped transport's session to be released. Only a
// playing character is retained linkdead (session_pump.go:46-61, DP-1323); a session
// still at the creation menu or earlier is cleaned up now.
func waitForSessionGone(t *testing.T, manager *Manager, name string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := manager.GetSession(name); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session %q was not released after its transport closed", name)
}
