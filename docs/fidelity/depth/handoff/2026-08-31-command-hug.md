# Depth-fidelity handoff: `hug`

Date: 2026-08-31

## Queue position and frontier

This session began from clean `main` after the `house` handoff; the `howl`
feature proof had also been completed, but its handoff PR remains open because
hosted checks did not fire after the one permitted retry. The pre-slice frontier
for `hug` was 2,219 total cases: 2,159 proven/delegated, 16 blocked, and 44
excluded. The `hug` manifest adds 11 cases, producing 2,230 total: 2,170
proven/delegated, 16 blocked, and 44 excluded (2,170 of 2,186 actionable
cases, 99.3%).

The special-procedure inventory remains exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table queue is complete through `hug`; the next unclaimed family
is `hum` at `src/interpreter.c:505`.

The slice was PR #954, branch `glm/depth-hug`, merged to `main` as
`ca821b251`. Its hosted test, lint, and security checks passed; build and
deploy were skipped by repository policy. No non-green PR was merged. The
prior howl handoff PR #953 is still open and not-green due to absent checks.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:504: { "hug", POS_RESTING, do_action, 0, 0 }
```

`src/act.social.c:102-151` is the complete `do_action` path. The `hug`
record at `lib/misc/socials:410-418` is `hug 1 5`: hide-invisible actor is
enabled and the victim must be at least POS_STANDING. Its eight message slots
provide the no-argument actor line with a `#` room sentinel, target actor/
room/victim lines, not-found line, and self-target actor/room lines.

The audited player-visible branches are:

- POS_RESTING command entry and the shared PLR_NOSHOUT refusal;
- no-argument output and the `#` room suppression;
- C `one_argument` fill-word/trailing-argument parsing;
- visible standing-target success with actor, TO_NOTVICT observer, and victim
  audiences;
- self target and missing target;
- sleeping-target rejection at the social's POS_STANDING minimum;
- hide=1 visibility behavior.

R1/R2/R3/R4/R5e apply. There is no RNG draw. The command position gate,
PLR_NOSHOUT gate, and target/Act visibility semantics are shared and delegated
under R5b/R5c to `fade.position-gate`, `dance-noshout`, and `socials-depth`;
the direct vehicles prove hug-specific messages, target minimum, and audience
topology.

## RED/ GREEN result

The first four-client vehicle placed the `force ... sleep` warmup before the
probe, so `--show-oracle` showed the expected sleeping-target refusal instead
of target success. That vehicle was corrected and is not counted as a proof
attempt. A second setup attempt exposed an infrastructure limit while adding
the fourth client (Go target setup reached EOF before any hug command); it was
also discarded before proof and no Go behavior was changed for either setup
issue.

The corrected awake and sleeping vehicles were GREEN on the unchanged Go
implementation. No confirmed Go divergence existed, so this slice is
proof-only:

- `cmd/dp-oracle-diff/scenarios/hug-depth.txt` proves no argument, fill-word
  target success, audience topology, self, and missing target;
- `cmd/dp-oracle-diff/scenarios/hug-sleeping-depth.txt` proves the standing
  victim gate with the wizard force vehicle;
- `pkg/session/hug_test.go` pins the C command gate, social metadata, and all
  eight authored messages;
- `docs/fidelity/depth/hug.tsv` records direct cases and shared delegations.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Verification

Both hug vehicles were GREEN with `--show-oracle` at seed 1 and without
divergence at seeds 2, 3, 5, and 8. The focused registration test passed. The
local gates all passed: `make fidelity-depth`, `go build ./...`, `go vet
./...`, `go test ./...`, `golangci-lint run ./...`, and a clean `gofumpt -l .`
check. PR #954's hosted test, lint, and security checks were green before
merge.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and the newest merged handoff, then take only
the unclaimed `hum` family at `src/interpreter.c:505`. Continue the
command-table sweep in source order with one slice/one PR and the non-green-
check safety rule.
