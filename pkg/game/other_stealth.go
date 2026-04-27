package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// do_not_here
// ---------------------------------------------------------------------------

func (w *World) doNotHere(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Sorry, but you cannot do that here!\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_sneak — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSneak(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't sneak around on a mount!\r\n")
		return true
	}

	skill := ch.GetSkill("sneak")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to sneak!\r\n")
		return true
	}

	percent := randRange(1, 101)
	if percent > skill+ch.Stats.Dex {
		ch.SendMessage("You try to sneak but fail.\r\n")
		return true
	}

	ch.SetAffect(affSneak, true)
	ch.SendMessage("Okay, you'll try to move silently for a while.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_hide — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doHide(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.IsAffected(affMounted) {
		ch.SendMessage("You can't hide while mounted!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.GetRoomVNum())
	if room != nil && !isOutdoors(room) {
		ch.SendMessage("You can't hide indoors!\r\n")
		return true
	}

	if room != nil && room.Sector == SECT_CITY {
		ch.SendMessage("There's nowhere to hide here!\r\n")
		return true
	}

	skill := ch.GetSkill("hide")
	if skill <= 0 {
		ch.SendMessage("You have no idea how to hide!\r\n")
		return true
	}

	percent := randRange(1, 101)
	if percent > skill+ch.Stats.Dex {
		ch.SendMessage("You try to hide but fail.\r\n")
		return true
	}

	ch.SetAffect(affHide, true)
	ch.SendMessage("You blend into the shadows.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_steal — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSteal(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// Parse arguments: "victim item" or "victim gold"
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		ch.SendMessage("Steal what from whom?\r\n")
		return true
	}

	victimName := parts[0]
	objName := ""
	if len(parts) > 1 {
		objName = parts[1]
	}

	victimPl, victimMob := w.findCharInRoom(ch, ch.GetRoomVNum(), victimName)
	if victimPl == nil && victimMob == nil {
		ch.SendMessage("They aren't here.\r\n")
		return true
	}

	// Determine victim
	var victNPC bool
	if victimPl != nil {
		// Stealing from player
		if victimPl.Level >= LVL_IMMORT {
			ch.SendMessage("You cannot steal from immortals!\r\n")
			return true
		}
		victNPC = false
	} else {
		victNPC = true
	}

	if (ch.Flags&(1<<PlrOutlaw)) != 0 && !victNPC {
		ch.SendMessage("You are an outlaw!  Wait until your crime is forgotten.\r\n")
		return true
	}

	// Check level difference — no stealing from players > 10 levels below
	victLevel := 1
	if victimPl != nil {
		victLevel = victimPl.Level
	}
	if victimMob != nil {
		victLevel = 1 // mobs are always stealable level-wise
	}

	if !victNPC && victLevel > ch.Level/2 {
		ch.SendMessage("You can't steal from someone so high above you.\r\n")
		return true
	}

	ohoh := false

	if objName != "" && !strings.EqualFold(objName, "coins") && !strings.EqualFold(objName, "gold") {
		ch.SendMessage("You can only steal coins for now.\r\n")
		return true
	}

	// Steal gold
	var victGold int
	if victimPl != nil {
		victGold = victimPl.Gold
	} else {
		victGold = 0 // mobs might have gold
	}

	if victGold <= 0 {
		ch.SendMessage("You couldn't get any gold...\r\n")
		ohoh = true
	} else {
		gold := (victGold * randRange(1, 10)) / 100
		if gold > 1782 {
			gold = 1782
		}
		if gold > 0 {
			ch.Gold += gold
			if victimPl != nil {
				victimPl.Gold -= gold
			}

			if gold > 1 {
				msg := fmt.Sprintf("Bingo!  You got %d gold coins.\r\n", gold)
				ch.SendMessage(msg)
				// improve_skill
				skillVal := ch.GetSkill("steal")
				if skillVal > 0 && skillVal < 97 {
					if randRange(1, 200) <= ch.Stats.Wis+ch.Stats.Int {
						inc := randRange(1, 3)
						skillVal += inc
						if skillVal > 97 {
							skillVal = 97
						}
						ch.SetSkill("steal", skillVal)
						if inc == 3 {
							ch.SendMessage("Your skill in steal improves.\r\n")
						}
					}
				}
			} else {
				ch.SendMessage("You manage to swipe a solitary gold coin.\r\n")
			}
		} else {
			ch.SendMessage("You couldn't get any gold...\r\n")
		}
	}

	// If victim is a mob and awake, they hit back
	if ohoh && victimMob != nil && victimMob.GetPosition() > combat.PosSleeping {
		// hit(vict, ch, TYPE_UNDEFINED) — simplified: start combat
// #nosec G104
		victimMob.Attack(ch, w)
	}

	if ohoh && victimPl != nil {
		ch.Flags |= 1 << PlrOutlaw
		ch.SendMessage("You are now an outlaw!\r\n")
	}

	return true
}
