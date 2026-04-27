package game

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// do_split — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doSplit(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("How many coins do you wish to split with your group?\r\n")
		return true
	}

	amount := 0
// #nosec G104
	fmt.Sscanf(arg, "%d", &amount)
	if amount <= 0 {
		ch.SendMessage("Sorry, you can't do that.\r\n")
		return true
	}
	ch.mu.Lock()
	if amount > ch.Gold {
		ch.mu.Unlock()
		ch.SendMessage("You don't seem to have that much gold to split.\r\n")
		return true
	}

	leaderName := ch.Following
	if leaderName == "" {
		leaderName = ch.Name
	}

	// Count group members in same room
	num := 0
	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.Following != leaderName && p.Name != leaderName {
			continue
		}
		if p.IsAffected(affGroup) {
			num++
		}
	}

	if num <= 1 || !ch.IsAffected(affGroup) {
		ch.mu.Unlock()
		ch.SendMessage("With whom do you wish to share your gold?\r\n")
		return true
	}

	share := amount / num
	ch.Gold -= share * (num - 1)
	ch.mu.Unlock()

	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.Following != leaderName && p.Name != leaderName {
			continue
		}
		if !p.IsAffected(affGroup) || p.Name == ch.Name {
			continue
		}
		p.mu.Lock()
		p.Gold += share
		p.mu.Unlock()
		p.SendMessage(fmt.Sprintf("%s splits %d coins; you receive %d.\r\n", ch.Name, amount, share))
	}

	ch.SendMessage(fmt.Sprintf("You split %d coins among %d members -- %d coins each.\r\n", amount, num, share))
	return true
}

// ---------------------------------------------------------------------------
// do_use — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doUse(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	parts := strings.SplitN(arg, " ", 2)
	itemArg := strings.TrimSpace(parts[0])
	_ = itemArg  // suppress unused
	if len(parts) > 1 {
		_ = strings.TrimSpace(parts[1]) // subArg placeholder
	}

	if itemArg == "" {
		ch.SendMessage(fmt.Sprintf("What do you want to %s?\r\n", cmd))
		return true
	}

	// Handle tattoo use
	if strings.EqualFold(itemArg, "tattoo") {
		ch.SendMessage("Tattoo functionality not yet implemented.\r\n")
		return true
	}

	// Find item via findObjNear
	item := w.findObjNear(ch, itemArg)

	if item == nil {
		ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", "a", itemArg))
		return true
	}

	// Simplified: just use the item (item-type routing TBD)
	_ = item.Prototype.TypeFlag

	// Call mag_objectmagic (simplified)
	ch.SendMessage(fmt.Sprintf("You use %s.\r\n", itemArg))
	return true
}
