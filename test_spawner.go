package main

import (
	"fmt"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func main() {
	// Create a minimal parsed world for testing
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1, Name: "Test Room", Description: "A test room."},
			{VNum: 2, Name: "Another Room", Description: "Another test room."},
		},
		Mobs: []parser.Mob{
			{
				VNum:      1001,
				Keywords:  "test mob",
				ShortDesc: "a test mob",
				LongDesc:  "A test mob stands here.",
				Level:     1,
				HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 0},
				Damage:    parser.DiceRoll{Num: 1, Sides: 4, Plus: 0},
				AC:        10,
				THAC0:     20,
			},
		},
		Objs: []parser.Obj{
			{
				VNum:      2001,
				Keywords:  "test object",
				ShortDesc: "a test object",
				LongDesc:  "A test object lies here.",
				Weight:    1,
				Cost:      10,
			},
		},
		Zones: []parser.Zone{
			{
				Number:    1,
				Name:      "Test Zone",
				TopRoom:   2,
				Lifespan:  15,
				ResetMode: 1,
				Commands: []parser.ZoneCommand{
					{Command: "M", IfFlag: 0, Arg1: 1001, Arg2: 5, Arg3: 1}, // Spawn mob 1001 in room 1, max 5
					{Command: "O", IfFlag: 0, Arg1: 2001, Arg2: 10, Arg3: 1}, // Spawn object 2001 in room 1, max 10
				},
			},
		},
	}

	// Create world
	world, err := game.NewWorld(parsed)
	if err != nil {
		fmt.Printf("Error creating world: %v\n", err)
		return
	}

	fmt.Println("World created successfully")
	fmt.Println(world.Stats())

	// Start zone resets
	err = world.StartZoneResets()
	if err != nil {
		fmt.Printf("Error starting zone resets: %v\n", err)
		return
	}

	fmt.Println("Zone resets started")

	// Start periodic resets (every 30 seconds for testing)
	world.StartPeriodicResets(30 * time.Second)
	fmt.Println("Periodic resets started (every 30 seconds)")

	// Keep the program running for a bit to see periodic resets
	fmt.Println("Waiting for 2 minutes to see periodic resets...")
	time.Sleep(2 * time.Minute)

	fmt.Println("Test completed")
}