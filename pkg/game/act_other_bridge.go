package game

// ---------------------------------------------------------------------------
// Exported wrappers — allow session-level access to unexported doXxx methods.
// Each exported ExecXxx method delegates to the corresponding doXxx, passing
// nil for the mob instance (since player sessions don't have one).
// ---------------------------------------------------------------------------

// ExecSave saves the player's character data.
func (w *World) ExecSave(ch *Player) { w.doSave(ch, nil, "save", "") }

// ExecReport shows the player's report (group/health/move summary).
func (w *World) ExecReport(ch *Player, arg string) { w.doReport(ch, nil, "report", arg) }

// ExecSplit splits coins among a group.
func (w *World) ExecSplit(ch *Player, arg string) { w.doSplit(ch, nil, "split", arg) }

// ExecWimpy sets the wimpy threshold.
func (w *World) ExecWimpy(ch *Player, arg string) { w.doWimpy(ch, nil, "wimpy", arg) }

// ExecDisplay sets display preferences.
func (w *World) ExecDisplay(ch *Player, arg string) { w.doDisplay(ch, nil, "display", arg) }

// ExecTransform transforms into an avatar form.
func (w *World) ExecTransform(ch *Player, arg string) { w.doTransform(ch, nil, "transform", arg) }

// ExecRide mounts a rideable creature.
func (w *World) ExecRide(ch *Player, arg string) { w.doRide(ch, nil, "ride", arg) }

// ExecDismount dismounts from a ridden creature.
func (w *World) ExecDismount(ch *Player, arg string) { w.doDismount(ch, nil, "dismount", arg) }

// ExecYank yanks an item from a follower.
func (w *World) ExecYank(ch *Player, arg string) { w.doYank(ch, nil, "yank", arg) }

// ExecPeek looks inside a container or inventory.
func (w *World) ExecPeek(ch *Player, arg string) { w.doPeek(ch, nil, "peek", arg) }

// ExecRecall recalls to home/start location.
func (w *World) ExecRecall(ch *Player, arg string) { w.doRecall(ch, nil, "recall", arg) }

// ExecStealth toggles stealth movement.
func (w *World) ExecStealth(ch *Player, arg string) { w.doStealth(ch, nil, "stealth", arg) }

// ExecAppraise estimates an item's value.
func (w *World) ExecAppraise(ch *Player, arg string) { w.doAppraise(ch, nil, "appraise", arg) }

// ExecScout scouts adjacent rooms.
func (w *World) ExecScout(ch *Player, arg string) { w.doScout(ch, nil, "scout", arg) }

// ExecRoll rolls dice (1-100 by default).
func (w *World) ExecRoll(ch *Player, arg string) { w.doRoll(ch, nil, "roll", arg) }

// ExecVisible makes the player visible.
func (w *World) ExecVisible(ch *Player, arg string) { w.doVisible(ch, nil, "visible", arg) }

// ExecInactive toggles inactive/auto-away status.
func (w *World) ExecInactive(ch *Player, arg string) { w.doInactive(ch, nil, "inactive", arg) }

// ExecAFK toggles away-from-keyboard status.
func (w *World) ExecAFK(ch *Player, arg string) { w.doAFK(ch, nil, "afk", arg) }

// ExecAuto toggles auto-assist mode.
func (w *World) ExecAuto(ch *Player, arg string) { w.doAuto(ch, nil, "auto", arg) }

// ExecGenWrite writes a bug/typo/idea/todo report.
// cmd should be "bug", "typo", "idea", or "todo".
func (w *World) ExecGenWrite(ch *Player, cmd, arg string) { w.doGenWrite(ch, nil, cmd, arg) }

// ExecGenTog toggles a player option (brief, compact, notell, etc.).
func (w *World) ExecGenTog(ch *Player, arg string) { w.doGenTog(ch, nil, "gentog", arg) }
