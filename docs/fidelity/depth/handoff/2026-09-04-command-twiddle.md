# Depth-fidelity handoff — `twiddle`

Date: 2026-09-04

Feature branch: `glm/depth-twiddle`

## Queue position and scope

This slice starts from merged main at `9b6350a98` after the `tug` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `twiddle` at line 788. No `twiddle` manifest,
scenario, or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remain
delegated to established social vehicles under R5b/R5c. This is a self-only
record, so typed target, missing target, and named self arguments remain on
the no-argument path by the C `char_found` condition.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:788 */
{ "twiddle"  , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, and, when the record has no target branch, emits the self-only
actor and room slots. The authored record at `lib/misc/socials:959-962` is:

```text
twiddle 0 0
You patiently twiddle your thumbs.
$n patiently twiddles $s thumbs.
#
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
0, and the victim-position minimum is the default 0. The record terminates
after the `#` room slot, so its parsed self-target actor slot is the exact
empty string. Shared command position, `PLR_NOSHOUT`, and room visibility
mechanics are not duplicated beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/twiddle-depth.txt` with the standard actor
and peer fixture; `pkg/session/twiddle_depth_test.go` to pin the C command
gate, social metadata, and all three parsed message slots; and
`docs/fidelity/depth/twiddle.tsv` with eight durable unit, delegated, and
oracle rows. The existing Go handler and data are faithful; this was a
pure-coverage slice and no player-visible Go behavior changed.

The final `twiddle-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact self-only actor/room
messages for no argument, typed target, missing target, and named self. Every
seed exited 0 with `result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,602 total, 4,497 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `6e7105665` and `aeaa7bc5f` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `tug` handoff reported 4,594 total cases: 4,489
proven/delegated, 54 blocked, and 51 excluded. This slice adds eight
proven/delegated rows, bringing the frontier to 4,602 total, 4,497
proven/delegated, 54 blocked, and 51 excluded. The next fresh social after
this slice is `violate` at `src/interpreter.c:801`.
