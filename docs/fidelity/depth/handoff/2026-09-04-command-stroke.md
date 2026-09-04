# Depth-fidelity handoff — `stroke`

Date: 2026-09-04

Feature branch: `glm/depth-stroke`

## Queue position and scope

This slice starts from merged main at `c4deabe91` after the `stretch` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `stroke` at line 746. No `stroke` manifest, scenario,
or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, Act-audience, target lookup, and
sleeping-victim behavior remains delegated or exercised through established
social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:746 */
{ "stroke"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:869-877` is:

```text
stroke 0 0
Whose thigh would you like to stroke?
#
You gently stroke $S inner thigh.
$n gently strokes $N's inner thigh... hmm...
$n gently strokes your inner thigh with feathery touches.
That person is not within reach.
You are about to do something you would rather not be caught doing.
$n starts to do something disgusting and then stops.
```

The C hide flag and victim-position minimum are both `0`. All eight authored
slots are reachable through the no-argument, visible-target, self-target,
missing-target, first-token/trailing-argument, mob-target, and
sleeping-target variants. The `#` no-argument room slot is intentionally
silent, while the self-target actor and room slots use their distinct
authored messages. Shared command position, `PLR_NOSHOUT`, and common
audience/lookup mechanics are not duplicated beyond the slice's differential
probes.

## Result and proof

Added `cmd/dp-oracle-diff/scenarios/stroke-depth.txt` with the standard actor,
observer, target, and generic-mob fixture; `pkg/session/stroke_depth_test.go`
to pin the C command gate and all eight parsed message slots; and
`docs/fidelity/depth/stroke.tsv` with twelve durable unit, delegated, and
oracle rows. The existing Go handler and data are faithful; this was a
pure-coverage slice and no player-visible Go behavior changed.

The final `stroke-depth` matrix used the C oracle at seeds 1, 2, 3, 5, and 8.
Seed 1 used `--show-oracle` and displayed the intentional silent no-argument
room slot, visible-target, generic-mob, self-target, not-found, and
sleeping-target audiences. Every seed exited 0 with
`result: no normalized divergence`. The sleeping-target result confirms C's
zero victim-position minimum admits the branch while `TO_VICT`/SENDOK
suppresses the sleeping recipient's private line.

The required local verification completed on 2026-09-04:

- `make fidelity-depth` — 4,418 total, 4,313 proven/delegated, 54 blocked,
  and 51 excluded; 98.8% actionable completion.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- `gosec -severity high -confidence high ./...` — 0 issues.
- `git diff --check` — passed.

The handoff and evidence are in commits `11196cf7f` and `73331a49a` at the
time of this note. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Starting frontier

The merged `stretch` handoff reported 4,406 total cases: 4,301
proven/delegated, 54 blocked, and 51 excluded. This slice adds twelve
proven/delegated rows, bringing main's frontier to 4,418 total, 4,313
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. The next fresh social after this slice is `strut` at
`src/interpreter.c:747`.
