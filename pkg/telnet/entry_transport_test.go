package telnet

import (
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

func listenerEntryDatabase(t *testing.T) *db.DB {
	t.Helper()
	dsn := os.Getenv("DP_ENTRY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DP_ENTRY_TEST_DATABASE_URL to run PostgreSQL entry transport tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Fatal("entry transport tests require a local PostgreSQL URL")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("entry_transport_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Errorf("drop isolated entry transport schema: %v", err)
		}
	})
	q := u.Query()
	// Use a lib/pq startup option so every pooled connection, including the
	// cleanup save path, resolves the isolated schema.
	q.Set("options", "-csearch_path="+schema)
	u.RawQuery = q.Encode()
	database, err := db.New(u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func listenerEntrySeed(t *testing.T, database *db.DB) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("oraclepass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CreatePlayer(&db.PlayerRecord{
		Name: "Aiko", Password: string(hash), RoomVNum: game.MortalStartRoom, Level: 1,
		Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20, Move: 100, MaxMove: 100,
		Class: game.ClassThief, Race: game.RaceKender,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
	}); err != nil {
		t.Fatal(err)
	}
}

func listenerEntryPort(t *testing.T) int {
	t.Helper()
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func readTelnetUntil(t *testing.T, conn net.Conn, needle string) string {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var raw bytes.Buffer
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("telnet read waiting for %q: %v; output=%q", needle, err, stripTelnetCommands(raw.Bytes()))
		}
		raw.Write(buf[:n])
		visible := string(stripTelnetCommands(raw.Bytes()))
		if strings.Contains(visible, needle) {
			return visible
		}
	}
}

// TestEntryTelnetSavedIdentityEntersWorld proves the real TCP listener
// against PostgreSQL, including case-insensitive lookup and post-MOTD entry.
func TestEntryTelnetSavedIdentityEntersWorld(t *testing.T) {
	t.Setenv("JWT_SECRET", "entry-telnet-test-jwt-secret-at-least-32")
	database := listenerEntryDatabase(t)
	listenerEntrySeed(t, database)
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	manager := session.NewManager(world, database)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)

	port := listenerEntryPort(t)
	if err := Listen(port, manager); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	readTelnetUntil(t, conn, "By what name do you wish to be known?")
	if _, err := conn.Write([]byte("aiko\r\n")); err != nil {
		t.Fatal(err)
	}
	readTelnetUntil(t, conn, "Password: ")
	if _, err := conn.Write([]byte("oraclepass\r\n")); err != nil {
		t.Fatal(err)
	}
	readTelnetUntil(t, conn, "PRESS RETURN: ")
	if _, err := conn.Write([]byte("\r\n")); err != nil {
		t.Fatal(err)
	}
	readTelnetUntil(t, conn, "Make your choice: ")
	if _, err := conn.Write([]byte("1\r\n")); err != nil {
		t.Fatal(err)
	}
	readTelnetUntil(t, conn, "A Burning Hut")
	stored, err := database.GetPlayer("AIKO")
	if err != nil || stored == nil {
		t.Fatalf("saved identity after telnet entry: stored=%v err=%v", stored, err)
	}
	if count, err := database.CountPlayers(); err != nil || count != 1 {
		t.Fatalf("telnet journey player count = %d, err=%v; want one row", count, err)
	}
}

// C close_socket retains only CON_PLAYING as linkdead, never MOTD/menu.
func TestEntryTelnetNewCharacterDisconnectResumes(t *testing.T) {
	for _, atMenu := range []bool{false, true} {
		t.Run(fmt.Sprintf("menu=%t", atMenu), func(t *testing.T) {
			t.Setenv("JWT_SECRET", "entry-telnet-test-jwt-secret-at-least-32")
			database := listenerEntryDatabase(t)
			listenerEntrySeed(t, database) // Ensure the new character is mortal.
			world, err := game.NewWorld(&parser.World{Rooms: []parser.Room{
				{VNum: game.NewbieStartRoom, Name: "Entry Nursery", Zone: 80},
				{VNum: game.NewbieHometownRoom(1), Name: "Entry Infirmary", Zone: 80},
			}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(world.StopAITicker)
			manager := session.NewManager(world, database)
			t.Cleanup(manager.Stop)
			t.Cleanup(Stop)
			port := listenerEntryPort(t)
			if err := Listen(port, manager); err != nil {
				t.Fatal(err)
			}
			dial := func() net.Conn {
				conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = conn.Close() })
				readTelnetUntil(t, conn, "By what name do you wish to be known?")
				return conn
			}
			step := func(conn net.Conn, input, prompt string) {
				if _, err := conn.Write([]byte(input + "\r\n")); err != nil {
					t.Fatal(err)
				}
				readTelnetUntil(t, conn, prompt)
			}
			conn := dial()
			step(conn, "Freshwire", "Did I get that right, Freshwire (Y/N)?")
			step(conn, "y", "Give me a password for Freshwire:")
			step(conn, "freshpass", "Please retype password:")
			step(conn, "freshpass", "Do you want ANSI color (Y/N)?")
			step(conn, "n", "What is your sex (M/F)?")
			step(conn, "m", "Race:")
			step(conn, "k", "Class:")
			step(conn, "t", "Select:")
			step(conn, "k", "reroll:")
			step(conn, "y", "PRESS RETURN:")
			if atMenu {
				step(conn, "", "Make your choice:")
			}
			accepted, err := database.GetPlayer("Freshwire")
			if err != nil || accepted == nil || accepted.Level != 0 {
				t.Fatalf("accepted row: %+v, %v", accepted, err)
			}
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(3 * time.Second)
			for {
				if _, exists := manager.GetSession("Freshwire"); !exists {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("pre-world character retained as a session after TCP EOF")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if _, exists := world.GetPlayer("Freshwire"); exists {
				t.Fatal("pre-world character retained in world")
			}
			conn = dial()
			step(conn, "FRESHWIRE", "Password:")
			step(conn, "freshpass", "PRESS RETURN:")
			step(conn, "", "Make your choice:")
			step(conn, "1", "Entry Nursery")
			stored, err := database.GetPlayer("freshwire")
			if err != nil || stored == nil || stored.ID != accepted.ID || stored.Level != 1 {
				t.Fatalf("resumed row: %+v, %v; accepted ID %d", stored, err, accepted.ID)
			}
			if stored.StatStr != accepted.StatStr || stored.StatInt != accepted.StatInt || stored.StatWis != accepted.StatWis || stored.StatDex != accepted.StatDex || stored.StatCon != accepted.StatCon || stored.StatCha != accepted.StatCha {
				t.Fatal("accepted stats changed on reconnect")
			}
			if count, err := database.CountPlayers(); err != nil || count != 2 {
				t.Fatalf("player count=%d, err=%v", count, err)
			}
			// The same EOF after world entry must still retain a linkdead player.
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
			deadline = time.Now().Add(3 * time.Second)
			for {
				p, exists := world.GetPlayer("Freshwire")
				if exists && p.IsLinkless() {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("playing character was not retained as linkdead")
				}
				time.Sleep(10 * time.Millisecond)
			}
			manager.Unregister("Freshwire")
		})
	}
}
