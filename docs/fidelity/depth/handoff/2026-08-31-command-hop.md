# Depth-fidelity handoff: `hop`

Date: 2026-08-31

## Queue position and frontier

This session began from clean `main` after the `home` handoff. The pre-slice
frontier was 2,180 total cases: 2,120 proven/delegated, 16 blocked, and 44
excluded. The `hop` manifest adds 11 cases, producing 2,191 total: 2,131
proven/delegated, 16 blocked, and 44 excluded (2,131 of 2,147 actionable
cases, 99.3%).

The special-procedure inventory remains exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table queue is complete through `hop`; the next unclaimed family
is `house` at `src/interpreter.c:502`.

The slice is PR #948, branch `glm/depth-hop`, merged to `main` as
`1f78cdbac`. GitHub initially reported no checks for the PR, so the single
permitted retry was run with `gh workflow run "Dark Pawns CI/CD" --ref
glm/depth-hop`; test, lint, and security then passed. Build and deploy were
skipped by repository policy. No non-green PR was merged.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:501: { "hop", POS_RESTING, do_action, 0, 0 }
```

`src/act.social.c:102-151` is the complete `do_action` path for the command:
`find_action`, the `PLR_NOSHOUT` refusal, conditional `one_argument` parsing,
the no-argument pair, visible-room target lookup, missing-target text,
self-target pair, minimum victim-position check, and the actor/TO_NOTVICT/
TO_VICT target trio. `lib/misc/socials:400-408` supplies the authoritative
`hop` record: `hide=0`, minimum victim position `0`, and eight messages.
The `act()` recipient and invisibility behavior is the shared path in
`src/comm.c:2397-2558`.

The audited player-visible branches are:

- POS_RESTING command entry and the shared `PLR_NOSHOUT` refusal;
- no argument and the first-token/trailing-argument boundary;
- visible target success with actor, non-victim observer, and victim audiences;
- self target and missing target;
- a sleeping target passing the record's zero minimum-position gate, while the
  victim act remains undelivered because C does not add `TO_SLEEP` there;
- recipient-specific visibility and the record's `hide=0` behavior.

R1/R2/R3/R5e apply. There is no command RNG draw to reconcile, and the
multi-seed run confirms the stable social transcript. The position, noshout,
and visibility branches are shared behavior and remain delegated under R5b/R5c
to `fade.position-gate`, `dance-noshout`, and `socials-depth`; duplicating
those matrices here would violate the depth-testing ownership rule.

## RED/ GREEN result

The first four-client setup attempt failed while establishing the Go target
peer, before any probe command ran; it was discarded and is not counted as a
proof attempt. The reduced actor/observer/sleeping-target vehicle then ran on
clean `main` and was GREEN. No confirmed Go divergence existed: the existing
social record, generic `DoAction`, argument parser, and `Act` routing already
matched the C path.

The durable changes are therefore proof-only:

- `cmd/dp-oracle-diff/scenarios/hop-depth.txt` covers the authored branches
  with a room observer and sleeping target;
- `pkg/session/hop_test.go` pins the C command gate, social metadata, and all
  eight authored messages;
- `docs/fidelity/depth/hop.tsv` records the direct cases and shared
  delegations.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Verification

`hop-depth.txt` was run with `--show-oracle` at seed 1 and reported no
normalized divergence. The same post-edit vehicle reported no normalized
divergence at seeds 2, 3, 5, and 8. The focused registration and social-table
tests passed. The local gates all passed: `make fidelity-depth`, `go build
./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a clean
`gofumpt -l .` check. PR #948's hosted test, lint, and security checks were
green before merge.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only the
unclaimed `house` family at `src/interpreter.c:502`. Continue the command-table
sweep in source order with one slice/one PR and the non-green-check safety
rule.
