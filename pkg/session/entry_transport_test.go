package session

import (
	"encoding/json"
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

func entryTransportManager(t *testing.T, database db.Database) *Manager {
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

func entryWebSocketJourney(t *testing.T, serverURL, name string) StateData {
	return entryWebSocketJourneyWithPassword(t, serverURL, name, "oraclepass")
}

func entryWebSocketJourneyWithPassword(t *testing.T, serverURL, name, password string) StateData {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	headers := http.Header{"Origin": []string{"https://darkpawns.org"}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer conn.Close()

	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": name})
	loginPrompt := wsReadUntilType(t, conn, MsgCharCreate)
	var prompt CharCreateData
	loginData, _ := json.Marshal(loginPrompt["data"])
	if err := json.Unmarshal(loginData, &prompt); err != nil {
		t.Fatal(err)
	}
	if prompt.Stage != "login_password" || prompt.Prompt != "Password: " || !prompt.Secret {
		t.Fatalf("name %q did not route to password state: %+v", name, prompt)
	}

	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": password})
	motd := wsReadUntilType(t, conn, MsgCharCreate)
	motdData, _ := json.Marshal(motd["data"])
	if err := json.Unmarshal(motdData, &prompt); err != nil {
		t.Fatal(err)
	}
	if prompt.Stage != "motd" {
		t.Fatalf("password success stage = %q, want motd", prompt.Stage)
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": ""})
	menu := wsReadUntilType(t, conn, MsgCharCreate)
	menuData, _ := json.Marshal(menu["data"])
	if err := json.Unmarshal(menuData, &prompt); err != nil {
		t.Fatal(err)
	}
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
// boundary against PostgreSQL: a case-variant saved name reaches the password
// state, enters through the menu, and reconnects to the same persisted row.
func TestEntryWebSocketSavedIdentityAndMenuResume(t *testing.T) {
	t.Setenv("JWT_SECRET", "entry-transport-test-jwt-secret-at-least-32")
	database := entryDatabase(t)
	want := entrySeed(t, database, "Aiko")
	manager := entryTransportManager(t, database)
	server := httptest.NewServer(http.HandlerFunc(manager.HandleWebSocket))
	t.Cleanup(server.Close)

	beforeFirstClose := entryUpdatedAt(t, database, "Aiko")
	first := entryWebSocketJourney(t, server.URL, "aiko")
	if first.Player.Name != "Aiko" || first.Room.VNum != game.MortalStartRoom {
		t.Fatalf("first WebSocket entry: player=%q room=%d", first.Player.Name, first.Room.VNum)
	}
	waitForSessionGone(t, manager, "Aiko")
	waitForEntrySave(t, database, "Aiko", beforeFirstClose)

	beforeSecondClose := entryUpdatedAt(t, database, "Aiko")
	second := entryWebSocketJourney(t, server.URL, "AIKO")
	if second.Player.Name != "Aiko" || second.Room.VNum != game.MortalStartRoom {
		t.Fatalf("reconnect WebSocket entry: player=%q room=%d", second.Player.Name, second.Room.VNum)
	}
	waitForSessionGone(t, manager, "Aiko")
	waitForEntrySave(t, database, "Aiko", beforeSecondClose)
	stored, err := database.GetPlayer("aiko")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.ID != want.ID {
		t.Fatalf("case-variant reconnect resolved ID %v, want %d", stored, want.ID)
	}
	if count, err := database.CountPlayers(); err != nil || count != 1 {
		t.Fatalf("WebSocket journey player count = %d, err=%v; want one row", count, err)
	}
}

// TestEntryWebSocketNewCharacterPersistsAtMenu proves that accepted stats are
// durable before first world entry, and that a disconnect at the menu resumes
// the same level-zero row without creating a duplicate.
func TestEntryWebSocketNewCharacterPersistsAtMenu(t *testing.T) {
	t.Setenv("JWT_SECRET", "entry-transport-test-jwt-secret-at-least-32")
	database := entryDatabase(t)
	entrySeed(t, database, "Founder")
	manager := entryTransportManager(t, database)
	server := httptest.NewServer(http.HandlerFunc(manager.HandleWebSocket))
	t.Cleanup(server.Close)

	conn, accepted := entryWebSocketNewCharacterToMenu(t, server.URL, "Freshweb")
	beforeClose := entryUpdatedAt(t, database, "Freshweb")
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	waitForSessionGone(t, manager, "Freshweb")
	waitForEntrySave(t, database, "Freshweb", beforeClose)

	stored, err := database.GetPlayer("freshweb")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.ID <= 0 {
		t.Fatalf("accepted character was not persisted: %+v", stored)
	}
	if stored.Level != 0 {
		t.Fatalf("accepted character level = %d, want level zero before first entry", stored.Level)
	}
	if got := (CharStatsDisplay{Str: stored.StatStr, Int: stored.StatInt, Wis: stored.StatWis, Dex: stored.StatDex, Con: stored.StatCon, Cha: stored.StatCha}); got != accepted {
		t.Fatalf("persisted stats = %+v, want accepted %+v", got, accepted)
	}
	if count, err := database.CountPlayers(); err != nil || count != 2 {
		t.Fatalf("accepted character count = %d, err=%v; want founder plus one candidate", count, err)
	}

	acceptedID := stored.ID
	state := entryWebSocketJourneyWithPassword(t, server.URL, "FRESHWEB", "freshpass")
	if state.Player.Name != "Freshweb" || state.Player.Level != 1 || state.Room.VNum != game.NewbieStartRoom {
		t.Fatalf("reconnected new character: player=%q level=%d room=%d", state.Player.Name, state.Player.Level, state.Room.VNum)
	}
	waitForSessionGone(t, manager, "Freshweb")
	stored, err = database.GetPlayer("FRESHWEB")
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

func entryUpdatedAt(t *testing.T, database *db.DB, name string) time.Time {
	t.Helper()
	var updatedAt time.Time
	if err := database.SQLDB().QueryRow(`SELECT updated_at FROM players WHERE LOWER(name) = LOWER($1)`, name).Scan(&updatedAt); err != nil {
		t.Fatalf("updated_at for %q: %v", name, err)
	}
	return updatedAt
}

func waitForEntrySave(t *testing.T, database *db.DB, name string, before time.Time) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		after := entryUpdatedAt(t, database, name)
		if after.After(before) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("cleanup save for %q did not update updated_at after %s", name, before)
}

func waitForSessionGone(t *testing.T, manager *Manager, name string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := manager.GetSession(name); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session %q was not removed after transport close", name)
}
