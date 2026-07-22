# BRIEF (codex) — zone-reset draw parity: R-command + O/P/G/E ordering (DP-1162 follow-up)

**Owner:** codex. **Gate:** Claude runs `combat-swing` red→green + full sweep under `DP_CLOCK`. **Branch off `main`, one PR.**
Diagnosis source: `docs/reports/boot-draw-parity-report.md` (mimo — measured per-zone draw deltas). **Read it, but apply the three corrections below — one of its stated mechanisms is wrong, and implementing to its literal rationale would add needless/incorrect code.** All C references verified against the oracle tree.

## The offset
`combat-swing` (and any post-boot scenario) shows Go **+2 RNG draws ahead of C**, accrued entirely during **boot zone reset**. Two independent Go bugs in `pkg/game/spawner.go` `ExecuteZoneReset` net to +2 (Bug A ≈ +6, Bug B ≈ −4). **Both must be fixed** — fixing one alone leaves a different nonzero offset.

## Bug A — `R` command sets `lastCmd` unconditionally
`spawner.go:446-453` sets `lastCmd = 1` even when the target isn't found. C (`db.c:2221-2245`, `case 'R'`) sets `last_cmd = 1` **only on a successful removal**. A failed `R` in Go wrongly enables subsequent `if_flag=1` commands (they run + draw); C skips them. `R` itself draws nothing — the damage is purely the `if_flag` cascade.

**Fix A:**
- Change `removeObjectFromRoom` and `removeMobFromRoom` (`spawner.go:~589-631`) to return `bool` (did anything get removed).
- Set `lastCmd = 1` only when the removal returned true. Preserve the existing Go parser arg convention (`Arg3==1`⇒object else mob; `Arg2`=vnum).

## Bug B — O/P/G/E create the object *after* `percentLoad`; C creates it *before*
C's `read_object` (`db.c:1799`) increments `obj_index[].number++` and, for `ITEM_RARE` objects, calls `init_rare` — **which draws** — and this happens **before** `percent_load` (`db.c`: `percent_load` is `GET_OBJ_LOAD(obj) > uniform()*100`, one draw). Go inverts it: `percentLoad(proto)` (`spawner.go:208`, one `Uniform()` draw) runs first, then `SpawnObject` → `initRare` (`spawner.go:526`). For rare items this shifts the shared stream position at the `percent_load` draw, flipping pass/fail and cascading through `if_flag` chains.

**Fix B — reorder every load command to C's order: create (→`initRare`) → then `percentLoad` → extract on fail.** Match C per-command exactly (guards that precede creation in C must precede it in Go, so a failed guard draws nothing):

| Cmd | C ref | Order to replicate |
|-----|-------|--------------------|
| `O` | `db.c:2149` | max-check → **create**; if `arg3>=0`: `percentLoad` → ok: keep,`lastCmd=1` / fail: extract. If `arg3<0` (floating): set NOWHERE, `lastCmd=1`, **no `percentLoad`**. |
| `P` | `db.c:2170` | max-check → **create** → find container; **if container missing: stop, no `percentLoad`** → else `percentLoad` → ok: put / fail: extract. (Note: Go currently finds the container *first* — move it to after create.) |
| `G` | `db.c:2186` | mob-check → max-check → **create** → `percentLoad` → ok: give / fail: extract. |
| `E` | `db.c:2199` | mob-check → max-check → **pos-check** → **create** → `percentLoad` → ok: equip / fail: extract. (Go currently pos-checks *after* `percentLoad` — move it before create.) |

**Extract-on-fail is mandatory and must decrement the instance count.** Because the reorder now increments the count at create time (before the `percentLoad` gate), a failed `percentLoad` must extract the just-created object AND decrement whatever `CanSpawn`/max-in-world reads (`s.objInstances` / obj-instance count) — exactly mirroring C's `read_object`(+1) then `extract_obj`(−1) (`handler.c:1030` decrements `obj_index[].number`). Net effect: +1 on success, 0 on fail — identical to C. You'll need a spawner-callable extract that undoes a spawn (create it if absent).

## Three corrections to mimo's report — do NOT implement its literal rationale
1. **Its "Bug B effect #2: max-in-world count divergence" does not exist.** C's `extract_obj` decrements the count on `percent_load` failure, so C and Go counts already match at every command boundary. Do **not** add logic to "reach max sooner" or diverge counts. The extract-on-fail decrement above is needed only because the *reorder* increments before the gate — not because of any count mismatch.
2. **Its per-zone cause attribution is loose** (e.g. it pins zone 48's "+2" on the nonexistent count divergence; it's actually Bug A / cascade). Trust the *measured deltas*, not the per-zone story. Your target is simply: both fixes applied ⇒ zero net offset.
3. **The floating (`arg3<0`) and invalid-`E`-pos and missing-`P`-container edges don't occur in the currently-failing zones** (43/48/70/191) — but the table above specifies them for full C-parity since you're rewriting these handlers anyway. Getting them right is free and correct; getting them wrong is a latent draw bug.

## Files
`pkg/game/spawner.go`: `O`(296-329), `G`(331-357), `E`(359-393), `P`(395-424), `R`(446-453), `removeObjectFromRoom`/`removeMobFromRoom`(~589-631), `percentLoad`(208), `SpawnObject`+`initRare`(~505-560), `CanSpawn`. Do **not** touch draw sites elsewhere, add compensating draw-burns, or hack fixtures.

## Acceptance (Claude-gated, from a PR-branch worktree)
1. `--scenario combat-swing` → `no normalized divergence`.
2. **Full committed sweep under `DP_CLOCK` stays green.** Fix B changes which objects load at boot in rare-object zones, so this is the real risk surface — every scenario must be re-run, not just combat-swing.
3. `DP_CLOCK`/`DP_SEED` unset ⇒ byte-identical world/reset behavior to today (no gameplay change when not oracle-testing — verify the reorder doesn't alter normal boots, only draw *order*).
