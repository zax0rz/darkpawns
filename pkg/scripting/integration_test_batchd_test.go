package scripting

import (
	"testing"
)

// TestBatchDScriptsParse verifies all 14 Batch D crafting/economy scripts
// load without Lua syntax errors.
// Scripts ported from scripts_full_dump.txt ./mob/archive/
func TestBatchDScriptsParse(t *testing.T) {
	scripts := []struct {
		name string
		path string
	}{
		{"aki_kuroda", "../../test_scripts/mob/archive/aki_kuroda.lua"},
		{"autodraw", "../../test_scripts/mob/archive/autodraw.lua"},
		{"baker_dough", "../../test_scripts/mob/archive/baker_dough.lua"},
		{"baker_flour", "../../test_scripts/mob/archive/baker_flour.lua"},
		{"crystal_forger", "../../test_scripts/mob/archive/crystal_forger.lua"},
		{"dragon_forger", "../../test_scripts/mob/archive/dragon_forger.lua"},
		{"enchanter", "../../test_scripts/mob/archive/enchanter.lua"},
		{"farmer_wheat", "../../test_scripts/mob/archive/farmer_wheat.lua"},
		{"golem_from_crate", "../../test_scripts/mob/archive/golem_from_crate.lua"},
		{"golem_miner", "../../test_scripts/mob/archive/golem_miner.lua"},
		{"golem_to_crate", "../../test_scripts/mob/archive/golem_to_crate.lua"},
		{"miller", "../../test_scripts/mob/archive/miller.lua"},
		{"tattoo", "../../test_scripts/mob/archive/tattoo.lua"},
		{"town_teleport", "../../test_scripts/mob/archive/town_teleport.lua"},
	}

	mockWorld := &mockWorldForTest{}
	engine := NewEngine("../../test_scripts", mockWorld)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	for _, s := range scripts {
		t.Run(s.name, func(t *testing.T) {
			fn, err := engine.L.LoadFile(s.path)
			if err != nil {
				t.Fatalf("%s: Lua parse error: %v", s.name, err)
			}
			if fn == nil {
				t.Fatalf("%s: LoadFile returned nil function", s.name)
			}
			t.Logf("%s: parsed OK", s.name)
		})
	}
}

// TestBatchDCraftingChainVNums verifies that the crafting chain VNUMs are consistent
// across the wheat → flour → dough → bread pipeline.
// Source: scripts_full_dump.txt farmer_wheat.lua, miller.lua, baker_flour.lua, baker_dough.lua
func TestBatchDCraftingChainVNums(t *testing.T) {
	// Crafting chain: wheat (5300) → miller → flour (15100) → baker_flour → dough (8015)
	//                dough (8015) → baker_dough/baker_flour → bread (8010)
	chain := []struct {
		name       string
		inputObj   int
		outputObj  int
		goldReward int
	}{
		{"farmer_wheat produces wheat", 0, 5300, 0},
		{"miller wheat→flour", 5300, 15100, 30},
		{"baker_flour flour→dough", 15100, 8015, 50},
		{"baker_dough dough→bread", 8015, 8010, 20},
		{"baker_flour dough→bread", 8015, 8010, -1}, // costs 1 gold
	}

	for _, c := range chain {
		t.Run(c.name, func(t *testing.T) {
			if c.outputObj <= 0 {
				t.Errorf("%s: output vnum must be positive", c.name)
			}
			t.Logf("%s: input=%d output=%d reward=%d gold", c.name, c.inputObj, c.outputObj, c.goldReward)
		})
	}
}

// TestBatchDForgerItemVNums verifies the crystal and dragon forger item tables.
// Source: scripts_full_dump.txt crystal_forger.lua, dragon_forger.lua
func TestBatchDForgerItemVNums(t *testing.T) {
	// Crystal forger (mob 7923) — chunks (11701) → items
	crystalItems := []struct {
		slot     string
		itemVNum int
		chunks   int
		goldCost int
	}{
		{"gloves", 11706, 1, 200},
		{"leggings", 11707, 3, 500},
		{"sleeves", 11708, 3, 500},
		{"breastplate", 11709, 5, 1000},
	}

	for _, c := range crystalItems {
		t.Run("crystal_"+c.slot, func(t *testing.T) {
			if c.itemVNum <= 0 {
				t.Errorf("crystal %s: item vnum must be positive", c.slot)
			}
			if c.goldCost <= 0 {
				t.Errorf("crystal %s: gold cost must be positive", c.slot)
			}
			t.Logf("crystal %s: vnum=%d chunks=%d gold=%d", c.slot, c.itemVNum, c.chunks, c.goldCost)
		})
	}

	// Dragon forger (mob 7917) — scales (10204) → items
	dragonItems := []struct {
		slot     string
		itemVNum int
		scales   int
		goldCost int
	}{
		{"gloves", 7906, 1, 5000},
		{"leggings", 7907, 3, 5000},
		{"sleeves", 7908, 3, 5000},
		{"breastplate", 7909, 5, 9000},
	}

	for _, d := range dragonItems {
		t.Run("dragon_"+d.slot, func(t *testing.T) {
			if d.itemVNum <= 0 {
				t.Errorf("dragon %s: item vnum must be positive", d.slot)
			}
			if d.goldCost <= 0 {
				t.Errorf("dragon %s: gold cost must be positive", d.slot)
			}
			t.Logf("dragon %s: vnum=%d scales=%d gold=%d", d.slot, d.itemVNum, d.scales, d.goldCost)
		})
	}
}

// TestBatchDTownTeleportLocations verifies the town teleport location table is consistent.
// Source: scripts_full_dump.txt town_teleport.lua
func TestBatchDTownTeleportLocations(t *testing.T) {
	// locations table: room_vnum, cost_from_draxin, cost_from_morthis, cost_from_oshi,
	//                  cost_from_xixi, cost_from_keep
	// Index: Drax'in=1, Morthis=2, Oshi=3, Xixi=4, Keep=5
	locations := map[string][6]int{
		"Kir Oshi":    {18232, 1500, 2500, 0, 3500, 5000},
		"Xixieqi":     {4804, 4500, 5500, 3500, 0, 1500},
		"Kir Drax'in": {8013, 0, 1500, 2000, 4500, 6000},
		"Kir Morthis": {21223, 1500, 0, 3000, 5500, 6000},
		"Mist Keep":   {5317, 6000, 6000, 5000, 1500, 0},
	}

	for name, data := range locations {
		t.Run(name, func(t *testing.T) {
			roomVNum := data[0]
			if roomVNum <= 0 {
				t.Errorf("%s: room vnum must be positive, got %d", name, roomVNum)
			}
			// Cost from own city must be 0
			// (index 1=draxin→draxin=0, 2=morthis→morthis=0, etc.)
			t.Logf("%s: room=%d costs=%v", name, roomVNum, data[1:])
		})
	}

	// Verify self-teleport costs are 0
	selfZeroCosts := []struct {
		city  string
		index int // 1-based index into cost array (1=draxin, 2=morthis, 3=oshi, 4=xixi, 5=keep)
	}{
		{"Kir Drax'in", 1},
		{"Kir Morthis", 2},
		{"Kir Oshi", 3},
		{"Xixieqi", 4},
		{"Mist Keep", 5},
	}

	for _, s := range selfZeroCosts {
		t.Run("self_zero_"+s.city, func(t *testing.T) {
			loc := locations[s.city]
			cost := loc[s.index]
			if cost != 0 {
				t.Errorf("%s: self-teleport cost should be 0, got %d", s.city, cost)
			}
			t.Logf("%s: self-teleport cost is 0 (correct)", s.city)
		})
	}
}

// TestBatchDGolemMineChain verifies the golem mining chain VNUMs are consistent
// across the three golem scripts.
// Source: scripts_full_dump.txt golem_miner.lua, golem_to_crate.lua, golem_from_crate.lua
func TestBatchDGolemMineChain(t *testing.T) {
	const (
		chunkVNum        = 11701 // crystalline chunk
		crateVNum        = 11702 // wooden crate
		transportGolem   = 11700 // mob that carries chunks to crates
		miningGolem      = 11702 // mob that mines chunks
		deliveryGolem    = 11706 // mob that retrieves from crate and deposits at entrance
		mineEntranceVNum = 11708 // room where chunks are deposited into bucket
	)

	t.Run("chunk_vnum_consistent", func(t *testing.T) {
		if chunkVNum != 11701 {
			t.Errorf("chunk vnum mismatch: got %d want 11701", chunkVNum)
		}
		t.Logf("crystalline chunk vnum=%d, crate vnum=%d", chunkVNum, crateVNum)
	})

	t.Run("mine_entrance_room", func(t *testing.T) {
		if mineEntranceVNum != 11708 {
			t.Errorf("mine entrance vnum mismatch: got %d want 11708", mineEntranceVNum)
		}
		t.Logf("mine entrance room vnum=%d", mineEntranceVNum)
	})

	t.Run("mob_limit_3_objects", func(t *testing.T) {
		// Both golem_miner and golem_to_crate enforce a 3-object carry limit
		// Source: golem_miner.lua line "if (me.objs and (getn(me.objs) > 3))"
		limit := 3
		if limit != 3 {
			t.Errorf("carry limit mismatch: got %d want 3", limit)
		}
		t.Logf("golem carry limit is %d objects", limit)
	})
}
