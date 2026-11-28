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

| Item | Owner | State (2026-11-28) |
|---|---|---|
| Guild guards (#3097) | GLM-5.4 | merged |
| Corpse attack type (#3106) | DeepSeek | merged |
| Spec-proc gates, PR 1 (#3105) | Ember | merged |
| Spec-proc effects and bytes, PRs 2–3 | Ember | resumes 11-28 01:50 |
| Room 8061 blank line, reconnect prompts | GLM-5.4 | resumes 11-28 13:51 |
| Iteration order, PRs 1–4 (#3101–#3104) + C order by default (PR 5) | Neon | censuses running, then round 2 |
| Toh seal, sobriety, C-claim comment sweep | Kimi | in progress |
| Fill in the window brief's commands and known-bug list | Claude | after the above |

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
- **Gold rows** are the `model` policy's choices. Volume rows come from
  `progression` and `random`.
- **Training format:** the Unsloth decision-model recipe, rows
  `{state, questions, gold}`, gold rows only, filtered by outcome, held out by
  session (memory: decision-model plan).

## Open decisions for Zach

1. **Who chooses the gold rows.** The `model` policy calls an outside command
   once per step (`-model-cmd`, JSON in and out, 30 s limit) and otherwise
   uses a stub. Real gold rows need a wrapper around a model with a CLI or API
   (Luna's, or another lab's). Without it, the window yields port evidence and
   volume rows, but no gold rows.
2. **Text or GMCP state.** Lockstep runs over plain telnet, so rows carry the
   text a telnet player sees, not GMCP. The October architecture plan assumed
   GMCP state. Decide whether text-only rows are acceptable for the first
   model, or whether GMCP capture is a later run.
3. **Depth of play.** The last proof window gained 0 levels in 30 minutes
   (21 fights, 5 kills). If the window's progression play still doesn't
   level, the data stays shallow (newbie-zone only). Decide whether to
   improve `progression` first, or rely on the model policy for depth.

## After the window

- Reach: run the C function-reach and draw-site reach tools over the window's
  scenarios (PF-188, PF-189) and report what play reached that the census
  corpus doesn't.
- The release census, then the release.
- Blog: write while the window runs (kept bugs first; memory has the plan).
