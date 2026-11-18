package game

import "log/slog"

// saveCharSiteInRoom is C's save_char(vict, vict->in_room) at the clan
// command sites (clan.c:233,266,378,431,486,539,627,718,983,1138): a
// character-only store-of-record write whose record carries the character's
// current room as the load room.
func (w *World) saveCharSiteInRoom(p *Player, why string) {
	if p == nil {
		return
	}
	if res := w.SavePlayerRecord(p, why, p.GetRoomVNum(), SaveCharOnly); res == SaveFailed {
		slog.Error("store-of-record save failed", "player", p.GetName(), "why", why)
	}
}

// ExecClanCommandActor routes clan commands for either a player or a switched
// mobile body (src/clan.c:1541).
func (w *World) ExecClanCommandActor(actor Actor, argument string) {
	if p, ok := actor.(*Player); ok {
		w.ExecClanCommand(p, argument)
		return
	}
	if actor == nil {
		return
	}
	arg1, arg2 := halfChop(argument)
	switch {
	case isAbbrev(arg1, "create"):
		if arg2 == "" {
			w.sendClanFormat(actor)
		} else {
			actor.SendMessage("You are not mighty enough to create new clans!\r\n")
		}
	case isAbbrev(arg1, "destroy"):
		if arg2 == "" {
			w.sendClanFormat(actor)
		} else {
			actor.SendMessage("Your not mighty enough to destroy clans!\r\n")
		}
	case isAbbrev(arg1, "who"), isAbbrev(arg1, "status"):
		actor.SendMessage("You do not belong to a clan!\r\n")
	case isAbbrev(arg1, "info"):
		w.doClanInfo(actor, arg2)
	case isAbbrev(arg1, "members"), isAbbrev(arg1, "quit"):
		actor.SendMessage("You aren't in a clan!\r\n")
	case isAbbrev(arg1, "enroll"), isAbbrev(arg1, "expel"), isAbbrev(arg1, "demote"),
		isAbbrev(arg1, "promote"), isAbbrev(arg1, "set"), isAbbrev(arg1, "private"),
		isAbbrev(arg1, "withdraw"), isAbbrev(arg1, "deposit"):
		if arg2 == "" && !isAbbrev(arg1, "enroll") {
			w.sendClanFormat(actor)
		} else {
			actor.SendMessage("You don't belong to any clan!\r\n")
		}
	default:
		w.sendClanFormat(actor)
	}
}

func (w *World) ExecClanCommand(ch *Player, argument string) {
	arg1, arg2 := halfChop(argument)

	if isAbbrev(arg1, "rename") {
		w.doClanRename(ch, arg2)
		return
	}
	if isAbbrev(arg1, "create") {
		w.doClanCreate(ch, arg2)
		return
	}
	if isAbbrev(arg1, "destroy") {
		w.doClanDestroy(ch, arg2)
		return
	}
	if isAbbrev(arg1, "enroll") {
		w.doClanEnroll(ch, arg2)
		return
	}
	if isAbbrev(arg1, "expel") {
		w.doClanExpel(ch, arg2)
		return
	}
	if isAbbrev(arg1, "who") {
		w.doClanWho(ch)
		return
	}
	if isAbbrev(arg1, "status") {
		w.doClanStatus(ch)
		return
	}
	if isAbbrev(arg1, "info") {
		w.doClanInfo(ch, arg2)
		return
	}
	if isAbbrev(arg1, "apply") {
		w.doClanApply(ch, arg2)
		return
	}
	if isAbbrev(arg1, "demote") {
		w.doClanDemote(ch, arg2)
		return
	}
	if isAbbrev(arg1, "promote") {
		w.doClanPromote(ch, arg2)
		return
	}
	if isAbbrev(arg1, "members") {
		w.doClanMembers(ch)
		return
	}
	if isAbbrev(arg1, "quit") {
		w.doClanQuit(ch)
		return
	}
	if isAbbrev(arg1, "set") {
		w.doClanSet(ch, arg2)
		return
	}
	if isAbbrev(arg1, "private") {
		w.doClanPrivate(ch, arg2)
		return
	}
	if isAbbrev(arg1, "withdraw") {
		w.doClanBank(ch, arg2, CBWithdraw)
		return
	}
	if isAbbrev(arg1, "deposit") {
		w.doClanBank(ch, arg2, CBDeposit)
		return
	}

	w.sendClanFormat(ch)
}

// doClanBank handles bank operations for a clan (deposit/withdraw).
// In C: do_clan_bank()
