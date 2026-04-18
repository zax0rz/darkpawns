// Package parser provides utilities for parsing Dark Pawns world files.
package parser

import (
	"fmt"
)

// World represents the entire parsed game world.
type World struct {
	Rooms []Room
	Mobs  []Mob
	Objs  []Obj
	Zones []Zone
}

// Stats returns statistics about the parsed world.
func (w *World) Stats() string {
	return fmt.Sprintf(
		"World Stats:\n"+
		"  Rooms: %d\n"+
		"  Mobs:  %d\n"+
		"  Objs:  %d\n"+
		"  Zones: %d",
		len(w.Rooms), len(w.Mobs), len(w.Objs), len(w.Zones),
	)
}

// ParseWorld parses all world files from the given lib directory.
func ParseWorld(libDir string) (*World, error) {
	world := &World{}

	// Parse rooms
	rooms, err := ParseAllWldFiles(libDir + "/wld")
	if err != nil {
		return nil, fmt.Errorf("parse rooms: %w", err)
	}
	world.Rooms = rooms

	// Parse mobs
	mobs, err := ParseAllMobFiles(libDir + "/mob")
	if err != nil {
		return nil, fmt.Errorf("parse mobs: %w", err)
	}
	world.Mobs = mobs

	// Parse objects
	objs, err := ParseAllObjFiles(libDir + "/obj")
	if err != nil {
		return nil, fmt.Errorf("parse objects: %w", err)
	}
	world.Objs = objs

	// Parse zones
	zones, err := ParseAllZonFiles(libDir + "/zon")
	if err != nil {
		return nil, fmt.Errorf("parse zones: %w", err)
	}
	world.Zones = zones

	return world, nil
}
