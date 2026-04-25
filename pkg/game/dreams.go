// Port of dream.c — dream processing and dream travel
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// This includes all original code done for Dark Pawns MUD by other authors.
// All code is the intellectual property of the author, and is used here
// by permission.
//
// No original code may be duplicated, reused, or executed without the
// written permission of the author. All rights reserved.
//
// See dp-team.txt or "help coding" online for members of the Dark Pawns
// Coding Team.

package game

import (
	"fmt"
	"math/rand"
	"time"
)

// NumDreams is the number of dream travel destinations (8 defined entries,
// but NUM_DREAMS is 8 for the loop in the original — note there are 9 entries).
const NumDreams = 8

// AFF_DREAM is the affect bit for dream travel.
// Source: dream.c, structs.h:345 — AFF_DREAM = 35
const AFF_DREAM = 35

// DtravelData represents a dream travel destination.
// Ported from dream.h:struct dtravel_data.
type DtravelData struct {
	SubCmd  int
	RoomNum int
	Descrip string
}

// dtravel is the dream travel destinations table.
// Ported from dream.c:const struct dtravel_data dtravel[].
var dtravel = []DtravelData{
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

// DreamResult describes the outcome of ProcessDream.
type DreamResult struct {
	PlayerMessage string // message to send to the player
	RoomMessage   string // message to send to the room (act())
	WakeUp        bool   // whether the player should be woken up
	TravelRoom    int    // destination room vnum (-1 if no travel)
	Travel        bool   // whether dream travel occurred
}

// ProcessDream generates dream messages for a sleeping player.
// Ported from dream.c:dream().
func ProcessDream(player *Player, lastDeathUnix int64) *DreamResult {
	if lastDeathUnix != 0 {
		// Has died recently
		if player.IsAffected(AFF_DREAM) {
			return DreamTravel(player, true)
		}

		now := time.Now().Unix()
		diff := now - lastDeathUnix

		if diff < (24 * 60 * 60) { // one real day
			if rand.Intn(6) == 0 { // 1-in-6 chance
				return &DreamResult{
					PlayerMessage: "You see the visions of your own death and wake up screaming!\r\n",
					RoomMessage: fmt.Sprintf("%s wakes up screaming, with a look of death in %s eyes.\r\n",
						player.Name, hisHer(player.GetSex())),
					WakeUp: true,
				}
			}
			return nil
		} else if diff < (2 * 24 * 60 * 60) { // two real days
			if rand.Intn(6) == 0 {
				return &DreamResult{
					PlayerMessage: "In your dreams you keep seeing a dark figure hunched over your corpse.\r\n",
					RoomMessage: fmt.Sprintf("%s shivers in %s sleep.\r\n",
						player.Name, hisHer(player.GetSex())),
				}
			}
			return nil
		} else if diff < (3 * 24 * 60 * 60) { // three real days
			if rand.Intn(6) == 0 {
				return &DreamResult{
					PlayerMessage: "You toss and turn as a dark cloud hovers over your dreams.\r\n",
					RoomMessage: fmt.Sprintf("%s tosses and turns in %s sleep, must be a bad dream.\r\n",
						player.Name, hisHer(player.GetSex())),
				}
			}
			return nil
		} else if diff < (5 * 24 * 60 * 60) { // five real days
			if rand.Intn(6) == 0 {
				return &DreamResult{
					PlayerMessage: "You sleep uneasily, as if something looms over your past.\r\n",
					RoomMessage: fmt.Sprintf("%s grunts in %s sleep.\r\n",
						player.Name, hisHer(player.GetSex())),
				}
			}
			return nil
		} else {
			// Too long ago, clear last death
			player.LastDeath = 0
		}
	}

	// Dream travel check
	if player.IsAffected(AFF_DREAM) {
		return DreamTravel(player, false)
	}

	// Level-based dreams
	lvl := player.Level
	switch {
	case lvl >= 0 && lvl <= 5:
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You have dreams of showing this world what you are really made of.\r\n",
				RoomMessage:   fmt.Sprintf("%s smiles in %s sleep.\r\n", player.Name, hisHer(player.GetSex())),
			}
		}
	case lvl >= 6 && lvl <= 10:
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You have a pleasant dream of safe travels to far places and a hero's welcome when you return.\r\n",
				RoomMessage:   fmt.Sprintf("%s begins to hum a happy ditty in %s sleep.\r\n", player.Name, hisHer(player.GetSex())),
			}
		}
	case lvl >= 11 && lvl <= 20:
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You dream of your conquest of the world.\r\n",
				RoomMessage:   fmt.Sprintf("%s begins to grin in %s sleep.\r\n", player.Name, hisHer(player.GetSex())),
			}
		}
	case lvl >= 21 && lvl <= 28:
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You dream of slaying the dark creatures of the night.\r\n",
				RoomMessage:   fmt.Sprintf("%s smirks in %s sleep.\r\n", player.Name, hisHer(player.GetSex())),
			}
		}
	case lvl >= 29 && lvl < LVL_IMMORT:
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You have a fantastic dream of one day attaining immortality.\r\n",
				RoomMessage:   fmt.Sprintf("%s looks like %s is having big dreams.\r\n", player.Name, heShe(player.GetSex())),
			}
		}
	case lvl == LVL_IMMORT:
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You have big, grand dreams of the power of the Gods.\r\n",
				RoomMessage:   fmt.Sprintf("%s glows in %s sleep.\r\n", player.Name, hisHer(player.GetSex())),
			}
		}
	default: // beyond immort
		if rand.Intn(16) == 0 {
			return &DreamResult{
				PlayerMessage: "You toss and turn under the constant fear of the wrath of Orodreth :-)\r\nYou find yourself wide awake!\r\n",
				RoomMessage:   fmt.Sprintf("%s awakens with fear in %s eyes.\r\n", player.Name, hisHer(player.GetSex())),
				WakeUp:        true,
			}
		}
	}

	return nil
}

// DreamTravel picks a random dream destination from the dtravel table.
// Ported from dream.c:dream_travel().
func DreamTravel(player *Player, isBad bool) *DreamResult {
	for i := range dtravel {
		dt := &dtravel[i]
		if rand.Intn(16) == 0 && !isBad && dt.SubCmd == 0 {
			return &DreamResult{
				PlayerMessage: fmt.Sprintf("You have a dream %s\r\n", dt.Descrip),
				RoomMessage:   "",
				Travel:        true,
				TravelRoom:    dt.RoomNum,
			}
		}
		if rand.Intn(16) == 0 && isBad {
			return &DreamResult{
				PlayerMessage: fmt.Sprintf("You have a dream %s\r\n", dt.Descrip),
				RoomMessage:   "",
				Travel:        true,
				TravelRoom:    dt.RoomNum,
			}
		}
	}
	return nil
}
