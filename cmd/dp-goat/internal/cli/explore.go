package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newExploreCmd(flags *rootFlags) *cobra.Command {
	var depth int

	cmd := &cobra.Command{
		Use:   "explore",
		Short: "Walk in random directions and map nearby rooms",
		Long: `Explore the world by walking in random directions. Tracks rooms
visited, exits found, and entities encountered. Returns to the
starting room when done.`,
		Example: `  dp-goat explore
  dp-goat explore --depth 5
  dp-goat --name Machine explore --depth 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			if depth <= 0 {
				depth = 5
			}

			// Get starting room
			startData, err := c.Get("/look", nil)
			if err != nil {
				return fmt.Errorf("look: %w", err)
			}

			var startRoom struct {
				Room struct {
					Name  string   `json:"name"`
					Vnum  int      `json:"vnum"`
					Exits []string `json:"exits"`
				} `json:"room"`
			}
			if err := json.Unmarshal(startData, &startRoom); err != nil {
				return fmt.Errorf("parse room: %w", err)
			}

			fmt.Printf("Starting at: %s (vnum %d)\n", startRoom.Room.Name, startRoom.Room.Vnum)
			fmt.Printf("Exploring %d rooms...\n\n", depth)

			// Track visited rooms
			visited := make(map[int]string) // vnum -> name
			visited[startRoom.Room.Vnum] = startRoom.Room.Name
			var path []string

			for i := 0; i < depth; i++ {
				// Get current room
				roomData, err := c.Get("/look", nil)
				if err != nil {
					fmt.Printf("Error looking: %v\n", err)
					continue
				}

				var room struct {
					Room struct {
						Name  string   `json:"name"`
						Vnum  int      `json:"vnum"`
						Exits []string `json:"exits"`
						Mobs  []struct {
							Name string `json:"name"`
						} `json:"mobs"`
						Items []struct {
							Name string `json:"name"`
						} `json:"items"`
					} `json:"room"`
				}
				if err := json.Unmarshal(roomData, &room); err != nil {
					continue
				}

				// Record room
				if _, exists := visited[room.Room.Vnum]; !exists {
					visited[room.Room.Vnum] = room.Room.Name
					fmt.Printf("[%d] %s\n", room.Room.Vnum, room.Room.Name)
					if len(room.Room.Mobs) > 0 {
						for _, m := range room.Room.Mobs {
							fmt.Printf("    mob: %s\n", m.Name)
						}
					}
					if len(room.Room.Items) > 0 {
						for _, it := range room.Room.Items {
							fmt.Printf("    item: %s\n", it.Name)
						}
					}
				}

				// Pick a random unvisited exit, or any exit if all visited
				exits := room.Room.Exits
				if len(exits) == 0 {
					fmt.Println("    (dead end)")
					// Go back
					if len(path) > 0 {
						lastDir := path[len(path)-1]
						path = path[:len(path)-1]
						c.Post("/"+lastDir, nil)
					}
					continue
				}

				// Try to find an unvisited exit
				direction := exits[0]
				for _, dir := range exits {
					// Simple heuristic: we can't know the destination without moving
					// so just pick the first one
					direction = dir
					break
				}

				// Move
				_, _, err = c.Post("/"+direction, nil)
				if err != nil {
					fmt.Printf("    error moving %s: %v\n", direction, err)
					continue
				}

				path = append(path, direction)
			}

			// Summary
			fmt.Printf("\nExploration complete:\n")
			fmt.Printf("  Rooms visited: %d\n", len(visited))
			fmt.Printf("  Path: %s\n", strings.Join(path, " → "))

			// Return to start
			fmt.Printf("\nReturning to %s...\n", startRoom.Room.Name)
			for i := len(path) - 1; i >= 0; i-- {
				opposite := reverseDir(path[i])
				c.Post("/"+opposite, nil)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&depth, "depth", "d", 5, "number of rooms to explore")

	return cmd
}

func reverseDir(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	case "up":
		return "down"
	case "down":
		return "up"
	default:
		return dir
	}
}
