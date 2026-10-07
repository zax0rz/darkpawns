package telnet

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

// parseMSSP decodes an MSSP subnegotiation frame (IAC SB MSSP ... IAC SE)
// into its VAR/VAL pairs.
func parseMSSP(t *testing.T, data []byte) map[string]string {
	t.Helper()
	if len(data) < 5 || data[0] != IAC || data[1] != SB || data[2] != OPT_MSSP {
		t.Fatalf("MSSP frame header = %v, want IAC SB OPT_MSSP", data[:min(3, len(data))])
	}
	fields := map[string]string{}
	var name, value strings.Builder
	inValue := false
	for i := 3; i < len(data); i++ {
		b := data[i]
		switch b {
		case IAC:
			// IAC SE terminates the frame.
			if name.Len() > 0 {
				fields[name.String()] = value.String()
			}
			return fields
		case MSSP_VAR:
			if name.Len() > 0 {
				fields[name.String()] = value.String()
			}
			name.Reset()
			value.Reset()
			inValue = false
		case MSSP_VAL:
			inValue = true
		default:
			if inValue {
				value.WriteByte(b)
			} else {
				name.WriteByte(b)
			}
		}
	}
	t.Fatalf("MSSP frame not terminated: %q", data)
	return nil
}

// Check 7: sendMSSP reports CREATED from the site's history source and
// the standard field set, with world-shape counts computed from the
// loaded test world and PLAYERS counting mortal-visible playing
// characters rather than raw sessions.
func TestSendMSSP(t *testing.T) {
	pw := &parser.World{
		Zones: []parser.Zone{{Number: 1, Name: "Zone One"}, {Number: 2, Name: "Zone Two"}},
		Rooms: []parser.Room{{VNum: 100}, {VNum: 101}, {VNum: 102}},
		Mobs:  []parser.Mob{{VNum: 100}},
		Objs:  []parser.Obj{{VNum: 100}, {VNum: 101}},
	}
	world, err := game.NewWorld(pw)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	world.HelpTable = make([]game.HelpEntry, 4)

	visible := game.NewCharacterWithStats(0, "Visible", game.ClassWarrior, game.RaceHuman, 0,
		game.CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10})
	visible.Level = 5
	if err := world.AddPlayer(visible); err != nil {
		t.Fatalf("AddPlayer visible: %v", err)
	}
	hidden := game.NewCharacterWithStats(0, "Hidden", game.ClassWarrior, game.RaceHuman, 0,
		game.CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10})
	hidden.Level = game.LVL_IMPL
	hidden.InvisLevel = game.LVL_IMPL
	if err := world.AddPlayer(hidden); err != nil {
		t.Fatalf("AddPlayer hidden: %v", err)
	}

	manager := session.NewManager(world, listenerEntryDatabase(t))

	run := func(t *testing.T) map[string]string {
		client, server := net.Pipe()
		defer func() { _ = client.Close() }()
		defer func() { _ = server.Close() }()
		tc := &telnetConn{
			Conn:    server,
			br:      bufio.NewReader(server),
			wmu:     make(chan struct{}, 1),
			manager: manager,
		}
		var mu sync.Mutex
		var buf bytes.Buffer
		go func() {
			tmp := make([]byte, 4096)
			for {
				n, err := client.Read(tmp)
				if n > 0 {
					mu.Lock()
					buf.Write(tmp[:n])
					mu.Unlock()
				}
				if err != nil {
					return
				}
			}
		}()
		tc.sendMSSP()
		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		return parseMSSP(t, buf.Bytes())
	}

	wantCommon := map[string]string{
		"NAME":          "Dark Pawns",
		"PLAYERS":       "1", // only the visible mortal; wizinvis immortal hidden, no sessions exist
		"CREATED":       "1994",
		"GENRE":         "Fantasy",
		"GAMEPLAY":      "Hack and Slash",
		"STATUS":        "Live",
		"AREAS":         "2",
		"ROOMS":         "3",
		"MOBILES":       "1",
		"OBJECTS":       "2",
		"HELPFILES":     "4",
		"LEVELS":        strconv.Itoa(game.LVL_IMPL),
		"CLASSES":       strconv.Itoa(len(game.ClassNames)),
		"RACES":         strconv.Itoa(len(game.RaceNames)),
		"UTF-8":         "0",
		"MSDP":          "0",
		"MXP":           "0",
		"MSP":           "0",
		"PAY TO PLAY":   "0",
		"PAY FOR PERKS": "0",
	}

	t.Run("grapevine configured advertises INTERMUD", func(t *testing.T) {
		t.Setenv("GRAPEVINE_CLIENT_ID", "test_client")
		t.Setenv("GRAPEVINE_CLIENT_SECRET", "test_secret")
		fields := run(t)
		for name, want := range wantCommon {
			if got := fields[name]; got != want {
				t.Errorf("MSSP %s = %q, want %q", name, got, want)
			}
		}
		if got := fields["INTERMUD"]; got != "Grapevine" {
			t.Errorf("MSSP INTERMUD = %q, want Grapevine", got)
		}
	})

	t.Run("grapevine unconfigured omits INTERMUD", func(t *testing.T) {
		t.Setenv("GRAPEVINE_CLIENT_ID", "")
		t.Setenv("GRAPEVINE_CLIENT_SECRET", "")
		fields := run(t)
		if got, ok := fields["INTERMUD"]; ok {
			t.Errorf("MSSP INTERMUD = %q, want the field omitted", got)
		}
		if got := fields["CREATED"]; got != "1994" {
			t.Errorf("MSSP CREATED = %q, want 1994", got)
		}
	})
}
