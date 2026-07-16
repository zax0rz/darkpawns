package game

// ---------------------------------------------------------------------------
// quit / reallyquit — C do_quit (src/act.other.c:72-181)
//
// C has two subcommands around one do_quit: quit (SCMD_QUIT) logs out safely
// only from temple/home/owned rooms; reallyquit (SCMD_REALLY_QUIT) logs out
// anywhere but, outside a safe room, skips Crash_rentsave so the quitter's
// equipment is not persisted (LOSTEQ). This file owns the decision logic and
// the equipment fork; the session layer owns descriptor teardown (room
// broadcast, infobar, duplicate-socket close, save, close).
// ---------------------------------------------------------------------------

// QuitOutcome is the game-owned decision for one do_quit invocation. The
// session layer tears down only on the QuitLogout* outcomes.
type QuitOutcome int

const (
	// QuitRefused — unsafe room: the REALLYQUIT block was sent, no logout.
	QuitRefused QuitOutcome = iota
	// QuitFighting — POS_FIGHTING gate, no logout.
	QuitFighting
	// QuitDied — POS <= POS_INCAP: "You die before your time..." + die(ch).
	QuitDied
	// QuitLogoutKeepEQ — logout; the save retains equipment (safe room/immort).
	QuitLogoutKeepEQ
	// QuitLogoutLoseEQ — logout; inventory+equipment were stripped pre-save.
	QuitLogoutLoseEQ
)

// DoQuit ports C do_quit. reallyQuit mirrors subcmd == SCMD_REALLY_QUIT.
//
// C's "You have to type quit--no less, to quit!" guard (branch on subcmd) is
// deliberately omitted: it exists because C's interpreter dispatches command
// abbreviations into do_quit with a non-quit subcmd. The Go command registry
// (pkg/command) resolves only exact command names, and quit/reallyquit are
// registered as two explicit entries, so that case cannot arise here.
func (w *World) DoQuit(ch *Player, reallyQuit bool) QuitOutcome {
	if ch == nil || ch.IsNPC() {
		// C: IS_NPC(ch) || !ch->desc -> return. Players reaching this from a
		// session always have a descriptor.
		return QuitRefused
	}

	roomVNum := ch.GetRoom()
	isokquit := w.isSafeQuitRoom(ch, roomVNum)
	immort := ch.GetLevel() >= LVL_IMMORT

	switch {
	case ch.GetPosition() == posFighting:
		ch.SendMessage("No way!  You're fighting for your life!\r\n")
		return QuitFighting
	case ch.GetPosition() <= posIncapacitated:
		ch.SendMessage("You die before your time...\r\n")
		// The C position implies non-positive hit points. Preserve die(ch)
		// even if a caller constructed an inconsistent positive-HP state.
		if ch.GetHP() > 0 {
			ch.SetHP(0)
		}
		w.HandleNonCombatDeath(ch) // C die(ch) — fight.c die(), no logout follows
		return QuitDied
	case !reallyQuit && !isokquit && !immort:
		msg := "Type REALLYQUIT to quit the game and lose your eq.\r\n" +
			"Return to the temple and QUIT to leave the game and keep your equipment.\r\n"
		if ch.GetLevel() <= 5 {
			msg += "You can type RECALL to return to your temple.\r\n"
		}
		ch.SendMessage(msg)
		return QuitRefused
	}

	BasicMudLogf("%s has quit the game.", ch.GetName())
	ch.SendMessage("Goodbye, friend.. Come back soon!\r\n")

	if !isokquit && !immort {
		// C: free_rent && !(isokquit || immort) -> no Crash_rentsave; the
		// equipment is not persisted. The Go port has no rent system, so the
		// faithful minimal model is to strip carried objects before the save.
		BasicMudLogf("LOSTEQ:%s has quit out of a save room.", ch.GetName())
		w.stripAllCarriedObjects(ch)
		if ch.IsMounted() {
			Unmount(ch, w.GetMount(ch))
		}
		return QuitLogoutLoseEQ
	}

	// Safe path: C sets GET_LOADROOM to this room so the player reloads where
	// they legally quit; the Go save already persists the current room.
	if ch.IsMounted() {
		Unmount(ch, w.GetMount(ch))
	}
	return QuitLogoutKeepEQ
}

// isSafeQuitRoom computes C's isokquit (src/act.other.c:87-112): temples
// 8004/8008, home rooms gated on hometown, otherwise house ownership.
func (w *World) isSafeQuitRoom(ch *Player, roomVNum int) bool {
	switch roomVNum {
	case 8004, 8008:
		return true
	case 18201:
		return ch.GetHometown() == 2
	case 21202, 21258:
		return ch.GetHometown() == 3
	default:
		return isOwner(w, ch, roomVNum)
	}
}

// stripAllCarriedObjects extracts every equipped and carried object so the
// upcoming save persists the character with empty equipment and inventory —
// the LOSTEQ half of C's rent fork. Objects vanish (as in C, where no rent
// file is written); gold and other char-record fields are untouched.
func (w *World) stripAllCarriedObjects(ch *Player) {
	var carried []*ObjectInstance
	if ch.Equipment != nil {
		carried = append(carried, ch.Equipment.extractAll()...)
	}
	if ch.Inventory != nil {
		carried = append(carried, ch.Inventory.extractAll()...)
	}
	for _, item := range carried {
		if item == nil {
			continue
		}
		// Detach first so ExtractObject does not try to move an equipped item
		// through a potentially full inventory on this destructive path.
		item.Location = LocNowhere()
		w.ExtractObject(item, ch.GetRoom()) // recursive over contents
	}
}
