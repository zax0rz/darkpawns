package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// specNoGet is the special procedure for the Grey Keep's no-get guards.
// C: new_cmds2.c:552-574. It intercepts one- or zero-argument GET, PALM, and
// TAKE before their ordinary command handlers and starts combat with the
// player after emitting the two hand-strike messages.
func specNoGet(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil {
		return false
	}
	if me.GetPosition() <= combat.PosSleeping || me.GetHP() <= 0 {
		return false
	}
	if cmd != "get" && cmd != "palm" && cmd != "take" {
		return false
	}
	// C's two_arguments() leaves the procedure active for zero or one token,
	// but returns FALSE when a second token is present.
	if len(strings.Fields(arg)) > 1 {
		return false
	}

	Act(w, true, me, ch, nil, nil, "$n strikes at $N's hand!", "", ToNotVict)
	Act(nil, false, me, ch, nil, nil, "$n strikes at your hand!", "", ToVict)
	if w != nil && w.combatEngine != nil {
		// C calls hit(me, ch, TYPE_UNDEFINED) here. The shared combat engine
		// owns the combat-entry state; TYPE_UNDEFINED has no loaded C skill
		// message, so this special must not invent a swing transcript.
		if err := w.combatEngine.StartCombat(me, ch); err != nil {
			slog.Debug("no_get combat entry already active", "mob", me.GetName(), "player", ch.GetName(), "error", err)
		}
	}
	return true
}

// specRecharger charges wands/staves for gold.
// Intercepts RECHARGE.
func specRecharger(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil {
		return false
	}
	if ch.GetPosition() < combat.PosSitting {
		return false
	}
	if cmd != "recharge" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("What do you want to recharge?\r\n")
		return true
	}

	var item *ObjectInstance
	if ch.Inventory != nil {
		item, _ = ch.Inventory.FindItem(arg)
	}
	if item == nil && ch.Equipment != nil {
		equipped := ch.Equipment.GetEquippedItems()
		for _, eqItem := range equipped {
			if eqItem != nil && (strings.Contains(strings.ToLower(eqItem.GetKeywords()), strings.ToLower(arg)) || strings.Contains(strings.ToLower(eqItem.GetShortDesc()), strings.ToLower(arg))) {
				item = eqItem
				break
			}
		}
	}
	if item == nil {
		ch.SendMessage("You don't seem to have that item.\r\n")
		return true
	}

	itemType := item.GetTypeFlag()
	if itemType != ITEM_WAND && itemType != ITEM_STAFF {
		ch.SendMessage("That is not a wand or staff.\r\n")
		return true
	}

	maxCharges := item.GetValue(1)
	currCharges := item.GetValue(2)
	spellLvl := item.GetValue(0)

	if currCharges >= maxCharges {
		ch.SendMessage("That item is already fully charged.\r\n")
		return true
	}

	cost := spellLvl * 100
	if cost < 100 {
		cost = 100
	}

	ch.mu.Lock()
	if ch.Gold < cost {
		ch.mu.Unlock()
		ch.SendMessage(fmt.Sprintf("It costs %d gold to recharge that, which you cannot afford.\r\n", cost))
		return true
	}
	ch.Gold -= cost
	ch.mu.Unlock()

	// Source: src/new_cmds2.c SPECIAL(recharger) — current charges and max
	// charges move independently (current+1, max-1), not to a shared value.
	// This lets current exceed max after enough recharges, a quirk the spec's
	// own "list" flavor text calls out explicitly.
	newCurr := currCharges + 1
	newMax := maxCharges - 1
	item.SetValue(2, newCurr)
	item.SetValue(1, newMax)

	w.roomMessage(ch.GetRoomVNum(), fmt.Sprintf("%s recharges %s.", me.Prototype.ShortDesc, item.GetShortDesc()))
	ch.SendMessage(fmt.Sprintf("The recharger works their magic on %s! It costs %d gold. The item now has %d charges remaining.\r\n", item.GetShortDesc(), cost, newCurr))
	return true
}

// specBeholder casts spells randomly in combat.
func specBeholder(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Block spellcasting in beholder's presence (C: intercepts cast and recite)
	if cmd == "cast" || cmd == "recite" {
		ch.SendMessage("You feel your magic dissipate in the beholder's presence!\r\n")
		return true
	}
	if cmd != "" {
		return false
	}
	if me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil {
		return false
	}

	switch randN(5) {
	case 0:
		w.roomMessage(me.RoomVNum, me.GetName()+"'s central eye glows with disintegrating power!")
		spells.Cast(me, melee, spells.SpellDisintegrate, me.GetLevel(), nil)
	case 1:
		w.roomMessage(me.RoomVNum, me.GetName()+" casts a sleep ray at "+melee.GetName()+"!")
		spells.Cast(me, melee, spells.SpellSleep, me.GetLevel(), nil)
	case 2:
		w.roomMessage(me.RoomVNum, me.GetName()+" casts a fear spell on "+melee.GetName()+"!")
		spells.Cast(me, melee, spells.SpellCurse, me.GetLevel(), nil)
	case 3:
		w.roomMessage(me.RoomVNum, me.GetName()+" disrupts "+melee.GetName()+"'s magic!")
		spells.Cast(me, melee, spells.SpellDisrupt, me.GetLevel(), nil)
	}
	return true
}

// specZenMaster teleports attackers in combat.
func specZenMaster(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil || number(0, 6) != 0 {
		return false
	}
	w.roomMessage(me.RoomVNum, me.GetName()+" touches "+melee.GetName()+" lightly on the forehead!")
	spells.Cast(me, melee, spells.SpellTeleport, me.GetLevel(), nil)
	return true
}

// specMoonGate teleports players entering night portals.
// Intercepts ENTER. Destination is Exit1 from the gatePhases table for the
// player's current room; falls back to MortalStartRoom if not found.
func specMoonGate(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "enter" {
		return false
	}
	arg = strings.TrimSpace(arg)
	if arg != "gate" && arg != "portal" && arg != "moon gate" {
		return false
	}

	sendToChar(ch, "You step into the shimmering, iridescent moon gate...")
	w.roomMessage(ch.GetRoomVNum(), "$n steps into the shimmering moon gate and vanishes!")

	dest := MortalStartRoom
	currentRoom := ch.GetRoomVNum()
	for _, ge := range gatePhases {
		if ge.Room == currentRoom && ge.Exit1 != 0 {
			dest = ge.Exit1
			break
		}
	}

	ch.SetRoom(dest)
	w.actToRoom(ch, "$n arrives through a shimmering moon gate!", nil, nil)
	return true
}

func init() {
	RegisterSpec("no_get", specNoGet)
	RegisterSpec("recharger", specRecharger)
	RegisterSpec("beholder", specBeholder)
	RegisterSpec("zen_master", specZenMaster)
	RegisterSpec("moon_gate", specMoonGate)
}
