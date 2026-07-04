package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestIsnameWithAbbrevs(t *testing.T) {
	cases := []struct {
		str, namelist string
		want          bool
	}{
		{"guard", "guard city", true},   // exact token
		{"gu", "guard city", true},      // abbreviation of "guard"
		{"ci", "guard city", true},      // abbreviation of "city"
		{"GU", "guard city", true},      // case-insensitive
		{"guardian", "guard city", false}, // longer than token
		{"post", "postman mail", true},  // abbreviation of "postman"
		{"postman", "the postman is here", true}, // "postman" is a token here
		{"xyz", "the postman is here", false},   // not a token/abbrev of any token
		{"", "guard", false},            // empty str
		{"guard", "", false},            // empty namelist
		{"x", "guard city", false},      // no match
	}
	for _, tc := range cases {
		got := isnameWithAbbrevs(tc.str, tc.namelist)
		if got != tc.want {
			t.Errorf("isnameWithAbbrevs(%q, %q) = %v, want %v", tc.str, tc.namelist, got, tc.want)
		}
	}
}

// newResolverTestWorld builds a world with one player (the viewer) and several
// mobs/players for resolver tests. The keyword namelist is what get_char_room_vis
// matches against, so mobs carry real Keywords.
func newResolverTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 100, Name: "Test Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	viewer := NewPlayer(1, "Hero", 100)
	viewer.Level = 5 // below LVL_IMMORT so canSee's immort short-circuit doesn't apply
	viewer.RoomVNum = 100
	w.AddPlayer(viewer)

	// Two guards (for ordinal tests) + a postman + a second player.
	for _, kmob := range []struct {
		vnum                                  int
		keywords, short                       string
	}{
		{200, "guard city", "a cityguard is here"},
		{201, "guard royal", "a royal guard stands here"},
		{202, "postman mail", "the city postman stands here"},
	} {
		m := NewMobInstance(&parser.Mob{
			VNum: kmob.vnum, Keywords: kmob.keywords, ShortDesc: kmob.short,
		}, 100)
		m.SetAlive(true)
		w.nextMobID++
		m.ID = w.nextMobID
		w.activeMobs[m.ID] = m
	}

	other := NewPlayer(2, "Zax", 100)
	other.RoomVNum = 100
	w.AddPlayer(other)

	return w, viewer
}

func TestResolveCharInRoomSelfAndMe(t *testing.T) {
	w, viewer := newResolverTestWorld(t)
	for _, name := range []string{"self", "me", "SELF", "Me"} {
		tgt, ok := w.ResolveCharInRoom(viewer, name)
		if !ok {
			t.Fatalf("ResolveCharInRoom(%q) not found, want self", name)
		}
		if tgt.Player != viewer {
			t.Errorf("ResolveCharInRoom(%q) = %v, want the viewer", name, tgt.Player)
		}
	}
}

func TestResolveCharInRoomAbbrevAndKeywords(t *testing.T) {
	w, viewer := newResolverTestWorld(t)

	// "postman" must resolve via the keyword namelist (not the ShortDesc,
	// which is "the city postman stands here"). This is the consider-vs-kick repro.
	tgt, ok := w.ResolveCharInRoom(viewer, "postman")
	if !ok {
		t.Fatal("ResolveCharInRoom(\"postman\") not found; keyword matching failed")
	}
	if tgt.Mob == nil || tgt.Mob.GetVNum() != 202 {
		t.Errorf("ResolveCharInRoom(\"postman\") = %+v, want mob vnum 202", tgt)
	}

	// Abbreviation "post" should also match "postman".
	tgt, ok = w.ResolveCharInRoom(viewer, "post")
	if !ok || tgt.Mob == nil || tgt.Mob.GetVNum() != 202 {
		t.Errorf("ResolveCharInRoom(\"post\") = %+v ok=%v, want mob vnum 202", tgt, ok)
	}

	// Player "zax" resolves via abbreviation.
	tgt, ok = w.ResolveCharInRoom(viewer, "zax")
	if !ok || tgt.Player == nil || tgt.Player.Name != "Zax" {
		t.Errorf("ResolveCharInRoom(\"zax\") = %+v ok=%v, want player Zax", tgt, ok)
	}
}

func TestResolveCharInRoomOrdinal(t *testing.T) {
	w, viewer := newResolverTestWorld(t)

	// Two guards share the "guard" keyword. Candidates are sorted by VNum then
	// instance ID, so ordinals are stable and reproducible: cityguard (200)
	// sorts before royal guard (201). "guard" and "1.guard" agree.
	first, ok := w.ResolveCharInRoom(viewer, "1.guard")
	if !ok || first.Mob == nil {
		t.Fatalf("ResolveCharInRoom(\"1.guard\") not found: %+v ok=%v", first, ok)
	}
	if first.Mob.GetVNum() != 200 {
		t.Errorf("\"1.guard\" = vnum %d, want 200", first.Mob.GetVNum())
	}
	plain, ok := w.ResolveCharInRoom(viewer, "guard")
	if !ok || plain.Mob == nil || plain.Mob.GetVNum() != 200 {
		t.Errorf("\"guard\" = %+v ok=%v, want vnum 200", plain, ok)
	}

	second, ok := w.ResolveCharInRoom(viewer, "2.guard")
	if !ok || second.Mob == nil {
		t.Fatalf("ResolveCharInRoom(\"2.guard\") not found: %+v ok=%v", second, ok)
	}
	if second.Mob.GetVNum() != 201 {
		t.Errorf("\"2.guard\" = vnum %d, want 201", second.Mob.GetVNum())
	}

	// "3.guard" → no third guard.
	if _, ok := w.ResolveCharInRoom(viewer, "3.guard"); ok {
		t.Error("ResolveCharInRoom(\"3.guard\") found a match, want none")
	}
}

func TestResolveCharInRoomMissing(t *testing.T) {
	w, viewer := newResolverTestWorld(t)
	if _, ok := w.ResolveCharInRoom(viewer, "dragon"); ok {
		t.Error("ResolveCharInRoom(\"dragon\") found a match in an empty-of-dragons room")
	}
	// The literal ShortDesc "the city postman stands here" should NOT match,
	// because matching is against keywords, not the display string.
	if _, ok := w.ResolveCharInRoom(viewer, "the city postman stands here"); ok {
		t.Error("ResolveCharInRoom matched on the ShortDesc literal, want keyword-only matching")
	}
}

func TestResolveCharInRoomConsidersKickParity(t *testing.T) {
	// The core DP-907 acceptance: consider and kick resolve identically. Both
	// now go through ResolveCharInRoom, so resolving "postman" once proves the
	// parity that previously diverged (consider's EqualFold missed it).
	w, viewer := newResolverTestWorld(t)
	if _, ok := w.ResolveCharInRoom(viewer, "postman"); !ok {
		t.Fatal("postman must resolve for both consider and kick")
	}
}
