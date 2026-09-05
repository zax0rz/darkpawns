# Depth-fidelity handoff — `sing`

Date: 2026-09-04

Feature branch: `glm/depth-sing`

## Queue position and scope

This slice starts from merged main at `3c6f669de` after the `sigh` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `sing` at line 701. No `sing` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, Act-audience, target lookup, and
sleeping-victim behavior remains delegated or exercised through established
social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:701 */
{ "sing"       , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token for a target-capable record,
handles no-target, not-found, self-target, and visible-target branches, and
routes the authored audience through `act`. The authored record at
`lib/misc/socials:705-713` is:

```text
sing 0 0
You raise your clear (?) voice towards the sky.
SEEK SHELTER AT ONCE!  $n has begun to sing.
You begin to seranade $M. Lalala...
$n raises $s voice and beings to seranade $N. Lalala.
$n raises $s voice and beings to seranade you. Lalala.
They can't hear you, wherever they are.
You sing to yourself.  Ladidadida.
$n sings to $mself.  Ladidadida.
```

The C hide flag and victim-position minimum are both `0`; all eight authored
slots are reachable through the no-argument, visible-target, self-target,
missing-target, first-token/trailing-argument, mob-target, and sleeping-target
variants. Shared command position, `PLR_NOSHOUT`, and common audience/lookup
mechanics are not duplicated beyond the slice's differential probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/sing-depth.txt` with the standard actor,
observer, target, and generic-mob fixture; `pkg/session/sing_depth_test.go` to
pin the C command gate and all eight parsed message slots; and
`docs/fidelity/depth/sing.tsv` with twelve durable unit, delegated, and oracle
rows. The existing Go handler and data are faithful; this was a pure-coverage
slice and no player-visible Go behavior changed.

The final `sing-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the intended no-argument,
player-target, generic-mob, self-target, not-found, and sleeping-target
audiences. Every seed exited 0 with `result: no normalized divergence`.
The sleeping-target result confirms C's zero victim-position minimum admits
the branch while `TO_VICT`/SENDOK suppresses the sleeping recipient's private
line.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,365 total, 4,260 proven/delegated, 54 blocked,
  and 51 excluded; 98.7% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The initial handoff and evidence are in commits `016f83c54` and
`f22041903` at the time of this note. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `sigh` handoff reported 4,353 total cases: 4,248
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing main's frontier to 4,365 total, 4,260
proven/delegated, 54 blocked, and 51 excluded. This is the tenth slice since
the last required worktree prune; prune after this slice is complete. Continue
the remaining Phase 1 social sweep before the later red/blocked and
off-command-table phases in the objective. The next fresh social after this
slice is `smile` at `src/interpreter.c:710`.
