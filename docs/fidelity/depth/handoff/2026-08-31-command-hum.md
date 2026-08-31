# Depth-fidelity handoff: `hum`

Date: 2026-08-31

## Queue position and frontier

This session began from clean `main` after the `hug` handoff. The pre-slice
frontier was 2,230 total cases: 2,170 proven/delegated, 16 blocked, and 44
excluded. The `hum` manifest adds 8 cases, producing 2,238 total: 2,178
proven/delegated, 16 blocked, and 44 excluded (2,178 of 2,194 actionable
cases, 99.3%).

The special-procedure inventory remains exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table queue is complete through `hum`; the next unclaimed family
is `hump` at `src/interpreter.c:506`.

The slice was PR #956, branch `glm/depth-hum`, merged to `main` as
`6af9aacd0`. Its hosted test, lint, and security checks passed; build and
deploy were skipped by repository policy. No non-green PR was merged. The
prior howl handoff PR #953 remains open and not-green because checks never
materialized after its single permitted retry.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:505: { "hum", POS_RESTING, do_action, 0, 0 }
```

`src/act.social.c:102-151` is the complete `do_action` path. The `hum`
record at `lib/misc/socials:1391-1394` is `hum 0 0`: no hide-invisible flag,
no victim-position minimum, and no `char_found` message (`#`). Since
`char_found` is absent, `do_action` does not parse or resolve any target and
always emits the no-argument actor/room pair.

The audited player-visible branches are:

- POS_RESTING command entry and the shared PLR_NOSHOUT refusal;
- no-argument actor and room output;
- visible, missing, and self-looking arguments ignored before target lookup;
- exact three-line social record bytes and ordinary room visibility.

R1/R2/R3/R4/R5e apply. There is no RNG draw. The command position gate,
PLR_NOSHOUT gate, and Act visibility behavior are shared and delegated under
R5b/R5c to `fade.position-gate`, `dance-noshout`, and `socials-depth`; the
direct vehicle proves the hum-specific self-only branch and bytes.

## RED/ GREEN result

The annotated `hum-depth` vehicle was run against the unchanged Go path from
clean `main`, with `--show-oracle` at seed 1. It was GREEN: the C oracle
showed the exact actor and observer lines for no argument, a visible named
argument with trailing words, a missing name, and a self-looking name. No
confirmed Go divergence existed, so no behavioral Go fix was made.

The durable proof-only changes are:

- `cmd/dp-oracle-diff/scenarios/hum-depth.txt`, with the actor/observer
  vehicle and four annotated self-only cases;
- `pkg/session/hum_test.go`, pinning the C command gate, social metadata, and
  all three authored messages;
- `docs/fidelity/depth/hum.tsv`, including shared gate and visibility
  delegations.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Verification

`hum-depth` was GREEN with `--show-oracle` at seed 1 and without divergence at
seeds 2, 3, 5, and 8. The focused registration test passed. The local gates
all passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, and a clean `gofumpt -l .` check.
PR #956's hosted test, lint, and security checks were green before merge.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and the newest merged handoff, then take only
the unclaimed `hump` family at `src/interpreter.c:506`. Continue the
command-table sweep in source order with one slice/one PR and the non-green-
check safety rule.
