# BRIEF (kimi k3) — `recall` fresh-char fidelity: port no-ops + re-skinned messages (DP-recall)

**Owner:** kimi k3. **Gate:** Claude establishes the oracle RED and runs `room-desc-exits` red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** One focused PR. `recall` is a player-facing command — match C byte-for-byte, including gate messages, room broadcasts, the hometown-dependent target, and the fact that the recaller sees **only the new room's look output**.

> **SCOPE GUARDRAILS (read first).** Do EXACTLY these two things and nothing else: (1) fix `doRecall` in `pkg/game/other_utility.go` to match C, and (2) symmetrize the `[setup:port]` block in `cmd/dp-oracle-diff/scenarios/room-desc-exits.txt` to match `[setup:oracle]` (append `look`/`recall`). Do NOT refactor neighboring code, do NOT touch other commands or files, do NOT "improve" the surrounding recall/teleport machinery, and do NOT attempt the suite-wide `[setup:port]` hygiene mentioned at the bottom of this brief (that is a SEPARATE task — skip it). One function + one scenario file. If something outside that scope looks broken, note it in the PR description; do not fix it here.

## The gap
Surfaced while gating DP-1173 (character-creation 1:1): the committed scenario `cmd/dp-oracle-diff/scenarios/room-desc-exits.txt` is **RED**, and the root cause is `recall`, not the creation change (it reproduces identically at the pre-DP-1173 commit `1ceec6bb`).

Both servers land a fresh char in **Temple Infirmary 8162** (single north exit). The scenario's `[setup:oracle]` block ends with `recall`; C moves the char to **"At the Temple Altar" (8004)** — the default `mortal_start_room` — and the room-sensitive probe (`look`/`exits`/`quit`) then matches the anchor room. The Go port's `recall` is broken **two ways**:

1. **Functional:** the character does not move. Adding `look`+`recall` to `[setup:port]` (symmetrizing with the oracle block) leaves the port standing in 8162 — proven by direct test. So even with the setup symmetrized, the scenario stays red until `recall` actually relocates the char.
2. **Cosmetic / message parity:** even where it "works," the port's recall is a **re-skin**, not a port of C. Gate messages, room broadcasts, and the self-output all diverge from C (details below).

## Read-only source of truth
C (never edit the oracle tree): `~/.openclaw/workspace/darkpawns-c-oracle/src/act.other.c` — `do_recall` (**1727-1748**); `~/.openclaw/workspace/darkpawns-c-oracle/src/spells.c` — `spell_recall` (**124-163**).
Go: `pkg/game/other_utility.go` — `doRecall` (**87-122**); `pkg/session/cmd_misc.go` — `cmdRecall` (**67**); `pkg/game/act_other_bridge.go` — `ExecRecall` (**40**). Room-flag plumbing: `pkg/game/room_flags.go`, `pkg/game/constants.go:135` (`RoomBitNames`).

## The C contract — reproduce exactly

**`do_recall` (act.other.c:1727) — gates, in this order:**
```c
if (GET_LEVEL(ch)>5 || IS_NPC(ch)) {           // "This command is not available for someone of your experience!\r\n"
if (ROOM_FLAGGED(ch->in_room, ROOM_BFR)) {     // "You can't recall from this magickal place.\r\n"
if (FIGHTING(ch)) {                            // "Your concentration is broken by your fighting!"  (NOTE: no \r\n in this one)
spell_recall(30, ch, ch, NULL, NULL);          // the actual work
```
Only `ROOM_BFR` (bit 17) blocks by room — there is **no `no_recall` room-flag check** in C. Drop the port's extra `hasRoomFlag(room, "no_recall")` clause.

**`spell_recall(level, ch, victim=ch, ...)` (spells.c:124):**
```c
if (victim == NULL || IS_NPC(victim)) return;
if (ROOM_FLAGGED(ch->in_room, ROOM_BFR) || ROOM_FLAGGED(victim->in_room, ROOM_BFR)) {
    // "Your magic ebbs and dissolves as you lose your concentration.\r\n"  (to victim); return
}
if (FIGHTING(ch)) { /* "Your concentration is broken by your fighting!\r\n" */ return; }

act("$n disappears.", TRUE, victim, 0, 0, TO_ROOM);   // old room sees "<Name> disappears."  (victim sees NOTHING)
char_from_room(victim);
if (GET_HOME(victim) == 2)      char_to_room(victim, real_room(kiroshi_start_room));
else if (GET_HOME(victim) == 3) char_to_room(victim, real_room(alaozar_start_room));
else                            char_to_room(victim, real_room(mortal_start_room));
/* unmount handling */
act("$n appears in the middle of the room.", TRUE, victim, 0, 0, TO_ROOM);  // new room sees "<Name> appears in the middle of the room."
if (AWAKE(victim)) look_at_room(victim, 0);           // victim sees ONLY the new room's look
else               stc("You have a strange dream about falling..\r\n", victim);
```

**Net player-facing output for the recaller (awake, default home):** nothing but the **new room's `look`**. No "You close your eyes and pray...", no "You are recalled!". Those are fabrications in the port and must go.

**Target is hometown-dependent — do NOT hardcode 8004.** Use `GET_HOME(victim)`: 2→`kiroshi_start_room`, 3→`alaozar_start_room`, else→`mortal_start_room`. For the default-home test char that resolves to 8004 (Temple Altar), which is why the scenario expects 8004 — but the code must key on home, matching the existing hometown table used elsewhere in the port (same 2/3/else split as quit's `isokquit`, see `pkg/game/quit.go`).

## The port's current divergence (other_utility.go:87-122)
- Gate messages all wrong (see above); extra bogus `no_recall` check.
- Fabricated self-messages `"You close your eyes and pray...\r\n"` and `"You are recalled!\r\n"`.
- Room broadcasts wrong: port emits `"<Name> closes his eyes and prays..."` (old room) and `"<Name> appears in the room."` (new room); C emits `"<Name> disappears."` and `"<Name> appears in the middle of the room."`.
- Hardcoded `recallRoom := 8004` instead of the home-based target.
- **And the char isn't actually relocating** — see diagnostic note.

## Diagnostic note (root-cause the no-op first)
For the test char none of the gates should fire: level 1, not fighting, room 8162 flag bitvector = `28` = PEACEFUL|INDOORS|NOMOB (bits 2,3,4) — **no BFR (bit 17), no no_recall**. Dispatch is wired (`cmdRecall → ExecRecall → doRecall`). Yet `ch.SetRoom(8004)` doesn't take: the follow-up `look` still shows 8162. Two candidates to confirm with a throwaway `fmt.Fprintf(os.Stderr,...)` (revert before final):
1. A gate fires anyway (room-flag parse/name-resolution mismatch — verify `hasRoomFlag(8162,"bfr")` is false and no early `return`).
2. `SetRoom`/room-keying no-ops for the recall target (vnum vs real-room-index, or 8004 not resolving) — note ordinary movement relocates fine, so compare how movement sets the room vs how `doRecall` does.

Add a focused `pkg/game` unit test: fresh L1 player in 8162 → `ExecRecall` → assert `GetRoomVNum()==<home target>` and assert the exact message bytes.

## Gate (Claude owns the red→green)
- Symmetrize `cmd/dp-oracle-diff/scenarios/room-desc-exits.txt` `[setup:port]` to match `[setup:oracle]` (append `look`/`recall`, and unify the creation keystrokes — see below). The fix must drive `--scenario room-desc-exits` from RED to **`no normalized divergence`**.
- Re-run the full suite; nothing else may regress. `combat-swing` stays red (separate DRAFT/mobact-parity item, out of scope).

## Secondary (optional, low-pri) — stale `[setup:port]` creation keystrokes
Independent hygiene surfaced in the same gate: **all ~28 `[setup:port]` blocks** still encode the pre-DP-1173 creation flow (lowercase `y` name-confirm + a stray `Y` that used to answer the deleted "Create new character?" prompt). Now that port creation is 1:1, the stray `Y` is consumed harmlessly inside creation (it answers C's ANSI question; the following `N` then re-prompts at the sex gate), so scenarios stay green — but it's fragile. Safe mechanical cleanup: make each `[setup:port]` creation portion byte-identical to its `[setup:oracle]` (uppercase the confirm, delete the stray `Y`); preserve any post-creation navigation. Provably position-neutral (the stray input never reaches the game world). Fold in only if it doesn't bloat the PR — the recall fix is the priority.
