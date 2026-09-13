# Recovered Phase 6.3 source excerpts

Date recovered: 2026-09-13

The original modernization audit was recovered from the read-only directory
`/home/zach/dp-modernization`. Its report files are not part of the tracked
application tree, so the excerpts below preserve the source facts used by the
Phase 6.3 reconciliation. The excerpts are copied verbatim, including the
original labels. Interpretations and corrections are in the dated handoff,
not silently folded into this evidence.

## Source identity and hashes

| source | SHA-256 |
|---|---|
| `/home/zach/dp-modernization/reports/06-roadmap.md` | `5a167fcdf2e7a3183cc7e84c14e75742e7d7d09bc52780be3da35827a50497c6` |
| `/home/zach/dp-modernization/reports/02-c-isms.md` | `d2ea1a557e6870a1cebfe13bca6a72c2b10a708b6057008a9b7754c61ecccf2c` |
| `/home/zach/dp-modernization/reports/raw/cisms-structural.md` | `49feac1bb347fc8b91612b7d668a2ff51445346f84474ce5451d74dcc452acc5` |
| `/home/zach/dp-modernization/reports/05-risk-tiers.md` | `9c08c536a54abe985cae7f16f14b1e4cf0537d50b5e789f6208a5913b2808a80` |

The source audit clone containing the C/Go comparison was inspected at
`/home/zach/dp-modernization/darkpawns`, branch `glm/depth-whimper`, commit
`0d6a9d1ff8a90793517264483f8f94c577504a56`.

## `reports/06-roadmap.md`

Source lines 69–75:

> ## Phase 6 — C-isms mechanical wave (~−2,300; subset of reports/02)
>
> | # | Target | Δ lines | Risk | Verifying cases |
> |---|---|---|---|---|
> | 6.1 | Giant switches → data tables where mechanical: `wiz_set` toggle majority (51 cases), `findExp` class/level ladders, equipment slot↔name maps, small lookup switches | −1,200–1,800 (mechanical subset only) | GREEN for the listed ones; spell-dispatch consolidation is **NOT here** | affected proven units |
> | 6.2 | String cleanup in proven files: 94 nested Sprintf → flatten; 57 loop-concats → Builder | −200–400 | GREEN | per-file unit scenarios |
> | 6.3 | Keyed tables + constants: key the unkeyed data tables (spellDB 509 L, THAC0, saving-throws 1,943 L), name the literal ladders, single-source LVL_IMMORT | ~0 net (churn) | GREEN | unit tests; highest bug-class value per line in census |

This is the original roadmap’s only Phase 6.3 row. It explicitly names the
three keyed data tables and `LVL_IMMORT`; “name the literal ladders” is a
scope phrase, not a finite inventory.

## `reports/02-c-isms.md`

Source lines 22–29:

> ## 2. Magic numbers — the real residue
>
> 7,795 bare ≥2-digit integer literals (5,021 excluding data-table rows). No constants package exists. Worst aspects:
> - LVL_IMMORT=31 duplicated at combat/fight_core.go:53 because import cycle — one place constant back-ported into literal.
> - Unkeyed composite-literal data tables — spellDB (spells/cast_cmds.go:31-120, 509 lines), THAC0 (combat/formulas.go:65), saving throws (saving_throws.go, 1,943 lines of positional numbers). ... No LevelCap/MaxLevel ... level thresholds literal ladder (combat/formulas.go:311-637).
> - Live literal dice...
> Naming these loses ~0–100...

The report’s explicit `LVL_IMMORT` duplication and the three unkeyed table
families map directly to the four merged slices. Its `formulas.go` range is a
lead to inspect, not authority that every value in that range has one meaning.

## `reports/raw/cisms-structural.md`

Source lines 80–94:

> **There is no constants package.** Constants are per-file `const (` blocks: 15+ files have ≥2 blocks (`pkg/game/item_helpers.go`:9, `pkg/game/death.go`:6, `pkg/combat/fight_core.go`:6, `pkg/spells/spell_info.go`:5, `pkg/combat/formulas.go`:5, `pkg/game/limits.go`:5, `pkg/game/clans.go`:5, …). Consequence worth flagging: **`LVL_IMMORT = 31` is duplicated** in `pkg/combat/fight_core.go:53` with the comment *"duplicated here to avoid import cycle with pkg/game"* — the package structure forces constant duplication.
>
> ### Quantification (sample-based, as briefed)
>
> Bare integer literals (≥2 digits, comment-stripped lines) in production code:
>
> - **7,795 occurrences in 251 files**; **5,021 after excluding pure data-table rows** (`{...},` literals — thac0/saving-throw/spell tables).
> - Top offenders (raw / non-table):
>
> | File | raw | non-table | Character |
> |---|---|---|---|
> | `pkg/spells/saving_throws.go` | 1,943 | — | saving-throw data tables (legit data, but 220-line unnamed literal block) |
> | `pkg/session/cast_cmds.go` | 509 | 509 | `spellDB` (:30-120): **unkeyed** `{1, "armor", 30, 15, 3, [12]int{}}` literals — positional struct literals, fragile |
> | `pkg/game/class_spells.go` | 409 | — | spell-learn tables |
> | `pkg/combat/formulas.go` | 396 | 348 | thac0 table (:65-110) **plus live thresholds** (:311-637) |

Source lines 110–118:

> **Level caps / thresholds as bare literals:**
> - `pkg/combat/formulas.go:311` `if level < 1` / :314 `if level > 40`; :582 `level >= 31`; :584/587/590/593 `level <= 30/27/20/10`; :610/616/622/627/632/637 — a whole ladder of weapon-skill learn thresholds with literals `60+level`, `30+level`, `75`, `39`
> - `pkg/game/level.go:30-31` `{0, 60}`, `{0, 65}` — con/int caps inline in tables
> - No `LevelCap`/`MaxLevel` constant exists anywhere (verified: zero matches)
>
> **Percentile RNG checks:**
> - `pkg/combat/formulas.go:610-711` `GetRoller().Number(1, 100) < (60+level)` (×3 variants), `< 75`, `Number(0, 100) >= defender.GetLevel()`
> - `pkg/game/act_movement.go:772` `doorNumber(1, 101)`; `pkg/game/skill_stealth.go:39` `dprng.Number(1, 101)`
> - `pkg/spells/affect_spells.go:941` `Number(0, 101)` in a seam var

The line-111 label is retained here because it is part of the recovered
source. Source verification shows that the cited formulas are attack-count
thresholds and percentile probes, not skill-learning thresholds.

## `reports/05-risk-tiers.md`

Source lines 3–7:

> Tiering combines the coverage overlay (reports/04) with the spine analysis (RNG draw order, combat math, creation draws, save format). Tier definitions:
>
> - **GREEN** — full oracle cover on every observable path the refactor can reach; mechanical refactor, safe to bite anytime. Proof = re-run the scenarios named in `coverage-mapping.tsv`.
> - **YELLOW** — partial cover: ≥1 blocked case, breadth-only, or indirect-only proof. Write new oracle cases *before* refactoring; refactor only branches the cases reach.
> - **RED** — load-bearing for RNG-stream position, combat math, character creation, or save format. Desync is invisible until deep play. Last or never; each item lists what would even permit touching it.

Source line 14:

> | `pkg/combat/engine.go`, `formulas.go`, `fight_core.go` | ~2,950 | Per-round draw order fixed by DP-1215/R3a (engine.go:495-509); THAC0 hit draw (formulas.go:379); damage pipeline (:527); `findFightingTarget` iterates a **Go map** (engine.go:473) — a latent order hazard vs C's list walk even today. Golden tests are the executable spec. |

This is why the two recovered `GetAttacksPerRound` candidates are retained as
separate, bounded future work rather than reclassified as a completed
constant-naming slice.
