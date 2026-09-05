package game

import "fmt"

const spellInvisible = 29 // C SPELL_INVISIBLE (src/spell_parser.c:85)

// DoInvis ports ACMD(do_invis) from src/act.wizard.c:1663-1688. The session
// dispatcher owns the LVL_IMMORT/POS_DEAD command gate; this method owns the
// handler's atoi branches, invisibility state, and room audiences.
func (w *World) DoInvis(ch *Player, argument string) {
	if ch == nil {
		return
	}

	arg, _ := oneArgument(argument)
	if arg == "" {
		if ch.GetInvisLevel() > 0 {
			w.performImmortVis(ch)
		} else {
			w.performImmortInvis(ch, ch.GetLevel())
		}
		return
	}

	level := atoiC(arg)
	if level > ch.GetLevel() {
		ch.SendMessage("You can't go invisible above your own level.\r\n")
	} else if level < 1 {
		w.performImmortVis(ch)
	} else {
		w.performImmortInvis(ch, level)
	}
}

// performImmortVis ports perform_immort_vis from src/act.wizard.c:1621-1636.
func (w *World) performImmortVis(ch *Player) {
	if ch.GetInvisLevel() == 0 &&
		(!ch.IsAffected(affHide) || !ch.IsAffected(affInvisible)) {
		ch.SendMessage("You are already fully visible.\r\n")
		return
	}

	ch.SetInvisLevel(0)
	w.appear(ch)
	ch.SendMessage("You are now fully visible.\r\n")
}

// performImmortInvis ports perform_immort_invis from src/act.wizard.c:1641-1659.
func (w *World) performImmortInvis(ch *Player, level int) {
	oldLevel := ch.GetInvisLevel()
	for _, target := range w.GetPlayersInRoom(ch.GetRoom()) {
		if target == ch {
			continue
		}
		if target.GetLevel() >= oldLevel && target.GetLevel() < level {
			Act(nil, false, ch, target, nil, nil,
				"You blink and suddenly realize that $n is gone.", "", ToVict)
		}
		if target.GetLevel() < oldLevel && target.GetLevel() >= level {
			Act(nil, false, ch, target, nil, nil,
				"You suddenly realize that $n is standing beside you.", "", ToVict)
		}
	}

	ch.SetInvisLevel(level)
	ch.SendMessage(fmt.Sprintf("Your invisibility level is %d.\r\n", level))
}

// appear ports the shared C appear() helper from src/fight.c:108-121.
func (w *World) appear(ch *Player) {
	ch.RemoveAffectBySpell(spellInvisible)
	ch.RemoveAffectBit(affInvisible)
	ch.RemoveAffectBit(affHide)

	if ch.GetLevel() < LVL_IMMORT {
		Act(w, false, ch, nil, nil, nil, "$n slowly fades into existence.", "", ToRoom)
	} else {
		Act(w, false, ch, nil, nil, nil,
			"You feel a strange presence as $n appears, seemingly from nowhere.", "", ToRoom)
	}
}
