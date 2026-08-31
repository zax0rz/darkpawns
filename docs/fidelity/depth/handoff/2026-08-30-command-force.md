# Depth handoff — 2026-08-30 — `force`

## Frontier and queue position

- Started from clean `main` at `3112254c3` after the merged `exits` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-30-command-exits.md`.
- The frontier before this slice was 1,657 total, with 1,602
  proven/delegated, 14 blocked, and 41 excluded. The force manifest adds 17
  cases: 15 proven and 2 blocked. The post-slice frontier is 1,675 total,
  1,618 proven/delegated, 16 blocked, and 41 excluded; actionable completion
  is 1,618/1,634 (99.0%).
- The source-order command gap was `force`, registered at
  `src/interpreter.c:438`. The next command-table gap is `fade` at line 439;
  the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:438` registers `force` with `POS_SLEEPING`, `LVL_GOD`, and
`do_force` in `src/act.wizard.c:1856-1906`. The handler uses `half_chop`, so
an empty target or command emits `Whom do you wish to force do what?`. For
ordinary targets it calls `get_char_vis`, rejects missing targets with
`NOPERSON`, rejects equal-or-higher victims with `No, no, no!`, acknowledges
with `Okay.`, optionally sends the victim `$n has forced you to '%s'.` when the
caster is below `LVL_IMPL`, and then calls
`command_interpreter(vict,to_force)`.

`get_char_vis` is the actual two-stage path in `src/handler.c:1276-1325`:
the room pass handles `self`/`me`, `0.<name>`, ordinals, abbreviations, and
visibility; the global pass then handles visible player names. The Go force
path now uses the shared C-faithful `ResolveCharWorld` resolver and maps a
connected player result to its session.

For `LVL_GRGOD` and above, C's `room` branch acknowledges once, then visits
lower-level characters in the caster's room; C's `all` branch acknowledges
once, then visits connected lower-level characters in every room. Both loops
unconditionally send the force notification, including for an implementor.
Below `LVL_GRGOD`, `room` and `all` fall through as ordinary target names.

The clean-main baseline was RED for the confirmed single-target ordering
vehicle: Go ran the forced command before sending its acknowledgement, while
C sends `Okay.` first. The baseline also exposed the existing handler-level
`LVL_GRGOD` gate, C's exact equal-level rejection, missing self/me resolution,
missing room/all support, and missing room/all victim notifications. The Go
implementation now follows the confirmed C branches, removes its C-absent
denylist/cooldown/transitive-force restrictions, preserves direct
`command_interpreter` behavior without target alias expansion, and keeps
forced-command errors diagnostic rather than inventing a second player-facing
error.

## Coverage proof and unresolved boundary

The live GREEN vehicles are:

- `force-depth`: single-target room audience, C acknowledgement ordering,
  `0.<name>` player-only ordinal, and in-room abbreviated player name;
- `force-gates`: usage, missing target, equal-level target, and `self`/`me`;
- `force-room-all`: room filtering, all filtering, and unconditional room/all
  victim notifications;
- `force-low-god`: LVL_GOD caster and victim notification before the forced
  command.

Each vehicle was run with `--show-oracle` at least once and all four were
GREEN for seeds `1,2,3,5,8`.

Two honest safe probes in `force-mob` (`force trainee sit` and then
`force trainee stand`) remain blocked. C resolves visible NPCs through
`get_char_vis` and invokes the generic command interpreter, producing
`Okay.` plus the mob's room action. Go's force path is session-backed and its
command handlers require a `*Session`/`*Player`; the port has no generic mob
command executor. A sit-only adapter would leave the general forced command
surface invented and incomplete, so these cases stay `blocked` rather than
being force-green. This is the sharp R4/R5b/R5c boundary for the next audit.

No `src/` or `darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4
and R5e: C bytes and dispatch gates remain authoritative, the actual resolver
and handler call paths were checked before changes, and the unresolved NPC
class is explicitly retained for a later shared-architecture decision.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,675 total / 1,618 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #850 (`fix: restore force command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, and begin
`fade`.
