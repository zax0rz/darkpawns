# Depth-fidelity handoff — `shake`

Date: 2026-09-04

Feature branch: `glm/depth-shake`

## Queue position and scope

This slice starts from merged main at `0f051b36c` after the `scream` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the one blocked `objmagic.sleep-entry-gates`
vehicle remain queued for their later passes. Phase 1 is continuing through
the remaining socials. `scream` is the last completed social handoff; the
next genuinely unmanifested reachable `do_action` row in `src/interpreter.c`
is `shake` at line 691. No `shake` manifest, scenario, or focused
registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R4/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, visible lookup, and Act-audience
behavior remains delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:691 */
{ "shake"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, conditionally consumes the first target token with
`one_argument`, handles no-argument and missing-target branches, checks the
self-target and victim-position cases, then dispatches actor, observer, and
victim messages. The authored record at `lib/misc/socials:665-673` is
`shake 0 5` followed by eight message slots:

```text
You shake your head.
$n shakes $s head.
You shake $S hand.
$n shakes $N's hand.
$n shakes your hand.
Sorry good buddy, but that person doesn't seem to be here.
You are shaken by yourself.
$n shakes and quivers like a bowlful of jelly.
```

The C hide flag is `0` and the victim-position minimum is `5` (`POS_RESTING`).
The reachable slice therefore includes no argument, visible player target and
three audiences, first-token parsing, visible NPC target, self target,
missing target, and sleeping-target rejection. The shared position failure,
lookup, and audience mechanisms are not duplicated here.

## Result and proof vehicle

Added `cmd/dp-oracle-diff/scenarios/shake-depth.txt` with `# depth-case:` tags
for each branch and the standard actor/observer/target plus trainee-mob
fixture. Added `pkg/session/shake_depth_test.go` to pin the C command gate and
all eight parsed social messages, and added `docs/fidelity/depth/shake.tsv`
with eleven durable rows. The existing Go handler and data are faithful; this
was a pure-coverage slice and no player-visible Go behavior changed.

The final `shake-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the intended no-argument, player
target, NPC target, self, missing-target, and sleeping-target blocks. Every
seed exited 0 with `result: no normalized divergence`.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,329 total, 4,224 proven/delegated, 54 blocked,
  and 51 excluded; 98.7% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff, evidence, and tests are in commits `dec6bc92c` and
`bedb22076` at the time of this note. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `scream` handoff reported 4,318 total cases: 4,213
proven/delegated, 54 blocked, and 51 excluded. This slice adds eleven
proven/delegated rows, bringing main's frontier to 4,329 total, 4,224
proven/delegated, 54 blocked, and 51 excluded. The next fresh social audit is
`shrug` at `src/interpreter.c:696`; `sharpen` and `shiver` are already owned
by their existing non-overlapping/manifested families. Continue the remaining
Phase 1 social sweep before the later red/blocked and off-command-table
phases in the objective.
