package game

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestExtraDescPresenceNeverChangesSaveBytes is the no-save-format-change proof
// (Zach, DP-1417): the presence flags are json:"-", so they can never appear in
// a serialised extra description, with or without the flags set; a pre-existing
// payload still decodes with its text intact. C's objsave.c stores no extra
// descriptions, so presence never has to survive a player save.
//
// Byte identity is asserted at the ExtraDesc encoding, not through SaveItemData.State:
// that field is a map[string]interface{}, so a payload round-trips through a
// generic map and its keys re-sort — true before this change as well.
func TestExtraDescPresenceNeverChangesSaveBytes(t *testing.T) {
	const want = `[{"Keywords":"altar","Description":"a stone altar"},{"Keywords":"","Description":"y"}]`
	present := []parser.ExtraDesc{
		{Keywords: "altar", Description: "a stone altar"},
		{Keywords: "", Description: "y"},
	}
	// The same text with C's NULL pointers set must encode identically.
	nullable := []parser.ExtraDesc{
		{Keywords: "altar", Description: "a stone altar", NullKeyword: true, NullDescription: true},
		{Keywords: "", Description: "y", NullKeyword: true, NullDescription: true},
	}
	for name, descs := range map[string][]parser.ExtraDesc{"present": present, "null": nullable} {
		raw, err := json.Marshal(descs)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if string(raw) != want {
			t.Fatalf("%s bytes = %s, want %s", name, raw, want)
		}
	}

	// A pre-existing save payload still decodes with its text intact, and no
	// presence key appears when it is encoded again.
	const preExisting = `{"vnum":3001,"count":1,"locate":0,"state":{"extra_descs":[{"Keywords":"altar","Description":"a stone altar"}]}}`
	var item SaveItemData
	if err := json.Unmarshal([]byte(preExisting), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "Null") {
		t.Fatalf("presence flags leaked into the save bytes: %s", raw)
	}
	for _, value := range []string{"altar", "a stone altar"} {
		if !strings.Contains(string(raw), value) {
			t.Fatalf("pre-existing extra description text lost: %s", raw)
		}
	}
}
