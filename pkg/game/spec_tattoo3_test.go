package game

import (
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newTattoo3TestWorld(t *testing.T, withPeer bool) (*World, *Player, *MobInstance, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 21281, Name: "A Tattoo Parlor", Zone: 212}},
		Mobs: []parser.Mob{{
			VNum:      21244,
			Keywords:  "tattooist artist polywig",
			ShortDesc: "Polywig",
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
	actor := NewPlayer(1, "TattooThreeActor", 21281)
	actor.SetLevel(levelCanShout)
	actor.Stats.Int = 10
	actor.Stats.Wis = 10
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if withPeer {
		peer := NewPlayer(2, "TattooThreePeer", 21281)
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
	}
	mob, err := w.SpawnMob(21244, 21281)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, mob, messages
}

func TestSpecTattoo3ListAndEntryGates(t *testing.T) {
	w, actor, tattooist, messages := newTattoo3TestWorld(t, false)

	if !specTattoo3(w, actor, tattooist, "list", "") {
		t.Fatal("list should be consumed")
	}
	wantList := "To buy a tattoo: BUY <number of tattoo>.\r\n" +
		"Available tattoos are:\r\n" +
		"[0] - (18000) tattoo of an open eye : see that which is normally unseen\r\n" +
		"[1] - (20000) tattoo of crossed swords : miss less and hit harder\r\n" +
		"[2] - (11000) tattoo of a ship : gain the ability of movement over water\r\n" +
		"[3] - (15000) tattoo of the word 'MOM' : the wisdom of your elders\r\n"
	if messages[actor.Name] != wantList {
		t.Fatalf("list output = %q, want %q", messages[actor.Name], wantList)
	}
	clearTattoo3Messages(messages)

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
			if !specTattoo3(w, actor, tattooist, "buy", test.arg) {
				t.Fatal("buy gate should be consumed")
			}
			if got := messages[actor.Name]; got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
			clearTattoo3Messages(messages)
		})
	}

	actor.Tattoo = TattooDragon
	if !specTattoo3(w, actor, tattooist, "buy", "0") {
		t.Fatal("already-tattooed gate should be consumed")
	}
	if got, want := messages[actor.Name], "Polywig tells you, 'Your mathickal thenter is awready tattooed. Your tattoo... ith enough mathick for such as yoursewf.'\r\n"; got != want {
		t.Fatalf("owned gate output = %q, want %q", got, want)
	}
	if actor.GetGold() != 0 {
		t.Fatalf("owned gate changed gold to %d", actor.GetGold())
	}

	if specTattoo3(w, actor, tattooist, "look", "") {
		t.Fatal("unrelated command should fall through")
	}
	if specTattoo3(w, nil, tattooist, "list", "") {
		t.Fatal("nil actor should fall through")
	}
}

func clearTattoo3Messages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecTattoo3PriceAndSuccessAudience(t *testing.T) {
	w, actor, tattooist, messages := newTattoo3TestWorld(t, true)
	baseStats := actor.Stats
	baseMove := actor.GetMaxMove()

	actor.SetGold(0)
	if !specTattoo3(w, actor, tattooist, "buy", "0") {
		t.Fatal("price gate should be consumed")
	}
	if got, want := messages[actor.Name], "Polywig tells you, 'You don't have enough cash, hot stuff.'\r\n"; got != want {
		t.Fatalf("price gate output = %q, want %q", got, want)
	}
	clearTattoo3Messages(messages)

	actor.SetGold(30000)
	if !specTattoo3(w, actor, tattooist, "buy", "0") {
		t.Fatal("successful purchase should be consumed")
	}
	wantActor := "Polywig starts to work on your tattoo...\r\n" +
		"The pain is incredible; it seems to eat into your soul.\r\n" +
		"A scream is ripped from your lips...\r\n" +
		"You shout, 'Arrrrrrrrrgggggggghhhh!'\r\n" +
		"You black out.\r\n"
	if messages[actor.Name] != wantActor {
		t.Fatalf("actor success output = %q, want %q", messages[actor.Name], wantActor)
	}
	wantPeer := "Polywig starts to work on TattooThreeActor's tattoo...\r\n" +
		"A ghastly scream is ripped from TattooThreeActor's lips just before he blacks out.\r\n" +
		"TattooThreeActor shouts, 'Arrrrrrrrrgggggggghhhh!'\r\n"
	if messages["TattooThreePeer"] != wantPeer {
		t.Fatalf("peer success output = %q, want %q", messages["TattooThreePeer"], wantPeer)
	}
	if actor.GetGold() != 12000 || actor.Tattoo != TattooEye {
		t.Fatalf("success state = gold %d tattoo %d, want 12000/%d", actor.GetGold(), actor.Tattoo, TattooEye)
	}
	if actor.GetPosition() != combat.PosStanding {
		t.Fatalf("success position = %d, want standing", actor.GetPosition())
	}
	if actor.Stats != baseStats || actor.GetMaxMove() != baseMove {
		t.Fatalf("open-eye effects = stats %+v max move %d, want no direct tattoo bonus", actor.Stats, actor.GetMaxMove())
	}
}

func TestSpecTattoo3AppliesEveryOfferedTattoo(t *testing.T) {
	for _, test := range []struct {
		name       string
		offerIndex int
		hitroll    int
		damroll    int
		wisDelta   int
	}{
		{name: "open eye", offerIndex: 0},
		{name: "crossed swords", offerIndex: 1, hitroll: 1, damroll: 1},
		{name: "ship", offerIndex: 2},
		{name: "mom", offerIndex: 3, wisDelta: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, actor, tattooist, _ := newTattoo3TestWorld(t, false)
			baseStats := actor.Stats
			baseHitroll := actor.Hitroll
			baseDamroll := actor.Damroll
			actor.SetGold(tattoo3Offers[test.offerIndex].price)
			if !specTattoo3(w, actor, tattooist, "buy", strconv.Itoa(test.offerIndex)) {
				t.Fatal("purchase should be consumed")
			}
			offer := tattoo3Offers[test.offerIndex]
			if actor.Tattoo != offer.number {
				t.Errorf("tattoo = %d, want %d", actor.Tattoo, offer.number)
			}
			if actor.Stats.Wis != baseStats.Wis+test.wisDelta || actor.Stats.Str != baseStats.Str || actor.Stats.Dex != baseStats.Dex || actor.Stats.Int != baseStats.Int {
				t.Errorf("stat effects = %+v, base %+v, want wis delta %d", actor.Stats, baseStats, test.wisDelta)
			}
			if actor.Hitroll != baseHitroll+test.hitroll || actor.Damroll != baseDamroll+test.damroll {
				t.Errorf("combat effects = hitroll %d damroll %d, want deltas %d/%d", actor.Hitroll, actor.Damroll, test.hitroll, test.damroll)
			}
		})
	}
}

func TestSpecTattoo3AutonomousEntry(t *testing.T) {
	w, _, tattooist, _ := newTattoo3TestWorld(t, false)
	if specTattoo3(w, nil, tattooist, "", "") {
		t.Fatal("autonomous no-command call should fall through")
	}
}

func TestTattoo3OfferDataIsStable(t *testing.T) {
	if got := strings.Join([]string{
		tattoo3Offers[0].name,
		tattoo3Offers[1].name,
		tattoo3Offers[2].name,
		tattoo3Offers[3].name,
	}, "|"); got != "of an open eye|of crossed swords|of a ship|of the word 'MOM'" {
		t.Fatalf("offer names = %q", got)
	}
}
