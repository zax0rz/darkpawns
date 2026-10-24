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
		// C do_visible calls appear(ch) (src/act.other.c:577), which removes
		// the SPELL_INVISIBLE affect record before clearing the AFF bits
		// (src/fight.c:108-121). Clearing the raw bit alone left the spell
		// record re-asserting AFF_INVISIBLE through IsAffected.
		w.appear(ch)
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

// doReport ports do_report (src/act.other.c:799-821) for either body. C has no
// IS_NPC gate: the reporter's numbers are formatted once per character in the
// room through that recipient's own color wrappers, so the same line reaches
// every occupant and always carries the *reporter's* stats (quirk included).
// A mobile has no descriptor, so send_to_char("You report:", ch) and every
// TO_CHAR audience line are invisible for it; the room header and the players'
// lines are not.
func (w *World) doReport(ch *Player, me *MobInstance, cmd string, arg string) bool {
	var (
		reporter           Actor
		roomVNum           int
		hp, maxHP          int
		mana, maxMana      int
		move, maxMove      int
		kills, pks, deaths int
	)
	switch {
	case me != nil:
		reporter = me
		roomVNum = me.GetRoomVNum()
		hp, maxHP = me.GetHP(), me.GetMaxHP()
		mana, maxMana = me.GetMana(), me.GetMaxMana()
		move, maxMove = me.GetMove(), me.GetMaxMove()
		// C reads GET_KILLS/PKS/DEATHS from player_specials; every mobile
		// shares the zeroed dummy_mob (src/db.c:104,1281), so those three
		// fields report 0 for a mobile.
	case ch != nil:
		reporter = ch
		roomVNum = ch.GetRoomVNum()
		hp, maxHP = ch.GetHP(), ch.GetMaxHP()
		mana, maxMana = ch.GetMana(), ch.GetMaxMana()
		move, maxMove = ch.GetMove(), ch.GetMaxMove()
		kills, pks, deaths = ch.Kills, ch.PKs, ch.Deaths
	default:
		return true
	}

	// act("$n reports:", FALSE, ch, 0, 0, TO_ROOM) — act appends the CRLF.
	Act(w, false, reporter, nil, nil, nil, "$n reports:", "", ToRoom)
	if ch != nil {
		ch.SendMessage("You report:\r\n")
	}

	// for (tch = world[ch->in_room].people; tch; tch = tch->next_in_room)
	//   act(buf, FALSE, tch, 0, 0, TO_CHAR);
	// tch supplies only the color wrappers; the numbers are ch's.
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		green := observationColors(p, ansiGreen)
		red := observationColors(p, ansiRed)
		normal := observationColors(p, ansiReset)
		line := fmt.Sprintf(
			"    [%s%d%s/%s%d%s]H [%s%d%s/%s%d%s]M [%s%d%s/%s%d%s]V"+
				" [%s%d%s]Kills [%s%d%s]PKs [%s%d%s]Deaths",
			green, hp, normal, green, maxHP, normal,
			green, mana, normal, green, maxMana, normal,
			green, move, normal, green, maxMove, normal,
			red, kills, normal, red, pks, normal, red, deaths, normal)
		// act(buf, FALSE, tch, 0, 0, TO_CHAR): the awake/writing gate is
		// SENDOK(tch, false), so a sleeping or writing occupant gets nothing.
		Act(w, false, p, nil, nil, nil, line, "", ToChar)
	}
	return true
}

// NPCReportCommand adapts do_report to a descriptorless mobile actor reached
// through command_interpreter (order, force, lua action).
func (w *World) NPCReportCommand(me *MobInstance) { w.doReport(nil, me, "report", "") }
