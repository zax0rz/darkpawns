package game

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// do_visible — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doVisible(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if ch.HasSpellAffect(skillNumKkZai) {
		ch.SendMessage("You cannot become visible until your zai ends!\r\n")
		return true
	}

	// Immort visibility
	if ch.GetLevel() >= LVL_IMMORT {
		w.performImmortVis(ch)
		return true
	}

	altered := false
	if ch.IsAffected(affInvisible) {
		ch.SetAffect(affInvisible, false)
		ch.SendMessage("You fade into view.\r\n")
		altered = true
	}
	if ch.IsAffected(affSneak) {
		ch.SendMessage("You stop sneaking.\r\n")
		// C affect_from_char() removes both the ordinary sneak and ninja
		// stealth records, not just the aggregate AFF_SNEAK bit.
		ch.RemoveAffectBySpell(skillNumSneak)
		ch.RemoveAffectBySpell(skillNumStealth)
		ch.SetAffect(affSneak, false)
		altered = true
	}
	if !altered {
		ch.SendMessage("You are already visible.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_report — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doReport(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	w.actToRoom(ch, "$n reports:\r\n", nil, nil)
	ch.SendMessage("You report:\r\n")

	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		msg := fmt.Sprintf("    [%d/%d]H [%d/%d]M [%d/%d]V [%d]Kills [%d]PKs [%d]Deaths\r\n",
			ch.GetHP(), ch.GetMaxHP(),
			ch.GetMana(), ch.GetMaxMana(),
			ch.GetMove(), ch.GetMaxMove(),
			ch.Kills, ch.PKs, ch.Deaths)
		p.SendMessage(msg)
	}
	return true
}
