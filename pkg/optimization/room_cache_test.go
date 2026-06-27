package optimization

import (
	"testing"
	"time"
)

func TestRoomCache_GetRoomReturnsIndependentCopy(t *testing.T) {
	cache := NewRoomCache(time.Hour)

	room := &CachedRoom{
		VNum:    100,
		Name:    "Town Square",
		Players: []string{"Alice"},
		Exits: []ExitData{
			{Direction: "north", ToRoom: 101, Flags: []string{"door"}},
		},
	}

	fetched, err := cache.GetRoom(100, func(vnum int) (*CachedRoom, error) {
		return room, nil
	})
	if err != nil {
		t.Fatalf("GetRoom failed: %v", err)
	}

	// Mutate the returned copy.
	fetched.Name = "Mutated"
	fetched.Players = append(fetched.Players, "Bob")
	fetched.Exits[0].Flags[0] = "secret"

	// Fetch again and verify cache retained original values.
	again, err := cache.GetRoom(100, func(vnum int) (*CachedRoom, error) {
		t.Fatal("fetchFunc should not be called for cached room")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("second GetRoom failed: %v", err)
	}

	if again.Name != "Town Square" {
		t.Errorf("name = %q, want %q", again.Name, "Town Square")
	}
	if len(again.Players) != 1 || again.Players[0] != "Alice" {
		t.Errorf("players = %v, want [Alice]", again.Players)
	}
	if len(again.Exits) != 1 || len(again.Exits[0].Flags) != 1 || again.Exits[0].Flags[0] != "door" {
		t.Errorf("exit flags = %v, want [[door]]", again.Exits[0].Flags)
	}
}

func TestRoomCache_UpdateRoomStoresCopy(t *testing.T) {
	cache := NewRoomCache(time.Hour)

	room := &CachedRoom{
		VNum:    200,
		Name:    "Forest",
		Players: []string{"Carol"},
	}
	cache.UpdateRoom(room)

	// Mutate the original pointer after giving it to the cache.
	room.Name = "Mutated"
	room.Players[0] = "Dave"

	fetched, err := cache.GetRoom(200, func(vnum int) (*CachedRoom, error) {
		t.Fatal("fetchFunc should not be called")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("GetRoom failed: %v", err)
	}

	if fetched.Name != "Forest" {
		t.Errorf("name = %q, want %q", fetched.Name, "Forest")
	}
	if len(fetched.Players) != 1 || fetched.Players[0] != "Carol" {
		t.Errorf("players = %v, want [Carol]", fetched.Players)
	}
}

func TestRoomCache_CloneIsDeep(t *testing.T) {
	original := &CachedRoom{
		VNum:    300,
		Players: []string{"Eve"},
		Mobs:    []MobData{{ID: 1, Name: "rat"}},
		Items:   []ItemData{{ID: 1, Name: "sword"}},
		Exits:   []ExitData{{Direction: "up", Flags: []string{"climb"}}},
	}

	clone := original.Clone()
	clone.Players[0] = "Frank"
	clone.Mobs[0].Name = "dragon"
	clone.Items[0].Name = "shield"
	clone.Exits[0].Flags[0] = "locked"

	if original.Players[0] != "Eve" {
		t.Errorf("original player mutated: %v", original.Players)
	}
	if original.Mobs[0].Name != "rat" {
		t.Errorf("original mob mutated: %v", original.Mobs[0])
	}
	if original.Items[0].Name != "sword" {
		t.Errorf("original item mutated: %v", original.Items[0])
	}
	if original.Exits[0].Flags[0] != "climb" {
		t.Errorf("original exit flags mutated: %v", original.Exits[0].Flags)
	}
}
