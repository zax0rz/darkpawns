# Depth-fidelity handoff — `lock` queue audit — 2026-09-01

## Queue position

This session started from clean `main` at the post-`lines` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md`, the standing brief, and
`docs/fidelity/depth/handoff/2026-09-01-command-lines.md`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked. The
source-order sweep reached `lock` at `src/interpreter.c:543`, but `lock` is
already claimed by the shared `do_gen_door` row
`docs/fidelity/depth/door.tsv:10`, previously handed off in
`docs/fidelity/depth/handoff/2026-08-27-depth-fidelity-loop.md` under
Continuation Round 3. It is therefore skipped under the no-repick rule. The
next unmanifested command is `load` at `src/interpreter.c:544`.

## C path and existing proof ownership

The C registration is `{ "lock", POS_SITTING, do_gen_door, 0, SCMD_LOCK }` in
`src/interpreter.c:543`. The shared C handler is
`src/act.movement.c:604-637`, with its lock-specific precondition ladder at
`src/act.movement.c:623-637`: two-argument object/direction parsing, object or
exit lookup, openability, closed/unlocked state gates, key ownership for
non-Gods, and `ok_pick` before `do_doorcmd`. The success path toggles the lock,
sends `*Click*\r\n`, and emits the exact room audience. It shares its full
lock/unlock/pick behavior class with the registered `open`, `close`, `unlock`,
and `pick` commands.

The existing `door-lock-pick-depth` vehicle already proves the keyed mortal
lock/unlock/pick gates, exact actor and room bytes, skill success, pickproof
resistance, lockpick breakage, reciprocal state, and seed matrix 1, 2, 3, 5,
and 8. The durable row is `door.lock-unlock-pick` with status
`oracle-green-multiseed`; the prior handoff records the two confirmed Go fixes
and their proof. Re-running this matrix as a standalone `lock` slice would
duplicate an already claimed shared callee and violate R5b/R5c.

No new oracle scenario, manifest row, or Go change was made in this audit; no
`src/` or `darkpawns-c-oracle/` file was edited.

## Durable frontier

The frontier remains `2478 total, 2417 proven/delegated, 17 blocked, 44
excluded`; actionable completion remains `2417/2434 = 99.3%`. The next round
must start from clean `main`, recheck the frontier and newest handoff, and take
`load` at `src/interpreter.c:544`.

This dated note records the required queue decision for the session. It follows
R1/R2/R4/R5e by preserving the command surface and C authority, and applies
R5b/R5c to avoid duplicating the already proven `do_gen_door` behavior class.
