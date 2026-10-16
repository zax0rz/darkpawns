package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestStatRoomIncludesExtraDescriptionKeywordsAtCColorLevel(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{
		VNum:        1001,
		Name:        "Map Room",
		Description: "A map room.",
		ExtraDescs: []parser.ExtraDesc{{
			Keywords: "map charts",
		}},
	}}}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })
	manager := newTestManager(t, world, nil)
	session := makeTestSession(t, manager, "Statgod", 1001, true)
	if err := world.AddPlayer(session.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	for _, test := range []struct {
		name  string
		color bool
		want  string
	}{
		{name: "color off", want: "Extra descs: map charts\r\n"},
		{name: "normal color", color: true, want: "Extra descs:\x1b[36m map charts\x1b[0m\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.color {
				session.player.SetPlrFlag(game.PrfColor1, true)
				session.player.SetPlrFlag(game.PrfColor2, true)
			}
			session.sendStatRoom()
			var output strings.Builder
			for len(session.send) > 0 {
				output.WriteString(readMsgText(t, session))
			}
			got := output.String()
			if !strings.Contains(got, test.want) {
				t.Fatalf("stat room output lacks %q: %q", test.want, got)
			}
		})
	}
}

func TestWizardObjectExtraDescriptionUsesCColorLevel(t *testing.T) {
	for _, test := range []struct {
		name  string
		flags uint64
		want  string
	}{
		{name: "color off", want: "Extra descs: oak chest\r\n"},
		{name: "color 1 only", flags: 1 << uint(game.PrfColor1), want: "Extra descs: oak chest\r\n"},
		// _clrlevel: COLOR_1 counts 1, COLOR_2 counts 2; C_NRM is 2 (src/screen.h:45-49).
		{name: "normal (color 2 only)", flags: 1 << uint(game.PrfColor2), want: "Extra descs:\x1b[36m oak chest\x1b[0m\r\n"},
		{name: "complete (both)", flags: 1<<uint(game.PrfColor1) | 1<<uint(game.PrfColor2), want: "Extra descs:\x1b[36m oak chest\x1b[0m\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := wizardExtraDescriptionLine([]string{"oak chest"}, wizardHasNormalColor(test.flags))
			if got != test.want {
				t.Fatalf("wizardExtraDescriptionLine() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestWizardObjectContentsMatchesCStatFormat(t *testing.T) {
	container := game.NewObjectInstance(&parser.Obj{
		TypeFlag: game.ITEM_CONTAINER,
		Values:   [4]int{20, 15, 12129, 0},
		Weight:   300,
	}, 0)
	pelt := game.NewObjectInstance(&parser.Obj{ShortDesc: "a seal pelt", Weight: 3}, 0)
	bars := game.NewObjectInstance(&parser.Obj{ShortDesc: "several gold bars", Weight: 15}, 0)
	if !container.AddToContainer(bars) || !container.AddToContainer(pelt) {
		t.Fatal("container rejected reset contents")
	}

	if got, want := wizardObjectValues(container), "Capacity: 20, Lock Type: CLOSEABLE PICKPROOF CLOSED LOCKED , Key Num: 12129, Corpse: NO"; got != want {
		t.Errorf("wizardObjectValues() = %q, want %q", got, want)
	}
	if got, want := container.GetTotalWeight(), 318; got != want {
		t.Errorf("container total weight = %d, want %d", got, want)
	}
	if got, want := wizardObjectContentsOutput(container.GetContents(), false), "\r\nContents: a seal pelt, several gold bars\r\n"; got != want {
		t.Errorf("uncolored contents = %q, want %q", got, want)
	}
	if got, want := wizardObjectContentsOutput(container.GetContents(), true), "\r\nContents:\x1b[32m a seal pelt, several gold bars\r\n\x1b[0m"; got != want {
		t.Errorf("colored contents = %q, want %q", got, want)
	}
}

func TestWizardObjectContentsUsesC62ByteWrap(t *testing.T) {
	children := []*game.ObjectInstance{
		game.NewObjectInstance(&parser.Obj{ShortDesc: "a"}, 0),
		game.NewObjectInstance(&parser.Obj{ShortDesc: strings.Repeat("x", 44)}, 0),
		game.NewObjectInstance(&parser.Obj{ShortDesc: "tail"}, 0),
	}
	want := "\r\nContents:\x1b[32m a, " + strings.Repeat("x", 44) + ",\r\n tail\r\n\x1b[0m"
	if got := wizardObjectContentsOutput(children, true); got != want {
		t.Fatalf("wizardObjectContentsOutput() = %q, want %q", got, want)
	}
}

func TestWizardObjectAffectOutputMatchesCOrderAndBytes(t *testing.T) {
	affects := []parser.ObjAffect{
		{Location: game.ApplyManaRegen, Modifier: 10},
		{Location: game.ApplySpell, Modifier: 10},
		{Location: game.ApplySpell, Modifier: 18},
	}
	want := "Affections: +10 to MANA_REGEN, casts permanent INFRA, casts permanent SNEAK\r\n"
	if got := wizardObjectAffectOutput(affects); got != want {
		t.Fatalf("wizardObjectAffectOutput() = %q, want %q", got, want)
	}
	if got, want := wizardObjectAffectOutput(nil), "Affections: None\r\n"; got != want {
		t.Fatalf("wizardObjectAffectOutput(nil) = %q, want %q", got, want)
	}
}

func TestWizardObjectValuesMatchesCItemTypeBranches(t *testing.T) {
	for _, test := range []struct {
		name     string
		itemType int
		values   [4]int
		want     string
	}{
		{name: "light finite", itemType: game.ITEM_LIGHT, values: [4]int{0, 0, 8, 0}, want: "Hours left: [8]"},
		{name: "light infinite", itemType: game.ITEM_LIGHT, values: [4]int{0, 0, -1, 0}, want: "Hours left: Infinite"},
		{name: "scroll", itemType: game.ITEM_SCROLL, values: [4]int{16, 2, 33, -1}, want: "Spells: (Level 16) shift reality, poison, UNUSED"},
		{name: "wand", itemType: game.ITEM_WAND, values: [4]int{16, 1, 2, 17}, want: "Spell: curse at level 16, 2 (of 1) charges remaining"},
		{name: "weapon", itemType: game.ITEM_WEAPON, values: [4]int{0, 2, 5, 3}, want: "Todam: 2d5, Message type: 3 Apr: 6.0"},
		{name: "armor", itemType: game.ITEM_ARMOR, values: [4]int{20, 0, 0, 0}, want: "AC-apply: [20]"},
		{name: "trap", itemType: game.ITEM_TRAP, values: [4]int{7, 12, 0, 0}, want: "Spell: 7, - Hitpoints: 12"},
		{name: "drink container", itemType: game.ITEM_DRINKCON, values: [4]int{20, 5, 3, 1}, want: "Capacity: 20, Contains: 5, Poisoned: YES, Liquid: ale"},
		{name: "note", itemType: game.ITEM_NOTE, values: [4]int{4, 0, 0, 0}, want: "Tongue: 4"},
		{name: "key", itemType: game.ITEM_KEY, want: ""},
		{name: "food", itemType: game.ITEM_FOOD, values: [4]int{5, 0, 0, 1}, want: "Makes full: 5, Poisoned: YES"},
		{name: "money", itemType: game.ITEM_MONEY, values: [4]int{100, 0, 0, 0}, want: "Coins: 100"},
	} {
		t.Run(test.name, func(t *testing.T) {
			object := game.NewObjectInstance(&parser.Obj{TypeFlag: test.itemType, Values: test.values}, 0)
			if got := wizardObjectValues(object); got != test.want {
				t.Fatalf("wizardObjectValues() = %q, want %q", got, test.want)
			}
		})
	}
}

// An APPLY_SPELL modifier outside affected_bits[] is C undefined behaviour;
// the port must not panic on a God's stat (R1a).
func TestWizardObjectAffectedBitNameOutOfRange(t *testing.T) {
	for _, index := range []int{-1, 39, 1000} {
		if got := wizardObjectAffectedBitName(index); got != "UNDEFINED" {
			t.Fatalf("wizardObjectAffectedBitName(%d) = %q, want UNDEFINED", index, got)
		}
	}
	if got := wizardObjectAffectedBitName(0); got != "BLIND" {
		t.Fatalf("index 0 = %q, want BLIND", got)
	}
}

func TestWizardRoomListMatchesC62ByteWrapAndColorBytes(t *testing.T) {
	long := strings.Repeat("x", 30)
	medium := strings.Repeat("y", 20)
	last := strings.Repeat("z", 61)
	want := "Chars present: " + long + ", " + medium + ",\r\n " + last + "\r\n"
	if got := wizardWrappedListOutput("Chars present:", []string{long, medium, last}, "\x1b[33m", false); got != want {
		t.Fatalf("uncolored room list = %q, want %q", got, want)
	}

	forty := strings.Repeat("x", 40)
	coloredWant := "Chars present:\x1b[33m " + forty + ", y,\r\n z, w\r\n\x1b[0m"
	if got := wizardWrappedListOutput("Chars present:", []string{forty, "y", "z", "w"}, "\x1b[33m", true); got != coloredWant {
		t.Fatalf("colored room list = %q, want %q", got, coloredWant)
	}
}
