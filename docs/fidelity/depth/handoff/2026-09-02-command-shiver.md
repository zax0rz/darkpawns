# Depth-fidelity handoff — `shiver`

Date: 2026-09-02

## Queue position and result

This session began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `2026-09-02-command-shishkabob.md` handoff. The special-procedure
inventory remains exhausted. The one-time blocked `objmagic.sleep-entry-gates`
row remains bounded after its cast-sleep outlaw/reagent vehicle and was not
repicked. The interpreter sweep consumed the next source-order social entry,
`shiver`, at `src/interpreter.c:693`.

The pre-slice frontier was 3,271 total cases, with 3,189 proven/delegated, 30
blocked, and 52 excluded. The shiver manifest contributes eight cases,
producing:

- 3,279 total cases
- 3,197 proven/delegated
- 30 blocked
- 52 excluded

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:693 */
{ "shiver"   , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. It first resolves the social and
checks `PLR_NOSHOUT`; because the `shiver` record has no `char_found` message,
C clears the target buffer before parsing and always takes the no-argument
actor/room path. The authored record at `lib/misc/socials:685-688` is
`shiver 0 0`, with `Brrrrrrrrr.`, `$n shivers uncomfortably.`, and `#` in its
three parsed message slots. Typed, missing, and self-looking arguments are
therefore ignored, with no target lookup, not-found byte, or self-target arm.
The POS_RESTING dispatcher gate, shared emote refusal, and room visibility
behavior remain delegated to their owning evidence under R5b/R5c.

The clean-main probes found no confirmed Go divergence. No implementation
change was warranted under R4, and no `src/` or `darkpawns-c-oracle/` file was
edited.

## Evidence and verification

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/shiver-depth.txt`, with a room observer and
  annotated no-argument, typed-target-ignored, missing-target-ignored, and
  self-target-ignored cases;
- `docs/fidelity/depth/shiver.tsv`, with eight rows;
- `pkg/session/shiver_depth_test.go`, whose focused test pins the C entry gate,
  zero social metadata, and all three authored message slots.

`shiver-depth` reported no normalized divergence at seeds 1, 2, 3, 5, and 8;
seed 1 used `--show-oracle` and showed the intended C actor and observer
blocks. The required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

The slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic seed matrix), R4 (no invention), R5/R5e (process and actual
C-call-path verification), and R5b/R5c (shared behavior ownership).

## Integration and continuation

Feature branch: `glm/depth-shiver`

Feature commit: `217a334bf` (`test: prove shiver depth fidelity`)

Feature PR: #1167 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. It was self-merged only after all
required checks were green, as main commit `793e240cb`.

The fresh source/manifest sweep places the next unclaimed interpreter-table
family at `show`, `src/interpreter.c:694`, registered to `do_show` with
`POS_DEAD` and `LVL_IMMORT`. The next session must return to clean `main`,
pull, rerun `make fidelity-depth`, reread the guide and this newest handoff,
then audit and prove `show` in table order.
