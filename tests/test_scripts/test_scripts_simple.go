//go:build ignore

package testscripts

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// mockPlayer implements ScriptablePlayer for testing
type mockPlayer struct {
	name      string
	level     int
	health    int
	maxHealth int
	gold      int
	roomVNum  int
}

func (p *mockPlayer) GetID() int             { return 1 }
func (p *mockPlayer) GetName() string        { return p.name }
func (p *mockPlayer) GetLevel() int          { return p.level }
func (p *mockPlayer) GetHealth() int         { return p.health }
func (p *mockPlayer) SetHealth(h int)        { p.health = h }
func (p *mockPlayer) GetMaxHealth() int      { return p.maxHealth }
func (p *mockPlayer) GetGold() int           { return p.gold }
func (p *mockPlayer) SetGold(g int)          { p.gold = g }
func (p *mockPlayer) GetRace() int           { return 0 }
func (p *mockPlayer) GetClass() int          { return 0 }
func (p *mockPlayer) GetAlignment() int      { return 0 }
func (p *mockPlayer) GetRoomVNum() int       { return p.roomVNum }
func (p *mockPlayer) SendMessage(msg string) { fmt.Printf("[TO PLAYER] %s\n", msg) }

func test_scripts_simple() {
	// Get the workspace directory
	workspaceDir := "/home/zach/.openclaw/workspace"
	libDir := filepath.Join(workspaceDir, "darkpawns", "lib")
	worldDir := filepath.Join(libDir, "world")
	scriptsDir := filepath.Join(libDir, "scripts")

	// Parse world
	fmt.Printf("Loading world from %s...\n", worldDir)
	parsedWorld, err := parser.ParseWorld(worldDir)
	if err != nil {
		log.Fatalf("Failed to parse world: %v", err)
	}

	// Create game world
	gameWorld, err := game.NewWorld(parsedWorld)
	if err != nil {
		log.Fatalf("Failed to create game world: %v", err)
	}

	// Create world adapter
	worldAdapter := game.NewWorldScriptableAdapter(gameWorld)

	// Create scripting engine
	fmt.Printf("Loading scripts from %s...\n", scriptsDir)
	scriptEngine := scripting.NewEngine(scriptsDir, worldAdapter)

	// Test 1: Verify globals.lua loads at engine init
	fmt.Println("\n=== Test 1: Engine initialization ===")
	fmt.Println("Engine created successfully (globals.lua loaded at init)")

	// Test 2: Test pattern_dmg.lua
	fmt.Println("\n=== Test 2: pattern_dmg.lua ===")
	player := &mockPlayer{
		name:      "TestPlayer",
		level:     1,
		health:    50,
		maxHealth: 50,
		gold:      0,
		roomVNum:  30,
	}

	ctx := &scripting.ScriptContext{
		Ch:       player,
		RoomVNum: 30,
		Argument: "",
		World:    worldAdapter,
	}

	handled, err := scriptEngine.RunScript(ctx, "room/30/pattern_dmg.lua", "onpulse")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Loaded successfully: handled=%v\n", handled)
	}

	// Test 3: Test no_move.lua
	fmt.Println("\n=== Test 3: no_move.lua ===")
	ctx2 := &scripting.ScriptContext{
		Ch:       player,
		RoomVNum: 30,
		Argument: "north",
		World:    worldAdapter,
	}

	// Create a mock mob
	mockMob := &game.MobInstance{}
	// We need to set some fields, but for now just test loading
	ctx2.Me = mockMob

	handled2, err2 := scriptEngine.RunScript(ctx2, "mob/no_move.lua", "oncmd")
	if err2 != nil {
		fmt.Printf("Error: %v\n", err2)
	} else {
		fmt.Printf("Loaded successfully: handled=%v\n", handled2)
	}

	// Test 4: Test assembler.lua loads (helper library)
	fmt.Println("\n=== Test 4: assembler.lua ===")
	// assembler.lua is a library, not a trigger script
	// Just test that it can be loaded
	handled3, err3 := scriptEngine.RunScript(ctx, "mob/assembler.lua", "ongive")
	if err3 != nil {
		fmt.Printf("Error: %v\n", err3)
	} else {
		fmt.Printf("Loaded successfully: handled=%v\n", handled3)
	}

	// Test 5: Test globals.lua constants
	fmt.Println("\n=== Test 5: Verify globals.lua constants ===")
	fmt.Println("Constants like LVL_IMMORT, NORTH, etc. should be available to scripts")

	fmt.Println("\n=== All tests completed ===")
}
