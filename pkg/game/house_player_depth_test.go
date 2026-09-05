package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoHouseMatchesCArgumentAndStateBoundaries(t *testing.T) {
	oldFilename := houseControlFilename
	houseControlFilename = filepath.Join(t.TempDir(), "house", "house_control.json")
	defer func() { houseControlFilename = oldFilename }()

	oldNameByID, oldIDByName := getPlayerNameByID, getPlayerIDByName
	RegisterHousePlayerLookup(
		func(id int64) string {
			switch id {
			case 1:
				return "Owner"
			case 2:
				return "Guest"
			default:
				return ""
			}
		},
		func(name string) int64 {
			switch {
			case strings.EqualFold(name, "Owner"):
				return 1
			case strings.EqualFold(name, "Guest"):
				return 2
			default:
				return -1
			}
		},
	)
	defer func() { RegisterHousePlayerLookup(oldNameByID, oldIDByName) }()

	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "House", Zone: 1, Flags: []string{"house"}}}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	defer w.StopAITicker()

	owner := NewPlayer(1, "Owner", 1001)
	guest := NewPlayer(2, "Guest", 1001)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatalf("AddPlayer(owner): %v", err)
	}
	if err := w.AddPlayer(guest); err != nil {
		t.Fatalf("AddPlayer(guest): %v", err)
	}
	w.HouseControl = []HouseControl{{VNum: 1001, Owner: 1}}

	var output strings.Builder
	w.MessageSink = func(_ string, message []byte) { output.Write(message) }
	reset := func() { output.Reset() }

	// C's do_house has no unknown-subcommand fallback.
	w.DoHouse(owner, "nonsense")
	if got := output.String(); got != "" {
		t.Fatalf("unknown subcommand output = %q, want silent C path", got)
	}

	// two_arguments() skips fill words and ignores the remainder.
	reset()
	w.DoHouse(owner, "guest the Guest trailing words are ignored")
	if got := output.String(); got != "Guest added.\r\n" {
		t.Fatalf("fill-word guest add = %q", got)
	}

	reset()
	w.DoHouse(owner, "guest")
	if got := output.String(); got != "Guests of your house:\r\nGuest\r\n" {
		t.Fatalf("guest list = %q", got)
	}

	reset()
	w.DoHouse(owner, "transfer Guest trailing words are ignored")
	if got := output.String(); got != "House transfered.\r\n" {
		t.Fatalf("transfer output = %q, want C spelling", got)
	}
	if got := w.HouseControl[0].Owner; got != 2 {
		t.Fatalf("owner after transfer = %d, want 2", got)
	}

	// C mutates the in-memory record but does not save it in do_house.
	data, err := os.ReadFile(houseControlFilename)
	if err != nil {
		t.Fatalf("read house control after transfer: %v", err)
	}
	var saved []HouseControl
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal house control: %v", err)
	}
	if len(saved) != 1 || saved[0].Owner != 1 {
		t.Fatalf("saved owner after transfer = %#v, want unsaved owner 1", saved)
	}

	// The transfer leaves the former owner behind the primary-owner gate.
	reset()
	w.DoHouse(owner, "guest")
	if got := output.String(); got != "Only the primary owner can set guests.\r\n" {
		t.Fatalf("former-owner gate = %q", got)
	}
}

func TestDoHouseHandlesMalformedAndFullGuestControls(t *testing.T) {
	oldFilename := houseControlFilename
	houseControlFilename = filepath.Join(t.TempDir(), "house", "house_control.json")
	defer func() { houseControlFilename = oldFilename }()

	oldNameByID, oldIDByName := getPlayerNameByID, getPlayerIDByName
	RegisterHousePlayerLookup(
		func(id int64) string {
			if id == 2 {
				return "Guest"
			}
			return ""
		},
		func(name string) int64 {
			if strings.EqualFold(name, "Guest") {
				return 2
			}
			return -1
		},
	)
	defer func() { RegisterHousePlayerLookup(oldNameByID, oldIDByName) }()

	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "House", Zone: 1, Flags: []string{"house"}}}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	defer w.StopAITicker()
	owner := NewPlayer(1, "Owner", 1001)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatalf("AddPlayer(owner): %v", err)
	}
	var output strings.Builder
	w.MessageSink = func(_ string, message []byte) { output.Write(message) }

	// A ROOM_HOUSE flag without a matching control record is a distinct C
	// early return.
	w.DoHouse(owner, "guest")
	if got := output.String(); got != "Um.. this house seems to be screwed up.\r\n" {
		t.Fatalf("malformed house output = %q", got)
	}

	output.Reset()
	w.HouseControl = []HouseControl{{VNum: 1001, Owner: 1, NumOfGuests: 2, Guests: [50]int64{2, 3}}}
	w.DoHouse(owner, "guest")
	if got := output.String(); got != "Guests of your house:\r\nGuest\r\n" {
		t.Fatalf("guest cleanup list = %q", got)
	}
	if got := w.HouseControl[0].NumOfGuests; got != 1 {
		t.Fatalf("guest count after cleanup = %d, want 1", got)
	}

	output.Reset()
	w.HouseControl = []HouseControl{{VNum: 1001, Owner: 1, NumOfGuests: MaxGuests}}
	w.DoHouse(owner, "guest Guest")
	if got := output.String(); got != "You've already reached the maximum number of guests in your house!\r\n" {
		t.Fatalf("full-house output = %q", got)
	}
}

func TestAddPlayerAssignsUniqueRuntimeIDsForHouseLookups(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	defer w.StopAITicker()

	first := NewPlayer(0, "First", 1001)
	second := NewPlayer(0, "Second", 1001)
	if err := w.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first): %v", err)
	}
	if err := w.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second): %v", err)
	}
	if first.GetID() != 0 || second.GetID() == first.GetID() {
		t.Fatalf("runtime IDs = %d and %d, want distinct IDs with first ID 0", first.GetID(), second.GetID())
	}
}
