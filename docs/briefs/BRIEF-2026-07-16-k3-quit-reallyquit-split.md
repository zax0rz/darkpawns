# BRIEF (kimi k3) — split `quit` / `reallyquit` + safe-room contract (DP-1115, O33)

**Owner:** kimi k3 (first run — welcome!). **Gate:** Claude establishes the oracle RED and runs red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** One focused PR. Player-facing message text (including exact whitespace/punctuation) is first-class fidelity — match C byte-for-byte.

## The gap
C has **two** logout subcommands around one `do_quit` (`src/act.other.c:72-181`): `quit` (`SCMD_QUIT`) is safe only in specific temple/home rooms; anywhere else it refuses and directs you to `reallyquit` (`SCMD_REALLY_QUIT`), which logs out **anywhere but loses your equipment**. Go collapsed both into one unrestricted handler (`reallyquit` is a bare alias) that blocks only fighting/death rooms then disconnects — so players keep equipment while quitting anywhere, erasing the safe-logout/rent constraint. Registrations: C `src/interpreter.c:630` (`quit`→SCMD_QUIT) and `:657` (`reallyquit`→SCMD_REALLY_QUIT). Go: `pkg/session/commands.go` quit registration, `pkg/session/cmd_inventory.go:12-44` handler, cleanup/save in `pkg/session/manager.go:821-852`.

## Read-only source of truth
C: `~/.openclaw/workspace/darkpawns-c-oracle/src/act.other.c` `do_quit` (72-181); `src/interpreter.c` registrations. **Never edit the oracle tree.**

## The C contract — reproduce exactly
`do_quit(ch, argument, cmd, subcmd)`:
1. `IS_NPC(ch) || !ch->desc` → return.
2. **Compute `isokquit` (safe-quit room)** by `rm->number`:
   - `8004` → safe. `8008` → safe.
   - `18201` → safe **iff** `hometown == 2`.
   - `21202` → safe **iff** `hometown == 3`. `21258` → safe **iff** `hometown == 3`.
   - default → safe iff `is_owner(ch, rm->number)` (player owns the house), else not.
3. Non-immort with `subcmd` neither `SCMD_QUIT` nor `SCMD_REALLY_QUIT` → `"You have to type quit--no less, to quit!\r\n"` (guards abbreviations; see registration note below).
4. `POS_FIGHTING` → `"No way!  You're fighting for your life!\r\n"`.
5. `POS <= POS_INCAP` → `"You die before your time...\r\n"` then `die(ch)`.
6. **`subcmd != SCMD_REALLY_QUIT && !(isokquit || immort)`** → refuse and **return** (no logout):
   ```
   Type REALLYQUIT to quit the game and lose your eq.
   Return to the temple and QUIT to leave the game and keep your equipment.
   ```
   plus, iff `GET_LEVEL(ch) <= 5`: `"You can type RECALL to return to your temple.\r\n"`.
7. Otherwise **perform logout**:
   - iff not invis: `act("$n has left the game.", TRUE, ch, 0, 0, TO_ROOM)`.
   - mudlog `"%s has quit the game."`.
   - `"Goodbye, friend.. Come back soon!\r\n"`.
   - infobar off if on.
   - **close duplicate sockets:** for every other descriptor whose character has the same `GET_IDNUM`, `close_socket(d)` (anti-dupe).
   - **rent/equipment:** iff `free_rent`:
     - **safe (`isokquit || immort`)** → set loadroom to this room (mortals), `Crash_rentsave` (or `Crash_cryosave` if `PLR_NODELETE`) — **saves with equipment**.
     - **unsafe (reallyquit from a non-safe room)** → **no rentsave**; just mudlog `"LOSTEQ:%s has quit out of a save room."` — the equipment is **not persisted** (lost on reload).
   - `IS_MOUNTED` → `unmount`.
   - `extract_char(ch)` (saves the char record).

## The Go fix (shape)
- Split into two subcommands around **one** game-owned logout op: `quit` (safe-gated) and `reallyquit` (destructive, works anywhere). Mirror C's `SCMD_QUIT`/`SCMD_REALLY_QUIT` — don't leave `reallyquit` as a bare alias.
- Port the `isokquit` room check (the exact vnums + hometown/ownership gates).
- Implement the refuse-and-return path with the exact 2-3 line message block (and the `<=5` RECALL hint).
- Fighting/incap gates with C's messages (Go currently blocks fighting/death rooms — align the messages + add the incap→die path if `die` is modeled; if not, gate it and file a follow-up).
- **Equipment loss:** on the unsafe `reallyquit` path, the character must be saved **without** its worn/carried equipment (match C: no rentsave → eq not persisted). On the safe path, save **with** equipment (current behavior). This is the economic core of the ticket — model it against Go's existing save path (`manager.go:821-852` saves the char today; you need the safe/unsafe fork). If Go has no rent system, the minimal faithful behavior is: unsafe logout persists the char with an **empty** equipment+inventory set; safe logout persists everything.
- Keep duplicate-socket close + dismount if those subsystems exist; otherwise gate cleanly and note as follow-ups.

## Registration note
In C the `"You have to type quit--no less"` guard exists because command abbreviation could reach `do_quit` via a non-quit subcmd. In Go, register `quit`→SCMD_QUIT and `reallyquit`→SCMD_REALLY_QUIT explicitly and make sure no abbreviation of another command dispatches here; if the Go interpreter can't produce that case, you may omit branch 3 — but leave a comment citing why.

## Oracle RED (Claude establishes + gates)
Fresh L1 char starts in room **8162** (Temple Infirmary) — **not** a safe-quit room (safe set is `{8004, 8008, homes, owned house}`). So:
- Probe `quit` from 8162 → C: the `Type REALLYQUIT to quit the game and lose your eq.` block (+ RECALL line, level ≤5). **Go currently just disconnects** → clean message divergence, tier-1.
- (Follow-up green checks Claude runs: `quit` from 8004 logs out with `Goodbye, friend..`; equipment-persistence asserted via save inspection.)
Note: `reallyquit`/safe-`quit` actually disconnect the session, which the harness handles as end-of-probe; the decisive RED is the **refuse** message from an unsafe room (no disconnect). Claude owns the scenario.

## Out of scope
- The full rent/cost economy (`Crash_rentsave` cost math) — model equipment kept-vs-lost, not rent pricing.
- `recall` itself (only the hint line).
- The C oracle tree, `website/static/map/world-sphere.json`, `docs/reports/reek/*`.

## Tests you own (deterministic)
- `quit` from a non-safe room → refuse message, session stays open, char unchanged.
- `quit` from 8004/8008 → logout + `Goodbye, friend..`, saved char **retains** equipment.
- `reallyquit` from a non-safe room → logout, saved char has **no** equipment/inventory.
- Home-room gating: 18201 safe only when `hometown==2`; 21202/21258 only when `hometown==3`.
- Fighting → refuse with C's message.

## PR hygiene
- Commits end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- PR body ends with: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`
