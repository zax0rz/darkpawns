package db

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Exercise the actual JSON columns produced by PlayerToRecord, not a stand-in
// schema. The synthetic inventory/equipment follow convert_test.go fixtures.
func TestGo126JSONBytes(t *testing.T) {
	for _, populated := range []bool{false, true} {
		name := "empty"
		if populated {
			name = "stateful"
		}
		t.Run(name, func(t *testing.T) {
			p := game.NewPlayer(456, "Rebooter", 1001)
			p.Birth = 1700000000
			if populated {
				p.Title = "<&\" é\u2028\xff"
				item := game.NewObjectInstance(&parser.Obj{VNum: 8020, ShortDesc: "a plain token", Keywords: "token"}, -1)
				item.Runtime.ShortDescOverride = "a personalized token"
				item.Runtime.MailText = "line one\r\nline two\x00tail"
				item.CustomData = map[string]any{"bytes": []byte{0, 255}, "nil": map[string]int(nil), "empty": map[string]int{}, "number": 1.25}
				item.Location = game.LocInventoryPlayer(p.Name)
				if err := p.Inventory.AddItem(item); err != nil {
					t.Fatal(err)
				}
				worn := game.NewObjectInstance(&parser.Obj{VNum: 8019, ShortDesc: "a tunic", Keywords: "tunic"}, -1)
				worn.Location = game.LocEquippedPlayer(p.Name, game.SlotBody)
				if err := p.Equipment.SetSlot(game.SlotBody, worn); err != nil {
					t.Fatal(err)
				}
			}
			record, err := PlayerToRecord(p, nil)
			if err != nil {
				t.Fatal(err)
			}
			assertGo126JSON(t, name+"-inventory", record.Inventory)
			assertGo126JSON(t, name+"-equipment", record.Equipment)
			assertGo126JSON(t, name+"-character", record.CharacterData)
		})
	}
}

func assertGo126JSON(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", "go126-json", name+".json")
	if os.Getenv("DP_CAPTURE_GO126_JSON") == "1" {
		if runtime.Version() != "go1.26.10" {
			t.Fatal("JSON baselines must be captured with Go 1.26.10")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s changed from Go 1.26.10 bytes:\n got %q\nwant %q", name, got, want)
	}
}
