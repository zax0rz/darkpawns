package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// The brief's proven live failure, through the real dispatch: buy a
// container, then put an item into it. On unregistered-main this prints
// "You can't put that in there." and the item stays out.
func TestBuyContainerThenPutSucceeds(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Shop Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 5001, ShortDesc: "the shopkeeper", Level: 5,
			HP: parser.DiceRoll{Num: 1, Sides: 10, Plus: 90},
		}},
		Objs: []parser.Obj{
			// TypeFlag 15 = container; Values[0]=0 means holds anything.
			{VNum: 7001, Keywords: "bag", ShortDesc: "a leather bag", LongDesc: "A bag.", TypeFlag: 15, Values: [4]int{0, 0, 0, 0}},
			{VNum: 7002, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword.", Cost: 10},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, nil)

	if _, err := world.SpawnMob(5001, 1001); err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	sm := game.NewShopManager()
	sm.AddShop(&game.Shop{
		VNum: 1, KeeperVNum: 5001,
		SellTypes: []int{7001, 7002}, BuyTypes: []int{},
		ProfitBuy: 1.0, ProfitSell: 1.0,
	})
	world.SetShopManager(sm)

	s := makeTestSession(t, m, "Buyer", 1001, true)
	s.player.SetGold(200)

	// Buy the container through the real command.
	if err := cmdBuy(s, []string{"bag"}); err != nil {
		t.Fatalf("cmdBuy bag: %v", err)
	}
	bag := s.player.Inventory.FindItems("bag")
	if len(bag) == 0 {
		t.Fatal("bag not bought — test preconditions wrong")
	}

	// Buy the item.
	if err := cmdBuy(s, []string{"sword"}); err != nil {
		t.Fatalf("cmdBuy sword: %v", err)
	}
	if len(s.player.Inventory.FindItems("sword")) == 0 {
		t.Fatal("sword not bought — test preconditions wrong")
	}

	// Put the sword in the bag — the exact live failure. Success is the
	// sword's Location: ObjInContainer with the (now registered) bag's ID.
	// On unregistered-main this path printed "You can't put that in there."
	// because MoveObjectToContainer rejected ContainerObjID 0.
	if err := cmdPut(s, []string{"sword", "bag"}); err != nil {
		t.Fatalf("put sword bag: %v", err)
	}
	swords := s.player.Inventory.FindItems("sword")
	if len(swords) == 0 {
		t.Fatal("sword vanished entirely")
	}
	for _, sw := range swords {
		if sw.Location.Kind != game.ObjInContainer || sw.Location.ContainerObjID != bag[0].GetInstanceID() {
			t.Fatalf("sword not in bag: location = %+v (bag id %d)", sw.Location, bag[0].GetInstanceID())
		}
	}
}
