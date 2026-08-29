package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newTattoo4TestWorld(t *testing.T, withPeer bool) (*World, *Player, *MobInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 2809, Name: "A Tattoo Shop", Zone: 27}},
		Mobs: []parser.Mob{{
			VNum:      2766,
			Keywords:  "artist tattoo man tattooist",
			ShortDesc: "a sleazy tattoo artist",
			Sex:       0,
			Level:     31,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "TattooFourActor", 2809)
	actor.SetLevel(levelCanShout)
	actor.Stats.Int = 10
	actor.Stats.Wis = 10
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if withPeer {
		peer := NewPlayer(2, "TattooFourPeer", 2809)
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
	}
	mob, err := w.SpawnMob(2766, 2809)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, mob, messages
}

func clearTattoo4Messages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecTattoo4ListEntryOwnedAndPriceGates(t *testing.T) {
	w, actor, tattooist, messages := newTattoo4TestWorld(t, false)

	if !specTattoo4(w, actor, tattooist, "list", "") {
		t.Fatal("list should be consumed")
	}
	wantList := "To buy a tattoo: BUY <number of tattoo>.\r\n" +
		"Available tattoos are:\r\n" +
		"[0] - (25000) tattoo of an ice worm : hit with the fierceness of the remorhaz\r\n" +
		"[1] - (30666) tattoo of a green dragon : grow stronger and hit harder\r\n" +
		"[2] - (18000) tattoo of a flaming skull : summon a flaming skull to aid against thy enemies.\r\n"
	if got := messages[actor.Name]; got != wantList {
		t.Fatalf("list output = %q, want %q", got, wantList)
	}
	clearTattoo4Messages(messages)

	for _, test := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "no argument", arg: "", want: "Buy what number?\r\n"},
		{name: "non numeric", arg: "x", want: "Buy by number!\r\n"},
		{name: "out of range", arg: "3", want: "Buy by number!\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !specTattoo4(w, actor, tattooist, "buy", test.arg) {
				t.Fatal("buy gate should be consumed")
			}
			if got := messages[actor.Name]; got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			clearTattoo4Messages(messages)
		})
	}

	actor.Tattoo = TattooDragon
	if !specTattoo4(w, actor, tattooist, "buy", "0") {
		t.Fatal("owned gate should be consumed")
	}
	if got, want := messages[actor.Name], "A sleazy tattoo artist tells you, 'Get outta here, punk, you already have one. '\r\n"; got != want {
		t.Fatalf("owned output = %q, want %q", got, want)
	}
	clearTattoo4Messages(messages)

	actor.Tattoo = TattooNone
	actor.SetGold(0)
	if !specTattoo4(w, actor, tattooist, "buy", "0") {
		t.Fatal("price gate should be consumed")
	}
	if got, want := messages[actor.Name], "A sleazy tattoo artist tells you, 'You don't have enough gold, get outta here!'\r\n"; got != want {
		t.Fatalf("price output = %q, want %q", got, want)
	}
	if actor.GetGold() != 0 || actor.Tattoo != TattooNone {
		t.Fatalf("price gate mutated state: gold=%d tattoo=%d", actor.GetGold(), actor.Tattoo)
	}

	if specTattoo4(w, actor, tattooist, "look", "") {
		t.Fatal("unrelated command should fall through")
	}
	if specTattoo4(w, nil, tattooist, "list", "") {
		t.Fatal("nil actor should fall through")
	}
}

func TestSpecTattoo4SuccessAudienceAndState(t *testing.T) {
	w, actor, tattooist, messages := newTattoo4TestWorld(t, true)
	baseStats := actor.Stats
	baseDamroll := actor.Damroll
	actor.SetGold(25000)

	if !specTattoo4(w, actor, tattooist, "buy", "0") {
		t.Fatal("successful purchase should be consumed")
	}
	wantActor := "A sleazy tattoo artist starts to work on your tattoo...\r\n" +
		"The pain is incredible; it seems to eat into your soul.\r\n" +
		"A scream is ripped from your lips...\r\n" +
		"You shout, 'Arrrrrrrrrgggggggghhhh!'\r\n" +
		"You black out.\r\n"
	if got := messages[actor.Name]; got != wantActor {
		t.Fatalf("actor output = %q, want %q", got, wantActor)
	}
	wantPeer := "A sleazy tattoo artist starts to work on TattooFourActor's tattoo...\r\n" +
		"A ghastly scream is ripped from TattooFourActor's lips just before he blacks out.\r\n" +
		"TattooFourActor shouts, 'Arrrrrrrrrgggggggghhhh!'\r\n"
	if got := messages["TattooFourPeer"]; got != wantPeer {
		t.Fatalf("peer output = %q, want %q", got, wantPeer)
	}
	if actor.GetGold() != 0 || actor.Tattoo != TattooWorm {
		t.Fatalf("success state = gold %d tattoo %d, want 0/%d", actor.GetGold(), actor.Tattoo, TattooWorm)
	}
	if actor.GetPosition() != combat.PosStanding {
		t.Fatalf("success position = %d, want standing", actor.GetPosition())
	}
	if actor.Stats != baseStats || actor.Damroll != baseDamroll+2 {
		t.Fatalf("worm effects = stats %+v damroll %d, want unchanged stats and damroll +2", actor.Stats, actor.Damroll)
	}
}

func TestSpecTattoo4AppliesEveryOfferedTattoo(t *testing.T) {
	for _, test := range []struct {
		name       string
		offerIndex int
		tattoo     int
		strDelta   int
		damroll    int
	}{
		{name: "worm", offerIndex: 0, tattoo: TattooWorm, damroll: 2},
		{name: "dragon", offerIndex: 1, tattoo: TattooDragon, strDelta: 2, damroll: 2},
		{name: "skull", offerIndex: 2, tattoo: TattooSkull},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, actor, tattooist, _ := newTattoo4TestWorld(t, false)
			baseStats := actor.Stats
			actor.SetGold(tattoo4Offers[test.offerIndex].price)
			if !specTattoo4(w, actor, tattooist, "buy", string(rune('0'+test.offerIndex))) {
				t.Fatal("purchase should be consumed")
			}
			if actor.Tattoo != test.tattoo || actor.Stats.Str != baseStats.Str+test.strDelta || actor.Damroll != test.damroll {
				t.Fatalf("offer state = tattoo %d stats %+v damroll %d", actor.Tattoo, actor.Stats, actor.Damroll)
			}
		})
	}
}

func TestTattoo4OfferDataIsStable(t *testing.T) {
	if got := strings.Join([]string{tattoo4Offers[0].name, tattoo4Offers[1].name, tattoo4Offers[2].name}, "|"); got != "of an ice worm|of a green dragon|of a flaming skull" {
		t.Fatalf("offer names = %q", got)
	}
}
