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

func test_scripts() {
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

	// Test loading globals.lua
	fmt.Println("Testing globals.lua...")
	
	// Create a test context
	ctx := &scripting.ScriptContext{
		RoomVNum: 30, // Pattern room
		Argument: "",
		World:    worldAdapter,
	}

	// Test loading pattern_dmg.lua
	fmt.Println("Testing pattern_dmg.lua...")
	handled, err := scriptEngine.RunScript(ctx, "room/30/pattern_dmg.lua", "onpulse")
	if err != nil {
		fmt.Printf("Error loading pattern_dmg.lua: %v\n", err)
	} else {
		fmt.Printf("pattern_dmg.lua loaded: handled=%v\n", handled)
	}

	// Test loading mob scripts
	fmt.Println("\nTesting mob scripts...")
	
	// Test no_move.lua
	ctx2 := &scripting.ScriptContext{
		RoomVNum: 30,
		Argument: "north",
		World:    worldAdapter,
	}
	
	handled2, err2 := scriptEngine.RunScript(ctx2, "mob/no_move.lua", "oncmd")
	if err2 != nil {
		fmt.Printf("Error loading no_move.lua: %v\n", err2)
	} else {
		fmt.Printf("no_move.lua loaded: handled=%v\n", handled2)
	}

	// Test assembler.lua (helper library, no trigger)
	fmt.Println("\nTesting assembler.lua...")
	// assembler.lua is a library, not a trigger script
	// Just verify it loads without error
	handled3, err3 := scriptEngine.RunScript(ctx, "mob/assembler.lua", "ongive")
	if err3 != nil {
		fmt.Printf("Error loading assembler.lua: %v\n", err3)
	} else {
		fmt.Printf("assembler.lua loaded: handled=%v\n", handled3)
	}

	fmt.Println("\nScript loading test complete!")
}