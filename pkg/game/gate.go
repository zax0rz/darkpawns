// Package game — ported from src/gate.c (moongate code).
//
// Moon-phase-gated portals in rooms 4001-4008 (8 moon phases), plus
// permanent gates in rooms 4011-4018 and a red portal spell (gate)
// that can only be cast in moon rooms.
//
// Source: dkarnes 970114 for Dark Pawns.

package game

import (
	"log"
)

// =========================================================================
// Constants from gate.c
// =========================================================================

const (
	gateLoadRoom  = 0
	gateLoadPhase = 1
	gateExitRoom  = 2
	gateExitRoom2 = 3
	gateNumParts  = 4
	gateNumGates  = 16
)

const (
	bluePortalVNum = 4001
	redPortalVNum  = 4002
)

var legalRedRooms = []int{4001, 4002, 4003, 4004, 4005, 4006, 4007, 4008}

var gatePhase = [gateNumGates][4]int{
	{4001, MoonNew, 4004, 4006},
	{4002, MoonQuarterFull, 4005, 4007},
	{4003, MoonHalfFull, 4006, 4008},
	{4004, MoonThreeFull, 4001, 4007},
	{4005, MoonFull, 4002, 4008},
	{4006, MoonQuarterEmpty, 4001, 4003},
	{4007, MoonHalfEmpty, 4002, 4004},
	{4008, MoonThreeEmpty, 4003, 4005},
	{4011, -1, 4001, 0},
	{4012, -1, 4002, 0},
	{4013, -1, 4003, 0},
	{4014, -1, 4004, 0},
	{4015, -1, 4005, 0},
	{4016, -1, 4006, 0},
	{4017, -1, 4007, 0},
	{4018, -1, 4008, 0},
}

// =========================================================================
// addNightGatePortal — creates a blue portal in the given room
// =========================================================================

func addNightGatePortal(w *World, roomVNum int) {
	if w.GetRoomInWorld(roomVNum) == nil {
		return
	}
	proto, ok := w.GetObjPrototype(bluePortalVNum)
	if !ok || proto == nil {
		log.Printf("SYSERR: addNightGatePortal: no obj prototype for vnum %d", bluePortalVNum)
		return
	}
	obj := NewObjectInstance(proto, 0)
	w.AddItemToRoom(obj, roomVNum)
	w.roomMessage(roomVNum, "A shimmering portal of blue light suddenly appears in the darkness!\r\n")
}

// =========================================================================
// removeNightGatePortal — removes blue portal objects from the given room
// =========================================================================

func removeNightGatePortal(w *World, roomVNum int) bool {
	items := w.GetItemsInRoom(roomVNum)
	var found bool
	for _, obj := range items {
		if obj.Prototype != nil && obj.Prototype.VNum == bluePortalVNum {
			if w.RemoveItemFromRoom(obj, roomVNum) {
				found = true
			}
		}
	}
	if found {
		w.roomMessage(roomVNum, "The shimmering blue portal of light fades out of existence.\r\n")
	}
	return found
}
