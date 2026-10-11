# PR 4 design: prove the 57 Lua seam producers (DP-1371 D7)

Base `origin/main` `ee8e93467`, worktree `/home/zach/dp-mudlog-pr4`, branch
`deepseek/mudlog-pr4`. Proof-only tier (Claude merges). GLM's Lua-fallback
removal is already merged (#1890), so every proof runs against the bridge
players actually use.

## Scope

The 57 `implemented-needs-proof` rows in
`2026-10-09-dp-1371-lua-needs-proof-survey.tsv`. Their Go bindings live in
`pkg/scripting/bindings_bridge.go` (the `bridged` wrapper), which production
always reaches. The survey's `reachable_via_real_lua_call` column is `yes` for
all 57: a script calling the binding with the bad argument.

## Ground truth (not transcribed)

`2026-10-11-dp-1371-lua-proofs.tsv` is **generated** by
`2026-10-11-dp-1371-lua-proofs-extract.py`, which walks `src/scripts.c` from
each survey site to the `mudlog` call it guards and reads the payload, type,
level and file flag. All 57 rows are identical in the last three columns:
**BRF / LVL_IMMORT / file FALSE**, and the payloads are C's `[Lua] …`
diagnostics verbatim (typos included, e.g. `lua_extchar`'s missing period).

## The test

Table-driven, one row per survey site, following the harness
`lua_seam_producers_test.go` established (a real `writeScript` +
`NewEngine` + `RunScript` through `ScriptContext{World: bridge, MeRef: &me,
OwnerType: "mob"}`), with `recordingBridge` as the observation point.

```go
type luaProofCase struct {
    site    string // src/scripts.c:NNN, the survey's site column
    name    string // binding name for the subtest
    call    string // a Lua call that passes the bad argument
    payload string // from the generated TSV
}
```

Each subtest asserts the observable contract exactly:

* exactly one `Bridge.Log` line equal to `payload`, and the file-TRUE sink
  (`fakeBridge.logFiles`, i.e. `Bridge.MudLog`) stayed **empty** — that pins the
  bytes and the file flag at the bridge boundary;
* the script still runs to completion (`RunScript` returns no error and counts
  as handled), and no other diagnostic appears in the log list.

`lua_tport`'s `return -1` (`src/scripts.c:1532`) is C undefined behaviour when
Lua reads it as a value, so that row is recorded as **R1a** and asserts the log
only — no assertion on the returned value.

**Where type and level are pinned.** The bindings emit through
`Bridge.Log(msg)`, which carries no type/level/file: the fake records it via
`fakeBridge.Log` as a bare message. So the 57-row table pins the payload and the
file flag (its assertion is the whole log sequence — exactly one expected
payload, so an unrelated extra log fails the row), but **not** the type and
level. Those live one layer out:
`game.WorldScriptableAdapter.Log` (pkg/game/world_bridge.go:484) is
`MudLog(msg, MudlogBrief, lvlImmort, false)` — BRF / `LVL_IMMORT` / file FALSE,
exactly the generated TSV's contract for all 57.
`TestScriptAdapterLogIsBriefImmortalNoFile` (pkg/game) pins that mapping once
for all 57: an observer with `LOG1` receives the line, one with syslog off does
not (BRF is the lowest type, so that is what distinguishes it from NRM/CMP), a
level-30 observer does not, and the file capture stays empty. Asserting
typ/level/toFile from the scripting table would have been asserting the fake,
not the port.

## Controls scope (eight families, not 57 mutations)

`2026-10-11-dp-1371-pr4-lua-proofs-controls.py` removes the producer of **one
representative binding per D7 family** (`lua-proof-01`…`08`) and requires that
family's subtest to fail on the tightened assertion — a green there claims each
family can fail, not that all 57 rows were mutated. The survey's group column
has eight families (seven of eight rows, one of one), which is what the data
supports.

## Controls (R5h)

One producer per family, family = the survey's D7 group (`lua-proof-01` …
`lua-proof-08`), as the existing `*-controls.py` scripts do: remove that
binding's producer, require the named subtest's assertion failure (never a build
failure), restore, re-prove green. Each control cites the C site it removes.

## Evidence

* A full combined census at the implementation tip.
* The generated TSV plus the extractor, so the table can be re-derived from
  `src/` at review time (R5g).
