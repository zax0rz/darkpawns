package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDumpOutputGatesMatchCAtRealDispatch(t *testing.T) {
	for _, test := range []struct {
		name             string
		invisLevel       int
		roomDrop         string
		witnessGetsAward bool
	}{
		{name: "visible actor", roomDrop: "Tester drops a steel sword.\r\n", witnessGetsAward: true},
		{name: "invisible actor", invisLevel: 10, roomDrop: "Someone drops a steel sword.\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			const dumpRoom = 8085
			world, err := game.NewWorld(&parser.World{
				Rooms: []parser.Room{{VNum: dumpRoom, Name: "Dump Room", Zone: 1}},
				Objs: []parser.Obj{{
					VNum:      3001,
					Keywords:  "sword steel",
					ShortDesc: "a steel sword",
					LongDesc:  "A steel sword lies here.",
					Cost:      100,
				}},
			})
			if err != nil {
				t.Fatalf("NewWorld: %v", err)
			}
			t.Cleanup(func() { world.StopAITicker() })
			if specName, assigned := game.GetRoomSpecName(dumpRoom); !assigned || specName != "dump" {
				t.Fatalf("room %d special = %q, assigned %v, want dump", dumpRoom, specName, assigned)
			}

			manager := newTestManager(t, world, nil)
			actor := makeCommandTestSession(t, manager, "Tester", 5, dumpRoom)
			actor.player.SetInvisLevel(test.invisLevel)
			witness := game.NewPlayer(2, "Witness", dumpRoom)
			witness.SetLevel(5)
			if err := world.AddPlayer(actor.player); err != nil {
				t.Fatalf("AddPlayer actor: %v", err)
			}
			if err := world.AddPlayer(witness); err != nil {
				t.Fatalf("AddPlayer witness: %v", err)
			}
			outputs := make(map[string]string)
			world.MessageSink = func(name string, message []byte) {
				outputs[name] += string(message)
			}

			roomItem, err := world.SpawnObject(3001, -1)
			if err != nil {
				t.Fatalf("SpawnObject room item: %v", err)
			}
			if err := world.MoveObjectToRoom(roomItem, dumpRoom); err != nil {
				t.Fatalf("MoveObjectToRoom: %v", err)
			}
			dropped, err := world.SpawnObject(3001, -1)
			if err != nil {
				t.Fatalf("SpawnObject carried item: %v", err)
			}
			if err := world.MoveObjectToPlayerInventory(dropped, actor.player); err != nil {
				t.Fatalf("MoveObjectToPlayerInventory: %v", err)
			}

			if err := ExecuteCommand(actor, "drop", []string{"sword"}); err != nil {
				t.Fatalf("ExecuteCommand(drop): %v", err)
			}
			// C's line is act("$p vanishes in a puff of smoke!", …): $p renders
			// through OBJS and act() CAPs the first character, so it reads
			// "A steel sword vanishes …" (src/spec_procs.c:264, src/comm.c:2477).
			// The oracle vehicle spec-npc-dump-depth proves the same form with
			// object 8000.
			vanish := "A steel sword vanishes in a puff of smoke!\r\n"
			if got, want := outputs["Tester"], vanish+"You drop a steel sword.\r\n"+vanish+"You are awarded for outstanding performance.\r\n"; got != want {
				t.Fatalf("actor bytes = %q, want %q", got, want)
			}
			witnessWant := vanish + test.roomDrop + vanish
			if test.witnessGetsAward {
				witnessWant += "Tester has been awarded by the gods!\r\n"
			}
			if got := outputs["Witness"]; got != witnessWant {
				t.Fatalf("witness bytes = %q, want %q", got, witnessWant)
			}

			outputs["Tester"], outputs["Witness"] = "", ""
			if err := ExecuteCommand(actor, "drop", []string{"sword"}); err != nil {
				t.Fatalf("ExecuteCommand(drop missing item): %v", err)
			}
			if got, want := outputs["Tester"], "You don't seem to have a sword.\r\n"; got != want {
				t.Fatalf("missing-item control bytes = %q, want %q", got, want)
			}
			if got := outputs["Witness"]; got != "" {
				t.Fatalf("missing-item control sent room bytes %q", got)
			}
		})
	}
}

type petShopFixture struct {
	world   *game.World
	actor   *Session
	witness *game.Player
	outputs map[string]string
}

func newPetShopFixture(t *testing.T) *petShopFixture {
	t.Helper()
	const shopRoom = 21235
	const storageRoom = shopRoom + 1
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: shopRoom, Name: "The Pet Shop", Zone: 212},
			{VNum: storageRoom, Name: "Pet Shop Storage", Zone: 212},
		},
		Mobs: []parser.Mob{
			{VNum: 21235, Keywords: "puppy", ShortDesc: "the puppy", LongDesc: "A small loyal puppy is here.\r\n", Level: 1, Cha: 10},
			{VNum: 21236, Keywords: "beagle", ShortDesc: "the beagle", LongDesc: "A small, quick, loyal beagle is here.\r\n", Level: 2, Cha: 10},
			{VNum: 21237, Keywords: "rottweiler", ShortDesc: "the rottweiler", LongDesc: "A large, loyal rottweiler is here.\r\n", Level: 3, Cha: 10},
			{VNum: 21238, Keywords: "wolf", ShortDesc: "the wolf", LongDesc: "A large, trained wolf is here.\r\n", Level: 4, Cha: 10},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })
	if specName, assigned := game.GetRoomSpecName(shopRoom); !assigned || specName != "pet_shops" {
		t.Fatalf("room %d special = %q, assigned %v, want pet_shops", shopRoom, specName, assigned)
	}
	manager := newTestManager(t, world, nil)
	actor := makeCommandTestSession(t, manager, "Tester", 5, shopRoom)
	actor.player.SetStat("CHA", 10)
	actor.player.CopyBaseAttributes()
	witness := game.NewPlayer(2, "Witness", shopRoom)
	witness.SetLevel(5)
	if err := world.AddPlayer(actor.player); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if err := world.AddPlayer(witness); err != nil {
		t.Fatalf("AddPlayer witness: %v", err)
	}
	outputs := make(map[string]string)
	world.MessageSink = func(name string, message []byte) {
		outputs[name] += string(message)
	}
	for _, vnum := range []int{21235, 21236, 21237, 21238} {
		if _, err := world.SpawnMob(vnum, storageRoom); err != nil {
			t.Fatalf("SpawnMob %d in storage: %v", vnum, err)
		}
	}
	return &petShopFixture{world: world, actor: actor, witness: witness, outputs: outputs}
}

func TestPetShopOutputGatesMatchCAtRealDispatch(t *testing.T) {
	t.Run("list uses C room order and keyword names", func(t *testing.T) {
		fixture := newPetShopFixture(t)
		if err := ExecuteCommand(fixture.actor, "list", nil); err != nil {
			t.Fatalf("ExecuteCommand(list): %v", err)
		}
		want := "Available pets are:\r\n     100 - the wolf\r\n      75 - the rottweiler\r\n      50 - the beagle\r\n      25 - the puppy\r\n"
		if got := fixture.outputs["Tester"]; got != want {
			t.Fatalf("list bytes = %q, want %q", got, want)
		}
		if got := fixture.outputs["Witness"]; got != "" {
			t.Fatalf("list sent room bytes %q", got)
		}
	})

	t.Run("missing pet refusal", func(t *testing.T) {
		fixture := newPetShopFixture(t)
		fixture.actor.player.SetGold(1000)
		if err := ExecuteCommand(fixture.actor, "buy", []string{"unicorn"}); err != nil {
			t.Fatalf("ExecuteCommand(buy missing pet): %v", err)
		}
		if got, want := fixture.outputs["Tester"], "There is no such pet!\r\n"; got != want {
			t.Fatalf("missing-pet bytes = %q, want %q", got, want)
		}
		if got := fixture.outputs["Witness"]; got != "" {
			t.Fatalf("missing-pet control sent room bytes %q", got)
		}
	})

	t.Run("follower-cap refusal precedes gold check", func(t *testing.T) {
		fixture := newPetShopFixture(t)
		fixture.actor.player.SetStat("CHA", 2)
		fixture.actor.player.CopyBaseAttributes()
		fixture.actor.player.SetGold(1000)
		follower, err := fixture.world.SpawnMob(21238, 21235)
		if err != nil {
			t.Fatalf("SpawnMob existing follower: %v", err)
		}
		game.AddFollowerQuietMob(follower, fixture.actor.player)
		if err := ExecuteCommand(fixture.actor, "buy", []string{"puppy"}); err != nil {
			t.Fatalf("ExecuteCommand(buy at follower cap): %v", err)
		}
		if got, want := fixture.outputs["Tester"], "You can't have any more followers!\r\n"; got != want {
			t.Fatalf("follower-cap bytes = %q, want %q", got, want)
		}
		if got := fixture.outputs["Witness"]; got != "" {
			t.Fatalf("follower-cap control sent room bytes %q", got)
		}
	})

	t.Run("insufficient-gold refusal", func(t *testing.T) {
		fixture := newPetShopFixture(t)
		fixture.actor.player.SetGold(24)
		if err := ExecuteCommand(fixture.actor, "buy", []string{"puppy"}); err != nil {
			t.Fatalf("ExecuteCommand(buy without gold): %v", err)
		}
		if got, want := fixture.outputs["Tester"], "You don't have enough gold!\r\n"; got != want {
			t.Fatalf("insufficient-gold bytes = %q, want %q", got, want)
		}
		if got := fixture.outputs["Witness"]; got != "" {
			t.Fatalf("insufficient-gold control sent room bytes %q", got)
		}
	})

	t.Run("successful named purchase", func(t *testing.T) {
		fixture := newPetShopFixture(t)
		fixture.actor.player.SetGold(100)
		if err := ExecuteCommand(fixture.actor, "buy", []string{"puppy", "Milo"}); err != nil {
			t.Fatalf("ExecuteCommand(buy named pet): %v", err)
		}
		actorWant := "The puppy starts following you.\r\nMay you enjoy your pet.\r\n"
		if got := fixture.outputs["Tester"]; got != actorWant {
			t.Fatalf("buyer bytes = %q, want %q", got, actorWant)
		}
		witnessWant := "The puppy starts to follow Tester.\r\nTester buys the puppy as a pet.\r\n"
		if got := fixture.outputs["Witness"]; got != witnessWant {
			t.Fatalf("witness bytes = %q, want %q", got, witnessWant)
		}
		var bought *game.MobInstance
		for _, mob := range fixture.world.GetMobsInRoom(21235) {
			if mob.GetVNum() == 21235 && mob.GetFollowing() == "Tester" {
				bought = mob
				break
			}
		}
		if bought == nil {
			t.Fatal("successful buy did not create a follower in the shop")
		}
		if bought.GetName() != "the puppy" || bought.Proto().Keywords != "puppy milo" || bought.GetShortDesc() != "the puppy" || bought.GetExp() != 0 || !bought.IsAffected(game.AffCharm) {
			t.Fatalf("bought pet state: name=%q short=%q exp=%d charmed=%v", bought.GetName(), bought.GetShortDesc(), bought.GetExp(), bought.IsAffected(game.AffCharm))
		}
		if got, want := bought.Proto().DetailedDesc, "A small sign on a chain around the neck says 'My name is milo'\r\n"; got != want {
			t.Fatalf("named pet description = %q, want %q", got, want)
		}
		if got := bought.GetLongDesc(); got != "A small loyal puppy is here.\r\n" {
			t.Fatalf("named pet changed room description: %q", got)
		}
		if got := fixture.actor.player.GetGold(); got != 75 {
			t.Fatalf("buyer gold = %d, want 75", got)
		}
		if got := fixture.world.NumFollowers("Tester"); got != 1 {
			t.Fatalf("buyer follower count = %d, want 1", got)
		}
	})
}
