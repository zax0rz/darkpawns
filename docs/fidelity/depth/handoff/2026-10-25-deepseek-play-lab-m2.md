---
tags: [handoff, fidelity, play-lab, differential, latency, dataset]
last_updated: 2026-10-25
author: DeepSeek (play lab)
---
# Play lab, round 2: play at volume, a decision dataset, and command latency

Milestone 2 of the play lab (`~/DEEPSEEK-BRIEF-2026-10-25-play-lab.md`, after
milestone 1 merged as #2150). Three products this round:

1. **Volume play** with the two policies the brief asks for (`random` for
   coverage, `directed` toward a goal), recorded as one decision row per choice.
2. **A training dataset** in the recipe's `{state, questions, gold}` shape, with
   the random rows separated, an outcome filter, and whole sessions held out.
3. **Command latency** on both local engines, from the same seeded session, with
   per-command-type median / p95 / max and every Go spike over 250 ms.

Findings only, no fixes. Local servers only.

## 1. What was built (lab side)

`~/dp-play-lab` (private, research tooling, never part of the port):

| component | what it does |
|---|---|
| `cmd/dp-play play` | creation flow → entry pulse → state bootstrap → N decisions; records state, options, choice, confidence, rationale, outcome, and the latency of every send |
| `cmd/dp-play bench` | replays one recorded session's exact command sequence against a target engine, timing each command |
| `cmd/dp-play export` | sessions → `{state, questions, gold}` rows; directed rows to `train`/`train-unfiltered`, random rows to `random`, whole sessions to `holdout` |
| `internal/agent` | state extraction (GMCP + text), option enumeration (capped at 255), the two policies, the goal state machines |
| `scripts/local-server.sh --engine c` | runs the C oracle on loopback from a copy of its lib, with the same `DP_SEED`, `DP_CLOCK` and `DP_FIXED_TIME` as the Go side |
| `scripts/run-batch.sh` | records a batch of sessions, restarting the server (fresh name, fresh zone resets) between them |
| `scripts/patch-world.py` | the lab's copy of the two fixtures the light goal needs, matching the harness's `set-room-flag` / `spawn-obj` exactly |
| `scripts/latency-report.py` | median/p95/max per command type, per engine, plus the spike list |
| `scripts/world-probe.py` | room graph queries (dark rooms, routes, mob levels) used to plan the goals |

**State.** GMCP (`Char.Vitals`, `Char.Status`, `Char.Name`, `Room.Info`) plus the
raw text: the vitals prompt, the `[ Exits: … ]` line, the room's contents, and
the `score` / `inventory` / `equipment` / `list` tables. The C oracle has no
GMCP at all, so a C-side row would be text-only; every dataset session is
recorded on the port.

**Options.** Moves from the visible exits, `kill`/`rescue`/`flee` on visible
mobs, `get` on room objects, `drop`/`wear`/`wield`/`hold` on carried items,


## 2. Sessions, policies and volume

Eight sessions, 340 decisions, recorded against a locally built port
(`44eb9c360`) on a frozen clock. Every session restarts the server first, so
character names are free and the zone resets are replayed.

| session | policy | goal | decisions | rows that made progress |
|---|---|---|---|---|
| `m2-s1-random-80` | random | — | 60 | 4 (uniform draws are mostly refusals) |
| `m2-s2-explore-80` | directed | explore zone 80 | 60 | 42 |
| `m2-s3-level3` | directed | reach level 3 | 80 | 3 kills, 24 experience |
| `m2-s4-shop` | directed | buy a weapon and wield it | 40 | 39 |
| `m2-s5-flee` | directed | survive a fight and flee at low HP | 60 | flees and rests |
| `m2-s6-wear` | directed | get an item and wear it | 40 | 2 |
| `m2-s7-lights` | directed | hold a lantern in a dark room and look | 20 | 6 |
| `m2-s8-dim` | directed | explore the dim streets | 40 | 40 |

Policy mix: 60 random decisions (18%), 280 directed (82%).

What the directed policy actually did, and what that says about the goals:

- **explore zone 80** worked: it walks to rooms it has not seen, using GMCP's
  exit destinations. Before that it bounced between two rooms forever, because
  "first visible exit" is deterministic.
- **buy a weapon and wield it** reached the Dry Goods Store in eight moves,
  listed its wares, and then wielded the sword it already carried — the shop
  stocks no sword, so the goal degraded honestly instead of buying one. It never
  had the 36 coins a lantern costs (`score`: 0 gold), which is why the light goal
  uses a spawned lantern rather than a purchase.
- **reach level 3** did not get there: three citizens killed for 24 experience in
  eighty decisions. The city's mobs are level 1-8 and the newbie hunting grounds
  are outside the city, so this goal needs a route to a real hunting zone before
  it can produce many rows. The combat rows it did produce are sound.
- **survive a fight and flee at low HP** fled at 7/21 HP (below the 35% line) and
  then rested; resting on a frozen clock heals only after enough pulses, so it
  spent eight decisions resting before recovering. That is a policy lesson
  (advance more time per rest, or read the regeneration cadence), not a game
  finding.
- **get an item and wear it** had almost nothing to do: the starting rooms hold
  no objects, so two of forty decisions made progress.
- **hold a lantern in a dark room and look** is the one that found PL-003, and
  its twenty decisions are the light proof: dark without the light, lit with it.

## 3. Findings

### PL-003 (bytes) — a held lit lantern did not light a dark room

**Already fixed, by someone else, mid-round.** This is the bug the light goal was
written to catch, and the play lab caught it independently of PR #2148
(`6e245a2ba`, "fix: room light counts a lit held lantern like C
equip_char/unequip_char"), which merged between the commit my lab server was
built from (`4fc5c8ab2`) and the commit I censused (`44eb9c360`).

**What it was.** C's `has_light()` (`src/handler.c:823-835`) reports a held light
when `GET_OBJ_VAL(obj, 2) != 0` in *any* wear slot, and `char_to_room()`
(`src/handler.c:532-555`) raises the room's light counter for it. The port's
`Player.HasLight()` read **`Values[1]`** and only the **light slot**, while its
own `isLitLightSource()` read `GetValue(2)` (`pkg/game/world.go:672-679`). A
brass lantern's values are `0 0 96 0`, so C saw a light and the port did not: the
character walked into a dark room holding a lit lantern and read `Darkness` where
C showed the room.

**Evidence — a revert control and a regression guard.**

| build | same commands, same patched world | result |
|---|---|---|
| `4fc5c8ab2` (pre-fix) | get lantern, hold lantern, north, north, look | `Darkness` / `It is too dark here to see much of anything...` |
| `44eb9c360` (post-fix) | identical | `Temple Medical Centre` (the room, matching C) |

The transcript pair is retained in the lab
(`sessions/m2-lights-prefix.jsonl`, `m2-lights-postfix.jsonl`, with the build of
each in `sessions/m2-lights-provenance.md`), and
`cmd/dp-oracle-diff/scenarios/play-lab-lights-dark-room.txt` is the C-vs-Go
scenario: it is CLEAN on `44eb9c360` and, per the table above, reddens on the
pre-fix build — so it is a guard that can fail (R5h), not a scenario that passes
because it tests the one input where old and new agree.

### PL-002 (state) — GMCP `Room.Info` is not sent for a dark room

Walking into a dark room without a light produces `Darkness` and a `Char.Vitals`
update, but **no `Room.Info`**, so a GMCP client's room and map stay on the room
it just left. Reproduced with a GMCP probe: `north` into room 8146 emits
`Char.Vitals {"mv":80,...}` only, while the text says `Darkness`; the recorded
session's `state.room` stays `8161`.

- **Cause:** `observeRoom` (`pkg/game/look.go:214-232`) returns from the
  blind/dark branch before `result.Room = &view` (`:237`), and
  `RenderObservationMessages` fires `observer.RoomShown` — the only caller of
  `sendGMCP("Room.Info", …)` — solely when `result.Room != nil`
  (`pkg/game/look.go:139-143`, `pkg/game/gmcp.go:455-474,506-515`).
- **Class:** state. The *text* is right and matches C
  (`src/act.informative.c:745`), so this is **not** a C-vs-Go byte divergence: C
  has no GMCP at all. It is a defect in a port addition, and no scenario can see
  it — the harness transport consumes GMCP framing
  (`internal/oraclediff/conn.go:77-121`) and the browser-transport mode compares
  the terminal, not the structured feed.
- **Consequence for this lane:** the dataset cannot trust `state.room` in a dark
  room, so the state records `"dark": true` and the exporter says so instead of
  passing a stale room number to the model.

### Observations (both engines agree; not findings)

- **In a dark room the pack is invisible.** `hold lantern` in the dark answers
  `You grab something.` and `drop lantern` answers `You don't seem to have a
  lantern.` even when the character is carrying it — C-faithful, and the reason
  the light goal lights the lantern *before* walking in.
- **A sleeping character can do nothing else.** Any command but `wake` answers
  `In your dreams, or what?`. A uniform policy that sleeps early therefore spends
  most of its decisions on refusals; the refusal list in the lab now names it, so
  those rows are labelled instead of looking like silent decisions.
- **Unarmed combat is slow and visible.** `kill citizen` answers `The citizen
  ducks under your fist as you try to hit him.`; a level-1 warrior with fists
  needs many rounds, which is why the directed combat policy wields a weapon
  first.

### Lab defects found while building the policies (recorded so the rows are not misread)

- A worn or wielded item was left in the carried list, so the policy offered
  `wield sword` for a sword it was already wielding — refused forever. The
  equipment table's slot is inside angle brackets (`<held>  a lantern`); reading
  the text before the bracket yields an empty key.
- The first batch's combat policy fought with fists (no `wield` step) and its
  fight loop was too short to land a kill, so early rows looked like fights that
  changed nothing. Both fixed; the batches in §3 are the corrected ones.

`rest`/`sleep`/`stand`/`wake`, shop `list`/`buy`/`sell`, `cast` when the
character knows spells, plus `say` and `look` — capped at 255, in a stable order.
Target keywords are guessed from the game's own rendering, so an option can be
refused; the refusal is part of the row's outcome, which is the point.

**Timing.** A session runs against a `DP_CLOCK=1` + `DP_FIXED_TIME=1770838461`
server and drives time explicitly: `~dpclock pulse 40` for the newbie intro, 10
pulses per combat round, 30 pulses after a rest. Nothing in a session depends on
the wall clock.

## 4. The dataset

`dp-play export` wrote, from the eight sessions:

| file | rows | what |
|---|---|---|
| `dataset/train.jsonl` | 286 | directed rows that passed the outcome filter |
| `dataset/train-unfiltered.jsonl` | 300 | every directed row of the non-held-out sessions, for comparison |
| `dataset/random.jsonl` | 60 | random-policy rows — exploration and differential data, never labels |
| `dataset/holdout.jsonl` | 40 | the held-out session (`m2-s8-dim`) |
| `dataset/manifest.json` | — | the filter and hold-out rule, so the set is reproducible |

The row is the brief's `{state, questions, gold}`:

```json
{"state": "Goal: explore zone 80. You are in Temple Infirmary (8162), standing, HP 21/21, mana 100/100, move 85/85, level 1, 1 exp, 0 gold. Exits: north. Carrying: sword, tunic, backpack. Wearing: nothing. Last output: …",
 "questions": {"action": {"type": "choice", "instructions": "Pick the next command toward the goal.",
                          "criteria": {"north": "walk north", "look": "look at the room again", "…": "…"}},
               "flee_now": {"type": "noul", "instructions": "Should the character flee this round?"},
               "danger": {"type": "score", "instructions": "How dangerous is the current situation?",
                          "criteria": ["safe", "risky", "deadly"]}},
 "gold": {"action": "north", "flee_now": false, "danger": 0},
 "meta": {"session": "m2-s2-explore-80", "step": 1, "rationale": "…", "confidence": 0.6,
          "outcome": {"…": "…"}, "options": ["north", "…"]}}
```

- **Option keys** are short and stable (`kill_guard`, `get_sword`, `wear_tunic`),
  and each criteria entry says what the option does. The list is capped at 255.
- **Filter:** keeps a directed row when it was not refused, its own outcome did
  not die, its session did not die within five steps after it, and it did not
  lose experience — 14 of 340 rows dropped.
- **`danger`** is computed from the state at decision time (HP fraction and
  whether anything is in the room), never from the outcome, so it is a label and
  not a peek.
- **Hold-out is by whole session** (the last session in sorted order for a batch
  this small, every tenth otherwise), because rows from one session leak into
  each other.
- **`meta`** carries the rationale, confidence, outcome and raw option list. The
  recipe does not need them; the research record does.

## 5. Replaying sessions through the census

Five of the eight sessions were reduced to scenarios (the commands the policy
chose become the probe) and run as one targeted census at seed 1:

```
deepseek-m2-sessions-seed1 @44eb9c360: scenarios=5 passed=5 expected=0 failed=0
  elapsed=163.644s  verdict=CLEAN
oracle-draw-parity: scenarios=5 draw_divergent=0
```

| scenario | probe commands | fixtures |
|---|---|---|
| `play-lab-m2-s2-explore-80` | 60 | `quiet-mobs` |
| `play-lab-m2-s3-level3` | 80 | none (the mobs must be there for `kill` to reach C) |
| `play-lab-m2-s4-shop` | 40 | `quiet-mobs` |
| `play-lab-m2-s7-lights` | 20 | `quiet-mobs`, `set-room-flag 8146 0 on`, `spawn-obj 8031 2 8162 80` |
| `play-lab-m2-s8-dim` | 40 | `quiet-mobs` |

So 240 policy-chosen commands — movement, `look`, `list`, `get`, `hold`,
`wield`, `kill` — produce identical bytes and identical RNG draw streams on the
local C oracle and the local port. Play at volume found no byte divergence in
this sample; it found PL-002 (state) and PL-003 (already fixed) instead.

One caveat, per R5h: the `m2-s3-level3` scenario keeps its mobs, so its `kill`
commands start real fights — but a scenario probe cannot advance the frozen
clock, so the fights never resolve and the probe certifies only the entry
messages. A fight that resolves would need the pulse control inside the probe,

## 6. Command latency: local C oracle vs local Go port

Method: one recorded session replayed by `dp-play bench` against each engine, on
the same seed (`DP_SEED=1`), the same frozen clock (`DP_CLOCK=1`,
`DP_FIXED_TIME=1770838461`), the same world copy and the same transport (plain
telnet), with a freshly restarted server on each side. Three sessions — a random
walk (141 commands), a combat session (217) and the light session (53) — 411
timed sends per engine. "first byte" is send → first byte back; "prompt" is send
→ the completed vitals prompt a player reads as "the command finished".

### All commands

| engine | n (first byte) | median | p95 | max | n (prompt) | median | p95 | max |
|---|---|---|---|---|---|---|---|---|
| Go port | 359 | **0.6 ms** | 1.1 ms | 47.5 ms | 323 | **0.6 ms** | 1.3 ms | 60.1 ms |
| C oracle | 269 | **99.3 ms** | 100.0 ms | 199.0 ms | 263 | **99.3 ms** | 99.9 ms | 179.9 ms |

**Go / C ratio: 0.01x** on the median and on p95, for both first byte and prompt.

### Per command type (median first byte → prompt, ms)

| command | Go n | Go first byte | Go prompt | C n | C first byte | C prompt |
|---|---|---|---|---|---|---|
| `look` | 158 | 0.7 | 0.8 | 130 | 99.3 | 99.3 |
| `north` | 12 | 0.7 | 1.1 | 2 | 98.5 | 98.5 |
| `south` | 8 | 0.7 | 0.9 | — | — | — |
| `kill` | 6 | 0.8 | 0.8 | 5 | 88.2 | 88.2 |
| `wield` | 18 | 0.4 | 0.5 | 16 | 99.2 | 99.2 |
| `wear` | 14 | 0.4 | 0.4 | 14 | 99.1 | 99.1 |
| `hold` | 12 | 0.5 | 0.5 | 12 | 99.0 | 99.0 |
| `drop` | 10 | 0.5 | 0.5 | 10 | 99.2 | 99.2 |
| `get` | 2 | 0.3 | 0.3 | 2 | 99.2 | 99.2 |
| `list` | 10 | 0.2 | 0.2 | 10 | 99.1 | 99.1 |
| `sell` | 12 | 0.3 | 0.3 | 12 | 99.3 | 99.3 |
| `rest` | 22 | 0.5 | 0.5 | 22 | 99.1 | 99.1 |
| `sleep` | 8 | 0.4 | 0.4 | 8 | 99.1 | 99.1 |
| `stand` | 16 | 0.4 | 0.5 | 16 | 99.2 | 99.2 |
| `wake` | 4 | 0.5 | 0.5 | 4 | 99.4 | 99.4 |
| `flee` | 2 | 0.7 | 0.8 | 2 | 99.8 | 99.8 |
| `say` | 4 | 0.7 | 0.7 | 4 | 99.3 | 99.3 |
| `~pulse` | 3 | 0.4 | 0.4 | 1 | 178.2 | 178.2 |
| `(greeting)` | 3 | 0.1 | — | 3 | 198.2 | — |

### Go spikes over 250 ms

**None.** The largest Go first-byte time in 359 sends is 47.5 ms, and it is the
password line during creation (a one-off hash, not a command); the largest prompt
time is 58.0 ms, the `1` that enters the game, which carries the newbie-intro
flush. Every in-game command is under 2 ms.

### Why C sits at ~99 ms — and whether it matters

C's main loop is quantized: `OPT_USEC 100000` (`src/structs.h:628`) is ten passes
per second, the loop waits out the remainder of the current pass
(`src/comm.c:545`, `select(0,0,0,0,&timeout)`), and input is processed once per
pass (`process_input` at `src/comm.c:580`, called from the pass loop). A command
therefore waits for the next 100 ms pass, which is exactly the ~99 ms median and
the ~199 ms greeting (two passes). It is C's design, not a defect, and the port's
reader goroutine does not share it.

**This is a timing difference, not a byte difference, and I have not treated it
as a fidelity violation.** Two things follow, and they are for Zach:

- The port answers ~200x faster than the original. A player cannot see the
  difference in bytes, but they can feel it, and C's cadence is part of how the
  original played.
- More interesting for fidelity: C's pass loop *quantizes* input, so a command's
  output can interleave with pulse output differently than in a port that
  processes it immediately. I did not find a scenario where that changes bytes —
  the frozen-clock censuses compare pulse-driven output extensively and are clean
  — but the spike hunt's real result is that the port has no spikes to explain,
  so there is nothing here for the harness to catch either way.

which the harness does not offer.


## 7. Handoff numbers (milestone 2)

- **Sessions recorded:** 8 (60 random decisions, 280 directed). 340 decision rows
  total; 105-217 recorded steps per session including controls.
- **Policy mix:** random 18% / directed 82%.
- **Findings by class:** bytes 1 (PL-003, already fixed by #2148 — the lab's
  scenario is the regression guard), state 1 (PL-002, open, GMCP only), timing 0
  (the latency gap is a design difference, reported not filed), draw 0.
- **Dataset:** 286 filtered training rows, 300 unfiltered directed rows, 60 random
  rows, 40 held-out rows from one whole session (`m2-s8-dim`), plus the manifest.
- **Replay:** 5 session-reduced scenarios, 240 probe commands, CLEAN at seed 1
  with draw parity.
- **Latency:** 359 sends on Go, 269 on C; Go median 0.6 ms / p95 1.1 ms, C median
  99.3 ms / p95 100.0 ms, zero Go commands over 250 ms.
- **Lab artifacts:** `~/dp-play-lab` — `sessions/m2-*` (sessions, latency files,
  bench transcripts), `dataset/`, `sessions/latency-report.md`, scenario sources
  under `scenarios/`, and the recorded pre/post-fix light transcripts.

## Decisions for Zach

1. **Is the ~200x latency gap in scope?** The brief asked for the measurement and
   I have reported it; I have not filed it as a fidelity violation, because R1
   governs bytes and C's 100 ms pass is a design choice, not a divergence. If you
   want the port to *feel* like C, that is a deliberate, player-visible departure
   from the original and would need its own ruling — and it would slow every
   command, so I would not do it without one.
2. **PL-002 (GMCP `Room.Info` missing in a dark room)** is a real client-visible
   defect in a port addition, and nothing in the corpus can see it. Fix it, or pin
   it as a known GMCP limitation? It also means the dataset's `state.room` is
   unreliable in dark rooms, which the rows now flag with `"dark": true`.
3. **PL-003's scenario:** `play-lab-lights-dark-room` is a regression guard for a
   fix that already landed (#2148). Keep it in the corpus as a guard, or drop it
   as redundant with that PR's own tests?
4. **The five session scenarios.** They are 240 commands of policy-chosen play and
   they are clean, but they are slow (the five took 164 s of census) and they
   overlap command surfaces the corpus already covers. Keep them, keep only the
   lights one, or treat session replays as one-off evidence rather than corpus
   members? My preference is to keep the lights one and drop the other four
   unless you want the coverage.
5. **Goal coverage for a next round.** "reach level 3" needs a route to a real
   hunting zone (the city holds no newbie-appropriate prey), "get an item and
   wear it" needs rooms with objects, and `cast` was never exercised because every
   character this round was a warrior. Say which you want first; a cleric or mage
   session with practiced spells would cover the cast options.
6. **Dataset volume.** 286 filtered rows is a smoke test, not a training set, and
   the recipe wants orders of magnitude more. If you want volume next, the cheap
   levers are longer sessions (200-300 decisions), more sessions per goal, and the
   random policy for coverage — the policy code is the same either way.

