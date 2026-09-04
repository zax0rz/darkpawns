# Depth-fidelity handoff — `wink`

Date: 2026-09-04

Feature branch: `glm/depth-wink`

## Queue position and scope

This slice starts from merged main at `3608e164f` after the `wiggle` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `wink` at line 828. No `wink` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:828 */
{ "wink"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, visible-target, and sleeping-target branches, and routes the
authored audiences through `act`. The authored record at
`lib/misc/socials:1004-1012` is:

```text
wink 0 5
Have you got something in your eye?
$n winks suggestively.
You wink suggestively at $N.
$n winks at $N.
$n winks suggestively at you.
No one with that name is present.
You wink at yourself?? -- what are you up to?
$n winks at $mself -- something strange is going on...
```

The command row requires `POS_RESTING`. The C social hide field is 0 and its
minimum victim position is 5 (represented by the legacy Go `HideFlag` field),
while the explicit Go override remains 0. This target-capable record reaches
no-argument, visible player/NPC, named self, missing target, first-token/
trailing-argument, and sleeping-target branches. Shared command position,
`PLR_NOSHOUT`, target lookup, visibility, and Act audience behavior are not
duplicated beyond this slice.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/wink-depth.txt` with the standard
actor/observer/target/mob fixture and the full target-capable probe matrix;
`pkg/session/wink_depth_test.go` to pin the C command gate, social metadata,
and all eight parsed message slots; and `docs/fidelity/depth/wink.tsv` with
twelve durable D1-D3 rows. This was a pure-coverage slice: the existing Go
handler and data were already faithful, so no player-visible Go source behavior
changed.

The `wink-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8. Seed 1
used `--show-oracle` and displayed the exact actor, observer, target, mob,
self-target, not-found, and sleeping-target outputs. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,722 total, 4,617 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `8be67f67e` and `d3d9e5917`. No file
under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `wiggle` handoff reported 4,710 total cases: 4,605
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing the frontier to 4,722 total, 4,617
proven/delegated, 54 blocked, and 51 excluded. The next fresh social is
`worship` at `src/interpreter.c:836`.
