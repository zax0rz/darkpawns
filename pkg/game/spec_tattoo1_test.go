package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newTattoo1TestWorld(t *testing.T, withPeer bool) (*World, *Player, *MobInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8052, Name: "Berzerker's Tattoos", Zone: 80}},
		Mobs: []parser.Mob{{
			VNum:      8086,
			Keywords:  "berzerker tattooist",
			ShortDesc: "Berzerker the tattoo guy",
			Sex:       1,
			Level:     33,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "TattooActor", 8052)
	actor.SetLevel(levelCanShout)
	actor.Stats.Int = 10
	actor.Stats.Wis = 10
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if withPeer {
		peer := NewPlayer(2, "TattooPeer", 8052)
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
	}
	mob, err := w.SpawnMob(8086, 8052)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	clearTattoo1Messages(messages)
	return w, actor, mob, messages
}

func clearTattoo1Messages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecTattoo1ListAndEntryGates(t *testing.T) {
	w, actor, tattooist, messages := newTattoo1TestWorld(t, false)

	if !specTattoo1(w, actor, tattooist, "list", "") {
		t.Fatal("list should be consumed")
	}
	wantList := "To buy a tattoo: BUY <number of tattoo>.\r\n" +
		"Available tattoos are:\r\n" +
		"[0] - (30666) tattoo of a green dragon : grow stronger and hit harder\r\n" +
		"[1] - (3000) tattoo in a tribal design : increase your dexterity\r\n" +
		"[2] - (10000) tattoo of a screaming eagle : move like the wind\r\n" +
		"[3] - (3000) tattoo of a fox : gain the intelligence of the fox\r\n" +
		"[4] - (3000) tattoo of an owl : gain the wisdom of the owl\r\n"
	if messages[actor.Name] != wantList {
		t.Fatalf("list output = %q, want %q", messages[actor.Name], wantList)
	}
	clearTattoo1Messages(messages)

	for _, test := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "no argument", arg: "", want: "Buy what number?\r\n"},
		{name: "non numeric", arg: "x", want: "Buy by number!\r\n"},
		{name: "out of range", arg: "5", want: "Buy by number!\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !specTattoo1(w, actor, tattooist, "buy", test.arg) {
				t.Fatal("buy gate should be consumed")
			}
			if got := messages[actor.Name]; got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			clearTattoo1Messages(messages)
		})
	}

	actor.Tattoo = TattooDragon
	if !specTattoo1(w, actor, tattooist, "buy", "0") {
		t.Fatal("already-tattooed gate should be consumed")
	}
	if got, want := messages[actor.Name], "Berzerker the tattoo guy tells you, 'Your magickal center is already tattooed. Get a new arm or get rid of that tattoo then come back.'\r\n"; got != want {
		t.Fatalf("owned gate output = %q, want %q", got, want)
	}
	if actor.GetGold() != 0 {
		t.Fatalf("owned gate changed gold to %d", actor.GetGold())
	}

	if specTattoo1(w, actor, tattooist, "look", "") {
		t.Fatal("unrelated command should fall through")
	}
	if specTattoo1(w, nil, tattooist, "list", "") {
		t.Fatal("nil actor should fall through")
	}
}

func TestSpecTattoo1PriceAndSuccessAudience(t *testing.T) {
	w, actor, tattooist, messages := newTattoo1TestWorld(t, true)
	baseStats := actor.Stats

	actor.SetGold(0)
	if !specTattoo1(w, actor, tattooist, "buy", "0") {
		t.Fatal("price gate should be consumed")
	}
	if got, want := messages[actor.Name], "Berzerker the tattoo guy tells you, 'You look a little short on the price there, kid.'\r\n"; got != want {
		t.Fatalf("price gate output = %q, want %q", got, want)
	}
	clearTattoo1Messages(messages)

	actor.SetGold(40000)
	if !specTattoo1(w, actor, tattooist, "buy", "0") {
		t.Fatal("successful purchase should be consumed")
	}
	wantActor := "Berzerker the tattoo guy starts to work on your tattoo...\r\n" +
		"The pain is incredible; it seems to eat into your soul.\r\n" +
		"A scream is ripped from your lips...\r\n" +
		"You shout, 'Arrrrrrrrrgggggggghhhh!'\r\n" +
		"You black out.\r\n"
	if messages[actor.Name] != wantActor {
		t.Fatalf("actor success output = %q, want %q", messages[actor.Name], wantActor)
	}
	wantPeer := "Berzerker the tattoo guy starts to work on TattooActor's tattoo...\r\n" +
		"A ghastly scream is ripped from TattooActor's lips just before he blacks out.\r\n" +
		"TattooActor shouts, 'Arrrrrrrrrgggggggghhhh!'\r\n"
	if messages["TattooPeer"] != wantPeer {
		t.Fatalf("peer success output = %q, want %q", messages["TattooPeer"], wantPeer)
	}
	if actor.GetGold() != 9334 || actor.Tattoo != TattooDragon {
		t.Fatalf("success state = gold %d tattoo %d, want 9334/%d", actor.GetGold(), actor.Tattoo, TattooDragon)
	}
	if actor.GetPosition() != combat.PosStanding {
		t.Fatalf("success position = %d, want standing", actor.GetPosition())
	}
	if actor.Stats.Str != baseStats.Str+2 || actor.Damroll != 2 {
		t.Fatalf("dragon effects = str %d damroll %d, want base+2/2", actor.Stats.Str, actor.Damroll)
	}
}

func TestSpecTattoo1AppliesEveryOfferedTattoo(t *testing.T) {
	for _, test := range []struct {
		name       string
		offerIndex int
		strDelta   int
		dexDelta   int
		intDelta   int
		maxMove    int
		wisDelta   int
	}{
		{name: "dragon", offerIndex: 0, strDelta: 2},
		{name: "tribal", offerIndex: 1, dexDelta: 1},
		{name: "eagle", offerIndex: 2, maxMove: 20},
		{name: "fox", offerIndex: 3, intDelta: 1},
		{name: "owl", offerIndex: 4, wisDelta: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, actor, tattooist, _ := newTattoo1TestWorld(t, false)
			base := actor.Stats
			baseMove := actor.GetMaxMove()
			actor.SetGold(tattoo1Offers[test.offerIndex].price)
			if !specTattoo1(w, actor, tattooist, "buy", string(rune('0'+test.offerIndex))) {
				t.Fatal("purchase should be consumed")
			}
			offer := tattoo1Offers[test.offerIndex]
			if actor.Tattoo != offer.number {
				t.Errorf("tattoo = %d, want %d", actor.Tattoo, offer.number)
			}
			if actor.Stats.Str != base.Str+test.strDelta || actor.Stats.Dex != base.Dex+test.dexDelta || actor.Stats.Int != base.Int+test.intDelta || actor.Stats.Wis != base.Wis+test.wisDelta {
				t.Errorf("stat effects = %+v, base %+v, want deltas str=%d dex=%d int=%d wis=%d", actor.Stats, base, test.strDelta, test.dexDelta, test.intDelta, test.wisDelta)
			}
			if actor.GetMaxMove() != baseMove+test.maxMove {
				t.Errorf("max move = %d, want %d", actor.GetMaxMove(), baseMove+test.maxMove)
			}
		})
	}
}

func TestSpecTattoo1AutonomousEntry(t *testing.T) {
	w, _, tattooist, _ := newTattoo1TestWorld(t, false)
	if specTattoo1(w, nil, tattooist, "", "") {
		t.Fatal("autonomous no-command call should fall through")
	}
}

func TestTattoo1OfferDataIsStable(t *testing.T) {
	if got := strings.Join([]string{
		tattoo1Offers[0].name,
		tattoo1Offers[1].name,
		tattoo1Offers[2].name,
		tattoo1Offers[3].name,
		tattoo1Offers[4].name,
	}, "|"); got != "of a green dragon|in a tribal design|of a screaming eagle|of a fox|of an owl" {
		t.Fatalf("offer names = %q", got)
	}
}
