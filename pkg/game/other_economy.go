//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/spells"
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
	if _, err := fmt.Sscanf(arg, "%d", &amount); err != nil {
		ch.SendMessage("That doesn't look like a number.\r\n")
		slog.Warn("split parse failed", "player", ch.Name, "arg", arg, "error", err)
		return true
	}
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

	// Handle tattoo use — from src/tattoo.c use_tattoo()
	if strings.EqualFold(itemArg, "tattoo") {
		if ch.TatTimer > 0 {
			suffix := "s"
			if ch.TatTimer == 1 {
				suffix = ""
			}
			ch.SendMessage(fmt.Sprintf("You can't use your tattoo's magick for %d more hour%s.\r\n",
				ch.TatTimer, suffix))
			return true
		}
		switch ch.Tattoo {
		case TattooNone:
			ch.SendMessage("You don't have a tattoo.\r\n")
		case TattooSkull:
			// Summon mob vnum 9 (skull), charm it, make it follow
			mob, err := w.SpawnMob(9, ch.RoomVNum)
			if err != nil {
				ch.SendMessage("Your tattoo fizzles...\r\n")
				break
			}
			w.SetFollower(mob.GetName(), ch.GetName(), true)
			// Apply charm affect (duration 20)
			mob.AddAffect(&engine.Affect{
				Type:      engine.AffectType(spells.SpellCharm),
				Duration:  20,
				Magnitude: 0,
				Flags:     1 << 3, // AFF_CHARM
			})
			w.roomMessage(ch.RoomVNum, fmt.Sprintf("%s's tattoo glows brightly for a second, and %s appears!", ch.Name, mob.Prototype.ShortDesc))
			ch.SendMessage(fmt.Sprintf("Your tattoo glows brightly for a second, and %s appears!\r\n", mob.Prototype.ShortDesc))
		case TattooEye:
			spells.Cast(ch, ch, spells.SpellGreatPercept, ch.Level, w, nil)
		case TattooShip:
			spells.Cast(ch, ch, spells.SpellChangeDensity, ch.Level, w, nil)
		case TattooAngel:
			spells.Cast(ch, ch, spells.SpellBless, ch.Level, w, nil)
		default:
			ch.SendMessage("Your tattoo can't be 'use'd.\r\n")
			return true
		}
		ch.TatTimer = 24
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
