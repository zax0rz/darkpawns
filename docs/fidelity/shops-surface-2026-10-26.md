# Shop behavior audit — 2026-10-26

The shop command surface now follows C for counted purchases and sales, partial
fills, value dispatch, customer/keeper gates, prices, expressions, object
selectors, pet descriptions, bank audiences, recharging and stock-list paging.
This is a branch audit with explicit residuals, not a declaration that either
whole system is complete (R1/R2/R3/R5c/R5e/R5h).

## Denominator and ownership

The source-order map is [surface/shops-behavior.tsv](depth/surface/shops-behavior.tsv).
It records silent state/math branches as well as output. Its dedicated validator
is `scripts/shop_surface.py`, invoked by `make fidelity-depth`; the exemption in
`scripts/surface_map.py` applies only to this custom-schema file.

| Inventory family | Before | After: explicit blocked units | Branches | Proven | Fixed | Delegated (R5e) | Unreachable |
|---|---:|---:|---:|---:|---:|---:|---:|
| core shop transaction surface | 1 coarse blocked system | 14 | 181 | 31 | 124 | 6 | 6 |
| special-procedure shop surface | 1 coarse blocked system | 4 | 184 | 114 | 22 | 42 | 2 |

The numeric increase in units exposes the previously hidden denominator; it does
not represent eighteen newly introduced defects. All other inventory rows,
Luna's per-C-file maps, GLM's lifecycle rows and governing documents are unchanged.
Shared parser, visibility, follower, spell-identify, remort, corpse and save
boundaries remain explicitly delegated to their existing owners and proof symbols.
An ordinary successful shop scenario is not substituted for their full matrices.

The core map follows `src/shop.c:74-1445`: customer/opening gates, expression
stack, trade/identity/production, count/description/selectors, prices, purchase,
sort/slide, sale, value, list, room/dispatch/protection, boot and wizard-report
boundaries. Special definitions follow their source order in `src/spec_procs.c`,
`src/spec_procs2.c`, `src/spec_procs3.c`, and `src/new_cmds2.c`. Seventeen native
assignment links are enumerated from `src/spec_assign.c:183-618`. Unassigned
`eviltrade` is excluded rather than invented.

## Oracle evidence

Evidence root: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-26/`.
The 112 named scenarios and their five-seed claims are retained in
[depth/shops-surface.tsv](depth/shops-surface.tsv), with annotations in each
scenario. Effective result: **560/560 PASS**, seeds **1,2,3,5,8**, draw parity
matched, no expected-divergence exemptions, timeout or infrastructure rows.
Every individual scenario completed below the 240-second limit.

| Run directories | Selection | Result |
|---|---|---|
| `sol-shops-final-s{1,2,3,5,8}` | all 112 shop scenarios | All player bytes matched. Seeds 3/5/8 initially exposed setup-only DRAW rows in six older fixtures. Those rows are superseded below, not counted as passing proof. |
| `sol-shops-isolated-s{1,2,3,5,8}` | six older fixtures after isolating authored keeper/stock resets with quiet-mobs | 30/30 PASS, zero draw divergences |
| `sol-shops-dispatch-negative-control` | suppress shop handlers and native specials in a frozen source snapshot | 112/112 FAIL, zero infrastructure rows |
| `sol-shops-isolated-negative-control` | same suppression, six corrected fixture versions | 6/6 FAIL, zero infrastructure rows |

The six isolated names are `shop-buy-count-capacity`,
`shop-buy-weight-sell-steal`, `shop-live-inventory-control`,
`shop-live-inventory-filters`, `shop-live-inventory-gate`, and
`shop-stack-list-live`. Their old full-world setup consumed different unrelated
C draws at some seeds before the warmup. The corrected disposable resets retain
keeper association and original stock; the game code was not changed to conceal
those setup differences. Old red/DRAW attempts remain retained.

Machine-readable effective results are in
`sol-shops-work/effective-five-seed-results.tsv`; `source-equivalence.txt` verifies
all production `pkg/**/*.go` files match every positive immutable snapshot.
The original final runs captured HEAD `d9c5ff2ce` plus the pending parenthesis
fix; immutable source, not the abbreviated dirty HEAD alone, identifies the
executed implementation. The later parenthesis commit contains that same code.
Only proof metadata/comments and the six fixture corrections changed afterwards.
Each run's generated `MANIFEST.md` retains source hashes, engine logs, normalized
C/Go blocks and draw logs. Reference oracle SHA-256:
`bd6c548c6fbc41d35eede384e4ce2c5a858ff2ce0f568a379d7fb366a9051b07`.

The new transaction vehicles inspect gold and inventory after their probes.
Keeper banking/budget scenarios additionally inspect shop accounts; NPC sales
inspect the live mobile's coins/carried objects; recharging inspects object charge
values. The con-seller uses a lit live-PC stat observer because a stunned buyer
cannot execute `gold`. Empty-player setup gives C and Go matching persistent IDs.
Older list-only scenarios remain supplemental list proofs, not blanket state
proofs. LFCR and pager boundaries are additionally pinned by byte-level registered
session tests because normalized differential output alone would hide them.

## Fixes and revert controls

Each row below names the logical fix commit and its retained negative test log
under `sol-shops-work/`. Controls restored the previous implementation, observed
assertion failures, then restored the fixed source; no stash was used.

| Commit | C behavior restored | Revert evidence |
|---|---|---|
| `0c8c25931` | leading counts, trailing-word semantics, affordable prefix and per-object capacity | `buy-revert.txt` |
| `1f8f8ccdb` | counted sales, partial inventory/cash, no invented sell-all behavior | `sell-revert.txt` |
| `b0a6686be` | registered value command quotes without mutation | `value-revert.txt` |
| `f67c09abe` | keeper room/awake gates and real say audiences/punctuation | `gates-revert.txt` |
| `932b014b1` | adjacent real room, pet short description, optional-name keywords and detailed sign description | `pet-revert.txt` |
| `dc45645f1` | C keyword/flag expression evaluator and dead-wand/staff refusals | `trade-revert.txt` |
| `c520e9d58` | hidden customer bank transactions use hide-invisible room acts | `bank-revert.txt` |
| `76070f75e` | float32 markup before double charisma and integer truncation | `float-revert.txt` |
| `06e9eb178` | partial-use labels, LFCR rows and literal C pager splits | `list-revert.txt` |
| `e892717ac` | NPCs pay even above god level, prefix sale names, digit-only count/hash/ordinal selectors | `npc-payment-revert.txt`, `npc-prefix-revert.txt`, `numeric-selector-revert.txt` |
| `7fc92aa4f` | recharger affordability before full-charge refusal; resting HELP remains native | `recharger-revert.txt` |
| `d9c5ff2ce` | stat reports C's installed shop/native callback rather than hardcoded None | `stat-revert.txt` |
| `e2cd8d693` | pop underflow returns zero while top underflow returns NOTHING; stray close is accepted | `parenthesis-revert.txt` |

The parenthesis error arm is unreachable under defined stack bounds: the
higher-precedence loop leaves either OPEN (zero) or an empty stack, whose pop
also returns zero (`src/shop.c:161-170,237-244`). Signed count-refusal and
"screwy" fallback exclusions likewise cite their actual C predecessor gates.
A shipped stableboy cannot lack fixed prototype8021; the map cites the world
index/prototype and C `read_mobile` lookup, rather than assuming a quiet reset
removes a prototype.

## Validation

All ten gates ran separately and passed. Logs are retained as
`sol-shops-work/final-{build,vet,test,game-test,lint,deadcode,depth,units,strings}.txt`.
`make fmt` also passed; `make check-fmt` and `git diff --check` are final checks.

1. `make fmt`
2. `go build ./...`
3. `go vet ./...`
4. `go test ./...`
5. `go test ./pkg/game/...`
6. `golangci-lint run ./...`
7. `make deadcode-ratchet` — 52 baselined, zero new
8. `make fidelity-depth` — branch validator self-controls and schema checks pass
9. `make fidelity-units` — 1951/1951 claims pass
10. `make string-census`

A transient unrelated `TestWhodListenerOffClosesAndOnReopens` failure during an
earlier full-suite run was retained and the suite rerun successfully. No whod
source change was made. No full/combined census was run: Claude's batch gate and
review remain the release boundary. This branch is neither merged nor deployed.

## Decisions for Zach

- **R1a semantics:** four arithmetic cases (float-to-int, purchase/sale signed
  totals, bank signed accounts) and buying a PC from the pet roster remain
  blocked. C does not guard these operations; copying an overflowing integer or
  invalid mob-index read from one build would not establish historical semantics.
  Define the intended historical result and patch/pin the reference seam before
  claiming parity. The ordinary bounded arithmetic and PC roster listing are
  proved independently.
- **Residual routing:** Sol owns stock-identity variants, unprototyped stock,
  heterogeneous partial stock, the remaining broke-temper variants, a secondary
  keeper callback fixture and insufficient collection funds. Zach routes the two
  combat acceptance branches; GLM retains persisted missing-mount-prototype
  handling. The map gives each of the eighteen residuals an owner and concrete
  missing condition; no governing rule change is proposed.
- **Fixture hardening:** `give-object` checks that a requested M reset exists but
  appends G to the last M reset in that zone. An initial NPC vehicle therefore
  gave stock to the customer instead of the keeper. The corrected scenario puts
  the customer in a separate zone and proves actual transactions. Recommend a
  harness assertion that the requested M is the last M, plus a failing fixture
  control, as a separate infrastructure change.
- **Draw verdict hardening:** the census currently prints CLEAN and exits zero
  with DRAW rows. This audit explicitly required PASS for every effective pair,
  and replaced the affected fixtures. Recommend making `--draw-parity` fail the
  overall verdict on DRAW in a separate census-tool change; a green label alone
  is insufficient evidence of R3 compliance.
