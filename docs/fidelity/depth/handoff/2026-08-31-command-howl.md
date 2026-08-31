# Depth-fidelity handoff: `howl`

Date: 2026-08-31

## Queue position and frontier

This session began from clean `main` after the `house` handoff. The pre-slice
frontier was 2,211 total cases: 2,151 proven/delegated, 16 blocked, and 44
excluded. The `howl` manifest adds 8 cases, producing 2,219 total: 2,159
proven/delegated, 16 blocked, and 44 excluded (2,159 of 2,175 actionable
cases, 99.3%).

The special-procedure inventory remains exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table queue is complete through `howl`; the next unclaimed family
is `hug` at `src/interpreter.c:504`.

The slice was PR #952, branch `glm/depth-howl`, merged to `main` as
`606517a6f`. Its hosted test, lint, and security checks passed; build and
deploy were skipped by repository policy. No non-green PR was merged.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:503: { "howl", POS_RESTING, do_action, 0, 0 }
```

`src/act.social.c:102-151` is the complete `do_action` path. It resolves the
social, rejects `PLR_NOSHOUT`, and then checks whether `char_found` exists.
The `howl` record has no `char_found` message (`#` in
`lib/misc/socials:1083-1086`), so every invocation—including a visible name,
missing name, or self-looking name—takes the no-argument actor/room pair.
The social record has hide=1 and minimum victim position 0.

The audited player-visible branches are:

- POS_RESTING command entry and the shared noshout refusal;
- no-argument actor and room output;
- visible, missing, and self-looking arguments ignored before target lookup;
- exact hide-aware room delivery and the three authored social record lines.

R1/R2/R3/R4/R5e apply. There is no RNG draw. The command position gate,
PLR_NOSHOUT gate, and Act visibility behavior are shared and delegated under
R5b/R5c to `fade.position-gate`, `dance-noshout`, and `socials-depth`; the
direct vehicle proves the howl-specific self-only branch and bytes.

## RED/ GREEN result

The annotated `howl-depth` vehicle was run against the unchanged Go path based
on clean `main`, with `--show-oracle` at seed 1. It was GREEN: the C oracle
showed the exact actor line and hide-aware observer line for no argument,
visible-name input with trailing words, a missing name, and a self-looking
name. No confirmed Go divergence existed, so no behavioral Go fix was made.

The durable proof-only changes are:

- `cmd/dp-oracle-diff/scenarios/howl-depth.txt`, with the actor and room
  observer vehicle and four annotated self-only cases;
- `pkg/session/howl_test.go`, pinning the C command gate, hide flag, victim
  position, and all three authored messages;
- `docs/fidelity/depth/howl.tsv`, including shared gate and visibility
  delegations.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Verification

`howl-depth` was GREEN with `--show-oracle` at seed 1 and without divergence
at seeds 2, 3, 5, and 8. The focused registration test passed. The local
gates all passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, and a clean `gofumpt -l .` check.
PR #952's hosted test, lint, and security checks were green before merge.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only the
unclaimed `hug` family at `src/interpreter.c:504`. Continue the command-table
sweep in source order with one slice/one PR and the non-green-check safety
rule.
