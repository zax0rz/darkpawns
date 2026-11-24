# Blind Cross-Audit of 120 Go Tests — Results (`2026-11-24`)

- **Cross-auditor:** Gemini 4 Carbon
- **Sample:** `docs/briefs/CROSS-AUDIT-2026-11-24-sample.tsv` (`120` post-oracle tests: `60` in `pkg/session`, `60` in `pkg/game`; verified SHA-256 `aade2e6c07143df528954ea776402ff30b9cf47d18474f28181bd5b44a42b9cb`)
- **Audited codebase commit (`origin/main`):** `ef2b14138`
- **Pre-unblinding lock commit:** `8f52ebdbe1040dbbdad141b20b72571eb0574391` (`predictions.md`, `blind.tsv`, `mutations.tsv` committed before opening `docs/research/test-audit/{session,game}/ledger.tsv`)

---

## 1. Executive Answer: Tests vs. Auditors

1. **Post-oracle test quality in `pkg/session` and `pkg/game` is similar; the ~20x ledger gap was primarily an auditor-strictness and workflow effect.**
   - On `origin/main` (`ef2b14138`), Carbon's blind mutation-backed audit found **4 / 60 (`6.7%`)** non-`R` in `pkg/session` (**3 / 60 = `5.0%`** after accounting for one intentional `mudlog.tsv`-cited static C proof) versus **5 / 60 (`8.3%`)** residual non-`R` among `Orig=R` rows in `pkg/game`.
   - Earlier ledgers reported **`0.52%` (`8 / 1,537`)** non-`R` overall and **`0.14%` (`2 / 1,392`)** post-oracle non-`R` in `pkg/session` (`PF-165`, audited by Kimi `T001–T1045` and Neon `T1046–T1537`), compared with **`9.84%` (`203 / 2,063`)** in `pkg/game` (`PF-168`). In our 60-test post-oracle `pkg/session` sample (`0 / 60` non-`R` in `session/ledger.tsv`), Kimi missed two incomplete registration tests (`TestGiggleRegistrationUsesCEntryGate`, `TestPointRegistrationUsesCEntryGate`) whose names and ledger contracts claimed registration gating while checking only the static `commandGates` map, and Neon missed an immediate same-file duplicate (`TestAcTextNakedIsExact100` right below `TestAcTextEveryBand`). This directly settles the open question in `PF-165`: the `0.14%` post-oracle weak rate in `pkg/session` understated residual weak/duplicate tests by roughly an order of magnitude (~`5.0%` substantive).
2. **Fix-as-you-go creates a commit-state confound on `Orig=F` rows, and independently validates 5 / 5 `pkg/game` repairs.**
   - Every test in the 120-test sample was drawn from surviving post-oracle functions on `origin/main` (`born_before_oracle = no` for all 120 rows; `C`/`D` deletions had already been removed, and `Orig=F` rows had already been edited in place).
   - All **5 `Orig=F` rows** in `pkg/game` (`T701`, `T733`, `T1433`, `T1941`, `T1984`) had been repaired in place on `main` by `codex/luna-ta-game-{9,23,34}` (`76f1be032`, `b53a99434`, `12a723ca5`) on 2026-11-21..22 before `ef2b14138`. Carbon blindly marked **all 5 / 5 (`100%`) as `R`** on `ef2b14138`, confirming that Luna's assertion repairs genuinely fixed the underlying test weaknesses.
3. **Why `pkg/game` still had 5 / 55 (`9.1%`) `Orig=R` misses:**
   - Four of the five `Orig=R, Carbon=non-R` disagreements in `pkg/game` stem from **call-graph reachability (`R5e`) and cross-test redundancy** rather than local assertion reading: two tests (`TestEncodeSave_*`, `T653`/`T654`) exercise dead legacy `encodeSave` (`0` production callers), one (`TestHouseSaveLoadFloorsDepthTwo`, `T1697`) exercises dead `houseCrashsave` (`0` production callers; C's `House_save_all` is commented out at `src/house.c:709-721`) plus a flat `houseLoad` loop already covered by `TestHouseSaveLoadFloorsContainerContents`, and one (`TestRestartDoesNotRestoreLegacyWorldSnapshot`, `T1284`) runs a test-local copy `bootRestartFixture` while gutting `IgnoreLegacyWorldState()` stays green. The fifth (`TestCanTakeObjUsesTakeBit`, `T370`, from GLM's post-oracle redo batch 1) tests only `(1<<0)|(1<<13)` with no negative case, staying green when `canTakeObj` checks the `WIELD` bit (`1<<13`) instead of `TAKE` (`1<<0`).

---

## 2. Predictions vs. Actual Blind Marks

Pre-audit predictions (`predictions.md`, written before opening `CROSS-AUDIT-2026-11-24-sample.tsv`):

| Package | Sample $N$ | Predicted non-`R` | Blind non-`R` (`F`/`C`/`D`) | Original ledger non-`R` | Second-look non-`R` on `ef2b14138` |
|---|---:|---:|---:|---:|---:|
| `pkg/session` | 60 | 14 (`23.3%`) | **4 (`6.7%`)** — `2 F, 1 C, 1 D` | 0 (`0.0%`) | **3 (`5.0%`)** — `2 F, 1 C` |
| `pkg/game` | 60 | 10 (`16.7%`) | **5 (`8.3%`)** — `2 F, 1 C, 2 D` | 5 (`8.3%`) — `5 F` (pre-fix) | **5 (`8.3%`)** — `2 F, 1 C, 2 D` |
| **Total** | **120** | **24 (`20.0%`)** | **9 (`7.5%`)** — `4 F, 2 C, 3 D` | **5 (`4.2%`)** — `5 F` | **8 (`6.7%`)** — `4 F, 2 C, 2 D` |

*(Carbon's prior of 20% overestimated weak-test prevalence by ~2.7x because all 120 sampled tests turned out to be post-oracle tests on an already-cleaned `main` where earlier `C`/`D` deletions and `F` repairs had already landed.)*

---

## 3. Inter-Rater Agreement & Cohen's Kappa

Note on duplicate ledger rows: none of the 120 sampled tests had multiple exact-name or `TestName/subtest` rows in `session/ledger.tsv` or `game/ledger.tsv`.

### 3A. Primary Comparison: Strict Blind Marks (`ef2b14138`) vs. Recorded `ledger.tsv` Marks

| Scope | $N$ | Carbon (`R`/`F`/`C`/`D`) | Orig Ledger (`R`/`F`/`C`/`D`) | 4-Class Raw Agreement | 4-Class $\kappa_4$ | Binary (`R` vs non-`R`) Raw Agreement | Binary $\kappa_2$ |
|---|---:|---|---|---:|---:|---:|---:|
| `pkg/session` | 60 | `56 / 2 / 1 / 1` | `60 / 0 / 0 / 0` | `56 / 60` (`93.3%`) | `0.0000` | `56 / 60` (`93.3%`) | `0.0000` |
| `pkg/game` | 60 | `55 / 2 / 1 / 2` | `55 / 5 / 0 / 0` | `50 / 60` (`83.3%`) | `-0.0619` | `50 / 60` (`83.3%`) | `-0.0909` |
| **Combined** | **120** | **`111 / 4 / 2 / 3`** | **`115 / 5 / 0 / 0`** | **`106 / 120` (`88.3%`)** | **`-0.0402`** | **`106 / 120` (`88.3%`)** | **`-0.0566`** |

**Why raw $\kappa \le 0$ despite 83%–93% raw agreement:**
- In `pkg/session`, `Orig` has zero variance (`60/60` `R`), which forces $p_e = \frac{56}{60} \cdot \frac{60}{60} = p_o$ and $\kappa = 0.0000$ algebraically regardless of how high $p_o$ is.
- In `pkg/game`, `Orig=F` records **pre-fix** defects that were repaired in place *before* `ef2b14138`, so all 5 `Orig=F` tests legitimately became `R` on `ef2b14138`, while Carbon's 5 non-`R` findings could only come from `Orig=R` survivors—creating structural anti-correlation ($0$ shared non-`R` cells on `ef2b14138`).

### 3B. Secondary Comparison: Pre-Fix Commit-Aligned View

When the 5 `Orig=F` tests in `pkg/game` are evaluated at their pre-repair commits (`6e897d38a`, `c630fd542`, `47661d3d3`, `7f75a042c`, `52d2f631f` — where `git log -L` confirms all 5 failed mutation checks for the exact `J3`/`J7`/`J9` reasons recorded in `game/ledger.tsv`), both auditors mark those 5 tests `F`:

| Scope (Pre-Fix Aligned) | $N$ | 4-Class Raw Agreement | 4-Class $\kappa_4$ | Binary Raw Agreement | Binary $\kappa_2$ |
|---|---:|---:|---:|---:|---:|
| `pkg/session` | 60 | `56 / 60` (`93.3%`) | `0.0000` | `56 / 60` (`93.3%`) | `0.0000` |
| `pkg/game` | 60 | `55 / 60` (`91.7%`) | **`+0.6319`** | `55 / 60` (`91.7%`) | **`+0.6250`** |
| **Combined** | **120** | **`111 / 120` (`92.5%`)** | **`+0.5012`** | **`111 / 120` (`92.5%`)** | **`+0.4953`** |

### 3C. Breakdown by Original Sub-Auditor

| Package | Ledger Slice & Auditor | Sampled $N$ | Carbon `R` / non-`R` | Orig `R` / non-`R` | Substantive Misses by Orig |
|---|---|---:|---|---|---|
| `pkg/session` | `T001–T1045` (Kimi, Batches 1–12) | 39 | `37 R / 2 F` | `39 R / 0 non-R` | 2 `F` (`TestGiggle*`, `TestPoint*` omit registration check) |
| `pkg/session` | `T1046–T1537` (Neon, Batches 13–17) | 21 | `19 R / 1 C, 1 D` | `21 R / 0 non-R` | 1 `C` (`TestAcTextNakedIsExact100`); `1 D` is `mudlog.tsv` policy keep |
| `pkg/game` | `T339–T388` (GLM post-oracle redo B1) | 1 (`T370`) | `0 R / 1 F` | `1 R / 0 non-R` | 1 `F` (`TestCanTakeObjUsesTakeBit` lacks non-TAKE case) |
| `pkg/game` | `T389–T2063` (Codex/Luna B2–B36) | 59 | `55 R / 1 F, 1 C, 2 D` | `54 R / 5 F (fixed)` | 4 (`2 D` dead `encodeSave`, `1 C` dead `houseCrashsave`, `1 F` copied boot helper); 5/5 `F` repairs verified `R` |

---

## 4. Blinding & Accidental Exposure Disclosure

- `predictions.md` was written before opening `CROSS-AUDIT-2026-11-24-sample.tsv` or any Go test.
- `blind.tsv` and `mutations.tsv` were committed at `8f52ebdbe` before opening `session/ledger.tsv`, `game/ledger.tsv`, or any `notes.md`/`BRIEF.md` file under `docs/research/test-audit/`.
- **Single accidental exposure (disclosed per brief rule):** While verifying whether unexported `encodeSave` (`pkg/game/save.go:131`) had any callers, a scoped grep (`grep -rn "encodeSave" cmd/ pkg/ internal/ | grep -v "_test.go"`) first proved `encodeSave` had zero production callers. Immediately afterward, an unscoped `grep -rn "encodeSave" .` matched two lines in `docs/research/test-audit/game/ledger.tsv` (`T653` `TestEncodeSave_CloseErrorReported` and `T654` `TestEncodeSave_EncodeErrorTakesPrecedence`, both showing `Orig=R`). All subsequent greps were restricted with `--exclude="*_test.go"` / `cmd/ pkg/ internal/ src/`. Carbon marked both `TestEncodeSave_CloseErrorReported` and `TestEncodeSave_EncodeErrorTakesPrecedence` **`D`** based on the already-established zero-caller dead-code proof (opposite to the glimpsed `R`). No other test's prior mark was seen before commit `8f52ebdbe`.

---

## 5. Every Disagreement (14 of 120 Tests) & Second-Look Judgements

All 9 mutations where Carbon marked non-`R` and the original ledger marked `R` were **re-run after unblinding** (`GOEXPERIMENT=nojsonv2 go test -count=1 ...`) with byte-for-byte SHA-256 restoration verified afterward; every single re-run reproduced the blind outcome identically. `blind.tsv` is left unchanged; the **Second-Look** column below records Carbon's post-unblinding view after weighing both sides.

| # | Package & Test | Ledger ID / Auditor | Carbon Blind | Orig Mark | Mutation Re-Run (`Carbon=non-R, Orig=R`) | Second-Look | Judgement After Weighing Both Sides |
|---:|---|---|---|---|---|---|---|
| 1 | `session` :: `TestAcTextNakedIsExact100` (`cmd_info_test.go:117`) | `T1321` (Neon) | `C` | `R` | `cmd_info.go:170` `ac == 100` -> `ac >= 100`: survivor `TestAcTextEveryBand` **RED** (`exit=1`) | **`C`** | **Carbon `C` stands.** `TestAcTextEveryBand` (`cmd_info_test.go:73-111`, 4 lines above) already tests `101, 100, 99, 71, 70` on `acText()`; `110` hits the identical `ac > 70` branch as `101`. |
| 2 | `session` :: `TestCReditDefaultUnreachable` (`olc_control_reachability_test.go:153`) | `T1141` (Neon) | `D` | `R` | `redit.go:472` `state.mode = reditCopy` -> `reditMode(250)`: `TestCReditDefaultUnreachable` **GREEN** (`exit=0`) | **`R` (policy)** | **Orig `R` wins under repo manifest policy (`PF-165`).** It tests zero Go code (regexes read-only `src/redit.c`), so under pure Go-mutation rules it is `D`; however, `docs/fidelity/depth/mudlog.tsv:97` cites this symbol as the machine-checked static C-unreachability proof for `redit_parse` `default:`. Keep unless migrated to a static script. |
| 3 | `session` :: `TestGiggleRegistrationUsesCEntryGate` (`giggle_test.go:9`) | `T383` (Kimi) | `F` | `R` | `commands.go:1116` add `&& cmd != "giggle"`: `TestGiggleRegistrationUsesCEntryGate` **GREEN** (`exit=0`) | **`F`** | **Carbon `F` stands.** Kimi's contract claims `"the giggle command/social registration matches its C interpreter.c entry gate"`, but the test only reads `commandGates["giggle"]` (already SHA-256 hashed in `command_gates_test.go`) and never checks `game.Socials["giggle"]`, `isCSocial("giggle")`, or `ExecuteCommand`. |
| 4 | `session` :: `TestPointRegistrationUsesCEntryGate` (`point_depth_test.go:9`) | `T506` (Kimi) | `F` | `R` | `commands.go:579` `registerCommand("point", ...)` -> `cmdRegistry.Register(..., 99, PosStanding)`: **GREEN** (`exit=0`) | **`F`** | **Carbon `F` stands.** Same Kimi false negative: unlike `credits`/`kick`/`rin`/`sleeper`/`zedit` gate tests, `point_depth_test.go` omitted `cmdRegistry.Lookup("point")`, so registering `point` with wrong level/position leaves the test green. |
| 5 | `game` :: `TestCanTakeObjUsesTakeBit` (`item_transfer_test.go:26`) | `T370` (GLM) | `F` | `R` | `item_transfer.go:28` `wf&1 != 0` -> `wf&(1<<13) != 0`: `TestCanTakeObjUsesTakeBit` **GREEN** (`exit=0`) | **`F`** | **Carbon `F` stands.** GLM wrote `contract="can_take_obj requires ITEM_WEAR_TAKE"`, but the test only probes `(1<<0)\|(1<<13)` (`TAKE\|WIELD`) with no non-`TAKE` (`1<<13`) negative case; checking bit 13 or `wf != 0` or `hasTake := true` stays green. |
| 6 | `game` :: `TestEncodeSave_CloseErrorReported` (`save_test.go:403`) | `T653` (Luna) | `D` | `R` | `save.go:425` drop `json.Marshal` error in live `SerializePlayer`: **GREEN** (`exit=0`) | **`D`** | **Carbon `D` stands** *(accidental grep exposure disclosed above)*. `encodeSave` (`save.go:131`) has zero non-test callers anywhere in the repository; `docs/cleanup/2026-10-09-dead-code-decisions.md:60-66` confirms `encodeSave` is dead legacy JSON-file save code superseded by SQLite. |
| 7 | `game` :: `TestEncodeSave_EncodeErrorTakesPrecedence` (`save_test.go:413`) | `T654` (Luna) | `D` | `R` | `save.go:425` drop `json.Marshal` error in live `SerializePlayer`: **GREEN** (`exit=0`) | **`D`** | **Carbon `D` stands** *(accidental grep exposure disclosed above)*. Same dead-code target `encodeSave` (`0` production callers; `R5e`). |
| 8 | `game` :: `TestHouseSaveLoadFloorsDepthTwo` (`house_save_nesting_test.go:129`) | `T1697` (Luna) | `C` | `R` | `house_save.go:89` skip `item.ContainerIndex != nil` in `houseLoad`: survivor `TestHouseSaveLoadFloorsContainerContents` **RED** (`exit=1`) | **`C`** | **Carbon `C` stands.** `houseCrashsave`/`collectHouseItems` has `0` production callers (`House_save_all` is commented out in `src/house.c:709-721`), while live `houseLoad` (`house_save.go:40-92`) is a flat loop ignoring `ContainerIndex` already covered by `TestHouseSaveLoadFloorsContainerContents`. |
| 9 | `game` :: `TestRestartDoesNotRestoreLegacyWorldSnapshot` (`save_world_test.go:113`) | `T1284` (Luna) | `F` | `R` | `world_state.go:67` early `return` in `IgnoreLegacyWorldState()`: **GREEN** (`exit=0`) | **`F`** | **Carbon `F` stands (borderline `F`/`C`).** Calls test-local `bootRestartFixture` (`NewWorld` + `StartZoneResets` + void `IgnoreLegacyWorldState()`) rather than a shared `cmd/server` boot entrypoint and checks no effect of `IgnoreLegacyWorldState()`. Mutating `world_state.go` or `cmd/server/main.go:1020` stays green. |
| 10 | `game` :: `TestSpecAssassinRejectsPlayerRosterMember` (`spec_assassin_test.go:171`) | `T701` (Luna) | `R` | `F` | n/a (`Carbon=R, Orig=F`) | **`R` (fixed)** | **Both right at their commits.** Pre-fix (`6e897d38a`) lacked gold/mob-count and per-recipient checks (`F`); repaired in `76f1be032` (`codex/luna-ta-game-9`), verified `R` on `ef2b14138`. |
| 11 | `game` :: `TestSpecCleric_BlindnessGateConsumesCDraw` (`spec_cleric_test.go:192`) | `T733` (Luna) | `R` | `F` | n/a (`Carbon=R, Orig=F`) | **`R` (fixed)** | **Both right at their commits.** Pre-fix (`c630fd542`) used `level=10, blindRoll=1` so removing `lspell >= 4` stayed green (`F`); repaired in `76f1be032` (`level=2, blindRoll=0`, blind victim stays blind), verified `R` on `ef2b14138`. |
| 12 | `game` :: `TestSpecEnterCircleCountsPlayersAndConsumesOtherLook` (`unassigned_specials_depth_test.go:140`) | `T1433` (Luna) | `R` | `F` | n/a (`Carbon=R, Orig=F`) | **`R` (fixed)** | **Both right at their commits.** Pre-fix (`47661d3d3`) checked only `true` return on `look north` without `do_look` output (`F`); repaired in `b53a99434` (`codex/luna-ta-game-23`) to assert `"You look northwards."`, verified `R` on `ef2b14138`. |
| 13 | `game` :: `TestTakeToJailBreedCallerAndRoomOrder` (`spec_take_to_jail_test.go:364`) | `T1941` (Luna) | `R` | `F` | n/a (`Carbon=R, Orig=F`) | **`R` (fixed)** | **Both right at their commits.** Pre-fix (`7f75a042c`) used loose `strings.Contains` (`F`); repaired in `12a723ca5` (`codex/luna-ta-game-34`) to assert exact C transcripts, verified `R` on `ef2b14138`. |
| 14 | `game` :: `TestUseTattooSkullVisibility` (`use_tattoo_depth_test.go:280`) | `T1984` (Luna) | `R` | `F` | n/a (`Carbon=R, Orig=F`) | **`R` (fixed)** | **Both right at their commits.** Pre-fix (`52d2f631f`) used `strings.HasPrefix(..., "Your tattoo glows")` (`F`); repaired in `12a723ca5` (`codex/luna-ta-game-34`) to assert exact activation bytes, verified `R` on `ef2b14138`. |
