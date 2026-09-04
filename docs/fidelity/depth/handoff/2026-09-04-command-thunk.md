# Depth-fidelity handoff — `thunk`

Date: 2026-09-04

Feature branch: `glm/depth-thunk`

## Queue position and scope

This slice starts from merged main at `a9cbf9b12` after the `thumbsup` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `thunk` at line 772. No `thunk` manifest, scenario,
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
/* src/interpreter.c:772 */
{ "thunk"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:1277-1285` is:

```text
thunk 0 5
You hit your head and hear a hollow thunk.
$n hits $s head with a hollow thunk.
You thunk $N hollowly on the head.
$n thunks $N hollowly on the head.
$n thunks you hollowly on the head.
Thunk who?
You hit your head and hear a hollow thunk.
$n hits $s head with a hollow thunk.
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
5, and the victim-position minimum is the default 0. This target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/thunk-depth.txt` with the standard actor,
observer, target, and generic-mob fixture;
`pkg/session/thunk_depth_test.go` to pin the C command gate, social metadata,
and all eight parsed message slots; and `docs/fidelity/depth/thunk.tsv` with
twelve durable unit, delegated, and oracle rows. The existing Go handler and
data are faithful; this was a pure-coverage slice and no player-visible Go
behavior changed.

The final `thunk-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact no-argument,
visible-player/NPC target, named self, missing target, first-token/trailing-
argument, and sleeping-target outputs. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,558 total, 4,453 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `16595ed96` and `838a8f58e` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `thumbsup` handoff reported 4,546 total cases: 4,441
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing the frontier to 4,558 total, 4,453
proven/delegated, 54 blocked, and 51 excluded. The next fresh social after
this slice is `tickle` at `src/interpreter.c:775`.
