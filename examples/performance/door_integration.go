// Example: Door System Integration
// This shows how to integrate the door system with Dark Pawns.

package main

import (
	"fmt"
	"log"

	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func doorIntegration() {
	// Example of integrating doors with the game world

	// 1. Create a door manager
	doorManager := systems.NewDoorManager()

	// 2. Create example parsed world data
	parsedWorld := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 100,
				Name: "A small room",
				Exits: map[string]parser.Exit{
					"north": {
						Direction: "north",
						ToRoom:    101,
						DoorState: 1, // closed
						Key:       -1,
					},
					"east": {
						Direction: "east",
						ToRoom:    102,
						DoorState: 2,   // locked
						Key:       500, // key VNum
					},
				},
			},
			{
				VNum: 101,
				Name: "A northern room",
				Exits: map[string]parser.Exit{
					"south": {
						Direction: "south",
						ToRoom:    100,
						DoorState: 1, // closed
						Key:       -1,
					},
				},
			},
			{
				VNum: 102,
				Name: "An eastern room",
				Exits: map[string]parser.Exit{
					"west": {
						Direction: "west",
						ToRoom:    100,
						DoorState: 2, // locked
						Key:       500,
					},
				},
			},
		},
	}

	// 3. Load doors from parsed world
	doorManager.LoadDoorsFromWorld(parsedWorld)

	fmt.Printf("Loaded %d doors\n", doorManager.Count())

	// 4. Example: Check if player can pass through north door
	canPass, msg := doorManager.CanPass(100, "north")
	fmt.Printf("Can pass north from room 100: %v (%s)\n", canPass, msg)

	// 5. Example: Open the north door
	success, msg := doorManager.OpenDoor(100, "north")
	fmt.Printf("Open north door: %v (%s)\n", success, msg)

	// 6. Now check again if player can pass
	canPass, msg = doorManager.CanPass(100, "north")
	fmt.Printf("Can pass north from room 100 after opening: %v (%s)\n", canPass, msg)

	// 7. Example: Try to open locked east door (should fail without key)
	success, msg = doorManager.OpenDoor(100, "east")
	fmt.Printf("Open east door (locked): %v (%s)\n", success, msg)

	// 8. Example: Try to pick the lock (with skill 60)
	success, msg = doorManager.PickDoor(100, "east", 60)
	fmt.Printf("Pick east door lock (skill 60): %v (%s)\n", success, msg)

	// 9. Example: Get all doors in room 100
	doors := doorManager.GetDoorsInRoom(100)
	fmt.Printf("Doors in room 100: %d\n", len(doors))
	for _, door := range doors {
		fmt.Printf("  - %s door to room %d: %s\n",
			door.Direction, door.ToRoom, door.GetStatus())
	}

	// 10. Integration with game World
	// In the actual game, you would add DoorManager to the World struct:
	// type World struct {
	//     // ... existing fields ...
	//     Doors *systems.DoorManager
	// }

	// And update MovePlayer to check doors:
	log.Println("\nIntegration example complete.")
	log.Println("See docs/doors.md for full integration guide.")
}
