# Depth-fidelity handoff — `wedgie`

Date: 2026-09-04

Feature branch: `glm/depth-wedgie`

## Queue position and scope

This slice starts from merged main at `f805a493e` after the `wave` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `wedgie` at line 811. No `wedgie` manifest, scenario,
or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:811 */
{ "wedgie"   , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:1312-1320` is:

```text
wedgie 0 5
You pull your underwear out of your crack!
You watch as $n un-wedgies $mself.
You grab $N's underwear and pull up HARD.
You see $n give a painful wedgie to $N.
$n almost rips your underwear off with an incredible wedgie.
Wedgie who?
You give yourself a wedgie.
You watch $n pull $s underwear nearly over $s head.
```

The command row requires `POS_RESTING`; the C social hide field is 0 and its
minimum victim position is 5 (represented by the legacy Go `HideFlag` field),
while the explicit Go override remains 0. This target-capable record reaches
no-argument, visible player/NPC, named self, missing-target, first-token/
trailing-argument, and sleeping-target branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/wedgie-depth.txt` with the standard actor,
observer, target, and generic-mob fixture;
`pkg/session/wedgie_depth_test.go` to pin the C command gate, social metadata,
and all eight parsed message slots; and `docs/fidelity/depth/wedgie.tsv` with
twelve durable unit, delegated, and oracle rows. The existing Go handler and
data are faithful; this was a pure-coverage slice and no player-visible Go
behavior changed.

The final `wedgie-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact no-argument,
visible-player/NPC target, named self, missing target, first-token/trailing-
argument, mob target, and sleeping-target position-gate outputs. Every seed
exited 0 with `result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,650 total, 4,545 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `2f61ee67e` and `10a3c99a7` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `wave` handoff reported 4,638 total cases: 4,533
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing the frontier to 4,650 total, 4,545
proven/delegated, 54 blocked, and 51 excluded. The next fresh social after
this slice is `wee` at `src/interpreter.c:812`.
