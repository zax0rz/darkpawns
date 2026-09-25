# Roadmap: 2026-09-25 to 2026-09-30, open-model work

Written by Claude on 2026-09-25 for work done by open models (DeepSeek, GLM,
Kimi and similar) while Claude is out of weekly budget until Tuesday. It turns
the open Linear backlog into ordered work, and says how to do that work the way
this project does it.

**Read first, in this order:** [`AGENTS.md`](../../AGENTS.md),
[`docs/fidelity/RULEBOOK.md`](../fidelity/RULEBOOK.md) (R1–R5g),
[`docs/fidelity/DEPTH_TESTING.md`](../fidelity/DEPTH_TESTING.md), and, before any
research writing, [`docs/research/README.md`](../research/README.md).

---

## 1. The working loop

Every fidelity change follows the same loop. A model that skips a step
produces work that looks finished and is not.

1. **Find the C.** Grep `src/` for the function, and read it. Write the
   function name and `src/<file>:<line>` from that read (R5g). Never cite from
   memory, and never cite `darkpawns-c-oracle/`: its `utils.c`, `comm.c`,
   `db.c` and `act.social.c` have harness instrumentation, so their line
   numbers differ from `src/`.
2. **Write the oracle scenario first** in `cmd/dp-oracle-diff/scenarios/`. Run
   it and confirm it is **red** on the unfixed code. A scenario that passes
   before the fix proves nothing.
3. **Fix the port** to match C's bytes, including C's typos and quirks (R1).
   Add nothing C does not have (R4): no friendlier message, no extra check, no
   "helpful" fallback.
4. **Run the scenario**: now green. Then ask whether it could still pass if
   the fix were deleted. If yes, make the effect visible (a later `look`,
   `inventory` or `score`) until the answer is no.
5. **Find the class** (R5c). Grep for every other site with the same shape
   and fix or file them. One finding is usually several.
6. **Record it**: a row in the matching `docs/fidelity/depth/*.tsv` (with
   `# depth-case:` in the scenario), a unit test where the oracle cannot
   reach, and a research ledger row if the finding is paper-worthy
   (`docs/research/EVIDENCE_LEDGER.tsv`, 8 tab-separated columns).
7. **Gate, census, PR** (section 2).

### Things semi-frontier models get wrong here

These are the failure shapes seen in this repository. Check for each one
before calling work done.

- **Plausible paraphrase.** A message or rule that reads right but is not C's
  bytes. The fix for `yank`, the invented `[gossip] ...` format, "Tiger punch
  whom?" (C: "Hit who?"), "<mob> appears." on every zone reset: all were
  plausible and all were wrong. Copy the C string exactly.
- **Symptom fitting.** Making one scenario green with a special case instead
  of porting the rule. `do_description`'s trailing space had been imitated in
  three separate places before the rule itself was ported. If a fix mentions
  one room, one mob or one scenario by number, it is probably fitting a
  symptom.
- **Recalled citations.** Nine of one PR's C line numbers were wrong because
  they were recalled rather than grepped (PF-032).
- **Coincidence green.** Both servers print nothing, so the diff matches. A
  seated caster fails `cast_spell` silently on both sides; a spell probe was
  green and proved nothing until the caster stood up.
- **Claiming gates passed.** AGENTS.md: "Subagents that self-report passing
  builds have lied before." Paste the gate output; do not summarize it.
- **Editing the oracle.** `src/` and `darkpawns-c-oracle/` are read-only.

---

## 2. Gates, census and PRs

```bash
make fmt                        # gofumpt, not gofmt
golangci-lint cache clean       # a stale cache has hidden real findings
bash -c 'set -euo pipefail
  test -z "$(gofumpt -l pkg cmd internal)"
  go build ./...
  go vet ./...
  golangci-lint run ./...
  go test -timeout 300s ./... > /tmp/gotest.log 2>&1
  ! grep -E "^(FAIL|--- FAIL|panic:)" /tmp/gotest.log
  go run github.com/securego/gosec/v2/cmd/gosec@v2.29.0 -severity=high -quiet ./...
  python3 scripts/gen_fidelity_depth.py >/dev/null
  python3 scripts/world_fidelity.py
  echo GATES_PASS'
```

- Never chain gates with `;`: a failed gate followed by `; echo ok` reports
  success. A failing test has been pushed that way.
- **One scenario:**
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario <name> --seed 1`
  (add `--show-oracle` to see C's output).
- **Census before every PR:** `ORACLE_REGRESSION_JOBS=24 make oracle-regression`
  (up to 36 jobs is fine; about 15 minutes). An `INFRA` or exit-1 result is
  usually a flake: rerun that scenario alone before believing it.
- Work in a **git worktree off `origin/main`**, never in `~/darkpawns` (it
  holds uncommitted webOLC work).
- Never commit `website-astro/src/generated/project-activity.json`.
- Conventional commits (`fix:`, `docs:`, ...), and cite rules by number in
  commits and PRs ("violates R4" is a complete reason).
- Verify a push with `git ls-remote origin refs/heads/<branch>` against
  `git rev-parse HEAD`.
- **Zach merges and deploys.** Models open PRs; they do not merge, deploy,
  or post outside the repository and Linear.

### Review

The loop that has worked: one model implements, a *different* model reviews
against the C source, and the implementer fixes what the review finds. GLM's
reviews have caught real bugs (mis-cited line ranges, a provider that
outlived its manager). A review is only useful if it re-reads C; a review
that checks the diff against itself is not a review.

---

## 3. The work, in order

Priority is set by player impact first, then by how well the task suits a
semi-frontier model: small, oracle-provable, one C function.

### Wave 1: contained fidelity fixes (good first tasks)

| Issue | What | C source | Proof |
|---|---|---|---|
| **DP-1337** | `mp_give` is unported. Dogs eat or play with gifts, the janitor thanks the giver, and the soul eater (14401) takes a soul (9900) and opens a portal (19611). **This blocks a quest.** | `mp_give`, `src/mobprog.c:113`; `IS_DOG`/`IS_JANITOR`/`IS_DEMON`, `src/mobprog.h:24-33`; call site in `perform_give` (`src/act.item.c`) | Oracle: give food to a dog, a non-soul and then a soul to the demon; `look` shows the portal. |
| **DP-1318** | Incapacitated line: the port smoothed C's typo "an will slowly die". Restore the typo (R1). | `src/fight.c:1572` | Oracle, or a unit test on the string if the state is hard to reach. |
| **DP-1330** | tiger punch, strike and shoot print invented messages instead of C's `damage()`/`skill_message` path. | the `do_*` handlers in `src/act.offensive.c` | Oracle on hit and miss (fix skills first; see `skillset`). |
| **DP-1332** | `stat` shows a player's current position as the default position. | `do_stat_character` (`src/act.wizard.c`) | Oracle. |
| **DP-1296** | Combat position messages bypass `act()`, so "a witch is stunned" is lowercase and not hidden. | `update_pos` and its callers (`src/fight.c`) | Oracle. |
| **DP-1308** | `sendText` output skips the blank line C prints before the prompt. | `process_output` (`src/comm.c:637-642`) | Needs a `keep-prompts` scenario (RULEBOOK R5f). |
| **DP-1171** | `do_hide`'s daytime sector/weather guard is missing. | `do_hide`, `src/act.other.c:247` | Oracle (weather needs fixtures). |
| **DP-1341** | `cast 'enchant weapon'`: C holds the next command, the port says "Nothing seems to happen." Investigate before fixing: why does C stay silent? | `cast_spell` / `do_cast` (`src/spell_parser.c`), `spell_enchant_weapon` (`src/spells.c`) | Oracle with `~dpclock pulse`. |

### Wave 2: mechanical, pattern-following (good for bulk work)

| Issue | What | How |
|---|---|---|
| **DP-1340** | ~120 C `mudlog` sites still unported (OLC saves, wizard actions, objsave, specs, steal/loot, kill milestones). | Follow #1644: `game.MudLog(text, type, level, toFile)` with C's exact type (`MudlogBrief`/`MudlogNormal`, CMP is 3) and level (`max(LVL_IMMORT, invis)` where C uses `MAX(LVL_IMMORT, GET_INVIS_LEV(ch))`). Never call it with the world lock held *and* anything that takes that lock (see the #1644 deadlock). One PR per C file. |
| **DP-1343** | 1286 C citations in the four drifted files may use oracle line numbers. | `python3 scripts/cite_drift_report.py` lists them; the 23 flagged first. For each: grep the statement the comment describes in `src/`, fix the number. Depth-manifest `c_site` first, then code comments. No behaviour changes. |
| **Reek hardening** (DP-1285 to DP-1294) | Small defects from the automated reviewer: unchecked type assertions, swallowed errors, a destructive migration test, a mutex held across an HTTP call. | Each is one function. Read the Reek finding, write a failing test, fix. These are not fidelity work; the gates still apply. |

### Wave 3: needs care (brief carefully, review hard)

| Issue | What | Risk |
|---|---|---|
| **DP-1336** | Specials and mob `oncmd` scripts run *before* command resolution and the position check, so `s` bypasses a guard that checks `south`. C runs `special()` after both, in room/equipment/inventory/mob/object order. | Touches the command dispatcher for every command. Census must stay clean; many spec scenarios depend on this path. |
| **DP-1295** | `canSee` lacks C's `LIGHT_OK`: visibility ignores darkness and holylight. | Changes visibility everywhere; `holylight` is a common fixture workaround (search scenarios for it). |
| **DP-1321** | After a successful flee, every command still answers as if fighting. | State bug; find where fighting state is cleared in C's `do_flee` (`src/act.offensive.c:360`). |
| **DP-1311 / DP-1310** | Idle timer voids after 60s (C: about 9 minutes) and typing never resets it; login places the character in the saved room, not C's load room. | Session lifecycle; test with DP_CLOCK pulses. |
| **DP-1331** | `mag_damage`: saving-throw floor, zero damage skipping `damage()`, nil world for mob casters. | RNG draw parity (R3): validate roll *values*, not just a green diff (R5a). |
| **DP-1319** | Mob `fight`/`death` scripts get no target when the opponent is a mob. | Lua bridge (`pkg/scripting/bridge.go`); read `docs/fidelity/depth/lua.tsv` first. |
| **DP-1326 / DP-1306 / DP-1227** | Prompt framing, pager prompt, and C's `page_string` pager. | Prompt bytes need `keep-prompts` / `no-settle` scenarios (R5f). |

### Wave 4: product and research (not fidelity)

- **Triage first.** The July transport and MCP epics (DP-1140 to DP-1176,
  DP-516/522-524) predate `/play` (shipped 2026-09-24), GMCP, TLS on
  darkpawns.org and the Mudlet work. Many are probably done or superseded.
  Check each against the code and close or rewrite it before starting any.
- Website items (DP-321 to DP-331, F-* series): the site is Astro in
  `website-astro/`; follow `website-astro/DESIGN.md` and
  `docs/brand-voice.md`. Deploy is `make deploy-site`, and **only Zach
  deploys**: dry-run first and read the deletions (AGENTS.md).
- DP-Goat agent-layer and research items (DP-213 to DP-230, DP-71/74/75):
  paper claims must go through the evidence ledger (`docs/research/README.md`).

### Leave for Claude (Tuesday onward)

Anything where a wrong-but-green result would be expensive to find later:
cross-cutting dispatcher changes (DP-1336 if the census wobbles), RNG draw-order
work, save-format or migration changes, and security ordering. Brief these,
but do not merge them without a second model's review against C.

---

## 4. How to brief a model

The briefs that landed cleanly (PRs #1490, #1491) had three things:

1. **Ground truth inline.** The C function, quoted or cited with a grepped
   `src/` line, and the port's current code path. Do not make the model find
   the C itself if you can hand it over.
2. **An objective check.** The scenario name and what it must print, or the
   test that must fail first and pass after.
3. **Decisions made.** If there is a judgment call (keep a Go-only command?
   which of two C paths?), decide it in the brief. Models guess when they
   must decide, and their guesses read plausibly.

A template:

```
Issue: DP-XXXX: <title>
Rules: AGENTS.md, RULEBOOK R1, R4, R5c, R5g. Work in a worktree off origin/main.
C source: <function>, src/<file>:<lines> (quote the relevant lines)
Port today: <file:function>, which does <X> where C does <Y>
Do: <the change>, and nothing C does not do.
Proof: scenario <name> red before, green after; census clean; gates pasted.
Class: grep for <pattern> and list other sites (fix or file them).
Record: depth row in docs/fidelity/depth/<file>.tsv; ledger row if it is a
  new kind of finding.
Do not: edit src/ or darkpawns-c-oracle/, merge, deploy, cite from memory.
```

---

## 5. State at handoff (2026-09-25)

- **Merged this week:** the DP-1333 Lua trilogy (#1637, #1641: C's world data,
  the C↔Lua bridge, 54 bindings certified against the oracle), `ignore` and
  the 35 other Go-only commands removed (#1638, #1646), syslog broadcast to
  immortals (#1644), the skill catalog→key table so the God and practice work
  (#1649), and R5g (#1648).
- **The oracle harness can now test Lua**: `[lua-script <path>]` and
  `mob-script <vnum> <path> <flags>` fixtures (see `lua-bind-*.txt`).
- **Recent research rows:** PF-027 to PF-044 in
  `docs/research/EVIDENCE_LEDGER.tsv`; field notes dated 2026-09-24/25.
- **Known open decisions for Zach:** none blocking. DP-1335 (reviving the
  archived Lua designs) is post-port.
