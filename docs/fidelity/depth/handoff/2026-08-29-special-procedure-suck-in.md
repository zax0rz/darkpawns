# 2026-08-29 — `suck_in` depth slice

## Frontier and queue

- Started from the clean `main` boundary after the `newbie_zone_entrance`
  handoff, pulled `b8c44128f`, ran `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- Main baseline for this session remains 930 total cases; 903
  proven/delegated; 6 blocked; 21 excluded; actionable completion 903/909
  (99.3%). The feature branch adds seven manifest cases, yielding 937 total;
  910 proven/delegated; 6 blocked; 21 excluded; actionable completion 910/916
  (99.3%) on that branch.
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(suck_in)` is assigned to room 20073 at
  `src/spec_assign.c:616` and defined at `src/spec_procs.c:2279-2299`.
  Room-special dispatch runs before ordinary command dispatch at
  `src/interpreter.c:1407-1415`; a `FALSE` result reaches the ordinary
  handler.
- The procedure first calls the `do_look` entry point, returns `FALSE` for
  `mini_mud` or any command other than exact `look`, and consumes one
  C-faithful `one_argument` token. Only the token `painting` enters the
  transition arm; other arguments, including `picture`, fall through.
- The painting arm calls `do_look` on the extra description, sends the exact
  dizzy message to the actor, emits `$n suddenly vanishes!` through
  `TO_ROOM` (excluding the actor), removes the actor from room 20073, moves it
  to `PAINTING_ROOM` 18101, renders the destination room, and returns `TRUE`.
  The seven manifest rows cover command and argument gates, look output,
  embedded blank-line bytes, audience, relocation, and return interception.
- Authoritative `lib/world/wld/181.wld` exists, but `181.wld` is absent from
  the production `lib/world/wld/index` in both repository copies. The
  production world and C oracle tree were not edited. The focused vehicle
  uses a disposable `add-wld-index 181.wld` fixture to load that existing
  authoritative room file and proves the intended C call path under the
  same temporary world in both engines. Without that fixture, C's
  `real_room(18101)` is invalid, so no production-world claim is made for
  the unreachable destination data.

## RED/GREEN evidence and port result

- RED on pre-fix `main`: the valid vehicle exposed a nil room-special receiver
  panic in Go. Once the disposable destination index made the intended path
  reachable, the same vehicle also exposed missing initial `do_look` bytes,
  incorrect room audience handling, missing destination rendering, and
  mismatched transition blank lines.
- GREEN on `glm/spec-suck-in`:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-suck-in --show-oracle`
  reports `result: no normalized divergence` for `look picture` fallthrough
  and `look painting` transition/output. The focused
  `TestSpecSuckIn_LookGateAudienceAndRelocation` test pins the exact
  actor/observer audience, actor exclusion, parser gate, relocation, and
  destination state.
- No `src/` or `darkpawns-c-oracle/` file was edited. The harness change is
  disposable-fixture support only; it does not alter production world data.

## Verification and integration

- Feature-branch gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #749 (`glm/spec-suck-in`) is open with commit `a1c7bfa77`. No checks
  fired initially. The required single retry was issued with
  `gh workflow run "Dark Pawns CI/CD" --ref glm/spec-suck-in`; lint,
  security, and test were still pending at inspection. Under the standing
  rule, this PR is treated as not-green and was not merged.

This slice applies R1 (player-facing bytes), R2 (registered room command
surface), R4 (no invented output), and R5/R5e (actual C dispatch/call path and
oracle verification). The open PR claims the slice so it must not be
re-picked.

## Next queue item

Continue special-procedure source order with `oro_quarters_room`, defined at
`src/spec_procs.c:2302` and assigned to room 18397 at
`src/spec_assign.c:614`. Do not re-pick `suck_in` or any earlier claimed
procedure. After the active special-procedure inventory is exhausted, attempt
the single blocked `objmagic.sleep-entry-gates` vehicle, then sweep remaining
un-manifested command families in `src/interpreter.c` table order.
