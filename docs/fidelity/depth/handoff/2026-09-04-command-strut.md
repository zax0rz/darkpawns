# Depth-fidelity handoff — `strut`

Date: 2026-09-04

Feature branch: `glm/depth-strut`

## Queue position and scope

This slice starts from merged main at `d5eb0089e` after the `stroke` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `strut` at line 747. No `strut` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:747 */
{ "strut"      , POS_STANDING, do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, checks the record's `char_found` slot, and because it is `#`
follows the self-only/no-target path. Typed targets, including a self alias
and an unresolved name, are ignored rather than looked up. The authored
record at `lib/misc/socials:879-882` is:

```text
strut 0 0
Strut your stuff.
$n struts proudly.
#
```

The command row requires `POS_STANDING`; the C hide flag and victim-position
minimum are both `0`, and only the actor and ordinary room message slots are
authored. The reachable slice is therefore the standing-gated no-argument
actor/room pair and typed-target, self-target, and missing-target variants
that all remain on that same path. Shared command position, `PLR_NOSHOUT`,
and room visibility mechanics are not duplicated.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/strut-depth.txt` with the standard actor
and peer fixture; `pkg/session/strut_depth_test.go` to pin the C command gate
and all three parsed message slots; and `docs/fidelity/depth/strut.tsv` with
eight durable unit, delegated, and oracle rows. The existing Go handler and
data are faithful; this was a pure-coverage slice and no player-visible Go
behavior changed.

The final `strut-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact actor and ordinary room
messages for no argument, visible peer, missing target, and named self. Every
seed exited 0 with `result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,426 total, 4,321 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `5dce28170` and `8f109cda7` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `stroke` handoff reported 4,418 total cases: 4,313
proven/delegated, 54 blocked, and 51 excluded. This slice adds eight
proven/delegated rows, bringing main's frontier to 4,426 total, 4,321
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. The next fresh social after this slice is `sulk` at
`src/interpreter.c:749`.
