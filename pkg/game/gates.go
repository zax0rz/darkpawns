//nolint:unused // Game logic port — not yet wired to command registry.
// gates.go — Ported from src/gate.c
//
// Moongate system: blue gates that appear at night based on moon phase,
// permanent blue gates, red portal spell gate, and moon gate spec_proc.

package game

import (
	"log/slog"
)

const (
	BluePortalVNum = 4001
	RedPortalVNum  = 4002
)

// gatePhase entry: [room, phase, exit1, exit2]
type gateEntry struct {
	Room     int
	Phase    int
	Exit1    int
	Exit2    int
}

var gatePhases = []gateEntry{
	{4001, MoonNew,          4004, 4006},
	{4002, MoonQuarterFull, 4005, 4007},
	{4003, MoonHalfFull,    4006, 4008},
	{4004, MoonThreeFull,   4001, 4007},
	{4005, MoonFull,        4002, 4008},
	{4006, MoonQuarterEmpty,4001, 4003},
	{4007, MoonHalfEmpty,   4002, 4004},
	{4008, MoonThreeEmpty,  4003, 4005},
	{4011, -1,               4001, 0},
	{4012, -1,               4002, 0},
	{4013, -1,               4003, 0},
	{4014, -1,               4004, 0},
	{4015, -1,               4005, 0},
	{4016, -1,               4006, 0},
	{4017, -1,               4007, 0},
	{4018, -1,               4008, 0},
}

const numGates = 16 //nolint:unused // gate count constant

// Blue portal room VNums that allow the red gate spell
var legalRedRooms = []int{4001, 4002, 4003, 4004, 4005, 4006, 4007, 4008}

// ---------------------------------------------------------------------------
// LoadNightGate — port of load_night_gate()
// Spawn blue portals in rooms matching the current moon phase.
// ---------------------------------------------------------------------------

func (w *World) LoadNightGate(moonPhase int) {
	for _, ge := range gatePhases {
		if ge.Phase == -1 {
			continue // permanent gates, always present
		}
		if moonPhase == ge.Phase {
			rnum := w.RealRoom(ge.Room)
			if rnum < 0 {
				continue
			}
			gate := w.CreateObject(BluePortalVNum, rnum)
			if gate != nil {
				gate.SetTimer(1) // will be checked in point_update
				w.SendToRoom(rnum, "A shimmering portal of blue light suddenly appears in the darkness!\r\n")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RemoveNightGate — port of remove_night_gate()
// Remove all blue portals from phase-matching rooms.
// ---------------------------------------------------------------------------

func (w *World) RemoveNightGate(moonPhase int) {
	for _, ge := range gatePhases {
		if ge.Phase == -1 {
			continue
		}
		rnum := w.RealRoom(ge.Room)
		if rnum < 0 {
			continue
		}
		objs := w.GetItemsInRoom(rnum)
		for _, obj := range objs {
			if obj.GetVNum() == BluePortalVNum {
				if err := w.MoveObjectToNowhere(obj); err != nil {
					slog.Warn("MoveObjectToNowhere failed in gate expiration", "obj_vnum", obj.GetVNum(), "error", err)
				}
				w.SendToRoom(rnum, "The shimmering blue portal of light fades out of existence.\r\n")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// SpellGate — port of spell_gate()
// Cast the Gate spell: creates red portal in blue-moongate rooms, or disaster
// if cast where a portal already exists.
// ---------------------------------------------------------------------------

func (w *World) SpellGate(caster *Player) bool {
	casterRoom := caster.GetRoom()

	// Check for existing portal in room
	for _, obj := range w.GetItemsInRoom(casterRoom) {
		vnum := obj.GetVNum()
		if vnum == BluePortalVNum || vnum == RedPortalVNum {
			// Portal collision — disaster!
			caster.SendMessage("The magick flows through you, then out into the world, changing it....\r\n")
			w.SendToRoom(casterRoom, "As you watch the red portal slowly fade into existence,\r\n"+
				"the existing portal pulses once, then begins to expand,\r\n"+
				"consuming the new portal, and then the entire grove.\r\n"+
				"The fabric of time and space warps and stretches\r\n"+
				"around you, and flashes of white light explode in the back of your mind.\r\n")
			caster.SendMessage("In your final moments, the only thing you can feel is a\r\n" +
				"wave of cosmic energy coursing through you, tearing your soul to shreds.\r\n")
			w.RawKill(caster, "blast")
			if err := w.MoveObjectToNowhere(obj); err != nil {
				slog.Warn("MoveObjectToNowhere failed in gate collision", "obj_vnum", obj.GetVNum(), "error", err)
			}
			return true
		}
	}

	// Check if caster is in a legal room
	for _, legalRoom := range legalRedRooms {
		rnum := w.RealRoom(legalRoom)
		if rnum == casterRoom {
			redGate := w.CreateObject(RedPortalVNum, casterRoom)
			if redGate != nil {
				timer := 2
				if caster.GetLevel() >= 30 {
					timer++
				}
				redGate.SetTimer(timer)
				caster.SendMessage("The magick flows through you, then out into the world, changing it....\r\n")
				w.SendToRoom(casterRoom, "A shimmering red portal fades into existence.\r\n")
			}
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// Helper methods used by gate functions

// RealRoom converts a room VNum to rnum. Currently identity (vnum == rnum).
func (w *World) RealRoom(vnum int) int {
	return vnum
}

// SendToRoom sends a message to all players in a room.
func (w *World) SendToRoom(rnum int, msg string) {
	w.roomMessage(rnum, msg)
}

// RawKill kills a player.
func (w *World) RawKill(ch *Player, attackType string) {
	at := 0
	switch attackType {
	case "slash", "slice":
		at = 1
	case "stab", "pierce":
		at = 2
	case "bludgeon", "bash", "crush":
		at = 3
	case "hit", "pound":
		at = 4
	}
	w.rawKill(ch, at)
}


