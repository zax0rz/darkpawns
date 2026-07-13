# BRIEF 2026-07-11 — GLM: three Reek MED bug fixes (DP-1016/1017/1018)

**Executor:** GLM (zai coding plan). **Branch:** `glm/reek-med-bugs-2026-07-11`
(fresh off current `main`). **One PR** when all three are green.

These are the three MED bugs from Reek's overnight sweep. All small, all
mechanical. Claude took the HIGH (DP-1019, game-loop recover) and the fidelity
MED (DP-1015, charm-skip) — both already in PRs #144 / #143, **do not touch
those files**.

## Ground rules (same as the sweep that went well)
1. **Work in your OWN clone or a `git worktree`.** Never share a working dir /
   HEAD with another agent. `git status` clean before you start and before each
   commit.
2. **Verify each finding against CURRENT `main` before fixing.** Line numbers
   below were checked today but may drift.
3. **One commit per item**, scoped, each with a regression test that fails
   before / passes after. End each message with your own `Co-Authored-By:` line.
4. Run `go build ./... && go test -race ./... -timeout 120s` before pushing.
   `gofmt -l` must be empty (CI runs `lint` + `test`).

---

## A1. DP-1018 — `pkg/engine/affect.go` `GetType()` is non-deterministic

`Affect.GetType()` (line ~143) resolves a status affect by ranging over the
`StatusAffectFlags` map (declared at line ~331, `map[int]uint64`):

```go
for affType, flags := range StatusAffectFlags {
    if a.Flags&flags != 0 {
        return affType
    }
}
```

Go map iteration order is randomized, so an affect with **multiple** AFF bits set
(e.g. `AFFBlind|AFFPoison`) returns a different `affType` on different calls. That
breaks the deprecated `Type` field's save/load round-trip: two saves of the same
affect can disagree.

**Fix:** make resolution deterministic — return the affType whose flag has the
**lowest set bit** among the affect's flags (i.e. iterate candidates in ascending
flag-value order, return the first that matches). Implementation options, pick the
simplest:
- Snapshot `StatusAffectFlags` into a `[]struct{affType int; flag uint64}` sorted
  by `flag` ascending (build once via `sync.Once`/package init), and range that
  slice in `GetType()`; **or**
- Walk bit positions 0..63 low-to-high, and for the first bit set in `a.Flags`,
  look up which affType owns `1<<bit`.

Keep `SetType` as-is. Don't change the `StatusAffectFlags` map contents.

**Test** (`pkg/engine/affect_test.go`): construct an `Affect` with two flags set
(`AFFBlind|AFFPoison`), call `GetType()` ~1000× (or a handful — determinism is
the point), assert the result is always the same value, and that it's the lower
affType (Blind=100 < Poison=111 → expect 100). Add a single-flag case too.

---

## A2. DP-1017 — `pkg/testutil/helpers.go` mock returns nil writer

`MockDatabase.NewDecisionLogWriter()` (line ~247) returns `nil`. Any test that
calls `RecordDecision`/`Stop` on the result nil-panics. No current test does, so
it's latent — but it's a footgun.

**Fix:** return a minimal usable writer:

```go
func (m *MockDatabase) NewDecisionLogWriter() *db.DecisionLogWriter {
    return &db.DecisionLogWriter{stopCh: make(chan struct{})}
}
```

**BUT** `stopCh` is an **unexported** field of `db.DecisionLogWriter`, so
`pkg/testutil` (a different package) **cannot** set it with a composite literal.
Two clean ways out — pick one:
- **Preferred:** add an exported constructor in `pkg/db`, e.g.
  `func NewMockDecisionLogWriter() *DecisionLogWriter { return &DecisionLogWriter{stopCh: make(chan struct{})} }`
  (no background flush goroutine, no `db` handle), and have the mock call it.
- Or make `NewDecisionLogWriter` on a nil/mock `*DB` return such a writer.

I verified the nil-safety of the minimal writer against `pkg/db/decision_log.go`:
`Stop()` → `close(stopCh)` + `flushWG.Wait()` (zero) + `Flush()`, and `Flush()`
**early-returns when both buffers are empty** (line ~325) so it never touches the
nil `db`. `RecordDecision` just appends (safe) unless the buffer reaches
`flushBatchSize` and triggers a `Flush` of real records → that would deref the nil
`db`. That heavy path is out of scope; document in the constructor doc-comment
that the mock writer buffers but does not persist, and is intended for
construct/record/Stop paths in tests.

**Test** (`pkg/testutil/helpers_test.go` or `pkg/db`): get the writer from the
mock, call `RecordDecision(&db.DecisionRecord{...})` then `Stop()`, assert no
panic.

---

## A3. DP-1016 — `pkg/admin/agent_store.go` swallows `MkdirAll` error

`NewAgentStore(filePath string) *AgentStore` (line ~93) logs but ignores the
`os.MkdirAll` error at line ~96 and returns a store anyway. Every later `Save()`
then fails silently because the dir doesn't exist — mutations look like they
succeed but nothing persists.

**Fix:** change the signature to `NewAgentStore(filePath string) (*AgentStore, error)`
and return the wrapped `MkdirAll` error so the caller fails loudly at boot.
Update the **one** caller: `pkg/admin/router.go:128`
(`agentStore := NewAgentStore(storePath)`) — propagate/handle the error there
(return it up, or log-fatal at boot, matching how sibling constructors in that
file report fatal init errors — follow local convention, don't invent a pattern).

`grep -rn "NewAgentStore" pkg cmd` (excluding tests) shows only `router.go:128` +
the definition. Re-grep including `_test.go` and fix any test callers too.

**Test** (`pkg/admin/agent_store_test.go`): call `NewAgentStore` with a path
whose parent is unwritable (e.g. a file where a directory is expected, or a path
under `t.TempDir()` with a file segment in the middle so `MkdirAll` fails), assert
a non-nil error is returned. Add/keep a happy-path test under `t.TempDir()`
asserting `err == nil` and that a subsequent `Save`+reload round-trips.

---

## Do NOT touch (other lanes)
- **Claude:** DP-1015 (`pkg/spells/affect_spells.go`, `pkg/game/charm.go` — PR #143),
  DP-1019 (`pkg/engine/gameloop.go` — PR #144). Leave those files alone.
- Anything already in an open PR — `git log`/`gh pr list` before assuming a
  finding is open.

When done: open ONE PR `glm/reek-med-bugs-2026-07-11 → main`, and in the body
list each item (DP-1016/1017/1018) with its verdict (fixed / skipped-why) and the
test name. Claude verifies the diff + closes the Linear issues.
