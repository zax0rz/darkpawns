package game

// ---------------------------------------------------------------------------
// Exported wrappers — allow session-level access to unexported doXxx methods.
// Each exported ExecXxx method delegates to the corresponding doXxx. The `me`
// argument is the switched body when the session's descriptor rides a mobile
// (pkg/session passes s.switchedMob, nil otherwise): C command_interpreter
// runs every command as d->character, so a switched mobile answers as the
// mobile (DP-1516's remaining half; the IS_NPC early-returns in the doXxx
// bodies then give C's mob bytes instead of the owning player's).
// ---------------------------------------------------------------------------

// ExecSave saves the player's character data.
func (w *World) ExecSave(ch *Player, me *MobInstance) { w.doSave(ch, me, "save", "") }

// ExecReport shows the player's report (group/health/move summary).
func (w *World) ExecReport(ch *Player, me *MobInstance, arg string) {
	w.doReport(ch, me, "report", arg)
}

// ExecSplit splits coins among a group.
func (w *World) ExecSplit(ch *Player, me *MobInstance, arg string) { w.doSplit(ch, me, "split", arg) }

// ExecWimpy sets the wimpy threshold.
func (w *World) ExecWimpy(ch *Player, me *MobInstance, arg string) { w.doWimpy(ch, me, "wimpy", arg) }

// ExecDisplay sets display preferences.
func (w *World) ExecDisplay(ch *Player, me *MobInstance, arg string) {
	w.doDisplay(ch, me, "display", arg)
}

// ExecTransform transforms into an avatar form.
func (w *World) ExecTransform(ch *Player, me *MobInstance, arg string) {
	w.doTransform(ch, me, "transform", arg)
}

// ExecRide mounts a rideable creature.
func (w *World) ExecRide(ch *Player, me *MobInstance, arg string) { w.doRide(ch, me, "ride", arg) }

// ExecDismount dismounts from a ridden creature.
func (w *World) ExecDismount(ch *Player, me *MobInstance, arg string) {
	w.doDismount(ch, me, "dismount", arg)
}

// ExecYank yanks an item from a follower.
func (w *World) ExecYank(ch *Player, me *MobInstance, arg string) { w.doYank(ch, me, "yank", arg) }

// ExecPeek looks inside a container or inventory.
func (w *World) ExecPeek(ch *Player, me *MobInstance, arg string) { w.doPeek(ch, me, "peek", arg) }

// ExecRecall recalls to home/start location.
func (w *World) ExecRecall(ch *Player, me *MobInstance, arg string) {
	w.doRecall(ch, me, "recall", arg)
}

// ExecAppraise estimates an item's value.
func (w *World) ExecAppraise(ch *Player, me *MobInstance, arg string) {
	w.doAppraise(ch, me, "appraise", arg)
}

// ExecScout scouts adjacent rooms.
func (w *World) ExecScout(ch *Player, me *MobInstance, arg string) { w.doScout(ch, me, "scout", arg) }

// ExecRoll rolls dice (1-100 by default).
func (w *World) ExecRoll(ch *Player, me *MobInstance, arg string) { w.doRoll(ch, me, "roll", arg) }

// ExecVisible makes the player visible.
func (w *World) ExecVisible(ch *Player, me *MobInstance, arg string) {
	w.doVisible(ch, me, "visible", arg)
}

// ExecInactive toggles inactive/auto-away status.
func (w *World) ExecInactive(ch *Player, me *MobInstance, arg string) {
	w.doInactive(ch, me, "inactive", arg)
}

// ExecAFK toggles away-from-keyboard status.
func (w *World) ExecAFK(ch *Player, me *MobInstance, arg string) { w.doAFK(ch, me, "afk", arg) }

// ExecAuto toggles auto-assist mode.
func (w *World) ExecAuto(ch *Player, me *MobInstance, arg string) { w.doAuto(ch, me, "auto", arg) }

// ExecGenWrite writes a bug/typo/idea/todo report.
// cmd should be "bug", "typo", "idea", or "todo".
func (w *World) ExecGenWrite(ch *Player, me *MobInstance, cmd, arg string) {
	w.doGenWrite(ch, me, cmd, arg)
}

// ExecGenTog toggles a named player preference (brief, compact, notell,
// etc.) — key is the real command name the player typed.
func (w *World) ExecGenTog(ch *Player, me *MobInstance, key string) { w.doGenTog(ch, me, key, "") }

// RoomHasFlag reports whether the room at vnum has the named flag (e.g. "death", "no_recall").
func (w *World) RoomHasFlag(vnum int, flag string) bool {
	room := w.GetRoomInWorld(vnum)
	if room == nil {
		return false
	}
	return hasRoomFlag(room, flag)
}

// ExecDrink drinks from a container or fountain.
func (w *World) ExecDrink(ch *Player, me *MobInstance, arg string) {
	w.DoDrink(ch, me, "drink", arg, scmdDrink)
}

// ExecSip sips from a container or fountain.
func (w *World) ExecSip(ch *Player, me *MobInstance, arg string) {
	w.DoDrink(ch, me, "sip", arg, scmdSip)
}

// ExecEat eats a food item.
func (w *World) ExecEat(ch *Player, me *MobInstance, arg string) {
	w.DoEat(ch, me, "eat", arg, scmdEat)
}

// ExecTaste tastes a food item (or sips a drink container).
func (w *World) ExecTaste(ch *Player, me *MobInstance, arg string) {
	w.DoEat(ch, me, "taste", arg, scmdTaste)
}

// ExecPour pours liquid between containers or out onto the ground.
func (w *World) ExecPour(ch *Player, me *MobInstance, arg string) {
	w.DoPour(ch, me, "pour", arg, scmdPour)
}

// ExecFill fills a container from a room fountain.
func (w *World) ExecFill(ch *Player, me *MobInstance, arg string) {
	w.DoPour(ch, me, "fill", arg, scmdFill)
}

// ExecInsult delivers a randomized insult to a named target in the room.
func (w *World) ExecInsult(ch *Player, arg string) { DoInsult(w, ch, arg) }

// ExecDream prints the do_dream flavor message (only meaningful while sleeping).
func (w *World) ExecDream(ch *Player) { DoDream(w, ch) }

// ExecBan handles the "ban" admin command (ported from ban.c do_ban).
// actorName and invis are the acting body's (R4). DoBan logs inside itself, in
// C's order (producer, acknowledgement, ban file); ack delivers the
// acknowledgement bytes.
func (w *World) ExecBan(actorName string, invis int, arg string, ack func(string)) {
	if w.Bans == nil {
		ack("Ban system not initialized.\r\n")
		return
	}
	w.Bans.DoBan("./data/badsites", actorName, invis, arg, ack)
}

// ExecUnban handles the "unban" admin command (ported from ban.c do_unban).
// C's order here is the reverse: the acknowledgement reaches the actor before
// the producer, so ack must write to the descriptor.
func (w *World) ExecUnban(actorName string, invis int, arg string, ack func(string)) {
	if w.Bans == nil {
		ack("Ban system not initialized.\r\n")
		return
	}
	w.Bans.DoUnban("./data/badsites", actorName, invis, arg, ack)
}

// ExecWhod handles the "whod" admin command (ported from whod.c do_whod).
// Returns the message to send to the player.
func (w *World) ExecWhod(ch *Player, arg string, output func(string)) string {
	if w.WhodDisplay == nil {
		return "WHOD not initialized.\r\n"
	}
	return w.WhodDisplay.DoWhod(ch.Name, arg, output)
}
