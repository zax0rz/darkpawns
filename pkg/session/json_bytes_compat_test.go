package session

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Capture the raw envelopes queued by the real GMCP senders, including the
// escaped inner JSON string. Do not decode/re-encode or normalize these bytes.
func TestGo126JSONBytes(t *testing.T) {
	s := newGMCPSession(t)
	s.gmcp.enabled = true
	s.gmcpSync()
	s.gmcpRoomInfoForPlayer(1001, s.player)
	s.gmcpRoomInfoForPlayer(1002, s.player)
	s.player.SetLevel(game.LVL_IMMORT)
	s.gmcpRoomInfoForPlayer(1001, s.player)
	s.gmcpChannelText("tell", "Someone", "Someone tells you, 'boo'\r\n")
	s.gmcpChannelText("say", "<&é", "quotes\" slash\\ NUL\x00 é\u2028\u2029\xff\r\n")
	s.sendGMCP("Client.GUI", gmcpClientGUI{URL: "https://example.invalid/a?x=1&y=2", Version: "3"})
	s.sendGMCP("Client.Map", gmcpClientMap{URL: "https://example.invalid/map.xml", Version: "fixture"})
	s.sendGMCP("Room.Info", gmcpRoomInfo{})
	s.sendGMCP("Room.Info", gmcpRoomInfo{Exits: map[string]int{}})
	s.HandleGMCP("Core.Ping", "")

	var got bytes.Buffer
	for range 13 {
		select {
		case raw := <-s.send:
			got.Write(raw)
			got.WriteByte('\n')
		default:
			t.Fatal("missing GMCP fixture envelope")
		}
	}
	select {
	case <-s.send:
		t.Fatal("unexpected GMCP fixture envelope")
	default:
	}
	path := filepath.Join("testdata", "go126-json", "gmcp-envelopes.jsonl")
	if os.Getenv("DP_CAPTURE_GO126_JSON") == "1" {
		if runtime.Version() != "go1.26.10" {
			t.Fatal("JSON baselines must be captured with Go 1.26.10")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("GMCP envelopes changed from Go 1.26.10:\n got %q\nwant %q", got.Bytes(), want)
	}
}
