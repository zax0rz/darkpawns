package game

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// These synthetic saves pin the Go 1.26.10 bytes across the JSON v2-backed
// encoding/json upgrade. Fixtures follow save_test.go; no player files are read.
func TestGo126JSONBytes(t *testing.T) {
	for _, name := range []string{"empty", "populated", "escaped"} {
		t.Run(name, func(t *testing.T) {
			p := NewPlayer(99, "SaveTestPlayer", 1001)
			p.Birth = 1700000000
			if name != "empty" {
				p.SetLevel(10)
				p.SetExp(5000)
				p.BankGold = 2500
				p.Title = "the Test Hero"
				p.Description = "A heroic look.\r\n"
				p.SpellMap = map[string]int{"zeta": 90, "alpha": 1}
				p.PoofIn, p.PoofOut = "arrives", "leaves"
				p.ActiveAffects = []*engine.Affect{{SpellID: 12, Duration: 7, Magnitude: -2, Flags: 8, Source: "fixture", StackID: "fixed", MaxStacks: 1}}
				item := NewObjectInstance(&parser.Obj{VNum: 5001, Keywords: "note", ShortDesc: "a note"}, -1)
				item.CustomData = map[string]any{"blob": []byte{0, 1, 255}, "nil": []string(nil), "empty": []string{}, "float": 1.25, "negative_zero": math.Copysign(0, -1)}
				item.Runtime.MailText = "first\r\nsecond\x00tail"
				item.Location = LocInventoryPlayer(p.Name)
				if err := p.Inventory.AddItem(item); err != nil {
					t.Fatal(err)
				}
				worn := NewObjectInstance(&parser.Obj{VNum: 5002, Keywords: "tunic", ShortDesc: "a tunic"}, -1)
				worn.Location = LocEquippedPlayer(p.Name, SlotBody)
				if err := p.Equipment.SetSlot(SlotBody, worn); err != nil {
					t.Fatal(err)
				}
			}
			if name == "escaped" {
				p.Title = "<&\" é\u2028\u2029\n\xff"
				p.Description = "tabs\tand\x00NUL\r\n"
				p.SpellMap = map[string]int{"<&é": 0, "\xff": -1}
			}
			compact, err := SerializePlayer(p)
			if err != nil {
				t.Fatal(err)
			}
			assertGo126JSON(t, name+"-save", []byte(compact))
			character, err := EncodeCharacterData(p)
			if err != nil {
				t.Fatal(err)
			}
			assertGo126JSON(t, name+"-character", character)
			var indented bytes.Buffer
			if err := encodeSave(&indented, func() error { return nil }, playerToSaveData(p)); err != nil {
				t.Fatal(err)
			}
			assertGo126JSON(t, name+"-indented", indented.Bytes())
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
