package game

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
