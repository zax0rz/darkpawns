//go:build ignore

package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// mockMinimalWorld implements ScriptableWorld for minimal testing
type mockMinimalWorld struct{}

func (w *mockMinimalWorld) GetPlayersInRoom(roomVNum int) []scripting.ScriptablePlayer {
	return nil
}
func (w *mockMinimalWorld) GetObjPrototype(vnum int) scripting.ScriptableObject {
	return nil
}
func (w *mockMinimalWorld) AddItemToRoom(obj scripting.ScriptableObject, roomVNum int) error {
	return nil
}
func (w *mockMinimalWorld) HandleNonCombatDeath(player scripting.ScriptablePlayer) {
}

func main() {
	workspaceDir := "/home/zach/.openclaw/workspace"
	libDir := filepath.Join(workspaceDir, "darkpawns", "lib")
	scriptsDir := filepath.Join(libDir, "scripts")

	fmt.Printf("Testing scripts in %s\n", scriptsDir)

	world := &mockMinimalWorld{}
	
	// Create engine
	engine := scripting.NewEngine(scriptsDir, world)
	fmt.Println("Engine created")
	
	// Try to get a constant
	fmt.Println("Testing if LVL_IMMORT is defined...")
	// We can't directly check Lua globals from Go easily
	// But we can test by running a simple script
}