# Plan: the 24-hour play window and decision-model data

Written by Claude for Zach, 2026-11-28. One run, two outputs. This page is the
map; the details live in the documents it links.

1. **Port evidence.** The stopping rule needs 24 consecutive wall-clock hours of
   play through both engines, on one Go commit, with no new divergence
   ([roadmap](ROADMAP-2026-11-10-port-completion.md), section 6).
2. **Training data.** Every step of that play is a decision row for the
   small "System One" decision model
   ([architecture plan](AGENT-DECISION-MODEL-PLAN.md), October; the
   lockstep design below replaces its data-capture section).

## How the play runs

Lockstep differential play (darkpawns#3094 and dp-play-lab#2, merged
2026-11-27). The C oracle and the Go port boot side by side on a frozen
clock. The policy reads C's output, picks one command, sends it to both
engines, and compares the two replies with the census's own comparator and
draw parity. On the first difference the session stops, is replayed through
`scripts/census.sh`, and is shrunk. Every session is also written as an
ordinary census scenario, so the census replay is the proof.

- Runner: `dp-play window` in the private `~/dp-play-lab`, from a worktree at
  the commit under test (never the shared checkout).
- Characters: all 36 race/class pairs, both sexes, three hometowns, a fresh
  seed each session.
- Policies: `progression` (goal-driven volume), `random` (reaches what plans
  don't), and `model` (an LLM chooses from the options list; these are the
  gold rows).
- Goals are checked by outcome in C's reply: leveled, died, ate, practised,
  and so on.

## Before the window can start

The window counts only clean hours, so every known divergence has to be
fixed first, or it fires again every few sessions.

| Item | Owner | State (2026-11-28, afternoon) |
|---|---|---|
| Guild guards (#3097) | GLM-5.4 | merged |
| Corpse attack type (#3106) | DeepSeek | merged |
| Spec-proc gates, effects, bytes (#3105, #3115, #3117) | Ember | merged |
| Spec-proc leftovers and unassigned stand-ins (#3119, #3121) | Ember | merged |
| Room 8061 blank line, reconnect prompts (#3116, #3118) | GLM-5.4 | merged |
| Iteration order, C order by default (#3101–#3104, #3114) | Neon | merged |
| Toh seal, sobriety, C-claim sweep (#3112, #3113) | Kimi | merged |
| Invented strings, Lua `raw_kill` (#3120) | Kimi | merged |
| World text framing, trailing blank lines ([follow-up](../briefs/FOLLOWUP-2026-11-28-glm-world-text-framing.md)) | GLM-5.4 | in progress |
| Wall guard `fighter(ch, ch)`, E-034 ([brief](../briefs/BRIEF-2026-11-28-wall-guard-fighter-player.md)) | Kimi | in progress |
| Typed spacing, PF-193 ([brief](../briefs/BRIEF-2026-11-28-ember-typed-spacing.md)) | Ember | in progress |
| **Pulse-wait read race in the harness** ([brief](../briefs/BRIEF-2026-11-28-pulse-barrier-read.md)): every 630-pulse rest can stop a session, and quiet waits cost 7.5s each | Neon (suggested) | brief written; **blocks the window** |
| Data fixes in `dp-play-lab` (PR #3: runner overrides, sources, per-round combat, filters, row format, escalate, teacher transport, answer-blind cap, overwrite guard) | Sol, Luna | done on PR #3, not merged |
| Progression that levels, round 2 ([follow-up](../briefs/FOLLOWUP-2026-11-28-play-lab-progression-round-2.md)): guild at level 1, class powers each round, loot and wear, C's `consider` arithmetic, per-mob memory | lab agent | brief written |
| A real teacher model for `model` sessions | Zach | open |
| Fill in the window brief's commands and known-bug list | Claude | after the above |

The data fixes don't touch the port, so they run in parallel with the rest.
They have to land first anyway: without them, the window still produces port
evidence, but its training rows are quietly wrong.

Then Luna starts the window
([draft brief](../briefs/BRIEF-2026-11-26-luna-24h-play-window.md)). Her
current divergence hunt
([brief](../briefs/BRIEF-2026-11-27-luna-divergence-hunt.md)) keeps finding new
classes until then.

## The window itself

- 24 consecutive hours on one commit with no new divergence. A ledger
  `EXPECTED` divergence doesn't count; a FAIL or a draw divergence does.
- If a divergence appears, the count restarts from it. If a fix lands and the
  window moves to the new commit, the count restarts again.
- Concurrency 4. Lockstep sessions take census worker permits, so the 02:30
  nightly census isn't starved.
- No wall-clock cutoff: the window ends when the report shows 24 clean hours on
  one commit, or when Zach says so.

## The data

- **What a row holds:** state, options, choice, confidence, rationale, goal,
  subgoal and outcome, plus seed, race, class, sex, hometown, step, pulses,
  the C and Go output digests, the policy, and `teacher_model`.
- **Where it lives:** `dp-play-lab` and `~/Archives` only, never the public
  repo. No real player data: local engines, test characters.
- **Gold rows** are the `model` policy's choices, in `train.jsonl`. Heuristic
  `progression` rows go to their own file, and `random` rows to theirs. A step
  where the teacher failed and the stub answered is labeled `stub-fallback`
  and is never gold.
- **The questions per row:** `action` (choice among the options) and
  `escalate` (should System One wake the big model now?). The escalate head
  is the most valuable thing the small model learns: knowing when it's out of
  its depth.
- **Row shape for the trainer:** the subgoal, position, fight target, vitals
  as percentages and the latest output come **first** in `state`, because the
  Unsloth trainer cuts the end of a long input. At most 20 options per row
  (labels get trimmed past about 20), with the chosen one always kept, and
  the option order shuffled per row.
- **Combat is decided round by round** (20 pulses), with `wait` as an option,
  so the data has the per-round choices: stand after a bash, flee at low HP,
  wait out skill lag.
- **Filtering:** defensive choices just before a death are kept (a failed flee
  was still the right call). First tries that got refused are kept; repeats
  are dropped. Rows before a divergence are still verified against C, so
  they're kept, tagged.

## Training, in two stages

1. **Stage 1, after the window:** supervised training with Unsloth's
   `DecisionTrainer` on the gold rows, held out by session, with the
   player's own previous command randomly masked so the model reads the state
   instead of copying itself. Train **Laya-421M** (bidirectional, runs on CPU)
   and a small Qwen (needs a GPU to serve) on the same split, and compare
   calibration and latency. Act on `probabilities`, not `confidence`
   (Unsloth's docs say so). Their guide disagrees with itself on the learning
   rate (2e-4 in the prose, 2e-5 in the code): try both, and check that
   `build_dataset` skipped no rows.
2. **Stage 2, after v1:** play the Stage 1 model in Go alone on the frozen
   clock (fast, no oracle needed once the port is done). Wherever it's unsure,
   escalates, loops, dies or gets refused, ask the big model what it should
   have done in that exact state, add those rows, and retrain (DAgger).
   Supervised training alone tends to fall apart once a model drifts into
   states the teacher never visited.

## Open decisions for Zach

Each has a recommendation from Carbon's review (2026-11-28), checked by Claude.

1. **Who chooses the gold rows.** The `model` policy calls an outside command
   once per step (`-model-cmd`) and otherwise uses a stub. *Recommended:* a
   persistent teacher service (the data-fix brief adds the interface), on 1 of
   the 4 workers, replying with the choice, subgoal and escalate. The window
   doesn't need to produce all the gold: a few thousand good rows bootstrap
   Stage 1, and Stage 2 and your own play add more after v1.
2. **Text or GMCP state.** Lockstep runs over plain telnet. *Recommended:*
   text-derived state is fine for v0. The real risk is train/serve skew, so
   the MCP/CLI observation built after v1 must use the same state
   serializer as these rows.
3. **Depth of play.** The last proof window gained 0 levels in 30 minutes.
   *Recommended:* fix it before the clock starts. The first bar, level 3 in a
   20-minute pre-flight, was unrealistic: level 2 needs 1,500 XP and level 3
   needs 3,000 (`src/class.c:1131-1140`). The bar is now a fast Go-only loop
   (four of six classes reach level 3 in 1,000 steps), then a 60-minute
   lockstep pre-flight where every stop is confirmed by a full replay and XP
   rises in every session.
   Shallow play would weaken both the port evidence (no level-ups, mid-level
   skills, hunger ticks) and the data.

## After the window

- Reach: run the C function-reach and draw-site reach tools over the window's
  scenarios (PF-188, PF-189) and report what play reached that the census
  corpus doesn't.
- The release census, then the release.
- Blog: write while the window runs (kept bugs first; memory has the plan).
