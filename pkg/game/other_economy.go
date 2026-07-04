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
	if amount > ch.GetGold() {
		ch.mu.Unlock()
		ch.SendMessage("You don't seem to have that much gold to split.\r\n")
		return true
	}

	leaderName := ch.GetFollowing()
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
		if p.GetFollowing() != leaderName && p.Name != leaderName {
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
	ch.SetGold(ch.GetGold() - share*(num-1))
	ch.mu.Unlock()

	for _, p := range players {
		if p.IsNPC() {
			continue
		}
		if p.GetFollowing() != leaderName && p.Name != leaderName {
			continue
		}
		if !p.IsAffected(affGroup) || p.Name == ch.Name {
			continue
		}
		p.mu.Lock()
		p.SetGold(p.GetGold() + share)
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
	_ = itemArg // suppress unused
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
			mob, err := w.SpawnMob(9, ch.GetRoom())
			if err != nil {
				ch.SendMessage("Your tattoo fizzles...\r\n")
				break
			}
			if err := w.SetFollower(mob.GetName(), ch.GetName(), true); err != nil {
				slog.Error("SetFollower failed for tattoo skull", "mob", mob.GetName(), "leader", ch.GetName(), "error", err)
			}
			// Apply charm affect (duration 20)
			mob.AddAffect(&engine.Affect{
				SpellID:   spells.SpellCharm,
				Type:      spells.SpellCharm, // backward compat
				Duration:  20,
				Magnitude: 0,
				Flags:     1 << 3, // AFF_CHARM
			})
			w.roomMessage(ch.GetRoom(), fmt.Sprintf("%s's tattoo glows brightly for a second, and %s appears!", ch.Name, mob.Prototype.ShortDesc))
			ch.SendMessage(fmt.Sprintf("Your tattoo glows brightly for a second, and %s appears!\r\n", mob.Prototype.ShortDesc))
		case TattooEye:
			spells.Cast(ch, ch, spells.SpellGreatPercept, ch.GetLevel(), w)
		case TattooShip:
			spells.Cast(ch, ch, spells.SpellChangeDensity, ch.GetLevel(), w)
		case TattooAngel:
			spells.Cast(ch, ch, spells.SpellBless, ch.GetLevel(), w)
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

	itemType := item.GetTypeFlag()
	if itemType != ITEM_WAND && itemType != ITEM_STAFF && itemType != ITEM_POTION && itemType != ITEM_SCROLL {
		ch.SendMessage("You can't use that item.\r\n")
		return true
	}

	spellLvl := item.GetValue(0)
	spellType := item.GetValue(3)

	switch itemType {
	case ITEM_WAND:
		currCharges := item.GetValue(2)
		if currCharges <= 0 {
			ch.SendMessage("The wand is out of charges!\r\n")
			return true
		}
		item.SetValue(2, currCharges-1)

		targetName := ""
		if len(parts) > 1 {
			targetName = strings.TrimSpace(parts[1])
		}

		var target interface{}
		if targetName != "" {
			if p := w.FindPlayerInRoom(ch.GetRoomVNum(), targetName); p != nil {
				target = p
			} else if m := w.FindMobInRoom(ch.GetRoomVNum(), targetName); m != nil {
				target = m
			}
		}
		if target == nil {
			if ch.Fighting != "" {
				if p, ok := w.GetPlayer(ch.Fighting); ok {
					target = p
				} else {
					for _, mob := range w.GetMobsInRoom(ch.GetRoomVNum()) {
						if mob.GetName() == ch.Fighting {
							target = mob
							break
						}
					}
				}
			}
		}
		if target == nil {
			target = ch
		}

		var targetNameDisp string
		if p, ok := target.(*Player); ok {
			targetNameDisp = p.Name
		} else if m, ok := target.(*MobInstance); ok {
			targetNameDisp = m.Prototype.ShortDesc
		} else {
			targetNameDisp = "someone"
		}

		ch.SendMessage(fmt.Sprintf("You point %s at %s.\r\n", item.GetShortDesc(), targetNameDisp))
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s points %s at %s.", ch.Name, item.GetShortDesc(), targetNameDisp))

		spells.Cast(ch, target, spellType, spellLvl, w)

	case ITEM_STAFF:
		currCharges := item.GetValue(2)
		if currCharges <= 0 {
			ch.SendMessage("The staff is out of charges!\r\n")
			return true
		}
		item.SetValue(2, currCharges-1)

		ch.SendMessage(fmt.Sprintf("You tap %s on the ground.\r\n", item.GetShortDesc()))
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s taps %s on the ground.", ch.Name, item.GetShortDesc()))

		// Cast spell on everyone in the room
		players := w.GetPlayersInRoom(ch.GetRoomVNum())
		for _, p := range players {
			if p != nil {
				spells.Cast(ch, p, spellType, spellLvl, w)
			}
		}
		mobs := w.GetMobsInRoom(ch.GetRoomVNum())
		for _, mob := range mobs {
			if mob != nil {
				spells.Cast(ch, mob, spellType, spellLvl, w)
			}
		}

	case ITEM_POTION:
		ch.SendMessage(fmt.Sprintf("You quaff %s.", item.GetShortDesc()))
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s quaffs %s.", ch.Name, item.GetShortDesc()))

		spells.Cast(ch, ch, spellType, spellLvl, w)

		ch.Inventory.RemoveItem(item)

	case ITEM_SCROLL:
		targetName := ""
		if len(parts) > 1 {
			targetName = strings.TrimSpace(parts[1])
		}

		var target interface{}
		if targetName != "" {
			if p := w.FindPlayerInRoom(ch.GetRoomVNum(), targetName); p != nil {
				target = p
			} else if m := w.FindMobInRoom(ch.GetRoomVNum(), targetName); m != nil {
				target = m
			}
		}
		if target == nil {
			target = ch
		}

		var targetNameDisp string
		if p, ok := target.(*Player); ok {
			targetNameDisp = p.Name
		} else if m, ok := target.(*MobInstance); ok {
			targetNameDisp = m.Prototype.ShortDesc
		} else {
			targetNameDisp = "someone"
		}

		ch.SendMessage(fmt.Sprintf("You recite %s targeting %s.", item.GetShortDesc(), targetNameDisp))
		w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s recites %s targeting %s.", ch.Name, item.GetShortDesc(), targetNameDisp))

		spells.Cast(ch, target, spellType, spellLvl, w)

		ch.Inventory.RemoveItem(item)
	}

	return true
}

// DoUse is the exported session-level entrypoint for item usage.
func (w *World) DoUse(ch *Player, arg string) bool {
	return w.doUse(ch, nil, "use", arg)
}
