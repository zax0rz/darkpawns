package game

// persistence_seam.go — the store-of-record save seam (DP-1365). Login
// reads the character from the database (pkg/session/session_login.go →
// db.GetPlayer), so every C save_char + Crash_crashsave point must write
// that store. pkg/game has no store handle by design; instead the server
// sets World.PlayerSaver to a callback that finds the player's session and
// performs the write through Session.saveCharacter, which preserves
// session-owned record fields. A nil PlayerSaver (tests, no store) means a
// save is SKIPPED, never that it succeeded.

// SaveResult is what the store of record did with a save request.
type SaveResult int

const (
	// SaveSkipped: no store, no session, or an ineligible character. The
	// crash flag is retained so a later eligible save retries.
	SaveSkipped SaveResult = iota
	// SaveSucceeded: the record was written to the store of record.
	SaveSucceeded
	// SaveFailed: a write was attempted and errored.
	SaveFailed
)

// SaveKind preserves each caller's C save kind (DP-1365: share the write
// machinery, keep the caller's semantics).
type SaveKind int

const (
	// SaveCharOnly is a bare C save_char site: it writes the character
	// record and leaves PLR_CRASH alone.
	SaveCharOnly SaveKind = iota
	// SaveCrash is a C save_char + Crash_crashsave site (do_save,
	// Crash_save_all, check_idling's void pull): a successful write clears
	// PLR_CRASH, as C's Crash_crashsave does (objsave.c:823).
	SaveCrash
)

// PlayerSaver writes one player's record to the store of record. The server
// wires it to the player's session; returning without a write must be
// reported as SaveSkipped, not SaveSucceeded.
type PlayerSaver func(p *Player, why string, loadRoom int) SaveResult

// SavePlayerRecord is the one save seam every game-layer save point calls.
// kind selects the caller's C save kind: SaveCrash clears PLR_CRASH after a
// successful write — and only if no inventory change raced the save, in
// which case the newer change keeps the flag for the next autosave. On a
// failed or skipped write the flag is retained for retry and the failure is
// logged by the seam; no player-visible text is added. This retry-on-
// failure is a deliberate durability exception to C's autosave behavior
// (C's Crash_save_all simply drops a failed save); "don't port C's data
// loss" governs.
func (w *World) SavePlayerRecord(p *Player, why string, loadRoom int, kind SaveKind) SaveResult {
	if w.PlayerSaver == nil || p == nil {
		return SaveSkipped
	}
	seq := p.CrashSeq()
	res := w.PlayerSaver(p, why, loadRoom)
	if kind == SaveCrash && res == SaveSucceeded {
		p.ClearCrashFlagIfUnchanged(seq)
	}
	return res
}

// -----------------------------------------------------------------------
// PLR_CRASH bookkeeping (C handler.c:569-571 obj_to_char, 596-598
// obj_from_char; cleared by Crash_crashsave, objsave.c:823, and
// Crash_save_all, objsave.c:1218).
// -----------------------------------------------------------------------

// MarkCrashNeeded sets PLR_CRASH and bumps the inventory-dirty generation.
// C sets the flag wherever an object enters or leaves a player's carrying
// list, for non-NPCs; the port's central points are the ObjectLocation
// attach/detach arms for player inventory and equipment.
func (p *Player) MarkCrashNeeded() {
	p.mu.Lock()
	p.Flags |= 1 << uint(PlrCrash)
	p.crashSeq++
	p.mu.Unlock()
}

// CrashSeq is the inventory-dirty generation: it changes on every
// MarkCrashNeeded, so a save can tell whether inventory changed after its
// snapshot was taken.
func (p *Player) CrashSeq() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.crashSeq
}

// NeedsCrashSave reports PLR_CRASH (C PLR_FLAGGED(ch, PLR_CRASH)).
func (p *Player) NeedsCrashSave() bool {
	return p.GetFlags()&(1<<uint(PlrCrash)) != 0
}

// ClearCrashFlagIfUnchanged clears PLR_CRASH only when the dirty generation
// still matches the save's snapshot — a change made after the saved
// snapshot keeps the flag. Caller: only the crash-save success path.
func (p *Player) ClearCrashFlagIfUnchanged(seq uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.crashSeq == seq {
		p.Flags &^= 1 << uint(PlrCrash)
	}
}
