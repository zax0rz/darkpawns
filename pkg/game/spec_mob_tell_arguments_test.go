package game

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type mobDirectCommandCall struct {
	command  string
	argument string
}

// assertMobDirectCommandCalls records the raw command call at the command-body
// boundary, then lets the normal mobile command body run. Every table case
// names its C call site so a caller-specific argument regression identifies
// the source row it invalidates.
func assertMobDirectCommandCalls(t *testing.T, w *World, cSite string, invoke func() bool, want ...mobDirectCommandCall) {
	t.Helper()
	var got []mobDirectCommandCall
	w.CharacterCommandDirectExecFunc = func(_ combat.Combatant, command, argument string) CharacterCommandResult {
		got = append(got, mobDirectCommandCall{command: command, argument: argument})
		return CharacterCommandUnsupported
	}
	if !invoke() {
		t.Fatalf("%s caller returned false before its direct command", cSite)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s direct command calls = %#v, want %#v", cSite, got, want)
	}
}

func TestSpecStableboyTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, argument, want string
		setup                                func(*testing.T, *World, *Player)
	}{
		{name: "buy bad argument", cSite: "src/spec_procs2.c:329", command: "buy", argument: "horseman", want: "Tester Buy what, fine adventurer?"},
		{name: "list", cSite: "src/spec_procs2.c:336", command: "list", want: "Tester You can buy a horse for 300 gold coins."},
		{name: "buy cannot afford", cSite: "src/spec_procs2.c:345", command: "buy", argument: "horse", want: "Tester You can't afford a mount!", setup: func(_ *testing.T, _ *World, ch *Player) { ch.SetGold(100) }},
		{name: "buy succeeds", cSite: "src/spec_procs2.c:364", command: "buy", argument: "horse", want: "Tester That'll be 300 coins, treat'er well", setup: func(_ *testing.T, _ *World, ch *Player) { ch.SetGold(300) }},
		{name: "buy missing prototype", cSite: "src/spec_procs2.c:369", command: "buy", argument: "horse", want: "Tester Sorry we are all out of mounts at the moment, try again later.", setup: func(_ *testing.T, w *World, ch *Player) {
			ch.SetGold(300)
			w.mu.Lock()
			delete(w.mobs, horseVnum)
			w.mu.Unlock()
		}},
		{name: "stable no followers", cSite: "src/spec_procs2.c:384", command: "stable", want: "Tester How do you expect to stable a mount, you don't have a mount!"},
		{name: "stable no mountable follower", cSite: "src/spec_procs2.c:398", command: "stable", want: "Tester How do you expect to stable a mount, you don't have a mount!", setup: func(t *testing.T, w *World, ch *Player) {
			follower := newSpecProcTestMob(t, w, ch.GetRoomVNum(), 1)
			follower.SetFollowing(ch.Name)
		}},
		{name: "stable succeeds", cSite: "src/spec_procs2.c:409", command: "stable", want: "Tester I will take good care of 'em, for 5 coins a day.", setup: func(t *testing.T, w *World, ch *Player) {
			horse, err := w.SpawnMob(horseVnum, ch.GetRoomVNum())
			if err != nil {
				t.Fatalf("SpawnMob horse: %v", err)
			}
			horse.SetFollowing(ch.Name)
		}},
		{name: "collect has no stored mount", cSite: "src/spec_procs2.c:415", command: "collect", want: "Tester Hey now, you need to have stabled a mount to pick one up."},
		{name: "collect cannot afford", cSite: "src/spec_procs2.c:428", command: "collect", want: "Tester Hey man, you can't afford the 15 gold you need to get your mount outa' hock.", setup: func(_ *testing.T, _ *World, ch *Player) {
			ch.MountVNum, ch.MountCostDay = horseVnum, 5
			ch.MountRentTime = time.Now().Unix() - 3*86400
		}},
		{name: "collect succeeds", cSite: "src/spec_procs2.c:449", command: "collect", want: "Tester Here ya go pal, all patted down and ready to go... cost ya 5 to keep 'em here.", setup: func(_ *testing.T, _ *World, ch *Player) {
			ch.MountVNum, ch.MountCostDay = horseVnum, 5
			ch.MountRentTime = time.Now().Unix()
			ch.SetGold(100)
		}},
		{name: "collect missing prototype", cSite: "src/spec_procs2.c:454", command: "collect", want: "Tester Sorry, we are unable to gather your mount, try back later.", setup: func(_ *testing.T, _ *World, ch *Player) {
			ch.MountVNum, ch.MountCostDay = 99999, 5
			ch.MountRentTime = time.Now().Unix()
			ch.SetGold(100)
		}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, ch, boy, _ := newStableboyBranchWorld(t)
			if tc.setup != nil {
				tc.setup(t, w, ch)
			}
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specStableboy(w, ch, boy, tc.command, tc.argument) }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecRemorterTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, want string
		level, gold                int
	}{
		{name: "buy/list advice", cSite: "src/spec_procs2.c:696", command: "buy", want: "Tester Type REMORT to remort!", level: 5},
		{name: "low level", cSite: "src/spec_procs2.c:707", command: "remort", want: fmt.Sprintf("Tester You can't remort until level %d!\r\n", LVL_IMMORT-1), level: LVL_IMMORT - 2},
		{name: "immortal", cSite: "src/spec_procs2.c:714", command: "remort", want: "Tester Immortals cannot remort!\r\n", level: LVL_IMMORT},
		{name: "not enough gold", cSite: "src/spec_procs2.c:722", command: "remort", want: "Tester It costs 60000 gold to work my magicks.", level: LVL_IMMORT - 1},
		{name: "successful remort", cSite: "src/spec_procs2.c:837", command: "remort", want: "Tester Enjoy your new life...", level: LVL_IMMORT - 1, gold: 60000},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, ch, _ := newSpecProcTestWorld(t)
			me := newSpecProcTestMob(t, w, ch.GetRoomVNum(), 10)
			ch.SetLevel(tc.level)
			ch.SetGold(tc.gold)
			if tc.name == "successful remort" {
				old := levelNumber
				levelNumber = func(_, _ int) int { return 1 }
				t.Cleanup(func() { levelNumber = old })
			}
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specRemorter(w, ch, me, tc.command, "") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecTattoo1TellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, want string
		owned             bool
	}{
		{name: "already tattooed", cSite: "src/spec_procs2.c:978", want: "TattooActor Your magickal center is already tattooed. Get a new arm or get rid of that tattoo then come back.", owned: true},
		{name: "cannot afford", cSite: "src/spec_procs2.c:1000", want: "TattooActor You look a little short on the price there, kid."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, mob, _ := newTattoo1TestWorld(t, false)
			if tc.owned {
				actor.Tattoo = TattooDragon
			}
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specTattoo1(w, actor, mob, "buy", "0") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecTattoo2TellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, want string
		owned             bool
	}{
		{name: "already tattooed", cSite: "src/spec_procs2.c:1055", want: "TattooTwoActor Your magickal center is already tattooed. Your tattoo... is enough magick for such as yourself.", owned: true},
		{name: "cannot afford", cSite: "src/spec_procs2.c:1064", want: "TattooTwoActor Without more coins, I can give no wisdom."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, mob, _ := newTattoo2TestWorld(t, false)
			if tc.owned {
				actor.Tattoo = TattooDragon
			}
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specTattoo2(w, actor, mob, "buy", "0") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecTattoo3TellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, want string
		owned             bool
	}{
		{name: "already tattooed", cSite: "src/spec_procs2.c:1120", want: "TattooThreeActor Your mathickal thenter is awready tattooed. Your tattoo... ith enough mathick for such as yoursewf.", owned: true},
		{name: "cannot afford", cSite: "src/spec_procs2.c:1129", want: "TattooThreeActor You don't have enough cash, hot stuff."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, mob, _ := newTattoo3TestWorld(t, false)
			if tc.owned {
				actor.Tattoo = TattooDragon
			}
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specTattoo3(w, actor, mob, "buy", "0") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func newEviltradeTellWorld(t *testing.T, itemVNum, alignment int) (*World, *Player, *MobInstance) {
	t.Helper()
	w := newFlagWorldWithMob(t)
	ch := NewPlayer(1, "EviltradeActor", 1001)
	ch.SetAlignment(alignment)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	me, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	me.Proto().Keywords = "eviltrade"
	obj := newFlagItem(w, itemVNum, 1)
	if err := w.MoveObjectToPlayerInventory(obj, ch); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}
	return w, ch, me
}

func TestSpecEviltradeTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, want string
		itemVNum, align   int
	}{
		{name: "wrong object", cSite: "src/spec_procs2.c:1159", want: "EviltradeActor Don't give me that crap.", itemVNum: 7101},
		{name: "good aligned player", cSite: "src/spec_procs2.c:1165", want: "EviltradeActor Get lost, goody-goody.", itemVNum: 13111, align: 1},
		{name: "accepted gold watch", cSite: "src/spec_procs2.c:1172", want: "EviltradeActor eneeswseswseswseswsesw", itemVNum: 13111, align: -1},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, ch, me := newEviltradeTellWorld(t, tc.itemVNum, tc.align)
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specEviltrade(w, ch, me, "give", "thing eviltrade") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecTattoo4TellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, want string
		owned             bool
	}{
		{name: "already tattooed", cSite: "src/spec_procs2.c:1323", want: "TattooFourActor Get outta here, punk, you already have one. ", owned: true},
		{name: "cannot afford", cSite: "src/spec_procs2.c:1332", want: "TattooFourActor You don't have enough gold, get outta here!"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, mob, _ := newTattoo4TestWorld(t, false)
			if tc.owned {
				actor.Tattoo = TattooDragon
			}
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specTattoo4(w, actor, mob, "buy", "0") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecIdentifierTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, argument string
		wantCalls                      []mobDirectCommandCall
	}{
		{name: "list", cSite: "src/spec_procs2.c:1202", command: "list", wantCalls: []mobDirectCommandCall{{command: "tell", argument: "IdentifierActor Just read the sign!"}}},
		{name: "value missing argument", cSite: "src/spec_procs2.c:1212", command: "value", wantCalls: []mobDirectCommandCall{{command: "tell", argument: "IdentifierActor Value what?"}}},
		{name: "value missing item", cSite: "src/spec_procs2.c:1219", command: "value", argument: "not-carried", wantCalls: []mobDirectCommandCall{{command: "tell", argument: "IdentifierActor You don't seem to have that."}}},
		{name: "value item", cSite: "src/spec_procs2.c:1224", command: "value", argument: "loaf", wantCalls: []mobDirectCommandCall{{command: "tell", argument: "IdentifierActor I'll identify that fully for about 1 coins."}}},
		{name: "give item cannot afford", cSite: "src/spec_procs2.c:1245 and :1247", command: "give", argument: "loaf Ferrenx", wantCalls: []mobDirectCommandCall{
			{command: "tell", argument: "IdentifierActor That's a fine item, but I'll need 1 coins from you to id it.. and you're a little short.."},
			{command: "tell", argument: "IdentifierActor Keep it until you get the gold."},
		}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, mob, _, _ := newIdentifierTestWorld(t, false)
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specIdentifier(w, actor, mob, tc.command, tc.argument) }, tc.wantCalls...)
		})
	}
}

func TestSpecKeySellerTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, argument, want string
		owner                                int64
		gold                                 int
	}{
		{name: "non-owner", cSite: "src/spec_procs2.c:1889", command: "list", owner: 2, want: "Tester Sorry, I only serve the house owner."},
		{name: "bad key argument", cSite: "src/spec_procs2.c:1901", command: "buy", argument: "hat", owner: 1, want: "Tester I only sell keys, currently."},
		{name: "cannot afford key", cSite: "src/spec_procs2.c:1907", command: "buy", argument: "key", owner: 1, want: "Tester You can't afford a key."},
		{name: "list", cSite: "src/spec_procs2.c:1927", command: "list", owner: 1, want: "Tester You can buy a key for 10,000 gold coins."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _ := newSpecProcTestWorld(t)
			me := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 10)
			w.HouseControl = []HouseControl{{VNum: actor.GetRoomVNum(), Owner: tc.owner, Key: 500}}
			actor.SetGold(tc.gold)
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specKeySeller(w, actor, me, tc.command, tc.argument) }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecClerkTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, argument, want string
		gold, hometown                       int
	}{
		{name: "bad citizenship argument", cSite: "src/spec_procs3.c:116", command: "buy", argument: "citizen", gold: 2000, hometown: 1, want: "ClerkActor BUY CITIZENSHIP, if you're interested."},
		{name: "list", cSite: "src/spec_procs3.c:121", command: "list", hometown: 1, want: "ClerkActor Citizenship costs 2,000 coins."},
		{name: "cannot afford", cSite: "src/spec_procs3.c:126", command: "buy", argument: "citizenship", hometown: 1, want: "ClerkActor You cannot afford it!"},
		{name: "already citizen", cSite: "src/spec_procs3.c:131", command: "buy", argument: "citizenship", gold: 2000, hometown: 2, want: "ClerkActor You are already a citizen here!"},
		{name: "citizenship purchase", cSite: "src/spec_procs3.c:138", command: "buy", argument: "citizenship", gold: 2000, hometown: 1, want: "ClerkActor You are now a citizen of Kir-Oshi."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, mob, _ := newClerkTestWorld(t)
			actor.SetGold(tc.gold)
			actor.Hometown = tc.hometown
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specClerk(w, actor, mob, tc.command, tc.argument) }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecConSellerTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, argument, want string
		gold, origCon, con                   int
	}{
		{name: "bad buy argument", cSite: "src/spec_procs3.c:263", command: "buy", argument: "bad", gold: 4000, origCon: 14, con: 10, want: "ConSellerActor BUY CON, if you really want to do it."},
		{name: "healthy list", cSite: "src/spec_procs3.c:272", command: "list", origCon: 10, con: 10, want: "ConSellerActor You seem perfectly healthy!"},
		{name: "list quote", cSite: "src/spec_procs3.c:279", command: "list", origCon: 14, con: 10, want: "ConSellerActor You can buy up to 4 points, at 400 per point."},
		{name: "cannot afford", cSite: "src/spec_procs3.c:287", command: "buy", argument: "con", origCon: 14, con: 10, want: "ConSellerActor You can't afford it!"},
		{name: "healthy buy", cSite: "src/spec_procs3.c:295", command: "buy", argument: "con", gold: 4000, origCon: 10, con: 10, want: "ConSellerActor You seem perfectly healthy!"},
		{name: "successful buy", cSite: "src/spec_procs3.c:302", command: "buy", argument: "con", gold: 4000, origCon: 14, con: 10, want: "ConSellerActor That'll be 400 coins, you should feel much better.. if you wake up."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, mob, _ := newConSellerTestWorld(t, true)
			actor.SetGold(tc.gold)
			actor.SetOrigCon(tc.origCon)
			actor.Stats.Con = tc.con
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specConSeller(w, actor, mob, tc.command, tc.argument) }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecProstituteTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, want string
	}{
		{name: "buy", cSite: "src/spec_procs3.c:693", command: "buy", want: "Velvet I ain't for sale, just rent. Give me 5 gold for a good time."},
		{name: "list", cSite: "src/spec_procs3.c:701", command: "list", want: "Velvet For five coins, I'll show you a good time."},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, mob, _ := newProstituteTestWorld(t, false)
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specProstitute(w, actor, mob, tc.command, "") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecMorticianTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, want string
		gold                       int
	}{
		{name: "list cost", cSite: "src/spec_procs3.c:827", command: "list", want: "Tester It will cost 580 coins to retrieve your corpse."},
		{name: "cannot afford", cSite: "src/spec_procs3.c:834", command: "retrieve", want: "Tester I'm sorry, you can't afford the cost."},
		{name: "corpse not found", cSite: "src/spec_procs3.c:851", command: "retrieve", want: "Tester I'm sorry, I can't find your corpse anywhere!", gold: 1000},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, mob, _ := newMorticianTestWorld(t)
			actor.SetGold(tc.gold)
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specMortician(w, actor, mob, tc.command, "") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}

func TestSpecRecruiterTellArgumentsMatchC(t *testing.T) {
	tests := []struct {
		name, cSite, command, want string
	}{
		{name: "training", cSite: "src/spec_procs3.c:917", command: "kill", want: "Tester Why don't you sign up for training?  Just head south through those doors!"},
		{name: "no casting", cSite: "src/spec_procs3.c:925", command: "cast", want: "Tester Hey now! None of that voodoo mumbo jumbo in my office!"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _ := newSpecProcTestWorld(t)
			me := newSpecProcTestMob(t, w, actor.GetRoomVNum(), 10)
			assertMobDirectCommandCalls(t, w, tc.cSite, func() bool { return specRecruiter(w, actor, me, tc.command, "") }, mobDirectCommandCall{command: "tell", argument: tc.want})
		})
	}
}
