# 2026-08-29 — `normal_checker` depth slice

## Frontier and queue

- Started from clean `main` after the merged horn boundary, pulled, ran
  `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the
  newest handoff. The pre-slice frontier was 966 total cases; this slice adds
  seven cases, yielding 973 total, 946 proven/delegated, 6 blocked, and 21
  excluded; actionable completion is 946/952 (99.4%).
- The C inventory was refreshed across `src/spec_procs.c`,
  `src/spec_procs2.c`, and `src/spec_procs3.c`, and checked against the active
  registration tables. The next active, unclaimed definition after
  `normal_checker` is `whirlpool` at `src/spec_procs2.c:244`, registered for
  mob vnum 12200 at `src/spec_assign.c:342`.
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(normal_checker)` is defined at `src/spec_procs2.c:162-187` and is
  registered for mob vnums 18301-18304 at `src/spec_assign.c:411-414`.
- The reachable caller is autonomous `mobile_activity()` at
  `src/mobact.c:68-93`: it skips non-mobs, fighting mobs, and non-awake mobs,
  invokes the registered special as `(ch, ch, 0, "")`, and skips to the next
  mobile when the special returns `TRUE`.
- The special itself rejects nonzero command, non-awake, negative-HP, and
  fighting states; scans the C room people list; selects the first non-NPC
  below `LVL_IMMORT`; emits the capitalized `TO_NOTVICT` jump line; sends the
  victim warning; calls `hit(ch, i, TYPE_UNDEFINED)`; and returns `TRUE`.
  No eligible target returns `FALSE`.

## RED/GREEN evidence and port result

- RED on clean `main` with `spec-proc-normal-checker` first exposed the Go
  `< 50` target gate incorrectly selecting the level-40 actor instead of the
  level-1 peer. After correcting that gate, the second honest RED exposed the
  Go placeholder attack path (`hit` versus `maul`, wrong miss wording) and the
  missing C `TO_NOTVICT`/victim audience split.
- GREEN on `glm/spec-normal-checker`:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-normal-checker --show-oracle`
  reports `result: no normalized divergence`. The two-client vehicle proves
  target selection, exact warning and observer bytes, synchronous native maul
  opener, and score state.
- The Go fix uses canonical `Act` routing for the two C message calls, routes
  the special through the synchronous `mobHit` seam, and wires parsed
  `BareHandAttack` into the shared combat attack-type callback. Focused proof
  is in `TestSpecNormalChecker_EntryGates` and
  `TestSpecNormalChecker_UsesCanonicalSynchronousHit`.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and integration

- Feature-branch gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #754 (`glm/spec-normal-checker`) received green lint, security, and test
  checks; build/deploy were skipped by workflow policy. It was squash-merged
  into `main` as `c811b8824`; no CI retry was needed because checks fired
  normally.

This slice applies R1 (player-facing bytes), R2 (autonomous registered
special/return surface), R3 (native attack verb and deterministic opener), R4
(no invented output), and R5/R5e (actual C dispatch/call path and oracle
verification).

## Next queue item

Continue the active special-procedure inventory with `whirlpool` from
`src/spec_procs2.c:244`, using registered mob vnum 12200. Do not re-pick
`normal_checker`, `horn`, `bank`, or any earlier claimed procedure. After the
active inventory is exhausted, attempt the single blocked
`objmagic.sleep-entry-gates` vehicle, then sweep remaining un-manifested
command families in `src/interpreter.c` table order.
