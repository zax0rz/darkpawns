package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mockShopSession implements common.CommandSession for shop command tests.
type mockShopSession struct {
	player   *game.Player
	messages []string
	roomVNum int
}

func (m *mockShopSession) Send(msg string)        { m.messages = append(m.messages, msg) }
func (m *mockShopSession) Close()                 {}
func (m *mockShopSession) GetPlayer() interface{} { return m.player }
func (m *mockShopSession) GetPlayerName() string  { return m.player.Name }
func (m *mockShopSession) GetPlayerRoomVNum() int { return m.roomVNum }
func (m *mockShopSession) IsAuthenticated() bool  { return true }
func (m *mockShopSession) HasPlayer() bool        { return m.player != nil }
func (m *mockShopSession) GetPlayerLevel() int    { return m.player.GetLevel() }

func TestParseItemNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "valid positive", input: "5", want: 5, wantErr: false},
		{name: "zero", input: "0", want: 0, wantErr: false},
		{name: "large number", input: "9999", want: 9999, wantErr: false},
		{name: "non-numeric", input: "abc", want: 0, wantErr: true},
		{name: "empty string", input: "", want: 0, wantErr: true},
		{name: "mixed trailing", input: "5abc", want: 5, wantErr: false},
		{name: "negative", input: "-3", want: -3, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseItemNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseItemNumber(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseItemNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenderPronoun(t *testing.T) {
	tests := []struct {
		name string
		sex  int
		want string
	}{
		{name: "male", sex: 1, want: "himself"},
		{name: "female", sex: 0, want: "herself"},
		{name: "neutral default", sex: 2, want: "itself"},
		{name: "negative unused", sex: -1, want: "itself"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genderPronoun(tt.sex); got != tt.want {
				t.Errorf("genderPronoun(%d) = %q, want %q", tt.sex, got, tt.want)
			}
		})
	}
}

// newShopTestWorld creates a minimal world, player, and ShopCommands for testing.
func newShopTestWorld(t *testing.T) (*game.World, *game.Player, *systems.ShopManager, *ShopCommands) {
	t.Helper()

	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 2001, Name: "Market Square", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	player := game.NewPlayer(1, "Shopper", 2001)
	player.Class = game.ClassWarrior
	player.SetLevel(5)
	player.Stats = game.CharStats{Str: 14, Dex: 14, Int: 10, Wis: 10, Con: 12, Cha: 10}
	player.SetPosition(combat.PosStanding)
	player.Gold = 5000
	if err := world.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	sm := systems.NewShopManager()
	sm.CreateShopConcrete(5001, "General Store", 2001)
	sc := NewShopCommands(sm, world)

	return world, player, sm, sc
}

func newShopTestItem(vnum int, keywords, shortDesc string, cost, typeFlag int) *game.ObjectInstance {
	return &game.ObjectInstance{
		VNum:      vnum,
		Prototype: &parser.Obj{VNum: vnum, Keywords: keywords, ShortDesc: shortDesc, Cost: cost, TypeFlag: typeFlag},
	}
}

func TestShopCommands_CmdListShop_NoShops(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	// Use a room VNum that has no shop
	session := &mockShopSession{player: player, roomVNum: 9999}

	err := sc.CmdListShop(session, nil)
	if err == nil {
		t.Fatal("expected error for room with no shops")
	}
	if !strings.Contains(err.Error(), "no shops here") {
		t.Errorf("expected 'no shops here', got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdListShop_WithItems(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	// Add items to shop inventory
	shop := sm.GetShopsInRoomConcrete(2001)[0]
	sword := newShopTestItem(100, "sword", "a sturdy sword", 500, 1)
	shield := newShopTestItem(101, "shield", "a wooden shield", 300, 1)
	shop.AddItem(sword)
	shop.AddItem(shield)

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdListShop(session, nil)
	if err != nil {
		t.Fatalf("CmdListShop: %v", err)
	}
	output := strings.Join(session.messages, "")
	if !strings.Contains(output, "General Store") {
		t.Errorf("expected shop name in output, got: %s", output)
	}
	if !strings.Contains(output, "sturdy sword") {
		t.Errorf("expected 'sturdy sword' in output, got: %s", output)
	}
	if !strings.Contains(output, "wooden shield") {
		t.Errorf("expected 'wooden shield' in output, got: %s", output)
	}
	if !strings.Contains(output, "5000 gold") {
		t.Errorf("expected player gold in output, got: %s", output)
	}
	_ = world
}

func TestShopCommands_CmdBuy_ByNumber(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	sm.GetShopsInRoomConcrete(2001)[0].AddItem(newShopTestItem(100, "sword", "a sturdy sword", 500, 1))
	preGold := player.Gold

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdBuy(session, []string{"1"})
	if err != nil {
		t.Fatalf("CmdBuy: %v", err)
	}
	if player.Gold >= preGold {
		t.Errorf("expected gold to decrease after purchase: pre=%d post=%d", preGold, player.Gold)
	}
	_ = world
}

func TestShopCommands_CmdBuy_ByName(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	shop := sm.GetShopsInRoomConcrete(2001)[0]
	shop.AddItem(newShopTestItem(100, "sword", "a sturdy sword", 500, 1))
	preGold := player.Gold

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdBuy(session, []string{"sword"})
	if err != nil {
		t.Fatalf("CmdBuy: %v", err)
	}
	if player.Gold >= preGold {
		t.Errorf("expected gold to decrease after purchase: pre=%d post=%d", preGold, player.Gold)
	}
	if !strings.Contains(session.messages[len(session.messages)-1], "buy") {
		t.Errorf("expected success message, got: %v", session.messages)
	}
	_ = world
}

func TestShopCommands_CmdBuy_InvalidItemNumber(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)
	// Add one item so shop isn't empty, but 999 is out of range
	sm.GetShopsInRoomConcrete(2001)[0].AddItem(newShopTestItem(100, "sword", "a sturdy sword", 500, 1))
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdBuy(session, []string{"999"})
	if err == nil {
		t.Fatal("expected error for out-of-range item number")
	}
	if !strings.Contains(err.Error(), "item number 999 is not available") {
		t.Errorf("expected item number error, got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdBuy_NotForSale(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)
	// Add items so we get past the "nothing for sale" check, but search for something not there
	sm.GetShopsInRoomConcrete(2001)[0].AddItem(newShopTestItem(100, "sword", "a sturdy sword", 500, 1))
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdBuy(session, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown item")
	}
	if !strings.Contains(err.Error(), "not for sale") {
		t.Errorf("expected 'not for sale', got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdBuy_NotEnoughGold(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	shop := sm.GetShopsInRoomConcrete(2001)[0]
	// Expensive item
	shop.AddItem(newShopTestItem(200, "gem", "a precious gem", 100000, 1))
	player.Gold = 10

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdBuy(session, []string{"1"})
	if err == nil {
		t.Fatal("expected error for insufficient gold")
	}
	if !strings.Contains(err.Error(), "need") && !strings.Contains(err.Error(), "gold") {
		t.Errorf("expected insufficient gold error, got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdSell(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	// Give player an item to sell
	item := newShopTestItem(300, "rock", "a普通 rock", 100, 1)
	player.Inventory.AddItem(item)
	preGold := player.Gold
	shop := sm.GetShopsInRoomConcrete(2001)[0]
	// Shop must buy this type
	shop.BuyTypes = []int{1}

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdSell(session, []string{"rock"})
	if err != nil {
		t.Fatalf("CmdSell: %v", err)
	}
	if player.Gold <= preGold {
		t.Errorf("expected gold to increase after sale: pre=%d post=%d", preGold, player.Gold)
	}
	_ = world
}

func TestShopCommands_CmdSell_NoItem(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdSell(session, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error when selling nonexistent item")
	}
	if !strings.Contains(err.Error(), "don't have") {
		t.Errorf("expected 'don't have', got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdSell_NoArgs(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdSell(session, nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage: sell") {
		t.Errorf("expected usage message, got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdValue(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	item := newShopTestItem(300, "rock", "a普通 rock", 100, 1)
	player.Inventory.AddItem(item)
	shop := sm.GetShopsInRoomConcrete(2001)[0]
	shop.BuyTypes = []int{1}

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdValue(session, []string{"rock"})
	if err != nil {
		t.Fatalf("CmdValue: %v", err)
	}
	output := strings.Join(session.messages, "")
	if !strings.Contains(output, "buy for") {
		t.Errorf("expected buy price in output, got: %s", output)
	}
	if !strings.Contains(output, "sells for") {
		t.Errorf("expected sell price in output, got: %s", output)
	}
	_ = world
}

func TestShopCommands_CmdValue_ShopNotInterested(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	item := newShopTestItem(300, "rock", "a普通 rock", 100, 2) // type 2 — not in BuyTypes
	player.Inventory.AddItem(item)
	shop := sm.GetShopsInRoomConcrete(2001)[0]
	shop.BuyTypes = []int{1} // only buys type 1

	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdValue(session, []string{"rock"})
	if err != nil {
		t.Fatalf("CmdValue: %v", err)
	}
	if !strings.Contains(session.messages[len(session.messages)-1], "isn't interested") {
		t.Errorf("expected 'not interested' message, got: %v", session.messages)
	}
	_ = world
}

func TestShopCommands_CmdRepair(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	item := newShopTestItem(300, "sword", "a chipped sword", 200, 1)
	player.Inventory.AddItem(item)
	shop := sm.GetShopsInRoomConcrete(2001)[0]
	shop.RepairCost = 5

	preGold := player.Gold
	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdRepair(session, []string{"sword"})
	if err != nil {
		t.Fatalf("CmdRepair: %v", err)
	}
	if player.Gold >= preGold {
		t.Errorf("expected gold to decrease after repair: pre=%d post=%d", preGold, player.Gold)
	}
	_ = world
}

func TestShopCommands_CmdRepair_NoItem(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdRepair(session, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent item")
	}
	if !strings.Contains(err.Error(), "don't have") {
		t.Errorf("expected 'don't have', got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdIdentify(t *testing.T) {
	world, player, sm, sc := newShopTestWorld(t)

	item := newShopTestItem(300, "ring", "a mysterious ring", 500, 1)
	player.Inventory.AddItem(item)
	shop := sm.GetShopsInRoomConcrete(2001)[0]
	shop.IdentifyCost = 10

	preGold := player.Gold
	session := &mockShopSession{player: player, roomVNum: 2001}
	err := sc.CmdIdentify(session, []string{"ring"})
	if err != nil {
		t.Fatalf("CmdIdentify: %v", err)
	}
	if player.Gold >= preGold {
		t.Errorf("expected gold to decrease after identify: pre=%d post=%d", preGold, player.Gold)
	}
	_ = world
}

func TestShopCommands_CmdIdentify_NoItem(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdIdentify(session, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent item")
	}
	if !strings.Contains(err.Error(), "don't have") {
		t.Errorf("expected 'don't have', got: %v", err)
	}
	_ = world
}

func TestShopCommands_CmdBuy_NoArgs(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	session := &mockShopSession{player: player, roomVNum: 2001}

	err := sc.CmdBuy(session, nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage: buy") {
		t.Errorf("expected usage message, got: %v", err)
	}
	_ = world
}

func TestShopCommands_NotLoggedIn(t *testing.T) {
	world, _, _, sc := newShopTestWorld(t)
	t.Cleanup(world.StopAITicker)

	session := &mockShopSession{player: nil, roomVNum: 2001}

	t.Run("List", func(t *testing.T) {
		err := sc.CmdListShop(session, nil)
		if err == nil {
			t.Error("expected error when not logged in")
		} else if !strings.Contains(err.Error(), "logged in") {
			t.Errorf("expected 'logged in' error, got: %v", err)
		}
	})
	t.Run("Buy", func(t *testing.T) {
		err := sc.CmdBuy(session, []string{"sword"})
		if err == nil {
			t.Error("expected error when not logged in")
		} else if !strings.Contains(err.Error(), "logged in") {
			t.Errorf("expected 'logged in' error, got: %v", err)
		}
	})
	t.Run("Sell", func(t *testing.T) {
		err := sc.CmdSell(session, []string{"sword"})
		if err == nil {
			t.Error("expected error when not logged in")
		} else if !strings.Contains(err.Error(), "logged in") {
			t.Errorf("expected 'logged in' error, got: %v", err)
		}
	})
	t.Run("Repair", func(t *testing.T) {
		err := sc.CmdRepair(session, []string{"sword"})
		if err == nil {
			t.Error("expected error when not logged in")
		} else if !strings.Contains(err.Error(), "logged in") {
			t.Errorf("expected 'logged in' error, got: %v", err)
		}
	})
	t.Run("Identify", func(t *testing.T) {
		err := sc.CmdIdentify(session, []string{"ring"})
		if err == nil {
			t.Error("expected error when not logged in")
		} else if !strings.Contains(err.Error(), "logged in") {
			t.Errorf("expected 'logged in' error, got: %v", err)
		}
	})
	t.Run("Value", func(t *testing.T) {
		err := sc.CmdValue(session, []string{"rock"})
		if err == nil {
			t.Error("expected error when not logged in")
		} else if !strings.Contains(err.Error(), "logged in") {
			t.Errorf("expected 'logged in' error, got: %v", err)
		}
	})
}

func TestShopCommands_CmdList_NoRoom(t *testing.T) {
	world, player, _, sc := newShopTestWorld(t)
	t.Cleanup(world.StopAITicker)
	session := &mockShopSession{player: player, roomVNum: 0}

	err := sc.CmdListShop(session, nil)
	if err == nil {
		t.Fatal("expected error when player has invalid room")
	}
	if !strings.Contains(err.Error(), "valid room") {
		t.Errorf("expected 'valid room' error, got: %v", err)
	}
}
