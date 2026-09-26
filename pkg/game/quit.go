package game

import "fmt"

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

	MudLog(fmt.Sprintf("%s has quit the game.", ch.GetName()), MudlogNormal, max(lvlImmort, ch.GetInvisLevel()), true) // act.other.c:135-136
	ch.SendMessage("Goodbye, friend.. Come back soon!\r\n")

	if !isokquit && !immort {
		// C: free_rent && !(isokquit || immort) -> no Crash_rentsave, so
		// nothing leaves with the character: extract_char drops everything
		// worn and carried where they quit (handler.c:1133-1136), for anyone
		// to pick up.
		MudLog(fmt.Sprintf("LOSTEQ:%s has quit out of a save room.", ch.GetName()), MudlogNormal, max(lvlImmort, ch.GetInvisLevel()), true) // act.other.c:171-173
		if ch.IsMounted() {
			Unmount(ch, w.GetMount(ch))
		}
		return QuitLogoutLoseEQ
	}

	// Safe path: a legal quitter through LVL_IMMORT reloads where they legally
	// quit (act.other.c:167-169 sets GET_LOADROOM only for GET_LEVEL(ch) <=
	// LVL_IMMORT). Above LVL_IMMORT the load room keeps its previous value —
	// typically NOWHERE from init_char (db.c:3078) — so the next login falls
	// back to the immortal start room. DP-1310.
	if ch.GetLevel() <= LVL_IMMORT {
		ch.SetLoadRoom(roomVNum)
	}
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

// RentOut is Crash_rentsave's object pass for a legal quit (objsave.c:912):
// unrentable objects (Crash_is_unrentable: NORENT, keys, negative vnum) are
// destroyed wherever they are, worn, carried or inside a container
// (Crash_extract_norents_from_equipped, Crash_extract_norents). What remains
// is what the rent file keeps.
func (w *World) RentOut(p *Player) {
	held := heldObjects(p)
	for _, obj := range held {
		w.extractNorents(obj, p.GetRoom())
	}
}

func (w *World) extractNorents(obj *ObjectInstance, roomVNum int) {
	if obj == nil {
		return
	}
	for _, child := range append([]*ObjectInstance(nil), obj.Contains...) {
		w.extractNorents(child, roomVNum)
	}
	if IsUnrentable(obj) {
		w.ExtractObject(obj, roomVNum)
	}
}

// ExtractRentedObjects is Crash_extract_objs after the rent file is written:
// the rented objects leave the world with the character instead of dropping
// at extraction. They come back from the store when the character re-enters.
func (w *World) ExtractRentedObjects(p *Player) {
	held := heldObjects(p)
	for _, obj := range held {
		w.ExtractObject(obj, p.GetRoom())
	}
}

// heldObjects is everything a player wears and carries (top level only).
func heldObjects(p *Player) []*ObjectInstance {
	var held []*ObjectInstance
	if p.Equipment != nil {
		for _, obj := range p.Equipment.GetEquippedItems() {
			held = append(held, obj)
		}
	}
	if p.Inventory != nil {
		held = append(held, p.Inventory.FindItems("")...)
	}
	return held
}
