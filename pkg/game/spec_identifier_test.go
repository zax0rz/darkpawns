package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newIdentifierTestWorld(t *testing.T, withPeer bool) (*World, *Player, *MobInstance, *ObjectInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8116, Name: "The Identifier's Room", Zone: 80}},
		Mobs: []parser.Mob{{
			VNum:      8087,
			Keywords:  "identifier Ferrenx",
			ShortDesc: "Ferrenx the identifier",
			Sex:       1,
			Level:     33,
			HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		}},
		Objs: []parser.Obj{{
			VNum:      8010,
			Keywords:  "loaf bread",
			ShortDesc: "a loaf of bread",
			TypeFlag:  ITEM_FOOD,
			Values:    [4]int{10, 0, 0, 0},
			Weight:    2,
			Cost:      3,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "IdentifierActor", 8116)
	actor.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if withPeer {
		peer := NewPlayer(2, "IdentifierWitness", 8116)
		peer.SetPosition(combat.PosStanding)
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
	}
	mob, err := w.SpawnMob(8087, 8116)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	obj, err := w.SpawnObject(8010, 8116)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, actor); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, mob, obj, messages
}

func TestSpecIdentifierEntryGatesAndValue(t *testing.T) {
	w, actor, mob, _, messages := newIdentifierTestWorld(t, false)

	if !specIdentifier(w, actor, mob, "list", "") {
		t.Fatal("list should be consumed")
	}
	if got, want := messages[actor.Name], "Ferrenx the identifier tells you, 'Just read the sign!'\r\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}
	messages[actor.Name] = ""

	for _, test := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "value without object", arg: "", want: "Ferrenx the identifier tells you, 'Value what?'\r\n"},
		{name: "value missing object", arg: "sword", want: "Ferrenx the identifier tells you, 'You don't seem to have that.'\r\n"},
		{name: "value bread", arg: "bread", want: "Ferrenx the identifier tells you, 'I'll identify that fully for about 1 coins.'\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !specIdentifier(w, actor, mob, "value", test.arg) {
				t.Fatal("value should be consumed")
			}
			if got := messages[actor.Name]; got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			messages[actor.Name] = ""
		})
	}

	if specIdentifier(w, actor, mob, "xyzzy", "") {
		t.Fatal("unrelated command should fall through")
	}
	if specIdentifier(w, nil, mob, "list", "") {
		t.Fatal("nil actor should fall through")
	}
}

func TestSpecIdentifierGiveGatesAndPrice(t *testing.T) {
	w, actor, mob, _, messages := newIdentifierTestWorld(t, false)

	for _, arg := range []string{"", "bread", "bread nobody"} {
		if specIdentifier(w, actor, mob, "give", arg) {
			t.Fatalf("give %q should fall through", arg)
		}
	}
	actor.SetGold(0)
	if !specIdentifier(w, actor, mob, "give", "bread Ferrenx") {
		t.Fatal("insufficient-gold give should be consumed")
	}
	if got, want := messages[actor.Name], "Ferrenx the identifier tells you, 'That's a fine item, but I'll need 1 coins from you to id it.. and you're a little short..'\r\n"+
		"Ferrenx the identifier tells you, 'Keep it until you get the gold.'\r\n"; got != want {
		t.Fatalf("price output = %q, want %q", got, want)
	}
	if actor.GetGold() != 0 {
		t.Fatalf("price gate changed gold to %d", actor.GetGold())
	}
}

func TestSpecIdentifierSuccessAudienceAndState(t *testing.T) {
	w, actor, mob, obj, messages := newIdentifierTestWorld(t, true)
	actor.SetGold(1)

	if !specIdentifier(w, actor, mob, "give", "bread Ferrenx") {
		t.Fatal("successful give should be consumed")
	}
	if actor.GetGold() != 0 {
		t.Fatalf("success gold = %d, want 0", actor.GetGold())
	}
	if found, ok := w.ResolveObjectInInventory(actor, "bread"); !ok || found != obj {
		t.Fatal("identified object should remain in the actor's inventory")
	}

	wantActor := "You give a loaf of bread to Ferrenx the identifier.\r\n" +
		"Ferrenx the identifier studies it carefully, comparing it to ancient texts,\r\n" +
		"weighing it on scales, and chanting a number of odd spells over its surface.\r\n" +
		"Finally looking up, Ferrenx the identifier gives you back a loaf of bread.\r\n" +
		"Ferrenx the identifier touches your forehead, and knowledge fills your mind.\r\n" +
		"\r\n" +
		"You feel informed:\r\n" +
		"Object 'a loaf of bread', Item type: FOOD\r\n" +
		"Item will give you following abilities:  NOBITS \r\n" +
		"Item is: NOBITS \r\n" +
		"Encumbrance: 2, Value: 3\r\n"
	if got := messages[actor.Name]; got != wantActor {
		t.Fatalf("actor output = %q, want %q", got, wantActor)
	}
	wantPeer := "IdentifierActor gives a loaf of bread to Ferrenx the identifier.\r\n" +
		"Ferrenx the identifier studies it carefully, comparing it to ancient texts,\r\n" +
		"weighing it on scales, and chanting a number of odd spells over its surface.\r\n" +
		"Finally looking up, Ferrenx the identifier gives back a loaf of bread to IdentifierActor.\r\n" +
		"Ferrenx the identifier touches IdentifierActor gently on the forehead.\r\n"
	if got := messages["IdentifierWitness"]; got != wantPeer {
		t.Fatalf("peer output = %q, want %q", got, wantPeer)
	}
}

func TestIdentifierValCost(t *testing.T) {
	cheap := NewObjectInstance(&parser.Obj{Cost: 3}, -1)
	if got := identifierValCost(cheap); got != 1 {
		t.Fatalf("cheap val_cost = %d, want 1", got)
	}
	expensive := NewObjectInstance(&parser.Obj{Cost: 5000, ExtraFlags: [4]int{1 << itemExtraMagic}}, -1)
	if got := identifierValCost(expensive); got != 950 {
		t.Fatalf("magic val_cost = %d, want 950", got)
	}
}
