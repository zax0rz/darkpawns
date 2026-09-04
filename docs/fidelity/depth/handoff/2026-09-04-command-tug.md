# Depth-fidelity handoff — `tug`

Date: 2026-09-04

Feature branch: `glm/depth-tug`

## Queue position and scope

This slice starts from merged main at `48a686555` after the `tilt` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `tug` at line 786. No `tug` manifest, scenario, or
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
/* src/interpreter.c:786 */
{ "tug"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:1108-1116` is:

```text
tug 1 0
You go around looking for someone to tug on.
$n looks for someone's sleeve to tug on.
You tug on $N's sleeve and whine.
$n tugs on $N's sleeve and whines.
$n tugs on your sleeve to get your attention.
You tug on a non-existent sleeve.
You tug on your own sleeve.  Whoops, you tore your shirt.
$n tugs on $s own sleeve, creating a big hole in $s shirt.
```

The command row requires `POS_RESTING`; the C social level is 1, hide flag is
0, and the victim-position minimum is the default 0. This target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/tug-depth.txt` with the standard actor,
observer, target, and generic-mob fixture;
`pkg/session/tug_depth_test.go` to pin the C command gate, social metadata, and
all eight parsed message slots; and `docs/fidelity/depth/tug.tsv` with twelve
durable unit, delegated, and oracle rows. The existing Go handler and data are
faithful; this was a pure-coverage slice and no player-visible Go behavior
changed.

The final `tug-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the exact no-argument,
visible-player/NPC target, named self, missing target, first-token/trailing-
argument, and sleeping-target outputs. Every seed exited 0 with
`result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,594 total, 4,489 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `3bfd19707` and `cb85447b8` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `tilt` handoff reported 4,582 total cases: 4,477
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing the frontier to 4,594 total, 4,489
proven/delegated, 54 blocked, and 51 excluded. The next fresh social after
this slice is `twiddle` at `src/interpreter.c:788`.
