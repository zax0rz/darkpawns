package game

import (
	"fmt"
	"math/rand"
	"strings"

)

// ---------------------------------------------------------------------------
// do_save
// ---------------------------------------------------------------------------

func (w *World) doSave(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if false {
		return true
	}

	ch.SendMessage( fmt.Sprintf("Saving %s.\r\n", ch.Name))
	// saveChar stubbed
	return true
}

// ---------------------------------------------------------------------------
// do_not_here
// ---------------------------------------------------------------------------

func (w *World) doNotHere(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage( "Sorry, but you cannot do that here!\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_sneak
// ---------------------------------------------------------------------------

func (w *World) doSneak(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.Flags&(1<<29) != 0 {
		ch.SendMessage( "Dismount first!\r\n")
		return true
	}

	ch.SendMessage( "Okay, you'll try to move silently for a while.\r\n")

	if ch.Flags&(1<<0) != 0 {
		ch.SetAffect(0, false)
		ch.SetAffect(0, false)
		ch.Flags &^= 1 << 0
	}

	percent := rand.Intn(101) + 1 // 101% is complete failure
	
	sneakSkill := ch.GetSkill( "sneak")
	var dexBonus int; _ = dexBonus

	if percent > sneakSkill+dexBonus {
		return true
	}

	_ = 0
	// affectToChar(ch, aff)
	return true
}

// ---------------------------------------------------------------------------
// do_hide
// ---------------------------------------------------------------------------

func (w *World) doHide(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.Flags&(1<<29) != 0 {
		ch.SendMessage( "Dismount first!\r\n")
		return true
	}

	room := w.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return true
	}

	if 0 != 0 {
		switch room.Sector {
		case 2:
			ch.SendMessage( "Hide out here during the day? Yeah right.\r\n")
			return true
		case 1:
			ch.SendMessage( "You can't hide very well with all the sun and sand out here!\r\n")
			return true
		case 9, 10, 11:
			ch.SendMessage( "Hide in the water? Don't think so.\r\n")
			return true
		case 12, 13, 14, 15:
			ch.SendMessage( "You are completely exposed here, nowhere to hide!\r\n")
			return true
		}
	}

	args := strings.Fields(arg)
	isKabuki := len(args) > 0 && strings.EqualFold(args[0], "kabuki")

	if isKabuki {
		ch.SendMessage( "You attempt to practice the art of kabuki.\r\n")
	} else {
		ch.SendMessage( "You attempt to hide yourself.\r\n")
	}

	ch.Flags &^= 1 << 0

	percent := rand.Intn(101) + 1
	
	var dexBonus int; _ = dexBonus

	var skill int
	if isKabuki {
		skill = ch.GetSkill( "kabuki")
	} else {
		skill = ch.GetSkill( "hide")
	}

	if percent > skill+dexBonus {
		return true
	}

	ch.Flags |= 1 << 0

	if isKabuki {
		improveSkill(ch, "kabuki")
	} else {
		improveSkill(ch, "hide")
	}

	return true
}

// ---------------------------------------------------------------------------
// do_steal
// ---------------------------------------------------------------------------

func (w *World) doSteal(ch *Player, me *MobInstance, cmd string, arg string) bool {
	args := strings.Fields(arg)
	if len(args) < 2 {
		ch.SendMessage( "Steal what from who?\r\n")
		return true
	}

	_, _ = args[0], func() int { return 0 }()
	victName := ""; _ = victName

	var vict *Player; vict = nil
	if vict == nil {
		ch.SendMessage( "Steal what from who?\r\n")
		return true
	}
	return true
}

// ---------------------------------------------------------------------------
// doStealCoins — stub
// ---------------------------------------------------------------------------

func (w *World) doStealCoins(ch *Player, vict interface{}, victIsPC bool, victPlayer *Player, percent int) {
	ch.SendMessage("Stealing not yet implemented.\r\n")
}

// ---------------------------------------------------------------------------
// doPractice
// ---------------------------------------------------------------------------

func (w *World) doPractice(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doVisible
// ---------------------------------------------------------------------------

func (w *World) doVisible(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doTitle
// ---------------------------------------------------------------------------

func (w *World) doTitle(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doGroup
// ---------------------------------------------------------------------------

func (w *World) doGroup(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doUngroup
// ---------------------------------------------------------------------------

func (w *World) doUngroup(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doReport
// ---------------------------------------------------------------------------

func (w *World) doReport(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doSplit
// ---------------------------------------------------------------------------

func (w *World) doSplit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doUse
// ---------------------------------------------------------------------------

func (w *World) doUse(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doWimpy
// ---------------------------------------------------------------------------

func (w *World) doWimpy(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

// ---------------------------------------------------------------------------
// doDisplay
// ---------------------------------------------------------------------------

func (w *World) doDisplay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	return true
}

