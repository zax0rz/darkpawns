# Depth-fidelity handoff — `tackle`

Date: 2026-09-04

Feature branch: `glm/depth-tackle`

## Queue position and scope

This slice starts from merged main at `7395379fd` after the `sweat` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `tackle` at line 756. No `tackle` manifest, scenario,
or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:756 */
{ "tackle"   , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:889-897` is:

```text
tackle 0 5
You tackle the air.  It stands not a chance.
$n starts running around $mself in a desperate attempt to tackle the air.
You ruthlessly tackle $M to the ground.
$n ruthlessly tackles $N, pinning $M to the ground.
$n suddenly lunges at you and tackles you to the ground!
That person isn't here (lucky for them, it would seem...)
Tackle yourself?  Yeah, right....
$n makes a dextrous move and kicks $s left leg away with $s right.
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
5, and the victim-position minimum is the default 0. The target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/tackle-depth.txt` with the standard actor,
observer, target, and generic-mob fixture; `pkg/session/tackle_depth_test.go`
to pin the C command gate, social metadata, and all eight parsed message
slots; and `docs/fidelity/depth/tackle.tsv` with twelve durable unit,
delegated, and oracle rows. The existing Go handler and data are faithful;
this was a pure-coverage slice and no player-visible Go behavior changed.

The final `tackle-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact no-argument, visible
player/NPC target, named self, missing-target, first-token/trailing-argument,
and sleeping-target outputs. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,454 total, 4,349 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `522474225` and `9bd89626b` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `sweat` handoff reported 4,442 total cases: 4,337
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. This slice adds twelve proven/delegated rows, bringing main's
frontier to 4,454 total, 4,349 proven/delegated, 54 blocked, and 51 excluded.
The next fresh social after this slice is `tango` at
`src/interpreter.c:759`.
