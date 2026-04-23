//go:build ignore

package testscripts

import (
	"fmt"
	"log"
	"path/filepath"

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

// mockWorld implements ScriptableWorld for testing
type mockWorld struct{}

func (w *mockWorld) GetPlayersInRoom(roomVNum int) []scripting.ScriptablePlayer {
	return nil
}

func (w *mockWorld) GetObjPrototype(vnum int) scripting.ScriptableObject {
	return nil
}

func (w *mockWorld) AddItemToRoom(obj scripting.ScriptableObject, roomVNum int) error {
	return nil
}

func (w *mockWorld) HandleNonCombatDeath(player scripting.ScriptablePlayer) {
}

func test_debug2() {
	workspaceDir := "/home/zach/.openclaw/workspace"
	libDir := filepath.Join(workspaceDir, "darkpawns", "lib")
	scriptsDir := filepath.Join(libDir, "scripts")

	fmt.Printf("Testing scripts in %s\n", scriptsDir)

	world := &mockWorld{}

	// Create engine
	engine := scripting.NewEngine(scriptsDir, world)
	fmt.Println("Engine created")

	// Create a player
	player := &mockPlayer{
		name:      "TestPlayer",
		level:     1,
		health:    50,
		maxHealth: 50,
		gold:      0,
		roomVNum:  30,
	}

	// Create context
	ctx := &scripting.ScriptContext{
		Ch:       player,
		RoomVNum: 30,
		Argument: "",
		World:    world,
	}

	// Test with simple script
	fmt.Println("\nTesting simple script...")
	handled, err := engine.RunScript(ctx, "test_simple.lua", "test")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Result: handled=%v\n", handled)
	}
}
