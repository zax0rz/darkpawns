# Depth-fidelity handoff — conjured

Date: 2026-08-30  
Slice: special procedure `conjured`  
Registrations: mob vnums `81-86` at `src/spec_assign.c:186-191`  
C declaration: `SPECIAL(conjured)` at `src/spec_assign.c:164`  
C definition: `src/spec_procs3.c:859-892`  
Code commit: `7aab7ca72`  
PR: #782, merged as `7c6f9d7a0`; hosted run `33292651374`

## Queue position

The special-procedure inventory was refreshed through `conjured` in
`src/spec_procs3.c`. `conjured` is claimed here and must not be repicked.
The next session must return to `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then select the next
unclaimed special procedure in source-and-registration order.

## C call path and branch map

The audit followed player-command special dispatch in
`src/interpreter.c:1407-1456`, autonomous mobile dispatch in
`src/mobact.c:54-93`, the registered assignments at
`src/spec_assign.c:186-191`, and extraction/deferred cleanup through
`src/handler.c:1194-1248` and the next heartbeat in `src/comm.c:815+`.

`conjured` ignores the command text. It returns FALSE while the live mob has
`AFF_CHARM`; this is a runtime affect gate, not a prototype flag. Otherwise
vnums `81-84` take the fizzle branch: if `mob->master` is a player, only that
player receives `You lose control and <mob name> fizzles away!`; the room
always receives `$n returns to its own plane of existence.`. Vnums `85-86`
take the default branch: C `do_say` with a period produces the room line
`<mob> states, 'My work here is done.'`, followed by the room line
`<mob> disappears in a flash of white light!`. The procedure then calls
`extract_char`; it does not invoke the player death/corpse pipeline. The same
body is reachable from commandless autonomous mobile activity when the mob is
awake and otherwise eligible for that dispatch.

## RED → GREEN

On untouched `main`, the fizzle vehicle showed Go broadcasting the master
notice room-wide, lowercasing the room message, and creating a corpse. The
default vehicle showed Go emitting `says` instead of C's punctuation-sensitive
`states` and also creating a corpse. The old Go gate used prototype-only charm
state.

The confirmed Go fix uses the live `AFF_CHARM` bit, resolves a player-only
following/master recipient for the direct fizzle notice, routes the room
messages through canonical `Act`, emits the exact `states` text, and removes
the mob directly with `ExtractMob` so no death side effects are invented.
The implementation preserves both player-command and autonomous special
dispatch. No `src/` or `darkpawns-c-oracle/` files were edited.

The first attempted `load mob` vehicle was discarded: the legacy C
`MobLimits` pointer is null with `num_moblim=1`, so that command is an oracle
crash/setup path rather than valid gameplay proof. The final registered-mob
vehicles use `spawn-mob` with the harness `no-settle` fixture option, keeping
the mob alive until the intended command probe without changing either oracle
tree.

## Proof and verification

Scenarios:

- `cmd/dp-oracle-diff/scenarios/spec-proc-conjured-fizzle.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-conjured-disappear.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-conjured-charmed.txt`

Focused tests: `pkg/game/spec_conjured_test.go`

The three vehicles matched the C oracle at seeds `1, 2, 3, 5, and 8`. The
proof covers the live charm early return, both fizzle audiences, default
speech and disappearance bytes, command-text independence, autonomous
dispatch, direct extraction, and the absence of a corpse/death announcement.

Required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

Hosted `test`, `security`, and `lint` checks all passed in run
`33292651374`. The optional `build-and-push` and `deploy` jobs were skipped by
workflow policy; no CI retry was needed because the checks fired normally.

The manifest now reports:

```text
1193 total; 1148 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1148/1161 = 98.9%
```

## Fidelity rulings

This slice follows R1 (exact direct, room, and punctuation-sensitive bytes),
R2 (the command and autonomous dispatch surfaces), R3 (runtime state and
direct extraction parity), R4 (no invented master audience or corpse), and
R5e (verified the actual `special`, `do_say`, `act`, mobile-dispatch, and
extraction call paths). R5b/R5c apply to the shared audience and extraction
primitives used here.

## Next action

Start the next session from `main`, pull, confirm the frontier, reread the
depth-testing instructions and this handoff, refresh the C registration
inventory against `docs/fidelity/depth/spec-procs.tsv`, and work the next
unclaimed special procedure in file-and-registration order. Preserve the
standing queue order: finish special procedures, then attempt the blocked
`objmagic.sleep-entry-gates` row once via cast-sleep, then sweep remaining
un-manifested `src/interpreter.c` command families. Leave one dated handoff
for that session.
