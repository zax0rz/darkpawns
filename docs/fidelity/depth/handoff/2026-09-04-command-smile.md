# Depth-fidelity handoff — `smile`

Date: 2026-09-04

Feature branch: `glm/depth-smile`

## Queue position and scope

This slice starts from merged main at `615467fe2` after the `sing` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `smile` at line 710. No `smile` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, social-level, `PLR_NOSHOUT`, Act-audience,
target-lookup, and sleeping-victim behavior remains delegated or exercised
through established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:710 */
{ "smile"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, applies the
social record's level check, rejects `PLR_NOSHOUT`, parses the first target
token, handles no-target, not-found, self-target, and visible-target
branches, and routes the authored audience through `act`. The authored record
at `lib/misc/socials:725-733` is:

```text
smile 1 0
You smile happily.
$n smiles happily.
You smile at $M.
$n beams a smile at $N.
$n smiles at you.
There's no one by that name around.
You smile at yourself.
$n smiles at $mself.
```

The command row has no command minimum, while the social record has level 1,
hide flag 0, and the default zero victim-position minimum. All eight authored
slots are reachable through the no-argument, visible-target, self-target,
missing-target, first-token/trailing-argument, mob-target, and
sleeping-target variants. Shared command position, social-level, `PLR_NOSHOUT`,
and common audience/lookup mechanics are not duplicated beyond the slice's
differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/smile-depth.txt` with the standard actor,
observer, target, and generic-mob fixture; `pkg/session/smile_depth_test.go`
to pin the C command gate and all eight parsed message slots, including
social level 1; and `docs/fidelity/depth/smile.tsv` with thirteen durable
unit, delegated, and oracle rows. The existing Go handler and data are
faithful; this was a pure-coverage slice and no player-visible Go behavior
changed.

The final `smile-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the intended no-argument,
player-target, generic-mob, self-target, not-found, and sleeping-target
audiences. Every seed exited 0 with `result: no normalized divergence`.
The sleeping-target result confirms C's zero victim-position minimum admits
the branch while `TO_VICT`/SENDOK suppresses the sleeping recipient's private
line.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,378 total, 4,273 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The initial handoff and evidence are in commits `f1d248123` and
`27e4b3f00` at the time of this note. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `sing` handoff reported 4,365 total cases: 4,260
proven/delegated, 54 blocked, and 51 excluded. This slice adds thirteen
proven/delegated rows, bringing main's frontier to 4,378 total, 4,273
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. The next fresh social after this slice is `steam` at
`src/interpreter.c:741`.
