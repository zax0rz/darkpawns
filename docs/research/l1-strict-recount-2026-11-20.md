# Strict L1 recount — 2026-11-20 (finish-L1)

## Method

Redo of the 2026-11-19 strict recount (PF-132): classify every C segment the
coverage tool reports uncovered, joined with the current depth manifests and
the batch TSVs. This pass joins FOUR batch TSVs (batch one #2861, batch two
#2891, batch three #2914, finish-L1 this PR) instead of the two the 11-19
recount had, and uses Claude's combined census on main 43cbca21b
(`/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/claude-integ-1120-full`)
for the coverage join.

**Status:** complete. Claude's combined census
`claude-integ-1120-full` (main `43cbca21b`) finished at 10879.8s:
full corpus CLEAN_AFTER_RECHECK, claims corpus NOT_CLEAN on three INFRA rows
(groinrip-sleeping-pk-depth, mudlog-olc-disk-depth,
spec-proc-remorter-equipment — infrastructure-shaped, re-run budget
exhausted; no FAIL rows; draw parity clean on both corpora). Those three are
Claude's to re-run; the census-dump is intact and was used as-is.

## Row-level recount (the 20 rows the finish-L1 brief carried)

| Rows | Ending after finish-L1 |
|---|---|
| new_cmds.c:2088 | settled-witnessed (compare-band-depth@2) |
| spells.c:191 | settled-unit (unit exception, DP-1202 class; C-cited test + mutant-red revert proof) |
| dream.c:76,78,83,85x2,91,93 (7) | settled-unit (unit exception; oracle cannot age the real-time stamp) |
| interpreter.c:2227 x2 | settled-unreachable (free_rent=YES + static world ⇒ Crash_load return-2 arms cannot fire) |
| zedit.c:677 | settled-unreachable (reset disables non-MOEPDRL letters before zedit's display; fixture + empty-list evidence) |
| limits.c:629 | settled-witnessed (batch three, corpse-decay-depth@2) |
| mobprog.c:577 | settled-witnessed (mob-sound-mime-box-depth; see the hollow-retain note below) |
| constants.c:691 | open (L4: transient descriptor state, DP-1618) |
| constants.c:712, 714, 716 (3) | open (L4: identd socket / same-pass staging, DP-1618) |
| constants.c:713, 715 (2) | settled-unreachable (fall-through ident states held only within one ident_check pass; matches informative.tsv excluded rows) |
| mobact.c:252 | open (race-hate staging) |

**Open after finish-L1: 5 of the 20.** Exact open list: constants.c:691,
712, 714, 716 (four transient-descriptor rows, DP-1618 class), and
mobact.c:252 (race-hate staging). Settled: the other 15 (3 witnessed,
4 unreachable, 8 unit-exception).

## Corrections found while finishing

- **mob-sound-mime-emote's retained blocks are hollow.** The shipped mime
  (vnum 8079) carries MOB_AGGRESSIVE|MOB_RANDZON (80.mob act word 1048648 =
  ISNPC\|AGGRESSIVE\|RANDZON); spawn-mob enters through C's normal 'M' reset
  path, so RANDZON relocates the body at db.c:2132-2145 before the observer
  ever sees it, and the aggressive body removes the fresh one. No retained
  census dump anywhere contains the mime line. The new
  mob-sound-mime-box-depth scenario clears both flags on the fixture mob and
  witnesses the line with the observer co-located
  (glm-l1-final-mime3/census-dump/mob-sound-mime-box-depth.txt:16,20;
  mutant red glm-l1-final-mut-mime, parity MATCH). The existing manifest row
  `mob.mp-sound-emote-line` (mobprog.tsv) keeps its green status on the
  emote-rendering claim but its retained proof does not contain the case
  bytes; flagged for Zach.
- **constants.c:713/715 are same-pass ident states.** CON_IDCONED and
  CON_IDREAD are fall-through assignments inside ident_check's switch and are
  left within the same pass; the waiting states are CON_IDCONING (712) and
  CON_IDREADING (714), which need an in-flight ident socket (DP-1618). This
  aligns the finish TSV with the already-excluded manifest rows.

## Coverage-based recount (complete)

Ran `GOEXPERIMENT=nojsonv2 go run ./cmd/dp-census-coverage --dump
/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/claude-integ-1120-full/census-dump`.
Output retained at `/home/zach/Archives/darkpawns/notes/coverage-2026-11-20/`
(coverage.tsv, coverage-summary.txt). The dump has 2,428 scenarios and
39,283 C blocks at main `43cbca21b`. It reports **4,326 of 4,902 measurable C
segments covered (88.2%)** — 576 uncovered, 1,336 unverifiable short
segments. (11-19 baseline: 4,285/4,902, 617 uncovered.)

The uncovered set was re-classified with the 11-19 precedence, re-joining the
CURRENT top-level manifests (which include Luna's batch-one rows that merged
in #2861 — after the 11-19 recount ran — so three segments move to settled on
manifest rows the 11-19 pass could not see), the batch TSVs (batch two
#2891, batch three #2914) and this PR's `l1-final` TSV. A batch ending
overrides the 11-19 bucket; segments the new dump covers outright leave the
set (42 flipped to covered on main between the dumps — the four say_spell
segments via #2898's matcher wildcard, Luna's mutation-repair witnesses, and
the rest of the merged batch scenarios).

| Bucket (after join) | Segment rows |
|---|---:|
| `settled-excluded` | 304 |
| `settled-unreachable` | 225 |
| `settled-witnessed` | 22 |
| `settled-unit` | 8 |
| `cited-not-witnessed` | 11 |
| `wide-cite-only` | 5 |
| `unit-unqualified` | 1 |
| **Still uncovered** | **576** |

**Open by formula: 16** (was 161 on 11-19) + 1 exposed unit-unqualified =
**17 conservative unresolved** (was 163).

**Moved to settled by current manifest rows (#2861, after the 11-19 run):**

- act.comm.c:1115 — `Monsters can't page.. go away.`: page.tsv:3
  `page.handler-npc` [excluded] carries the exact-line cite; do_page is
  reachable only through the LVL_IMMORT command-table row and
  command_interpreter rejects IS_NPC switched bodies before dispatch, so `ch`
  is never an NPC there.
- comm.c:915 x2 — the `nusage:` record_usage literals (PF-133 class):
  comm.tsv:54 `comm.socket-stats-internal-log` [excluded] carries the
  exact-line cite; the sink is `log()` at comm.c:916, a non-player sink.

**The 16 carried open rows:**

- `cited-not-witnessed` (11): act.offensive.c:558 (`You fail the rescue!`,
  the percent>prob arm; the rescue-roll rows cite it widely and this dump's
  rescue blocks never roll the failure), act.wizard.c:1818 (`Current machine
  time: %s` — the whole line is wall-clock text the dump normalizer redacts,
  the DP-1496 class; date.machine-time is oracle-green at the exact line but
  a retained block cannot carry the bytes), fight.c:1179 x3, 1183, 1187,
  1196, 1209 x2 (the attitude-loot/brag quip lines: oracle-green quip rows
  cite them narrowly but this dump's quip blocks print other arms, and the
  case5/possessive rows are unit-green only), fight.c:1553 (the mounted
  poison-tick line; only a unit-green row covers it).
- `wide-cite-only` (5): constants.c:712, 714, 716 (ident-socket states,
  DP-1618), mobact.c:252 (race-hate staging) — this PR's open rows — and
  fight.c:1591 (see below).

Plus one unit-unqualified: constants.c:691 (CON_CLOSE same-pass state,
DP-1618) — also this PR's open row.

**One new arrival:** `src/fight.c:1591#1` (`screams in pain!`, the
`act("$N screams in pain!", ...)` arm behind the `!number(0,2)` draw at
fight.c:1590). The 11-19 dump covered it via
`spellmatrix-staff-olc-07-depth`; in this dump that scenario ran but the
1-in-3 draw did not fire, so the segment flickers back to uncovered. Its only
manifest cites are wide (cast.tsv spellmatrix rows, fight.c:1587-1614), so it
classifies `wide-cite-only`; a multi-seed run of the staff-olc scenario
matrix would re-cover it.

Note on scope: my branch's own witness scenarios (compare-band-depth,
mob-sound-mime-box-depth, corpse-decay witnesses, zedit skip, dream unit
rows) are not in this main dump — their segments stay uncovered in the table
above and are settled through the batch-TSV join with their retained runs,
as the brief's join prescribes.
