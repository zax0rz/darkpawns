# Depth-fidelity handoff — `smackheads` merge correction

Date: 2026-09-03

Existing feature branch: `glm/depth-smackheads` (feature merged); correction handoff branch:
`handoff/2026-09-03-command-smackheads-merge`

Feature PR: #1248 (merged green)
Original feature commit: `a1f513264`
Rebase commit: `08489dcf5`
Feature merge commit: `7346a588c`

## Queue position and result

This is a continuity correction for the already-claimed `smackheads` slice; it is not a repick. The earlier handoff `2026-09-03-command-smackheads.md` left PR #1248 open because its hosted test was pending. After concurrent main work landed, GitHub reported a merge conflict in `pkg/game/death.go`; the existing feature was updated in an isolated worktree, preserving current main's `announceCorpse` guard and the proven `SkillSmackheadsNum` suppression. All gates then passed, hosted checks reran green, and PR #1248 was self-merged. Do not repick smackheads.

The special-procedure inventory remains exhausted. `objmagic.sleep-entry-gates` remains blocked after its one allowed cast-sleep outlaw/reagent attempt and was not repicked.

Before integrating smackheads, current main frontier was 3,989 total, 3,885 proven/delegated, 48 blocked, 56 excluded. The existing smackheads manifest adds 20 proven/delegated rows. Current main frontier is 4,009 total, 3,905 proven/delegated, 48 blocked, 56 excluded; actionable completion is 3,905/3,953=98.8%.

## C path/evidence boundary

The original handoff and feature PR already documented C path and proof:
`src/interpreter.c:711` -> `src/new_cmds.c:966-1102`, two targets, skill/target/hands/mount/fighting/peaceful gates, miss/hit branches, ordered damage/waits, and all durable scenarios/manifest/unit tests. No src/oracle edits.

The only integration change was the safe `pkg/game/death.go` condition:
`announceCorpse` remains required, and `SkillSmackheadsNum` is also excluded from generic corpse announcement. This preserves R1/R3/R4/R5e/R5c.

## Gates/review

Isolated rebase local gates passed: make fidelity-depth, go build, go vet, go test, golangci-lint, gofumpt, git diff --check. Updated PR #1248 run 33751020735: lint/security/test green; build-and-push/deploy skipped. No workflow retry; CI fired normally. PR self-merged only after all applicable checks green.

## Continuation

The next fresh main session must rerun the frontier and newest handoff, then take the next dedicated un-manifested source-order family `muhaha` at `src/interpreter.c:560`. The `:` alias is already owned by emote/communication coverage. Do not repick smackheads or any earlier claimed family.
