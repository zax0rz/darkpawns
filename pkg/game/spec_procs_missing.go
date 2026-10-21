package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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
	// C calls hit(me, ch, TYPE_UNDEFINED) after the two acts
	// (new_cmds2.c:561-569): the swing consumes hit()'s draws and damage()'s
	// gates run before anyone enrols. The old plain enrolment skipped the
	// swing entirely.
	w.mobFirstHit(me, ch)
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
	if cmd == "list" || cmd == "help" {
		Act(w, false, me, ch, nil, nil, "$n sighs loudly.", "", ToRoom)
		Act(w, false, me, ch, nil, nil, "$n says 'I recharge wands and staves, of course!'", "", ToRoom)
		Act(w, false, me, ch, nil, nil, "$n says 'My price is 100 coins per spell level per charge..'", "", ToRoom)
		Act(w, false, me, ch, nil, nil, "$n says 'So a wand casting heal at level 25 would cost 2500 coins per charge.'", "", ToRoom)
		Act(w, false, me, ch, nil, nil, "$n says 'If it had 10 max charges at first, it would have 9 max after charging.", "", ToRoom)
		Act(w, false, me, ch, nil, nil, "$n says 'That means that an item could have 1 charge maximum, with 2 charges remaining (after a recharge)'", "", ToRoom)
		Act(w, false, me, ch, nil, nil, "$n says 'To recharge an item type: recharge <staff or wand>.'", "", ToRoom)
		Act(w, true, me, ch, nil, nil, "$n stares expectantly at $N.", "", ToNotVict)
		Act(w, false, me, ch, nil, nil, "$n stares expectantly at you.", "", ToVict)
		return true
	}
	if cmd != "recharge" {
		return false
	}
	fields := strings.Fields(arg)
	arg = ""
	if len(fields) > 0 {
		arg = fields[0]
	}
	ordinal := GetNumber(&arg)

	var item *ObjectInstance
	if ordinal > 0 {
		matches := 0
		for _, carried := range COrderedCarrying(ch) {
			if carried == nil || (!CanSeeObject(ch, carried) && carried.GetTypeFlag() != ITEM_LIGHT) || !isnameWithAbbrevs(arg, carried.GetKeywords()) {
				continue
			}
			matches++
			if matches == ordinal {
				item = carried
				break
			}
		}
	}
	if item == nil {
		Act(w, false, me, ch, nil, nil, "$n tells you, 'You don't have that!'", "", ToVict)
		Act(w, true, me, ch, nil, nil, "$n whaps $N upside the head.", "", ToNotVict)
		Act(w, false, me, ch, nil, nil, "$n whaps you upside the head.", "", ToVict)
		return true
	}

	itemType := item.GetTypeFlag()
	if itemType != ITEM_WAND && itemType != ITEM_STAFF {
		Act(w, false, me, ch, nil, nil, "$n tells you, 'Ummm... does that look like a wand or staff to you?'", "", ToVict)
		Act(w, false, me, ch, nil, nil, "$n shakes his head in disgust.", "", ToRoom)
		return true
	}

	maxCharges := item.GetValue(1)
	currCharges := item.GetValue(2)
	spellLvl := item.GetValue(0)

	if currCharges >= maxCharges {
		Act(w, false, me, ch, nil, nil, "$n tells you, 'The item does not need recharging, stop wasting my time!'", "", ToVict)
		return true
	}

	cost := spellLvl * 100
	if cost < 100 {
		cost = 100
	}

	ch.mu.Lock()
	if ch.Gold < cost {
		ch.mu.Unlock()
		Act(w, false, me, ch, nil, nil, "$n tells you, 'You don't have enough gold!'", "", ToVict)
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

	Act(w, false, me, ch, nil, nil, "$n waves his hands around and chants strange things.", "", ToRoom)
	ch.SendMessage(fmt.Sprintf("The item now has %d charges remaining.\r\n", newCurr))
	return true
}

// beholderNumber is the C number() seam for SPECIAL(beholder)'s pulse draws.
var beholderNumber = dprng.Number

// specBeholder ports SPECIAL(beholder) (src/new_cmds2.c:328-398).
//
// C has three arms in this order:
//
//   - cmd: cast and recite are intercepted with the two eyestalk acts;
//     every other command is declined (:334-343).
//   - not fighting: count the room's people, return when targets <= 1,
//     pick one with number(0, targets), skip a zero pick, then cast sleep,
//     charm or curse on number(0, 2) (:345-373). C returns FALSE, so the
//     rest of the mobile's pulse still runs.
//   - fighting: scan the room in people order and take the first body whose
//     FIGHTING is the beholder and whose !number(0,4) fires; fall back to
//     FIGHTING(ch), then number(0,10) -> disrupt, else number(0,5) ->
//     disintegrate (:377-397).
//
// The previous Go body invented a five-way randN(5) spell table and an HP
// gate that C does not have, so it drew nothing in the peaceful arm and the
// wrong draws in the fighting arm (R4, R3).
func specBeholder(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil {
		return false
	}
	if cmd != "" {
		if cmd == "cast" || cmd == "recite" {
			Act(w, true, me, ch, nil, nil, "A ray of light shoots out from one of $n's eyestalks, hitting $N!", "", ToNotVict)
			Act(w, false, me, ch, nil, nil, "A ray of light shoots out from one of $n's eyestalks, breaking your concentration!", "", ToVict)
			return true
		}
		return false
	}

	people := cRoomPeople(w, me.RoomVNum)

	if me.GetPosition() != combat.PosFighting {
		// C counts every occupant, itself included.
		if len(people) <= 1 {
			return false
		}
		pick := beholderNumber(0, len(people))
		if pick <= 0 || pick > len(people) {
			return false
		}
		vict := people[pick-1]
		if vict == Actor(me) || vict.IsNPC() {
			return false
		}
		switch beholderNumber(0, 2) {
		case 0:
			spells.Cast(me, vict, spells.SpellSleep, me.GetLevel(), nil)
		case 1:
			spells.Cast(me, vict, spells.SpellCharm, me.GetLevel(), nil)
		case 2:
			spells.Cast(me, vict, spells.SpellCurse, me.GetLevel(), nil)
		}
		return false
	}

	var target Actor
	for _, body := range people {
		fighter, ok := body.(interface{ GetFightingBody() combat.Combatant })
		if !ok || fighter.GetFightingBody() != combat.Combatant(me) {
			continue
		}
		if beholderNumber(0, 4) == 0 {
			target = body
			break
		}
	}
	if target == nil {
		if opponent := me.GetFightingBody(); opponent != nil {
			if actor, ok := opponent.(Actor); ok {
				target = actor
			}
		}
	}
	if target == nil {
		// C would pass a NULL victim to cast_spell here; guard instead of
		// dereferencing it. The arm is only reachable with POS_FIGHTING and no
		// opponent, which combat cannot leave behind in practice.
		return false
	}
	if beholderNumber(0, 10) == 0 {
		spells.Cast(me, target, spells.SpellDisrupt, me.GetLevel(), nil)
	} else if beholderNumber(0, 5) == 0 {
		spells.Cast(me, target, spells.SpellDisintegrate, me.GetLevel(), nil)
	} else {
		return false
	}
	return true
}

// specZenMaster ports the Kir-Oshi zen master's combat-only pulse.
func specZenMaster(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || me == nil || cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() <= 0 {
		return false
	}

	// C scans world[room].people head-first and gives every body fighting this
	// mob an independent 1-in-5 selection draw. If none is selected, it falls
	// back to FIGHTING(me), then always draws the action number.
	var victim combat.Combatant
	for _, candidate := range cOrderedRoomCombatants(w, me.GetRoomVNum()) {
		if candidate.GetFightingBody() != me || number(0, 4) != 0 {
			continue
		}
		victim = candidate
		break
	}
	if victim == nil {
		victim = me.GetFightingBody()
	}

	switch number(0, 20) {
	case 0:
		var victimActor Actor
		if actor, ok := victim.(Actor); ok {
			victimActor = actor
		}
		Act(w, true, me, victimActor, nil, nil, "$n touches $N on the arm.", "", ToNotVict)
		Act(nil, true, me, victimActor, nil, nil, "$n touches you lightly on the arm.", "", ToVict)
		spells.CastFromSpecial(me, victim, spells.SpellWordOfRecall, me.GetLevel(), w)
		return true
	case 20:
		Act(w, false, me, nil, nil, nil, "$n says, 'You have violence, but not thought.'", "", ToRoom)
		spells.CastFromSpecial(me, victim, spells.SpellTeleport, me.GetLevel(), w)
		return true
	default:
		return false
	}
}

// specMoonGate ports SPECIAL(moon_gate) (src/gate.c:237-325). The assigned
// object supplies its name and the red/blue destination rules.
func specMoonGate(w *World, ch *Player, obj *ObjectInstance, cmd string, arg string) bool {
	if ch == nil || obj == nil {
		return false
	}

	switch cmd {
	case "enter":
		if !isName(arg, obj.GetName()) {
			return false
		}

		portalRoom := 0
		if obj.GetVNum() == BluePortalVNum {
			for _, gate := range gatePhases {
				if gate.Room != ch.GetRoomVNum() {
					continue
				}
				exitRoom := gate.Exit1
				if combat.GetRoller().Number(0, 3) >= 2 {
					exitRoom = gate.Exit2
				}
				if exitRoom == 0 {
					exitRoom = gate.Exit1
				}
				portalRoom = exitRoom
				break
			}
		} else if obj.GetCost() > 0 {
			portalRoom = obj.GetCost()
		} else {
			portalRoom = 4000
		}

		ch.SendMessage("You enter the portal and are transferred...\n\r")
		Act(w, true, ch, nil, nil, nil,
			"$n enters the gate which suddenly starts glowing brightly,\r\nobscuring your view of $m!", "", ToRoom)
		if err := w.PlayerTransfer(ch, portalRoom); err != nil {
			slog.Error("moon gate transfer failed", "player", ch.GetName(), "to", portalRoom, "error", err)
			return true
		}
		Act(w, true, ch, nil, nil, nil,
			"With a sound like a raging river, a hole in space opens up and $n steps out.", "", ToRoom)
		w.lookAtRoom(ch, true)
		return true
	case "look":
		lowered := strings.ToLower(strings.TrimSpace(arg))
		if lowered != "gate" && lowered != "moongate" && lowered != "portal" {
			// C moon_gate delegates non-portal targets to do_look and consumes
			// the command (src/gate.c:309-316).
			w.doLook(ch, nil, cmd, lowered)
			return true
		}
		ch.SendMessage("You see a shimmering light... you could ENTER the gate.\r\n")
		return false
	default:
		return false
	}
}

func init() {
	RegisterSpec("no_get", specNoGet)
	RegisterSpec("recharger", specRecharger)
	RegisterSpec("beholder", specBeholder)
	RegisterSpec("zen_master", specZenMaster)
	// Keep the assigned-special registry validation satisfied; concrete
	// moon_gate dispatch uses the object-aware handler below.
	RegisterSpec("moon_gate", func(_ *World, _ *Player, _ *MobInstance, _, _ string) bool { return false })
	RegisterObjSpec("moon_gate", specMoonGate)
}
