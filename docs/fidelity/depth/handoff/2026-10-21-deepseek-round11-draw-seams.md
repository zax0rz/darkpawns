# DeepSeek — round 11: draw-seam fixes and the flaky-test class

Date: 2026-10-21 · Branch: `deepseek/round11` (off `origin/main` `1dd97cdb0`) ·
Rules: R1, R3, R5c, R5e, R5h

Four independent causes this round, three of them draw rows. Two are in the
port's own seams rather than in game logic, which is worth stating plainly:
a *draw* row can be a logging/parity artefact, and the comparator cannot tell
the difference.

## Part 1 — the intermittent shop test, and the class sweep

`TestSpecShopKeeperActorPortsC` asserted `shop.LastSort == 1` after the `list`
arm, but the sort's `slide_obj` consumes `number(0,5)` per non-produced item
(`src/shop.c:679`): a zero discards the stock instead of sorting it. The draw
came from the process-wide stream, which `pkg/dprng` seeds from the wall clock
at package init (`stream = New(uint32(time.Now().Unix()))`), so the assertion
rode the clock: 12 failures in 40 with `-race -count=40`.

Fixed by pinning the draw with the package's existing recorder
(`installDrawRecorder(t, 1)`), not by weakening the assertion — it still pins
C's behaviour, it just stops depending on which stream position the test lands
on. Revert control: removing the pin reintroduces the 1-in-6 failure.

**Class sweep.** Every test file in `pkg/game`, `pkg/session`, `pkg/command`,
`pkg/parser` and `pkg/combat` that does *not* install a roller pin was run 40
times per test (`-count=40`) and any failure inspected. Result: SWEEP_RESULT

## Part 2 — the 19–24 cluster: it was the race default, not `loud_mobs`

The round-10 lead was `loud_mobs` reading `proto.Noise` where C reads the
runtime `GET_NOISE`. That was a false trail: C never gives a live mobile a noise
different from its prototype — the only writers are the mob-file loader
(`src/db.c:1198`), OLC (prototype only) and medit's live refresh *from* the
prototype (`src/medit.c:235-243`) — so reading the prototype is faithful.

The real cause is one constant. `boot_mobiles` seeds every prototype with
`RACE_OTHER` (`src/db.c:1110`), which is **16** (`src/structs.h:147`), and the
`Race:` extra line overrides it (`src/db.c:1195-1201`). The parser defaulted to
**7**, `RACE_LYCANTHROPE` — a *speakable* race. `can_speak()` is
`is_intelligent()` (`src/utils.c:675-688`), whose table
(`src/constants.c:353-373`) is `0-11` plus `18-20` and `27-30`, so race 16 is
not in it.

Consequence: `brag_message` (`src/fight.c:1168`) returns before its
`!IS_MOB(victim) || !number(0,20)` gate for every mobile without a `Race:` line,
and Go did not. Instrumented run of `peek-gates-depth` seed 3:

```
TMP20   site=combat idx=64750 roll=18
TMPBRAG ch="a gulguthydra" victim="a skull rider" roll=18
```

Mob 14438 has no `Race:` line, so C's gulguthydra cannot speak and draws
nothing; Go's could. That is the whole cluster — the ten rows clear:

```
peek-gates-depth    seed 3  MATCH entries=31192 stream=64752
peek-gates-depth    seed 8  MATCH entries=30530 stream=64088
checkload-depth     seed 3  MATCH entries=98289 stream=540646
checkload-depth     seed 8  MATCH entries=97631 stream=539986
sedit-entry-depth   seed 3  MATCH entries=31476 stream=65036
sedit-entry-depth   seed 8  MATCH entries=30819 stream=64377
sedit-session-depth seed 3  MATCH entries=31473 stream=65033
sedit-session-depth seed 8  MATCH entries=30816 stream=64374
idlist-depth        seed 3  MATCH entries=31476 stream=65036
idlist-depth        seed 8  MATCH entries=30819 stream=64377
```

R5c: one constant, so every `is_intelligent()`/`can_speak()` caller was
affected — `brag_message`, hunt/quarrel (`src/graph.c:281-294`), mobact chatter
and the mobile special that speaks. Parser tests that pinned 7 now pin
`RACE_OTHER`. Revert control: restoring `mob.Race = 7` reddens all ten rows.

## Part 3 — `circle-aware-depth`: a zero-dice draw-log artefact

C's `dice()` returns *before* its draw-log seam when the roll is empty:

```c
if (size <= 0 || num <= 0)
  return 0;                       /* oracle src/utils.c, dice() */
```

Go's `dprng.Dice` returned 0 too and consumed the same (zero) draws, but it
still wrote a `Dice(0,4) = 0` line. The comparator pairs entries, so that
phantom line took C's next real draw's slot: `c=Number(0,500) go=Dice(0,4)`.
The RNG stream itself never differed.

`circle-aware-depth` is exactly this path: an aware mobile counter-hits with a
`0d4` damage roll (`src/new_cmds.c:2434-2441` → `hit(vict, ch)` → `dice(0,4)`).
Fix: mirror C's short-circuit in `dprng.Dice` so an empty roll consumes nothing
*and* logs nothing. All four rows clear:

```
circle-aware-depth seed 2  MATCH entries=4110 stream=4110
circle-aware-depth seed 3  MATCH entries=4111 stream=4111
circle-aware-depth seed 5  MATCH entries=4110 stream=4110
circle-aware-depth seed 8  MATCH entries=4109 stream=4109
```

Tier note: this is `pkg/dprng` (the game's RNG seam), not the census harness and
not DP_CLOCK. No file under `cmd/dp-oracle-diff`, `internal/oraclediff` or
`scripts/census.sh` was touched. Revert control: removing the short-circuit
restores the phantom line and the rows go red.

## Part 4 — `spec-proc-elements-galeru-alive-dead`: a #2121 regression

Claude asked why the round-10 handoff called this red on main when the v0.1.2
release census (on `41f19a5cd`) had it green. The answer: it **is** a regression,
and it is mine.

- Command: `DP_ORACLE_BIN=$HOME/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario spec-proc-elements-galeru-alive-dead --seed 1`
- Oracle binary: `$HOME/darkpawns-c-oracle/bin/circle` (the shared `-g -O0
  -fcommon` reference; unchanged all round).
- Worktree for the "main" claim: `/home/zach/wt/r10-maincheck`, a detached
  worktree of `origin/main` at **`65bfaeb4f`** — which already contained #2121.
  So the comparison was mine-against-mine, and the "pre-existing on main"
  attribution in the round-10 handoff was wrong. `41f19a5cd` predates #2121,
  where the player lane used `player.SendMessage` directly
  (`git show 41f19a5cd:pkg/game/spec_procs3.go`).

Cause: `SPECIAL(elements_galeru_alive)` opens with
`send_to_char("You begin to feel very dizzy and the world around you
fades...\r\n\n", ppl)` (`src/spec_procs3.c:1204`). Round 9's unification routed
that line through `elementTeleportPerson`'s `Act(..., ToChar)`, which appends its
own `\r\n`, so Go printed one extra blank line between the fade message and the
destination room's look. Fixed with a `rawDirect` flag for the `send_to_char`
sites; `platforms` and `master_column` keep `Act` because C drives them through
`act()`.

```
before: say hello [actor]  +1 blank line before "A Tropical Island"
after : no normalized divergence, draw-parity MATCH 4087/4087
```

Also re-ran `spec-proc-elements-galeru-alive`, `…-platforms`,
`…-master-column-all` and `spec-npc-elements-guardian-depth`: all clean.

## Draw-row state after round 11

| scenario | rows | state |
|---|---|---|
| `carve-depth` | 5 | cleared (round 10) |
| `circle-aware-depth` | 4 | **cleared** — zero-dice draw-log artefact |
| `checkload-depth`, `peek-gates-depth`, `sedit-session-depth`, `sedit-entry-depth`, `idlist-depth` | 12 | **cleared** — mob race default |
| `house-depth`, `shop-*`, `will-zen-meditation-depth`, `mudlog-crash-load-entry`, `spec-proc-*` | 15 | unchanged; setup-phase / undiagnosed |

## Decisions for Zach

- **Three PRs, not four.** The race default (12 rows) and the empty-dice seam
  (4 rows) are separate root causes and get their own PRs; the test pin and the
  galeru regression share a third because both are sweep follow-ups with no
  player-facing overlap. Alternative: four PRs.
- **The `pkg/dprng` logging change is tooling, not game behaviour.** It changes
  no draw and no byte; it makes the seam's log match C's `dice()` contract. If
  you would rather keep `pkg/dprng` frozen, the alternative is to teach the
  comparator to ignore zero-consumption entries — that file is stop tier, so it
  would be your call.
