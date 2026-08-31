# Depth-fidelity handoff: `hump`

Date: 2026-08-31

## Queue position and frontier

This session began from clean `main` after the `hum` handoff. The pre-slice
frontier was 2,238 total cases: 2,178 proven/delegated, 16 blocked, and 44
excluded. The `hump` manifest adds 11 cases, producing 2,249 total: 2,189
proven/delegated, 16 blocked, and 44 excluded (2,189 of 2,205 actionable
cases, 99.3%).

The special-procedure inventory remains exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table queue is complete through `hump`; the next unclaimed family
is `hush` at `src/interpreter.c:507`.

The slice was PR #958, branch `glm/depth-hump`, merged to `main` as
`f86e9a201`. Its hosted test, lint, and security checks passed; build and
deploy were skipped by repository policy. No non-green PR was merged. The
prior howl handoff PR #953 remains open and not-green because checks never
materialized after its single permitted retry.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:506: { "hump", POS_RESTING, do_action, 0, 0 }
```

`src/act.social.c:102-151` is the complete `do_action` path. The `hump`
record at `lib/misc/socials:1232-1240` is `hump 0 5`: no hide-invisible
suppression and a POS_STANDING victim minimum. All eight message slots are
present: no-argument actor/room, target actor/room/victim, not-found, and
self-target actor/room lines.

The audited player-visible branches are:

- POS_RESTING command entry and the shared PLR_NOSHOUT refusal;
- no-argument actor and room output;
- C `one_argument` fill-word/trailing-argument parsing;
- visible standing-target success with actor, TO_NOTVICT observer, and victim
  audiences;
- self target and missing target;
- sleeping-target rejection at the social's POS_STANDING minimum;
- ordinary target/Act visibility behavior.

R1/R2/R3/R4/R5e apply. There is no RNG draw. The command position gate,
PLR_NOSHOUT gate, and target/Act visibility semantics are shared and delegated
under R5b/R5c to `fade.position-gate`, `dance-noshout`, and `socials-depth`;
the direct vehicles prove hump-specific messages, target minimum, and audience
topology.

## RED/ GREEN result

The first awake/sleeping vehicle placed the force-to-sleep warmup before the
whole probe stream, so `--show-oracle` reached the intended sleeping-target
refusal instead of awake target success. That setup was corrected and is not
counted as a proof attempt. The corrected awake vehicle and separate sleeping
vehicle were GREEN on the unchanged Go implementation. No confirmed Go
divergence existed, so this slice is proof-only:

- `cmd/dp-oracle-diff/scenarios/hump-depth.txt` proves no argument, fill-word
  target success, audience topology, self, and missing target;
- `cmd/dp-oracle-diff/scenarios/hump-sleeping-depth.txt` proves the standing
  victim gate with the wizard force vehicle;
- `pkg/session/hump_test.go` pins the C command gate, social metadata, and all
  eight authored messages;
- `docs/fidelity/depth/hump.tsv` records direct cases and shared delegations.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Verification

Both hump vehicles were GREEN with `--show-oracle` at seed 1 and without
divergence at seeds 2, 3, 5, and 8. The focused registration test passed. The
local gates all passed: `make fidelity-depth`, `go build ./...`, `go vet
./...`, `go test ./...`, `golangci-lint run ./...`, and a clean `gofumpt -l .`
check. PR #958's hosted test, lint, and security checks were green before
merge.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and the newest merged handoff, then take only
the unclaimed `hush` family at `src/interpreter.c:507`. Continue the
command-table sweep in source order with one slice/one PR and the non-green-
check safety rule.
