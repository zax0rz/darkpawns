# DP-1511 — final triage batch

This batch dispositions the last 19 pending scenarios: **18 repaired and one
Go disagreement with its oracle claim blocked**. No shipping scenario retains
a `pending DP-1511 triage` allowance. The historical 19-name handoff is retained
in `../batch3-pending-triage.json`; the current pending ledger is empty.

The final 18-scenario target is CLEAN with zero draw divergence. All 51 original
scenario/seed pairs were run before repair with the temporary allowance removed:
all failed setup. The repaired original pairs pass; the genuinely overflowing
clan editor instead fails reproducibly and is retained outside the green corpus.
No Go game code, matcher, normalization exception, governing document or
expected-divergence pin changes in this batch.

## Scenario dispositions

`triage.tsv` and `witnesses.json` retain actual C event lines, command/audience
blocks, paths and line numbers. `before-refusals.tsv` verifies all 51 original red pairs.
`seed-outcomes.tsv` identifies every repaired
original pair and its retained C dump. These are event proofs, not conclusions
inferred from a CLEAN comparison alone (R5h/R5j).

| Scenario | Verdict | Setup repair and actual C witness |
|---|---|---|
| aid-failure-depth | Repaired | Target level 39 is below the level-40 setter. Aid=1 guarantees failure: `You fumble and ruin the bandages.` plus both room recipients. |
| carve-depth | Repaired | Clear the fixture room's peaceful flag; the real dead fish yields the no-tool branch, `You carve up some meat from the the corpse of fish.`, observer act and carried meat. Remove the unrelated fighting-no-argument tag; that case stays unit-proven. |
| clan-applicant-depth | Repaired | Remove redundant leader enrollment after create. `You've applied to the clan!` and `You applied to Wolves`. |
| clan-depth | Repaired | Same redundant enrollment removed; real rename, treasury, fees, privacy and destroy outputs. |
| clan-member-depth | Repaired | Same removal; actual leader status, applicant enrollment and named membership/rank refusals. |
| clan-plan-depth | Repaired | Same removal; first two-line plan, replacement and `@ trailing` clearing are reflected in clan info. |
| clan-plan-mortal-depth | Repaired | Same removal; member edits and clan info contains `Mortal member plan.` |
| clan-plan-overflow | Go disagrees with C | Create a real mortal leader and send a line that actually crosses 1024 bytes. C skips it and remains in the editor; Go exits early. Claim blocked, red source moved to `diagnostic-scenarios/`. |
| fighter-wimpy-window-depth | Repaired | Use registered `wimpy 120`, below maxhit/3 and above hit=100, rather than an unsupported set field. `You wimp out, and attempt to flee!` at all nine seeds. |
| idlist-depth | Repaired | Use a separate level-37 peer for `Huh?!?`; keep the implementor at 40 for both `Ok. Id list complete.` outputs. Repeated section headers do not interleave setup/probes. |
| lifecycle-idle-rent-norent | Repaired | Load the non-TAKE cloak directly into inventory, then actually wear it. Idle/close/relogin removes NORENT cloak and fruit while retaining `Idleforcerent's tunic`. |
| lua-cast-spell-gates-depth | Repaired | Remove invalid level 50; existing level 40 suffices. All six cast/will target/group gates and five scripted position refusals appear in C. |
| reagent-waterwalk-decoy-vnum-depth | Repaired | Remove self-give of the already carried decoy. Wait out the cast before the second inventory: the keyword-edited wand remains and no frog-leg dust narration occurs. |
| retaliation-flamestrike-save-depth | Repaired | Use C's canonical `flame strike` in both skillset and cast. Applied at seeds 1,5; `Nothing seems to happen.` on saves at 2,3,8; actual incubus retaliation appears in both arms. |
| spell-parser-item-output-gaps | Repaired | Three spawn directives create three scrolls; a reset maximum is not a copy count. Actual equipped-target recite, scroll/potion room action descriptions, staff/wand outputs and position refusals occur. |
| spell-points-spoken-depth | Repaired | Remove level 100, use `align`, cast ordinary points spells as cleric, switch to Psionic and **will** cell adjustment, then Paladin for lay hands. Each actual healing/vitality/invigoration message is retained. |
| violence-npc-switched-output-depth | Repaired | Use legal numeric bounds, explicitly save the Lua NPC position/timer, switch bodies and advance 400 pulses. C's `You dodge!` and `You drag yourself to your feet.` plus peer acts really occur. |
| wizard-file-output-depth | Repaired | Save peer at 39, reallyquit the level-1 peer, mutate its password at 40, then lower actor to 38. Actual password/saved acknowledgements and both higher-file refusals appear. |
| wizard-stat-clan-followers-depth | Repaired | Remove redundant leader enrollment. Real Wolves membership and both PC/mobile six-follower wrapped reports are visible. |

## C call paths and class audit

The repairs follow C rather than scenario names or old prose:

- `set level` rejects values at/above its caller (`src/act.wizard.c:2898-2904`).
  Bootstrap actor 40 does not need an impossible promotion to 50/100. The aid
  failure compares a 1..(101+victim level) draw to skill 1 and sets the actual
  wait (`src/new_cmds2.c:140-189`).
- Clan create assigns and saves the leader's clan/rank itself
  (`src/clan.c:230-234`). The six immediate re-enrollments are unnecessary
  refused operations, not proof that create failed. Editor entry is
  `src/clan.c:1392-1438`.
- `wimpy` is its own command with a maxhit/3 cap
  (`src/act.other.c:978-1019`); automatic flee is `src/fight.c:1601-1610`.
  Fighter dispatch remains `src/spec_procs.c:509-535`.
- C idlist requires LVL_GRGOD=38 (`src/interpreter.c:512`,
  `src/structs.h:612`). The old level-38 probe was allowed, and a second
  warmup's restore to 40 was refused. The harness appends all warmups before
  all probes; section repetition never represented the alleged level sequence.
- Cloak 4291 is NORENT but lacks TAKE (`lib/world/obj/42.obj:163-172`). C load
  puts it in inventory (`src/act.wizard.c:1368`). The retained warmup proves
  it was worn before force-rent, rather than inferring removal from absence.
- Spell position/target/group gates are `src/spell_parser.c:825-900`.
  `will` reaches do_cast (`src/interpreter.c:825`); class tokens are parsed at
  `src/class.c:122-169`. Points messages/formulas are `src/magic.c:1764-1828`.
- Material matching compares vnums (`src/magic.c:546-552`); waterwalk's
  positive reagent narration is conditional (`src/magic.c:1274-1288`). The
  OEDIT warmup proves `frog leg decoy` keywords, and both inventory probes
  show the same unrelated wand. This proves rejection as a reagent, not
  failure of the waterwalk spell, which still applies.
- C's canonical name is `flame strike` (`src/spell_parser.c:145`). Both
  made-save retaliation and applied aggro follow `src/magic.c:1109-1129,1421`.
- `spawn-obj`'s second number is a maximum existing count, not a requested
  count (`internal/oraclediff/scenario.go:309`). Stock scroll 9906 is not
  invisible (`lib/world/obj/99.obj:76-84`). The equipped target is legal via
  generic_find (`src/spell_parser.c:565-568`), so that first recite consumes
  a scroll; the target-missing recite does not. Parser output dispatch is
  `src/spell_parser.c:581-709`.
- NPC position edits require explicit `save_char(me)`
  (`src/scripts.c:1292-1302`). Ordinary return reads the ch table at Lua stack
  index 1 again for me (`src/scripts.c:1805-1813,1980-2018`); merely assigning
  me.pos produced no stand-up event. Existing `pf105-mob-timer-fight-depth`
  documents the same seam. The final warmup stat proves Sitting, then the
  switched NPC receives the actual combat outputs
  (`src/fight.c:1965-1972,1982-1986`). The two new manifest rows cover the
  live direct NPC and peer messages; existing unit rows retain their scope.
- Reallyquit bypasses the ordinary mortal quit-room restriction
  (`src/act.other.c:88-180`). Saved-file protection/password/save paths are
  `src/act.wizard.c:1027-1044,2648-2671,2973-2990,3063-3071`.

`audit-setup-classes.py` reproduces whole-corpus syntactic inventories from the
pinned batch base and the working tree: repeated warmup sections **1→0**,
warmup level≥40 inputs **6→0**, create/same-leader-enroll pairs **6→0**, and
me.pos/timer writes without explicit save **1→0**. The batch-2 set-field audit
now returns `[]` (`invalid-set-fields-after.json`; run its `--working-tree`
mode). These are precisely scoped token audits, not a claim that every script
or integer assignment in the corpus is correct. All found candidates belong
to this batch; no extra sibling scenario is silently left pending.

## Go disagreement: clan editor exits early

`sol-dp1511-b4-witness-repair-seed1` retains the red comparison in both captures.
The old scenario never created its clan (the implementor cannot be leader), and
its alleged overflow text totaled only 1012 bytes, below the 1024-byte cap.
After both repairs, the first divergent normalized actor block is:

- C: `String too long.  Last line skipped.\n\n]\n`
- Go: `String too long.  Last line skipped.\n`

The following `@` is consumed by C's still-active editor, but Go emits
`Huh?!?`. `clan info` shows the same retained four-line plan. The reference C
uses the improved-editor branch: overflow sets STRINGADD_ACTION and continues
(`src/modify.c:145-151`). Go's reachable session handler calls
`HandleClanPlanInput` (`pkg/session/session_login.go:442-448`), which hardcodes
save-and-exit for the non-improved branch (`pkg/game/clans.go:315-359`). The
first-line overflow branch has the same source assumption; that is a class-audit
candidate, not an additional live proof.

Both attempts also show a draw mismatch at stream 4087: C ends, while Go draws
`Number(0,3) = 2` for the subsequent clan-info step. The summary's zero draw
**tally excludes this FAIL** and must not be read as equality for it.
`clan-overflow-divergence.json` retains the first capture's actual blocks and
this detail. The diagnostic source is preserved outside the ordinary corpus;
`clan.plan-overflow-later-line-skipped` is blocked. No exception or new pin
conceals the difference. Claude files the game issue.

The earlier batch-3 door destination newline gap remains blocked and reported
in `../batch3/README.md`; this batch does not repair or relabel it.

## Retained runs and verification

All directories are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/`; `runs.tsv` retains every
batch-4 summary. Wrapper manifests freeze the exact source inputs and engine
hashes; both engines' normalized blocks, warmups, logs and enabled draws remain
in every attempt. No red or inadequate green has been overwritten.

| Run family | Result and limitations |
|---|---|
| `sol-dp1511-b4-before-seed{1,2,3,5,8,11,13,17,21}` | All 51 original pairs fail setup with allowances removed. |
| `sol-dp1511-b4-after-seed{1,2,3,5,8}` | Basic repairs; retal skill spelling and low-peer quit still fail. Early overflow stays below cap; early points/violence/reagent greens lack claimed events. Retained as inadequate evidence. |
| `sol-dp1511-b4-after-seed{11,13,17,21}` | Additional fighter claims CLEAN with real wimpy output and draw parity. |
| `sol-dp1511-b4-witness-repair-seed1` | Three repaired pairs PASS; corrected actual clan overflow FAIL as above. |
| `sol-dp1511-b4-witness-repair-seed{2,3,5,8}` | Retal, wizard file and points each CLEAN with draw parity and real events. |
| `sol-dp1511-b4-round-and-reagent-witness` | Both comparisons CLEAN. Delayed reagent inventory is now valid; NPC dodge occurs, but stand-up is still hollow. |
| `sol-dp1511-b4-direct-peer-standup`, `sol-dp1511-b4-npc-wait-standup`, `sol-dp1511-b4-admin-standup` | CLEAN comparisons still lack stand-up. Admin stat explicitly shows Fighting despite unsaved me.pos. Not accepted as stand-up evidence. |
| `sol-dp1511-b4-saved-npc-standup` | Explicit save establishes Sitting and actual stand-up/dodge messages; CLEAN with draw parity. |
| `sol-dp1511-b4-final` | All 18 shipping scenarios CLEAN, zero draw divergences. Subsequent evidence/manifest/comment edits do not alter engine inputs. |

The basic seed-1 clan-depth first attempt ended with C EOF (INFRA); the wrapper's
isolated recheck passed. This is retained, not erased by a manual whole-run
retry. The final target is independently clean. No concurrency acceptance claim
is made from a rechecked verdict.

All eight required gates pass separately: build, vet, all Go tests, lint,
formatting, deadcode ratchet, fidelity depth and expected-divergence checks.
The expected-divergence inventory retains its existing unresolved-proof warnings;
no new pin was added. These changes do not modify exported game symbols.

## Gate inventory, R5h and full census

The gate is unchanged from the reviewed batches: **312 added exact strings,
81 bounded anchored patterns, 393 real C witnesses**, plus a negative case for
every pattern. Existing exact entries are unchanged. The original verb inventory
is `../warmup-verbs.tsv`: **10,696 direct inputs, 223 raw direct tokens plus five
force-only names, 76 registered command-name candidates**. Counts remain pinned
to the original base; later scenario repair is not a rewritten inventory.

R5h remains the retained **red**
`sol-dp1511-r5h-old-water-retained` before #2389 (C refused `get chalice`) and
**CLEAN** `sol-dp1511-r5h-current-water` (real `A golden chalice is filled.`).
The initial full census `sol-dp1511-gate` found the 65 setup failures. Historical
full `sol-dp1511-gate-batch1-r2` is CLEAN: 1971 scenarios, 1965 PASS, four EXPECTED,
two known unstable. It predates these repairs and is not an exact-final release
gate. Per current AGENTS instructions, coding PRs run necessary targets; Claude
runs the batched exact-commit `--combined --release` gate.

## Decisions for Zach

Have Claude review this final triage batch and file the reproducible clan-editor
disagreement (R1/R3), then merge if accepted. No pending allowance remains, so
this final PR may carry bare `(DP-1511)` in its title. The batch-3 door finding
also remains for Claude to file if not already tracked. Neither game defect is
fixed in this harness-only brief. No governing-rule amendment is required for
this batch; no Go 1.27 work has started. That next brief waits for DP-1511 to close.
