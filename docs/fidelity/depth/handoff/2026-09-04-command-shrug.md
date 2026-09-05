# Depth-fidelity handoff — `shrug`

Date: 2026-09-04

Feature branch: `glm/depth-shrug`

## Queue position and scope

This slice starts from merged main at `a1199e1fe` after the `shake` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the one blocked `objmagic.sleep-entry-gates`
vehicle remain queued for their later passes. Phase 1 is continuing through
the remaining socials. `shake` is the last completed social handoff; the next
genuinely unmanifested reachable `do_action` row in `src/interpreter.c` is
`shrug` at line 696. No `shrug` manifest, scenario, or focused registration
test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R4/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:696 */
{ "shrug"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, checks the record's `char_found` slot, and because it is `#`
follows the self-only/no-target path. Typed targets, including a self alias
and an unresolved name, are ignored rather than looked up. The authored
record at `lib/misc/socials:690-693` is:

```text
shrug 0 0
You shrug.
$n shrugs.
#
```

The C hide flag and victim-position minimum are both `0`; only the actor and
ordinary room message slots are authored. The reachable slice is therefore
the no-argument actor/room pair and the typed-target, self-target, and
missing-target variants that all remain on that same pair. Shared command
position, `PLR_NOSHOUT`, and room visibility mechanics are not duplicated.

## Result and proof vehicle

Added `cmd/dp-oracle-diff/scenarios/shrug-depth.txt` with `# depth-case:` tags
for the no-argument and ignored-argument branches, plus the standard actor
and peer fixture. Added `pkg/session/shrug_depth_test.go` to pin the C command
gate and all three parsed message slots, and added
`docs/fidelity/depth/shrug.tsv` with eight durable rows. The existing Go
handler and data are faithful; this was a pure-coverage slice and no
player-visible Go behavior changed.

The final `shrug-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the intended actor/peer blocks for
no argument, typed player target, missing target, and self target. Every seed
exited 0 with `result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,337 total, 4,232 proven/delegated, 54 blocked,
  and 51 excluded; 98.7% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff, evidence, and tests are in commits `e4040a417` and
`2e712a77d` at the time of this note. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `shake` handoff reported 4,329 total cases: 4,224 proven/delegated,
54 blocked, and 51 excluded. This slice adds eight proven/delegated rows,
bringing main's frontier to 4,337 total, 4,232 proven/delegated, 54 blocked,
and 51 excluded. The next fresh social audit is `shudder` at
`src/interpreter.c:697`. Continue the remaining Phase 1 social sweep before
the later red/blocked and off-command-table phases in the objective.
