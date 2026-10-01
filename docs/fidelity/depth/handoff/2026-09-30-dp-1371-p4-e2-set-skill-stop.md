# DP-1371 Phase 4 E2: set-skill stopped at a shared attribute gap

Base: `origin/main` at `0fb0d662d46e9343c82f8d889ad963ca2f64cbda`
(after #1731). Batch: `lua.bind-set-skill` only. **The row remains blocked**;
no manifest status, production code, scenario corpus, oracle or governing
rule changes. This PR contains the diagnosis and proposed follow-up.

## Confirmed finding (R1, R5e/R5g)

`lua_set_skill` writes SET_SKILL then calls `affect_total`
(`src/scripts.c:1365-1383`, specifically :1378-1379).
Go's `WorldScriptableAdapter.SetSkill` maps spell number to a player skill,
but its comment claims that affect_total changes nothing in the port.
A real Lua-engine test confirms that skill 134 sets kick to 37 correctly,
then fails on effective DEX: **20, expected C's 18**.

The shipped spider silk vest, `lib/world/obj/144.obj:357-374` (prototype
14425), carries APPLY_DEX +2. A first-player God with base DEX set to 18
loads and wears it, then executes `stat Skillprobe`, the Lua set-skill probe,
and `stat Skillprobe` again. C prints DEX 18 in both stat blocks; Go prints
20 in both. The healer's `skill set` line matches, confirming that the Lua
probe executed. This is a pre-existing equipment/stat defect **before**
set-skill, not solely a missing Lua callback.

C's `affect_total` removes equipment and spell modifiers, removes tattoo
modifiers, restores real abilities, reapplies each family, then caps DEX,
INT, WIS and CON to [0,18] for PCs or [0,25] for NPCs
(`src/handler.c:314-358`). STR is nonnegative; NPC STR caps at 25; PC STR
above 18 converts excess into exceptional strength, capped at 100
(`:359-368`). Alignment caps at ±1000 (`:370-373`). C does **not** apply
this same upper cap to CHA; a repair must preserve that distinction.

The broader callers include equipment add/remove (`src/handler.c:748,792`),
affect add/remove (`:395,419,437`), wizard set, and Lua save/set-skill.
Go's effective attribute getters sum base, spell and equipment modifiers
without those bounds (`pkg/game/player_combat.go:53-82`,
`pkg/game/player_social.go:27-30`). Thus repairing this faithfully requires
shared behavior beyond the named Lua batch.

## Retained reproductions

All evidence is under `~/Archives/darkpawns/oracle-runs/2026-09-30/`:

- `dp-1371-p4-e2-skill-clamp-diagnostic/`: initial targeted reference census.
- `dp-1371-p4-e2-skill-clamp-main/`: independent detached `origin/main`
  checkout, same diagnostic scenario and reference oracle, same red.
- `dp-1371-p4-e2-skill-reproducer/`: exact test and scenario, failing main
  unit log, development `--show-oracle` report, C caller grep and Go getter
  grep, revision/hash/command metadata. Neither reproducer is shipped as a
  passing proof or placed in the checked-in scenario corpus.

The main unit exits 1 **on the DEX assertion**, while the kick=37 assertion
passes. Main oracle exits 1 for content divergence, not infrastructure.

```text
oracle-regression: scenarios=1 passed=0 expected=0 unpinnable=0 stale=0 failed=1 infra=0 timed_out=0 unstable=0 elapsed=27.364s started=2026-09-30T21:50:24-0400 finished=2026-09-30T21:50:51-0400 verdict=NOT_CLEAN
oracle-regression: scenarios=1 passed=0 expected=0 unpinnable=0 stale=0 failed=1 infra=0 timed_out=0 unstable=0 elapsed=22.593s started=2026-09-30T21:52:07-0400 finished=2026-09-30T21:52:30-0400 verdict=NOT_CLEAN
```

The report also contains STL 0/2, AC 110/100 and omitted Poofin/Poofout
lines. These are **additional observed differences, not diagnosed repairs**.
C report sites: `src/act.wizard.c:768,843,902`. They remain outside this
batch; do not normalize them away or invent a causal explanation.

## Proposed follow-up and decision

The goal brief says to stop when “Production code is wrong in a way outside
the current batch.” This shared attribute defect meets that condition.

Propose a separately authorized **shared affect_total/effective-attribute
repair** before completing set-skill:

1. Map every C affect_total caller and the Go base/effective-stat representation,
   including PC/NPC equipment, spell affects, tattoo, and exceptional strength.
   Establish which callers require recomputation and which can use an
   equivalent derived view; preserve C's ordering and base values.
2. Implement one shared C-equivalent effective-ability computation at those
   proven boundaries. No set-skill-only correction or mutation of permanent
   base stats to hide an equipment bonus. Keep save format unchanged.
3. Prove positive/negative modifiers, lower/upper caps, PC/NPC limits, STR
   conversion and removal restoration, with assertion-based revert pairs.
   Use the retained vest reproduction; diagnose the other stat-report reds
   in their own scopes rather than declaring a whole report green.
4. Run full and claims censuses because this changes shared game/Lua paths,
   then resume the set-skill readback/effective-state proof.

Zach's decision needed: authorize this shared repair (and its class audit),
or specify a narrower next scope. No new divergence approval, pin, dropped
seed or governing-document amendment is proposed. This report does not
claim to have audited every caller or completed the repair design.

## Branch validation

The diagnostic fixtures and failing test are retained outside the branch.
All required PR gates passed individually: `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint cache clean`,
`golangci-lint run ./...`, `git diff --check`, `make fidelity-depth`,
`make fidelity-units`, `make string-census`. These validate this docs-only
PR; they do not turn either diagnostic census green. There remain **71
blocked cases**, including `lua.bind-set-skill`.
