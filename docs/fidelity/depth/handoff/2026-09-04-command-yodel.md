# Depth-fidelity handoff — `yodel`

Date: 2026-09-04

Feature branch: `glm/depth-yodel`

## Queue position and scope

This slice starts from merged main at `c68810398` after the `yawn` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `yodel` at line 841. No `yodel` manifest, scenario,
or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remain
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:841 */
{ "yodel"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, parses no target when `char_found` is absent, sends the authored
no-argument bytes to the actor, and routes the authored room audience through
`act`. The authored record at `lib/misc/socials:1029-1032` is:

```text
yodel 0 0
You start yodelling loudly and rather beautifully in your own ears.
$n starts a yodelling session that goes right to the bone.
#
```

This is a self-only record: typed targets, a named self, and an unresolved
target all remain on the same no-argument actor/room path because there is no
`char_found`, `vict_found`, or not-found slot. The command row requires
`POS_RESTING`; the C social hide field and minimum victim position are both 0
(represented by the legacy Go `MinLevel` and `HideFlag` fields). Position,
`PLR_NOSHOUT`, and shared Act visibility are delegated rather than duplicated.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/yodel-depth.txt` with the standard
actor/peer fixture and four self-only probes; `pkg/session/yodel_depth_test.go`
to pin the C command gate, social metadata, and all three parsed message slots;
and `docs/fidelity/depth/yodel.tsv` with eight durable D1-D3 rows. This was a
pure-coverage slice: the existing Go handler and data were already faithful,
so no player-visible Go source behavior changed.

The `yodel-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8. Seed 1
used `--show-oracle` and displayed the exact actor/room output for no argument,
typed peer, missing target, and named self. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,750 total, 4,645 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `66ab4471f` and `e1f37dc8c`. No file
under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `yawn` handoff reported 4,742 total cases: 4,637
proven/delegated, 54 blocked, and 51 excluded. This slice adds eight
proven/delegated rows, bringing the frontier to 4,750 total, 4,645
proven/delegated, 54 blocked, and 51 excluded. The next fresh social is
`yuball` at `src/interpreter.c:842`.
