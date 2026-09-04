# Depth-fidelity handoff — `threaten`

Date: 2026-09-04

Feature branch: `glm/depth-threaten`

## Queue position and scope

This slice starts from merged main at `4b670fcab` after the `thpbt` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `threaten` at line 769. No `threaten` manifest,
scenario, or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:769 */
{ "threaten" , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:1332-1340` is:

```text
threaten 0 5
You threaten the room wildly.
You watch as $n threatens everyone in the room.
You threaten $N, that dirty rat!
You see $n threaten $N, maybe you should step in and calm things down.
$n threatens you, how mean!
Threaten who?
#
```

The C record terminates after the `#` self-target actor slot, so the parsed
eighth slot is the exact empty string. The command row requires
`POS_RESTING`; the C social level is 0, hide flag is 5, and the victim-position
minimum is the default 0. The target-capable record reaches no-argument,
visible-player/NPC, named self, missing-target, and first-token/trailing-
argument branches. Shared command position, `PLR_NOSHOUT`, target lookup, and
room visibility mechanics are not duplicated beyond the slice's differential
probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/threaten-depth.txt` with the standard
actor, observer, target, and generic-mob fixture;
`pkg/session/threaten_depth_test.go` to pin the C command gate, social
metadata, and all eight parsed message slots; and
`docs/fidelity/depth/threaten.tsv` with twelve durable unit, delegated, and
oracle rows. The existing Go handler and data are faithful; this was a
pure-coverage slice and no player-visible Go behavior changed.

The final `threaten-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and
8. Seed 1 used `--show-oracle` and displayed the exact no-argument,
visible-player/NPC target, named self, missing-target, first-token/trailing-
argument, and sleeping-target outputs. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,522 total, 4,417 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `d9c7dd1d9` and `61b57f760` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `thpbt` handoff reported 4,510 total cases: 4,405
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. This slice adds twelve proven/delegated rows, bringing main's
frontier to 4,522 total, 4,417 proven/delegated, 54 blocked, and 51 excluded.
The next fresh social after this slice is `throttle` at
`src/interpreter.c:770`.
