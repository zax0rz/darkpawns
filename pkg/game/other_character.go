package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// ---------------------------------------------------------------------------
// do_visible — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doVisible(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// Kai zai check (simplified: skill name "kai_zai" or "kz")
	hasKaiZai := false
	ch.mu.RLock()
	affects := make([]*engine.Affect, len(ch.ActiveAffects))
	copy(affects, ch.ActiveAffects)
	ch.mu.RUnlock()
	for _, a := range affects {
		if strings.Contains(strings.ToLower(a.Source), "zai") {
			hasKaiZai = true
			break
		}
	}
	if hasKaiZai {
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
