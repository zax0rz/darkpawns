package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// C loads a saved identity before selecting Password (interpreter.c:1762-1795).
// These are Go SQLite restore-failure boundaries, not corrupt C fread bytes.
func TestEntryRestoreFailureFailsClosed(t *testing.T) {
	for _, raw := range []string{`{`, `{"skills":{"track":"bad"}}`} {
		for _, route := range []string{"json-password", "json-interactive", "terminal", "retry"} {
			t.Run(route+raw, func(t *testing.T) {
				database := entryDatabase(t)
				saved := entrySeed(t, database, "Aiko")
				saved.CharacterData = []byte(raw)
				if err := database.SavePlayer(saved); err != nil {
					t.Fatal(err)
				}
				before, err := database.GetPlayer("Aiko")
				if err != nil {
					t.Fatal(err)
				}
				s := entrySession(t, database)
				if p, err := db.RecordToPlayer(before, s.manager.world); err == nil || p != nil {
					t.Fatal("fixture did not cause a genuine conversion failure")
				}
				switch route {
				case "json-password":
					if err := s.handleLogin(loginMsg("AIKO", "oraclepass")); err != nil {
						t.Fatal(err)
					}
				case "json-interactive":
					if err := s.handleLogin(loginMsg("AIKO", "")); err != nil {
						t.Fatal(err)
					}
					if err := entryInput(s, "oraclepass"); err != nil {
						t.Fatal(err)
					}
				case "terminal":
					s.TerminalLine("AIKO")
					s.TerminalLine("oraclepass")
				case "retry":
					s.startNewCharFlow("Othername")
					for _, line := range []string{"N", "AIKO", "oraclepass"} {
						if err := entryInput(s, line); err != nil {
							t.Fatal(err)
						}
					}
				}
				assertFailedRestore(t, s)
				after, err := database.GetPlayer("Aiko")
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatal("failed restore changed the saved identity")
				}
				if n, err := database.CountPlayers(); err != nil || n != 1 {
					t.Fatal("failed restore changed player count")
				}
			})
		}
	}
}

func assertFailedRestore(t *testing.T, s *Session) {
	t.Helper()
	if !s.SendClosed() || s.player != nil || s.authenticated || s.charCreating || s.menuActive || s.creationSaved {
		t.Fatal("failed restore retained an enterable candidate")
	}
	if _, ok := s.manager.GetSession("Aiko"); ok {
		t.Fatal("failed restore registered a session")
	}
	if _, ok := s.manager.world.GetPlayer("Aiko"); ok {
		t.Fatal("failed restore entered the world")
	}
	if err := entryInput(s, "1"); err == nil {
		t.Fatal("failed restore accepted menu input")
	}
}

func TestEntryRestoreFailureNoObjects(t *testing.T) {
	for _, raw := range []string{`{`, `{"skills":{"track":"bad"}}`} {
		t.Run(raw, func(t *testing.T) { testEntryRestoreFailureNoObjects(t, raw) })
	}
}

func testEntryRestoreFailureNoObjects(t *testing.T, raw string) {
	database := entryDatabase(t)
	saved := entrySeed(t, database, "Aiko")
	saved.Inventory = []byte(`[{"vnum":8023}]`)
	saved.Equipment = []byte(`[{"vnum":8019,"locate":6}]`)
	saved.CharacterData = []byte(raw)
	if err := database.SavePlayer(saved); err != nil {
		t.Fatal(err)
	}
	s := entrySession(t, database)
	before := len(s.manager.world.GetAllObjects())
	if err := s.handleLogin(loginMsg("Aiko", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	assertFailedRestore(t, s)
	if got := len(s.manager.world.GetAllObjects()); got != before {
		t.Fatalf("failed character restore leaked objects: before=%d after=%d", before, got)
	}
	// A valid-data control proves these object fixtures really restore.
	saved.CharacterData = []byte(`{}`)
	if err := database.SavePlayer(saved); err != nil {
		t.Fatal(err)
	}
	next := entrySession(t, database)
	if err := next.handleLogin(loginMsg("Aiko", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if next.player == nil || !next.authenticated || len(next.manager.world.GetAllObjects()) != 2 {
		t.Fatal("valid control did not restore both fixture objects")
	}
}

func TestEntryRecordDisappearsAtPassword(t *testing.T) {
	for _, route := range []string{"json", "terminal"} {
		t.Run(route, func(t *testing.T) {
			database := entryDatabase(t)
			saved := entrySeed(t, database, "Aiko")
			s := entrySession(t, database)
			if route == "json" {
				if err := s.handleLogin(loginMsg("AIKO", "")); err != nil {
					t.Fatal(err)
				}
			} else {
				s.TerminalLine("AIKO")
			}
			if s.charStage != "login_password" {
				t.Fatal("fixture did not reach stored password prompt")
			}
			if err := database.DeletePlayer(saved.ID); err != nil {
				t.Fatal(err)
			}
			if route == "json" {
				if err := entryInput(s, "oraclepass"); err != nil {
					t.Fatal(err)
				}
			} else {
				s.TerminalLine("oraclepass")
			}
			assertFailedRestore(t, s)
			if n, err := database.CountPlayers(); err != nil || n != 0 {
				t.Fatal("disappeared password identity was recreated")
			}
		})
	}
}

func TestEntryCharacterDataLegacyAndMissingControls(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte(`{}`), []byte(`null`)} {
		t.Run(string(raw), func(t *testing.T) {
			database := entryDatabase(t)
			saved := entrySeed(t, database, "Aiko")
			saved.CharacterData = raw
			if err := database.SavePlayer(saved); err != nil {
				t.Fatal(err)
			}
			before, err := database.GetPlayer("Aiko")
			if err != nil {
				t.Fatal(err)
			}
			s := entrySession(t, database)
			if err := s.handleLogin(loginMsg("AIKO", "oraclepass")); err != nil {
				t.Fatal(err)
			}
			if s.SendClosed() || s.player == nil || !s.authenticated || !s.menuActive {
				t.Fatal("legacy data was rejected")
			}
			got, err := database.GetPlayer("Aiko")
			if err != nil || !reflect.DeepEqual(got, before) {
				t.Fatal("legacy control changed data")
			}
		})
	}
	database := entryDatabase(t)
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Unknown", "")); err != nil {
		t.Fatal(err)
	}
	if s.charStage != "confirm_name" || s.SendClosed() || s.player != nil {
		t.Fatal("initial missing identity did not start fresh creation")
	}
	// Pin a real conversion error rather than depending on a mocked return.
	raw := []byte(`{"skills":{"track":"bad"}}`)
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordToPlayer(&db.PlayerRecord{Name: "Aiko", CharacterData: raw}, s.manager.world); err == nil || !strings.Contains(err.Error(), "decode character data") {
		t.Fatal("valid JSON type corruption did not fail conversion")
	}
}

func TestEntryRestoreFailureWebSocket(t *testing.T) {
	t.Setenv("JWT_SECRET", "entry-restore-test-jwt-secret-at-least-32")
	database := entryDatabase(t)
	saved := entrySeed(t, database, "Aiko")
	saved.CharacterData = []byte(`{"skills":{"track":"bad"}}`)
	if err := database.SavePlayer(saved); err != nil {
		t.Fatal(err)
	}
	before, err := database.GetPlayer("Aiko")
	if err != nil {
		t.Fatal(err)
	}
	m := entryTransportManager(t, database)
	server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	defer server.Close()
	conn := entryWebSocketDial(t, server.URL)
	defer func() { _ = conn.Close() }()
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": "AIKO"})
	var prompt CharCreateData
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "login_password" || !prompt.Secret {
		t.Fatal("corrupt saved name skipped the password boundary")
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": "oraclepass"})
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	sawError := false
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !sawError || !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				t.Fatalf("failed restore did not send error and close: errorSeen=%v err=%v", sawError, err)
			}
			break
		}
		var msg ServerMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type == MsgState || msg.Type == MsgCharCreate {
			t.Fatalf("failed restore admitted entry: %s", raw)
		}
		if msg.Type == MsgError {
			var envelope struct {
				Data ErrorData `json:"data"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(envelope.Data.Message, "restore character: decode character data:") {
				t.Fatalf("restore error frame: %s", raw)
			}
			sawError = true
		}
	}
	if _, ok := m.world.GetPlayer("Aiko"); ok {
		t.Fatal("failed WebSocket restore entered world")
	}
	after, err := database.GetPlayer("Aiko")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("WebSocket failure rewrote corrupt row")
	}
}
