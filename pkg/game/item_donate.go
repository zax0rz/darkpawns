package game

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// performDispose implements perform_drop() for the junk/donate paths —
// src/act.item.c:480-525. mode is scmdJunk or scmdDonate; donationRoom is
// the dice-selected destination room for a donate (unused for junk).
// Returns the junk-value reward (0 for a successful donate, or when the
// item couldn't be disposed of at all).
func (w *World) performDispose(ch *Player, obj *ObjectInstance, mode int, sname string, donationRoom int) int {
	if obj.HasExtraFlag(0, extraFlagNoDrop) && ch.GetLevel() < lvlImmort {
		w.actToChar(ch, fmt.Sprintf("You can't %s $p, it must be CURSED!", sname), obj, nil)
		return 0
	}

	// VANISH(mode) — act.item.c:477-478. Always applies here since this
	// helper is only ever called with mode == junk or donate.
	const vanish = "  It vanishes in a puff of smoke!"
	w.actToChar(ch, fmt.Sprintf("You %s $p.%s", sname, vanish), obj, nil)
	w.actToRoom(ch, fmt.Sprintf("$n %ss $p.%s", sname, vanish), obj, nil)
	ch.Inventory.removeItem(obj)

	// A donated item flagged NODONATE is silently junked instead — act.item.c:498-499.
	if mode == scmdDonate && obj.HasExtraFlag(0, extraFlagNoDonate) {
		mode = scmdJunk
	}

	switch mode {
	case scmdDonate:
		if err := w.MoveObjectToRoom(obj, donationRoom); err != nil {
			slog.Error("donation move failed",
				"player", ch.GetName(), "item_vnum", obj.VNum, "error", err)
			ch.SendMessage("Something went wrong. Your item was not donated.\r\n")
			//nolint:errcheck // best-effort rollback — player already saw failure message
			ch.Inventory.AddItem(obj)
			return 0
		}
		w.roomMessage(donationRoom, strings.ReplaceAll("$p suddenly appears in a puff a smoke!", "$p", obj.GetShortDesc()))
		return 0
	case scmdJunk:
		value := min(max(obj.GetCost()>>4, 1), 200)
		w.ExtractObject(obj, ch.GetRoomVNum())
		return value
	}
	return 0
}

// performDisposeGold implements perform_drop_gold() for the junk/donate
// gold paths — src/act.item.c:430-474. mode is scmdJunk or scmdDonate.
func (w *World) performDisposeGold(ch *Player, amount int, mode int, donationRoom int) {
	if amount <= 0 {
		ch.SendMessage("Heh heh heh.. we are jolly funny today, eh?\r\n")
		return
	}
	if ch.GetGold() < amount {
		ch.SendMessage("You don't have that many coins!\r\n")
		return
	}

	if mode == scmdJunk {
		w.actToRoom(ch, fmt.Sprintf("$n drops %s which disappears in a puff of smoke!", createMoneyDesc(amount)), nil, nil)
		ch.SendMessage("You drop some gold which disappears in a puff of smoke!\r\n")
	} else {
		ch.SetWaitState(1) // WAIT_STATE(ch, PULSE_VIOLENCE) — discourage coin-bombing
		ch.SendMessage("You throw some gold into the air where it disappears in a puff of smoke!\r\n")
		w.actToRoom(ch, "$n throws some gold into the air where it disappears in a puff of smoke!", nil, nil)
		moneyObj := w.createMoneyObject(amount)
		if err := w.MoveObjectToRoom(moneyObj, donationRoom); err != nil {
			slog.Error("donation gold move failed",
				"player", ch.GetName(), "error", err)
			ch.SendMessage("Something went wrong. Your gold was not donated.\r\n")
			return
		}
		w.roomMessage(donationRoom, strings.ReplaceAll("$p suddenly appears in a puff of orange smoke!", "$p", moneyObj.GetShortDesc()))
	}

	ch.SetGold(ch.GetGold() - amount)
}

// doDispose implements the shared argument handling of ACMD(do_drop) for
// the junk/donate subcommands — src/act.item.c:529-666. sname is the
// literal typed command ("junk" or "donate"); the junk-only exp reward is
// gated on sname, not on any per-item mode flip (donate never grants exp,
// even when the dice roll secretly junks an item).
func (w *World) doDispose(ch *Player, arg string, mode int, sname string, donationRoom int) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage(fmt.Sprintf("What do you want to %s?\r\n", sname))
		return
	}

	parts := strings.Fields(arg)
	arg1 := parts[0]

	if amount, err := strconv.Atoi(arg1); err == nil {
		rest := strings.Join(parts[1:], " ")
		if rest == "coins" || rest == "coin" {
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
		items := make([]*ObjectInstance, len(ch.Inventory.Items))
		copy(items, ch.Inventory.Items)
		found := false
		for _, obj := range items {
			if isname(keyword, obj.GetKeywords()) {
				found = true
				amount += w.performDispose(ch, obj, mode, sname, donationRoom)
			}
		}
		if !found {
			ch.SendMessage(fmt.Sprintf("You don't seem to have any %ss.\r\n", keyword))
		}
	default:
		var obj *ObjectInstance
		for _, o := range ch.Inventory.Items {
			if isname(arg1, o.GetKeywords()) {
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
		w.actToRoom(ch, "$n has been rewarded by the gods!", nil, nil)
		w.GainExp(ch, min(10, amount))
	}
}

// DoJunk implements the junk command — src/act.item.c ACMD(do_drop) with
// subcmd SCMD_JUNK.
func (w *World) DoJunk(ch *Player, arg string) {
	w.doDispose(ch, arg, scmdJunk, "junk", 0)
}

// junkCheapItems removes inventory items with a base cost of 150 or less,
// matching the attitude_loot() behavior in src/fight.c:1128.
func (w *World) junkCheapItems(ch *Player) {
	items := make([]*ObjectInstance, len(ch.Inventory.Items))
	copy(items, ch.Inventory.Items)
	for _, obj := range items {
		if obj.GetCost() <= 150 {
			w.performDispose(ch, obj, scmdJunk, "junk", 0)
		}
	}
}

// DoDonate implements the donate command — src/act.item.c ACMD(do_drop)
// with subcmd SCMD_DONATE. A single dice roll up front decides the fate of
// everything in the command (act.item.c:551-567): 25% chance (case 0) the
// donation secretly destroys the item/gold instead — mode silently becomes
// junk, but messages still say "donate" and no exp is granted; otherwise
// the destination room is picked 50/50 between donation_room_1 (8053) and
// donation_room_2 (18204). These donation rooms intentionally don't match
// the dump-spec rooms (8085/21223) — a pre-existing inconsistency in the
// original game, preserved here rather than "fixed".
func (w *World) DoDonate(ch *Player, arg string) {
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
