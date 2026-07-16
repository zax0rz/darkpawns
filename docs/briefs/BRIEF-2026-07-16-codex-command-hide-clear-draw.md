# BRIEF (codex) — per-command `AFF_HIDE` clear draw (DP-1170 real root cause)

**Owner:** codex (frontier). **Gate:** Claude runs the differential oracle red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** Sized to one small PR.

## TL;DR
C's `command_interpreter` draws `number(0, 3)` at the **top of every player command** and, on a 0, clears the actor's `AFF_HIDE`. The Go port does **not** — its in-game command dispatch (`ExecuteCommand`) draws nothing per command. That missing draw is a **per-command shared-stream desync** and is the **confirmed, complete root cause** of the DP-1170 steal divergence (see proof below). It also is a real behavior bug: a hidden Go player is never revealed by acting.

This is a tiny fix but a **foundational** one: it shifts the shared PRNG stream by one draw for *every* in-game command, so it silently desynced every roll-sensitive oracle probe that followed any command. Getting it right unblocks live command-driven scenarios.

## The C source (read-only)
`~/.openclaw/workspace/darkpawns-c-oracle/src/interpreter.c`, `command_interpreter()` — the **first two statements**, before command lookup/dispatch:
```c
void command_interpreter(struct char_data *ch, char *argument)
{
  int cmd, length;
  ...
  if (!number(0,3))
    REMOVE_BIT_AR(AFF_FLAGS(ch), AFF_HIDE);   /* interpreter.c:889-890 */

  skip_spaces(&argument);
  if (!*argument)
    return;
  ... /* command lookup + dispatch */
}
```
Semantics that matter:
- **One `number(0,3)` draw per command, unconditionally**, drawn *before* the command is even looked up — so it fires for valid, invalid, and no-op commands alike.
- It is the **first** RNG draw associated with the command; the command's own handler (steal, hide, cast, …) draws *after* it. Order is law.
- `command_interpreter` runs only for **playing** characters. Character creation / the menu go through `nanny()` in C, which does **not** draw this. So the draw must **not** fire during login/creation.
- On a 0 (1-in-4), clear the actor's hide affect. (This is why a hidden player is revealed ~25% of the time per action — a behavior the port is also missing.)

## The Go fix
Entry point: `pkg/session/commands.go` → `func ExecuteCommand(s *Session, cmdStr string, args []string)`. This is the in-game command dispatch (the `command_interpreter` analog). Add, as the **very first thing** in the function (before the moderation pre-check, alias expansion, and command lookup):

```go
// C command_interpreter draws number(0,3) at the top of EVERY player command
// and clears AFF_HIDE on 0 (interpreter.c:889-890). This draw is first, before
// any command handler, and desyncs the shared stream if omitted (DP-1170).
// #nosec G404 — game RNG, not cryptographic
if s.player != nil {
    if dprng.Number(0, 3) == 0 {
        s.player.SetAffect(affHide, false) // affHide == 1
    }
}
```
- Import `github.com/zax0rz/darkpawns/pkg/dprng`. The hide-affect bit is `1` (see `SetAffect` usage in `pkg/game/skill_stealth.go` / `act_movement.go`; there's no exported `affHide` in `pkg/session`, so use the literal `1` with a comment, or add/exported a named constant — your call, but keep it obviously the hide bit).
- **Gate on playing state.** Confirm `ExecuteCommand` is only reached for in-game commands, not during account login / char-creation / the entry menu (in C those are `nanny`, no draw). `s.player != nil` was sufficient in a spike, but verify there's no creation/menu path that reaches `ExecuteCommand` with a non-nil player — if there is, gate on the actual "playing" session state instead.
- **Placement is first, once per dispatched command.** Put it above the moderation check and alias expansion. One `ExecuteCommand` call = one draw, matching C's one-per-`command_interpreter`.

### Edge case to note (don't over-engineer, just be aware)
C draws once per `command_interpreter` call. A multi-command alias (`;`-separated) in C runs `command_interpreter` per sub-command → one draw each. Go expands aliases inside `ExecuteCommand`; if a single `ExecuteCommand` call fans out to multiple sub-commands, C would draw per sub-command. The common case (one typed command → one draw) is the priority; match that. If you can cheaply make multi-command aliases draw per sub-command, do; otherwise leave a `// TODO(DP-alias-draw)` and note it in the PR.

## Proof this is the fix (already done by Claude)
Instrumented the C oracle's `random.c` with a draw counter (temporary, reverted, oracle rebuilt clean) and measured the `steal coins <mob>` scenario at `DP_SEED=1`: **C `drawsBefore=4073` vs Go `4070`** (Go 3 behind → C rolls a success value, Go a fail value). Ruling-out chain: heartbeat draws exactly 6 (3 mobile ticks × 2, already fixed by #374); `room_activity`/`object_activity` draw nothing on this path; `do_move` draws 0. The gap was the per-command `number(0,3)`. **Spiking exactly this draw into `ExecuteCommand` turned the steal scenario GREEN** ("no normalized divergence"). So this single draw closes DP-1170's steal divergence.

## Out of scope (do NOT touch)
- The steal / hide / sneak / cast handlers themselves — all already C-faithful.
- `mobile_activity` (fixed in #374), `room_activity`/`object_activity` (they don't draw here).
- The C oracle tree, `website/static/map/world-sphere.json`, `docs/reports/reek/*`.

## Tests you own (golden, deterministic)
In `pkg/session` (or wherever `ExecuteCommand` is unit-testable), using `dprng.ResetStream(seed)` (NOT `Seed`):
1. **One draw per command.** With a playing session, dispatch a command and assert exactly one shared-stream draw is consumed at the top (a `dprng.DrawCount()`-style seam is fine if you add a minimal test-only one; otherwise assert via a known-seed value sequence).
2. **Hide clears on 0.** Seed so the first `number(0,3)` is 0; set the player's hide affect; dispatch any command; assert hide is cleared. Seed so it's nonzero; assert hide persists.
3. **No draw during creation/login.** Assert a command dispatched before the player is "playing" (or via the creation path) consumes **zero** draws.

## Acceptance / how Claude gates it
Claude re-creates the `steal coins <mob>` oracle scenario (fixture mob 16303 at 8105, warmup `n/e/s/e`, probe `steal coins trainee`), currently **RED on `main`**, and confirms it goes **GREEN** on your branch — the red→green proof. Claude also re-runs existing green scenarios (hide/sneak) to confirm no regression. Note the scenario name + expected-green in the PR body; Claude owns the run.

## PR hygiene
- Commit messages end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- PR body ends with: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`
