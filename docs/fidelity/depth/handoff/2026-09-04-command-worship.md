# Depth-fidelity handoff — `worship`

Date: 2026-09-04

Feature branch: `glm/depth-worship`

## Queue position and scope

This slice starts from merged main at `7ce92d45f` after the `wink` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `worship` at line 836. No `worship` manifest,
scenario, or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:836 */
{ "worship"  , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, visible-target, and sleeping-target branches, and routes the
authored audiences through `act`. The authored record at
`lib/misc/socials:1014-1022` is:

```text
worship 0 5
You find yourself head-down in the dirt, worshipping.
$n starts worshipping nothing at all.
You fall to your knees and worship $M deeply.
$n falls to $s knees, worshipping $N with uncanny dedication.
$n kneels before you in solemn worship.
Uh.. who?  They're not here, pal.
You seem sure to have found a true deity.....
$n falls to $s knees and humbly worships $mself.
```

The command row requires `POS_RESTING`. The C social hide field is 0 and its
minimum victim position is 5 (represented by the legacy Go `HideFlag` field),
while the explicit Go override remains 0. This target-capable record reaches
no-argument, visible player/NPC, named self, missing target, first-token/
trailing-argument, and sleeping-target branches. Shared command position,
`PLR_NOSHOUT`, target lookup, visibility, and Act audience behavior are not
duplicated beyond this slice.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/worship-depth.txt` with the standard
actor/observer/target/mob fixture and the full target-capable probe matrix;
`pkg/session/worship_depth_test.go` to pin the C command gate, social metadata,
and all eight parsed message slots; and `docs/fidelity/depth/worship.tsv` with
twelve durable D1-D3 rows. This was a pure-coverage slice: the existing Go
handler and data were already faithful, so no player-visible Go source behavior
changed.

The `worship-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8. Seed
1 used `--show-oracle` and displayed the exact actor, observer, target, mob,
self-target, not-found, and sleeping-target outputs, including the C `$M`/`$s`
pronoun substitutions. Every seed exited 0 with `result: no normalized
divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,734 total, 4,629 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `81aa63351` and `40fe7341b`. No file
under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `wink` handoff reported 4,722 total cases: 4,617
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing the frontier to 4,734 total, 4,629
proven/delegated, 54 blocked, and 51 excluded. The next fresh social is
`yawn` at `src/interpreter.c:840`.
