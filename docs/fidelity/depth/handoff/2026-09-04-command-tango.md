# Depth-fidelity handoff — `tango`

Date: 2026-09-04

Feature branch: `glm/depth-tango`

## Queue position and scope

This slice starts from merged main at `34a815da3` after the `tackle` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `tango` at line 759. No `tango` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:759 */
{ "tango"    , POS_STANDING, do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:899-907` is:

```text
tango 0 8
With whom would you like to tango?
$n puts a rose between $s teeth, but takes out it since noone joins $m.
You put a rose between your teeth and tango with $M seductively.
$n puts a rose between $s teeth and tangos with $N seductively.
$n puts a rose between $s teeth and tangos with you seductively.
That person isn't around.  Better sit this one out.
Feels rather stupid, doesn't it?
$n puts a rose between $s teeth and tries to tango with $mself.
```

The command row requires `POS_STANDING`; the C social level is 0, hide flag is
8, and the victim-position minimum is the default 0. The target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/tango-depth.txt` with the standard actor,
observer, target, and generic-mob fixture; `pkg/session/tango_depth_test.go`
to pin the C command gate, social metadata, and all eight parsed message
slots; and `docs/fidelity/depth/tango.tsv` with twelve durable unit,
delegated, and oracle rows. The existing Go handler and data are faithful;
this was a pure-coverage slice and no player-visible Go behavior changed.

The final `tango-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact no-argument, visible
player/NPC target, named self, missing-target, first-token/trailing-argument,
and sleeping-target outputs. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,466 total, 4,361 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `3d53d3f9f` and `667078196` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `tackle` handoff reported 4,454 total cases: 4,349
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. This slice adds twelve proven/delegated rows, bringing main's
frontier to 4,466 total, 4,361 proven/delegated, 54 blocked, and 51 excluded.
The next fresh social after this slice is `tap` at
`src/interpreter.c:760`.
