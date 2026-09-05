package game

import (
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newTattoo2TestWorld(t *testing.T, withPeer bool) (*World, *Player, *MobInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 18252, Name: "A Dark Building", Zone: 182}},
		Mobs: []parser.Mob{{
			VNum:      18213,
			Keywords:  "confucius tattooist",
			ShortDesc: "Confucius",
			Sex:       1,
			Level:     29,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "TattooTwoActor", 18252)
	actor.SetLevel(levelCanShout)
	actor.Stats.Int = 10
	actor.Stats.Wis = 10
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if withPeer {
		peer := NewPlayer(2, "TattooTwoPeer", 18252)
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
	}
	mob, err := w.SpawnMob(18213, 18252)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, mob, messages
}

func TestSpecTattoo2ListAndEntryGates(t *testing.T) {
	w, actor, tattooist, messages := newTattoo2TestWorld(t, false)

	if !specTattoo2(w, actor, tattooist, "list", "") {
		t.Fatal("list should be consumed")
	}
	wantList := "To buy a tattoo: BUY <number of tattoo>.\r\n" +
		"Available tattoos are:\r\n" +
		"[0] - (14000) tattoo of a leaping tiger : the nimbleness and stamina of the tiger\r\n" +
		"[1] - (17000) tattoo of a heart : live longer through trust in your heart\r\n" +
		"[2] - (17000) tattoo of a star : gain the magic of the stars\r\n" +
		"[3] - (19000) tattoo of the symbol of the Jyhad : the power of fighting a holy war\r\n"
	if messages[actor.Name] != wantList {
		t.Fatalf("list output = %q, want %q", messages[actor.Name], wantList)
	}
	clearTattoo2Messages(messages)

	for _, test := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "no argument", arg: "", want: "Buy what number?\r\n"},
		{name: "non numeric", arg: "x", want: "Buy by number!\r\n"},
		{name: "out of range", arg: "4", want: "Buy by number!\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !specTattoo2(w, actor, tattooist, "buy", test.arg) {
				t.Fatal("buy gate should be consumed")
			}
			if got := messages[actor.Name]; got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			clearTattoo2Messages(messages)
		})
	}

	actor.Tattoo = TattooDragon
	if !specTattoo2(w, actor, tattooist, "buy", "0") {
		t.Fatal("already-tattooed gate should be consumed")
	}
	if got, want := messages[actor.Name], "Confucius tells you, 'Your magickal center is already tattooed. Your tattoo... is enough magick for such as yourself.'\r\n"; got != want {
		t.Fatalf("owned gate output = %q, want %q", got, want)
	}
	if actor.GetGold() != 0 {
		t.Fatalf("owned gate changed gold to %d", actor.GetGold())
	}

	if specTattoo2(w, actor, tattooist, "look", "") {
		t.Fatal("unrelated command should fall through")
	}
	if specTattoo2(w, nil, tattooist, "list", "") {
		t.Fatal("nil actor should fall through")
	}
}

func clearTattoo2Messages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecTattoo2PriceAndSuccessAudience(t *testing.T) {
	w, actor, tattooist, messages := newTattoo2TestWorld(t, true)
	baseStats := actor.Stats
	baseMove := actor.GetMaxMove()

	actor.SetGold(0)
	if !specTattoo2(w, actor, tattooist, "buy", "0") {
		t.Fatal("price gate should be consumed")
	}
	if got, want := messages[actor.Name], "Confucius tells you, 'Without more coins, I can give no wisdom.'\r\n"; got != want {
		t.Fatalf("price gate output = %q, want %q", got, want)
	}
	clearTattoo2Messages(messages)

	actor.SetGold(30000)
	if !specTattoo2(w, actor, tattooist, "buy", "0") {
		t.Fatal("successful purchase should be consumed")
	}
	wantActor := "Confucius starts to work on your tattoo...\r\n" +
		"The pain is incredible; it seems to eat into your soul.\r\n" +
		"A scream is ripped from your lips...\r\n" +
		"You shout, 'Arrrrrrrrrgggggggghhhh!'\r\n" +
		"You black out.\r\n"
	if messages[actor.Name] != wantActor {
		t.Fatalf("actor success output = %q, want %q", messages[actor.Name], wantActor)
	}
	wantPeer := "Confucius starts to work on TattooTwoActor's tattoo...\r\n" +
		"A ghastly scream is ripped from TattooTwoActor's lips just before he blacks out.\r\n" +
		"TattooTwoActor shouts, 'Arrrrrrrrrgggggggghhhh!'\r\n"
	if messages["TattooTwoPeer"] != wantPeer {
		t.Fatalf("peer success output = %q, want %q", messages["TattooTwoPeer"], wantPeer)
	}
	if actor.GetGold() != 16000 || actor.Tattoo != TattooTiger {
		t.Fatalf("success state = gold %d tattoo %d, want 16000/%d", actor.GetGold(), actor.Tattoo, TattooTiger)
	}
	if actor.GetPosition() != combat.PosStanding {
		t.Fatalf("success position = %d, want standing", actor.GetPosition())
	}
	if actor.Stats.Str != baseStats.Str || actor.Stats.Dex != baseStats.Dex+1 || actor.Stats.Int != baseStats.Int || actor.Stats.Wis != baseStats.Wis || actor.GetMaxMove() != baseMove+10 {
		t.Fatalf("tiger effects = stats %+v max move %d, want base dex+1 and max move+10", actor.Stats, actor.GetMaxMove())
	}
}

func TestSpecTattoo2AppliesEveryOfferedTattoo(t *testing.T) {
	for _, test := range []struct {
		name       string
		offerIndex int
		strDelta   int
		dexDelta   int
		intDelta   int
		maxHealth  int
		maxMana    int
		maxMove    int
		wisDelta   int
		damroll    int
	}{
		{name: "tiger", offerIndex: 0, dexDelta: 1, maxMove: 10},
		{name: "heart", offerIndex: 1, maxHealth: 20},
		{name: "star", offerIndex: 2, maxMana: 20},
		{name: "jyhad", offerIndex: 3, damroll: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, actor, tattooist, _ := newTattoo2TestWorld(t, false)
			baseStats := actor.Stats
			baseHealth := actor.GetMaxHP()
			baseMana := actor.GetMaxMana()
			baseMove := actor.GetMaxMove()
			actor.SetGold(tattoo2Offers[test.offerIndex].price)
			if !specTattoo2(w, actor, tattooist, "buy", strconv.Itoa(test.offerIndex)) {
				t.Fatal("purchase should be consumed")
			}
			offer := tattoo2Offers[test.offerIndex]
			if actor.Tattoo != offer.number {
				t.Errorf("tattoo = %d, want %d", actor.Tattoo, offer.number)
			}
			if actor.Stats.Str != baseStats.Str+test.strDelta || actor.Stats.Dex != baseStats.Dex+test.dexDelta || actor.Stats.Int != baseStats.Int+test.intDelta || actor.Stats.Wis != baseStats.Wis+test.wisDelta {
				t.Errorf("stat effects = %+v, base %+v, want deltas str=%d dex=%d int=%d wis=%d", actor.Stats, baseStats, test.strDelta, test.dexDelta, test.intDelta, test.wisDelta)
			}
			if actor.GetMaxHP() != baseHealth+test.maxHealth || actor.GetMaxMana() != baseMana+test.maxMana || actor.GetMaxMove() != baseMove+test.maxMove || actor.Damroll != test.damroll {
				t.Errorf("resource effects = hp %d mana %d move %d damroll %d, want deltas hp=%d mana=%d move=%d damroll=%d", actor.GetMaxHP(), actor.GetMaxMana(), actor.GetMaxMove(), actor.Damroll, test.maxHealth, test.maxMana, test.maxMove, test.damroll)
			}
		})
	}
}

func TestSpecTattoo2AutonomousEntry(t *testing.T) {
	w, _, tattooist, _ := newTattoo2TestWorld(t, false)
	if specTattoo2(w, nil, tattooist, "", "") {
		t.Fatal("autonomous no-command call should fall through")
	}
}

func TestTattoo2OfferDataIsStable(t *testing.T) {
	if got := strings.Join([]string{
		tattoo2Offers[0].name,
		tattoo2Offers[1].name,
		tattoo2Offers[2].name,
		tattoo2Offers[3].name,
	}, "|"); got != "of a leaping tiger|of a heart|of a star|of the symbol of the Jyhad" {
		t.Fatalf("offer names = %q", got)
	}
}
