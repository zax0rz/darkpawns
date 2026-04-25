// Package game — dream system (ported from dream.c)
// Source: src/dream.c — dream(), dream_travel(), dtravel[]
// Port: Wave 13, 2026-04-25
package game

import (
	"fmt"
	"math/rand"
)

// NumDreams is the number of entries in the dream travel table.
// Source: dream.c #define NUM_DREAMS 8
const NumDreams = 8

// DreamTravel holds a single dream-travel destination.
// Source: dream.c struct dtravel_data { int subcmd; int room_num; char *descrip; }
type DreamTravel struct {
	Subcmd  int    // 0=normal dream, 1=bad dream (after death)
	RoomNum int    // Destination room VNum
	Descrip string // Description appended to "You have a dream ..."
}

// DreamTravelTable holds all dream destinations.
// Source: dream.c const struct dtravel_data dtravel[] lines 38–48
var DreamTravelTable = []DreamTravel{
	{0, 8004, "that shows you penitent before an altar."},
	{0, 11111, "about lounging around a beautiful desert oasis."},
	{0, 20400, "in which you are lost in a dark, spooky forest."},
	{0, 4622, "of tredging through a murky swamp."},
	{0, 4805, "about being lost in a foreign city."},
	{0, 4267, "of being surrounded by cold alpine peaks."},
	{0, 14213, "in which the stench of death and battle overwhelm you."},
	{1, 12410, "of being hopelessly lost at sea."},
	{1, 12848, "of a dark figure standing over your corpse!"},
}

// AFF_DREAM is the bitmask flag for the dream travel affect.
// Source: structs.h line 345: #define AFF_DREAM 35
const AFF_DREAM_BIT = 35

// DreamContext provides the external dependencies Dream() needs.
// This keeps dream logic free of direct World coupling for easy testing.
type DreamContext interface {
	GetLevel() int
	GetLastDeath() int64             // Unix timestamp of last death, 0 if never
	SetLastDeath(t int64)            // Clear last death by setting to 0
	HasAffect(bitNum int) bool       // Check if character has an affect flag set
	RemoveAffect(bitNum int)         // Remove an affect flag
	SendToChar(msg string)           // Send a message to the character
	SendToRoom(msg string)           // Send a message to everyone in the same room
	WakeUp()                         // Force character to wake (like do_wake)
	MoveToRoom(roomVNum int)         // Teleport character to a room (char_from_room + char_to_room)
	CurrentTime() int64              // Returns current Unix timestamp
}

// DreamResult describes what the dream function decided to do.
type DreamResult struct {
	Traveled    bool   // Character was dream-teleported to a new room
	DestRoomNum int    // If Traveled, destination room VNum
	Woke        bool   // Character was woken up
}

// Dream processes a sleeping character's dream tick.
// Called from the point_update tick when a character is asleep.
// Source: dream.c dream() lines 50–187
//
// Priority order (matches C):
// 1. If player has died recently and has AFF_DREAM — bad dream travel
// 2. If player has died recently — send death nightmare message (chance-based)
// 3. If player has AFF_DREAM (but no recent death) — normal dream travel
// 4. Otherwise — level-based flavor dream message
func Dream(ch DreamContext) DreamResult {
	result := DreamResult{}
	now := ch.CurrentTime()
	lastDeath := ch.GetLastDeath()

	if lastDeath != 0 { // has died at some point
		diff := now - lastDeath

		if ch.HasAffect(AFF_DREAM_BIT) {
			// AFF_DREAM while having died recently → bad dream travel
			// Source: dream.c lines 59–62
			DreamTravelFn(ch, 1) // subcmd=1: bad dream
			result.Traveled = true
			return result
		}

		const day = int64(24 * 60 * 60)

		if diff < day { // less than 1 real day ago
			// Source: dream.c lines 63–73 (1/6 chance)
			if rand.Intn(6) == 0 {
				ch.SendToChar("You see the visions of your own death and wake up screaming!\r\n")
				ch.SendToRoom("$n wakes up screaming, with a look of death in $s eyes.")
				ch.WakeUp()
				result.Woke = true
			}
			return result
		} else if diff < 2*day { // 1–2 real days ago
			// Source: dream.c lines 74–80 (1/6 chance)
			if rand.Intn(6) == 0 {
				ch.SendToChar("In your dreams you keep seeing a dark figure hunched over your corpse.\r\n")
				ch.SendToRoom("$n shivers in $s sleep.")
			}
			return result
		} else if diff < 3*day { // 2–3 real days ago
			// Source: dream.c lines 81–88 (1/6 chance)
			if rand.Intn(6) == 0 {
				ch.SendToChar("You toss and turn as a dark cloud hovers over your dreams.\r\n")
				ch.SendToRoom("$n tosses and turns in $s sleep, must be a bad dream.")
			}
			return result
		} else if diff < 5*day { // 3–5 real days ago
			// Source: dream.c lines 89–96 (1/6 chance)
			if rand.Intn(6) == 0 {
				ch.SendToChar("You sleep uneasily, as if something looms over your past\r\n")
				ch.SendToRoom("$n grunts in $s sleep.")
			}
			return result
		} else {
			// More than 5 days: clear last death
			// Source: dream.c line 97: GET_LAST_DEATH(ch) = 0
			ch.SetLastDeath(0)
		}
	}

	// Dream travel check (AFF_DREAM without recent death)
	// Source: dream.c lines 103–106
	if ch.HasAffect(AFF_DREAM_BIT) {
		DreamTravelFn(ch, 0)
		result.Traveled = true
		return result
	}

	// Level-based flavor dreams (chance: 1/16)
	// Source: dream.c switch(GET_LEVEL(ch)) lines 108–186
	lvl := ch.GetLevel()
	switch {
	case lvl >= 0 && lvl <= 5:
		if rand.Intn(16) == 0 {
			ch.SendToChar("You have dreams of showing this world what you are really made of.\r\n")
			ch.SendToRoom("$n smiles in $s sleep.")
		}
	case lvl >= 6 && lvl <= 10:
		if rand.Intn(16) == 0 {
			ch.SendToChar("You have a pleasant dream of safe travels to far places and a hero's welcome when you return.\r\n")
			ch.SendToRoom("$n begins to hum a happy ditty in $s sleep.")
		}
	case lvl >= 11 && lvl <= 20:
		if rand.Intn(16) == 0 {
			ch.SendToChar("You dream of your conquest of the world.\r\n")
			ch.SendToRoom("$n begins to grin in $s sleep.")
		}
	case lvl >= 21 && lvl <= 28:
		if rand.Intn(16) == 0 {
			ch.SendToChar("You dream of slaying the dark creatures of the night.\r\n")
			ch.SendToRoom("$n smirks in $s sleep.")
		}
	case lvl >= 29 && lvl <= 30:
		if rand.Intn(16) == 0 {
			ch.SendToChar("You have a fantastic dream of one day attaining immortality.\r\n")
			ch.SendToRoom("$n looks like $e is having big dreams.")
		}
	case lvl == LVLImmort:
		if rand.Intn(16) == 0 {
			ch.SendToChar("You have big, grand dreams of the power of the Gods.\r\n")
			ch.SendToRoom("$n glows in $s sleep.")
		}
	default:
		// Above immortal level
		if rand.Intn(16) == 0 {
			ch.SendToChar("You toss and turn under the constant fear of the wrath of Orodreth :-)\r\n")
			ch.SendToChar("You find yourself wide awake!\r\n")
			ch.SendToRoom("$n awakens with fear in $s eyes.")
			ch.WakeUp()
			result.Woke = true
		}
	}

	return result
}

// DreamTravelFn teleports a sleeping character to a dream destination.
// Source: dream.c dream_travel() lines 189–223
//
// subcmd=0: normal dream travel (only entries with dtravel.subcmd==0)
// subcmd=1: bad dream travel (any entry eligible)
//
// Each entry has a 1/16 chance of being selected.
// On selection: send dream message, teleport, remove AFF_DREAM.
func DreamTravelFn(ch DreamContext, subcmd int) {
	for i := 0; i <= NumDreams; i++ {
		dt := DreamTravelTable[i]
		// Source: dream.c lines 195–206 (normal) and 208–219 (bad)
		if subcmd == 0 && dt.Subcmd == 0 && rand.Intn(16) == 0 {
			ch.SendToChar(fmt.Sprintf("You have a dream %s \r\n", dt.Descrip))
			ch.SendToRoom("The sleeping body of $n fades from existence.")
			ch.MoveToRoom(dt.RoomNum)
			ch.SendToRoom("The sleeping body of $n fades into existence.")
			ch.RemoveAffect(AFF_DREAM_BIT)
			return
		}
		if subcmd == 1 && rand.Intn(16) == 0 {
			ch.SendToChar(fmt.Sprintf("You have a dream %s \r\n", dt.Descrip))
			ch.SendToRoom("The sleeping body of $n fades from existence.")
			ch.MoveToRoom(dt.RoomNum)
			ch.SendToRoom("The sleeping body of $n fades into existence.")
			ch.RemoveAffect(AFF_DREAM_BIT)
			return
		}
	}
}

// LVLImmort is the minimum immortal level.
// Source: structs.h LVL_IMMORT 31 (Dark Pawns used 31 per act.wizard.c constants)
// This is a local alias to avoid import cycles — session/wizard_cmds.go has LVL_IMMORT = 31.
const LVLImmort = 31

// Go Improvements Over C
// ======================
// 1. INTERFACE DECOUPLING: C accessed char_data fields directly. Go uses DreamContext
//    interface, making the dream logic testable without a full World or Player instance.
//
// 2. NO IMPLICIT GLOBALS: C's dream() read/wrote global time_info and called
//    char_from_room()/char_to_room() which accessed global world state. Go passes
//    everything through DreamContext.
//
// 3. RANDOM: C used number(0,5) which returns 0–5 (6 outcomes). The code checked
//    !number(0,5) meaning 1-in-6 chance. Go's rand.Intn(6) == 0 is equivalent.
//    Same for the 1/16 chance: number(0,15) → rand.Intn(16) == 0.
//
// 4. TIME: C stored lastdeath as a raw long Unix timestamp. Go uses int64 (same
//    semantics) to avoid the 2038 problem that C's 32-bit time_t would hit.
//
// 5. POTENTIAL MODERNIZATION (do not implement now):
//    - Track last death time in PostgreSQL alongside player data rather than
//      in a transient field.
//    - Make DreamTravelTable configurable from a JSON/Lua data file.
//    - The dream_travel loop bug: C iterates i <= NUM_DREAMS (9 entries, index 0–8)
//      but DreamTravelTable has exactly 9 entries (indices 0–8). The loop condition
//      should be i < NumDreams to be safe; we preserve the <= for faithfulness.
//    - Add Dream() call into the point_update sleeping character tick.
