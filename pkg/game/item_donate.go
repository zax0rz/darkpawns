package game

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/common"
)

func disposeInventorySnapshot(ch Actor) []*ObjectInstance {
	switch v := ch.(type) {
	case *Player:
		return v.Inventory.Snapshot()
	case *MobInstance:
		return v.InventorySnapshot()
	default:
		return nil
	}
}

func disposeAddInventory(ch Actor, obj *ObjectInstance) {
	switch v := ch.(type) {
	case *Player:
		//nolint:errcheck // best-effort rollback
		v.Inventory.AddItem(obj)
	case *MobInstance:
		v.AddToInventory(obj)
	}
}

func disposeRemoveInventory(ch Actor, obj *ObjectInstance) {
	switch v := ch.(type) {
	case *Player:
		v.Inventory.RemoveItem(obj)
	case *MobInstance:
		v.RemoveFromInventory(obj)
	}
}

func disposeSpendGold(ch Actor, amount int) bool {
	switch v := ch.(type) {
	case *Player:
		return v.SpendGold(amount)
	case *MobInstance:
		return v.SpendGold(amount)
	default:
		return false
	}
}

func disposeAddGold(ch Actor, amount int) {
	switch v := ch.(type) {
	case *Player:
		v.AddGold(amount)
	case *MobInstance:
		v.SetGold(v.GetGold() + amount)
	}
}

func disposeSetWaitState(ch Actor, rounds int) {
	switch v := ch.(type) {
	case *Player:
		v.SetWaitState(rounds)
	case *MobInstance:
		v.SetWaitState(rounds)
	}
}

// performDispose implements perform_drop() for the junk/donate paths —
// src/act.item.c:480-525. mode is scmdJunk or scmdDonate; donationRoom is
// the dice-selected destination room for a donate (unused for junk).
// Returns the junk-value reward (0 for a successful donate, or when the
// item couldn't be disposed of at all).
func (w *World) performDispose(ch Actor, obj *ObjectInstance, mode int, sname string, donationRoom int) int {
	chLevel := 0
	if l, ok := ch.(interface{ GetLevel() int }); ok {
		chLevel = l.GetLevel()
	}
	if obj.HasExtraFlag(0, extraFlagNoDrop) && chLevel < lvlImmort {
		Act(w, false, ch, nil, obj, nil, fmt.Sprintf("You can't %s $p, it must be CURSED!", sname), "", ToChar)
		return 0
	}

	// VANISH(mode) — act.item.c:477-478. Always applies here since this
	// helper is only ever called with mode == junk or donate.
	const vanish = "  It vanishes in a puff of smoke!"
	Act(w, false, ch, nil, obj, nil, fmt.Sprintf("You %s $p.%s", sname, vanish), "", ToChar)
	Act(w, true, ch, nil, obj, nil, fmt.Sprintf("$n %ss $p.%s", sname, vanish), "", ToRoom)
	disposeRemoveInventory(ch, obj)

	// A donated item flagged NODONATE is silently junked instead — act.item.c:498-499.
	if mode == scmdDonate && obj.HasExtraFlag(0, extraFlagNoDonate) {
		mode = scmdJunk
	}

	switch mode {
	case scmdDonate:
		// C perform_drop's SCMD_DONATE arm is obj_to_room (act.item.c:510).
		if err := w.MoveObjectToRoomFront(obj, donationRoom); err != nil {
			slog.Error("donation move failed",
				"player", ch.GetName(), "item_vnum", obj.VNum, "error", err)
			ch.SendMessage("Something went wrong. Your item was not donated.\r\n")
			disposeAddInventory(ch, obj)
			return 0
		}
		// C act() receives a NULL actor here (src/act.item.c:511), so it uses the
		// object's room and resolves $p separately for each recipient.
		Act(w, false, nil, nil, obj, nil, "$p suddenly appears in a puff a smoke!", "", ToRoom)
		return 0
	case scmdJunk:
		value := min(max(obj.GetCost()>>4, 1), 200)
		w.ExtractObject(obj, ch.GetRoom())
		return value
	}
	return 0
}

// performDisposeGold implements perform_drop_gold() for the junk/donate
// gold paths — src/act.item.c:430-474. mode is scmdJunk or scmdDonate.
func (w *World) performDisposeGold(ch Actor, amount int, mode int, donationRoom int) {
	if amount <= 0 {
		ch.SendMessage("Heh heh heh.. we are jolly funny today, eh?\r\n")
		return
	}
	if !disposeSpendGold(ch, amount) {
		ch.SendMessage("You don't have that many coins!\r\n")
		return
	}

	if mode == scmdJunk {
		Act(w, false, ch, nil, nil, nil, fmt.Sprintf("$n drops %s which disappears in a puff of smoke!", createMoneyDesc(amount)), "", ToRoom)
		ch.SendMessage("You drop some gold which disappears in a puff of smoke!\r\n")
	} else {
		disposeSetWaitState(ch, 1) // WAIT_STATE(ch, PULSE_VIOLENCE) — discourage coin-bombing
		ch.SendMessage("You throw some gold into the air where it disappears in a puff of smoke!\r\n")
		Act(w, false, ch, nil, nil, nil, "$n throws some gold into the air where it disappears in a puff of smoke!", "", ToRoom)
		moneyObj := w.createMoneyObject(amount)
		// C perform_drop_gold's donate arm is obj_to_room (act.item.c:452).
		if err := w.MoveObjectToRoomFront(moneyObj, donationRoom); err != nil {
			slog.Error("donation gold move failed",
				"player", ch.GetName(), "error", err)
			// The reservation is returned: the donation never landed.
			disposeAddGold(ch, amount)
			ch.SendMessage("Something went wrong. Your gold was not donated.\r\n")
			return
		}
		// C act() receives a NULL actor here (src/act.item.c:453), so it uses the
		// money object's room and applies recipient visibility and sleep gates.
		Act(w, false, nil, nil, moneyObj, nil, "$p suddenly appears in a puff of orange smoke!", "", ToRoom)
	}
}

// doDispose implements the shared argument handling of ACMD(do_drop) for
// the junk/donate subcommands — src/act.item.c:529-666. sname is the
// literal typed command ("junk" or "donate"); the junk-only exp reward is
// gated on sname, not on any per-item mode flip (donate never grants exp,
// even when the dice roll secretly junks an item).
func (w *World) doDispose(ch Actor, arg string, mode int, sname string, donationRoom int) {
	// C do_drop (junk/donate) parses with one_argument: fill words dropped,
	// argument lowercased. A fill-word-only argument reads as empty.
	arg1, rest := oneArgument(arg)
	if arg1 == "" {
		ch.SendMessage(fmt.Sprintf("What do you want to %s?\r\n", sname))
		return
	}

	// C do_drop gates the coin form with is_number (src/act.item.c:587-589)
	// and then atoi: only all-digit tokens reach perform_drop_gold, and an
	// overflowing token wraps negative into its "Heh heh heh.." arm.
	allDigits := arg1 != ""
	for _, digit := range arg1 {
		if digit < '0' || digit > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		amount := common.AtoiC(arg1)
		arg2, _ := oneArgument(rest)
		if arg2 == "coins" || arg2 == "coin" {
			w.performDisposeGold(ch, amount, mode, donationRoom)
		} else {
			ch.SendMessage("Sorry, you can't do that to more than one item at a time.\r\n")
		}
		return
	}

	amount := 0
	switch findAllDots(arg1) {
	case findAll:
		// Can't junk or donate everything in one command — act.item.c:605-616.
		if sname == "junk" {
			ch.SendMessage("Go to the dump if you want to junk EVERYTHING!\r\n")
		} else {
			ch.SendMessage("Go do the donation room if you want to donate EVERYTHING!\r\n")
		}
		return
	case findAlldot:
		keyword := arg1[4:]
		if keyword == "" {
			ch.SendMessage(fmt.Sprintf("What do you want to %s all of?\r\n", sname))
			return
		}
		items := disposeInventorySnapshot(ch)
		found := false
		for _, obj := range items {
			if isnameWithAbbrevs(keyword, obj.GetKeywords()) {
				found = true
				amount += w.performDispose(ch, obj, mode, sname, donationRoom)
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
	default:
		var obj *ObjectInstance
		for _, o := range disposeInventorySnapshot(ch) {
			if isnameWithAbbrevs(arg1, o.GetKeywords()) {
				obj = o
				break
			}
		}
		if obj == nil {
			ch.SendMessage(fmt.Sprintf("You don't seem to have %s %s.\r\n", an(arg1), arg1))
			return
		}
		amount += w.performDispose(ch, obj, mode, sname, donationRoom)
	}

	if amount > 0 && sname == "junk" {
		ch.SendMessage("You have been rewarded by the gods!\r\n")
		Act(w, true, ch, nil, nil, nil, "$n has been rewarded by the gods!", "", ToRoom)
		if p, ok := ch.(*Player); ok {
			w.GainExp(p, min(10, amount))
		}
	}
}

// DoJunk implements the junk command — src/act.item.c ACMD(do_drop) with
// subcmd SCMD_JUNK.
func (w *World) DoJunk(ch Actor, arg string) {
	w.doDispose(ch, arg, scmdJunk, "junk", 0)
}

// DoDonate implements the donate command — src/act.item.c ACMD(do_drop)
// with subcmd SCMD_DONATE. A single dice roll up front decides the fate of
// everything in the command (act.item.c:551-567): 25% chance (case 0) the
// donation secretly destroys the item/gold instead — mode silently becomes
// junk, but messages still say "donate" and no exp is granted; otherwise
// draws 1/2 select donation_room_1 (8053), and draw 3 selects
// donation_room_2 (18204). These donation rooms intentionally don't match
// the dump-spec rooms (8085/21223) — a pre-existing inconsistency in the
// original game, preserved here rather than "fixed".
func (w *World) DoDonate(ch Actor, arg string) {
	mode := scmdDonate
	donationRoom := 0

	switch randRange(0, 3) {
	case 0:
		mode = scmdJunk
	case 1, 2:
		donationRoom = DonationRoom1
	case 3:
		donationRoom = DonationRoom2
	}

	// act.item.c:568-572 — RDR==NOWHERE rejects the whole command. RDR only
	// ever gets set on the case 1/2/3 branches, so this is a no-op when the
	// dice picked the junk-destroy outcome (case 0).
	if mode == scmdDonate {
		if _, ok := w.GetRoom(donationRoom); !ok {
			ch.SendMessage("Sorry, you can't donate anything right now.\r\n")
			return
		}
	}

	w.doDispose(ch, arg, mode, "donate", donationRoom)
}
