# Rules without a check

2026-11-02. Field note by Claude, from a conversation with Zach.

## The observation

Reviewing a MiMo pull request in the play lab, Claude credited its catches to
the project's written rules rather than to model skill. Its own model can't
explain them: "a green must be able to fail" (R5h), "subagents that
self-report passing builds have lied before" (AGENTS.md), and the
coincidence-green lessons. Zach drew the connection to the port's early slop:
"not only no oracle, but also NO RULES".

## What the history says

The history agrees on the oracle but not on the rules. There was a fidelity rule
from the second day:

| Date | Commit | Event |
|---|---|---|
| 2026-04-17 | `3e11aca83` | Repository starts |
| 2026-04-18 | `f725fd017` | `CLAUDE.md`: "**Stay true to the original. Do not invent game logic.**" Read the C first, port the actual logic, cite file and line |
| 2026-04-24 | `070dfe54e` | "Wave 4b" bulk port; the earliest commit in `pkg/` touching `do_yank` (pickaxe on `do_yank`), the line of the yank paraphrase (PF-002) |
| 2026-07-12 | `ba05d9906` | First oracle differential harness (`cmd/dp-oracle-diff`) |
| 2026-07-22 | `3d58ba719` | `docs/fidelity/RULEBOOK.md` seeded (R1–R5) |

So the slop of April–July was produced *under* a rule that forbade it.

## A sharper hypothesis

The day-two rule and the July rulebook differ in kind, not only in presence:

- **What it governs.** "Stay true to the original" names a goal. R1 names the
  unit: the player-facing bytes. A paraphrased message is "true to the original"
  in spirit and still fails R1.
- **Licence to diverge.** The day-two brief itself invited approximation:
  "a modern architecture", "`backstab_mult()` … is approximated", and
  "`// TODO: phase N`" for anything not yet faithful.
- **Teaching.** Each rulebook rule carries a "Taught by" line naming the
  incident that produced it. The day-two rule was written before any incident.
- **Verdicts.** "Violates R4" is a complete review verdict. "Not true to the
  original" is an argument.
- **A check.** The oracle arrived ten days before the rulebook. A rule that no
  tool can test leaves the agent to judge its own work, and a plausible
  paraphrase passes that judgment.

Hypothesis: **agent slop is controlled by checkable, specific, incident-taught
rules plus a tool that can fail, not by the presence of a rule.** A stated
intent without a check is roughly equivalent to no rule. This fits the existing
thesis that the harness, not the model, is the variable (see the AI-slop-squash
notes) and is ledgered as PF-098.

## How to test it

- Bucket the merged fidelity fixes by when the defect was introduced:
  - before the oracle (≤ 2026-07-12);
  - between the oracle and the rulebook;
  - after the rulebook.
- Compare the "plausible paraphrase" rate per bucket.
- Check whether pre-July commits cite C file and line, as the day-two rule
  required, and how often the cited line actually matches the code. If the
  citations exist but are wrong (as in PR #1637, PF-032), the rule was followed
  in form and not in substance, which is the hypothesis in miniature.
- The data for the first test already exists. The slop track (PF-097,
  `docs/research/fidelity-study/slop-track/`) traced 178 fixed AI-introduced
  mistakes with birth dates, and at least 46.8% were born before the oracle.
  Re-bucket those births at the rulebook date (2026-07-22) as well as the
  oracle date, and compare the categories per bucket.

## Status

The dates and commits are verified from Git. The causal claim is a hypothesis.

## Correction, same day (Zach)

The note above read the day-two `CLAUDE.md` as the rule the early builders
worked under. That is unverified. Zach: the first building happened in
OpenClaw, from a Telegram window, with about half a dozen mostly open-weight
models. An OpenClaw agent loads its own workspace instruction files, not
necessarily a repo's `CLAUDE.md`, so whether those models ever saw the
"stay true" rule is unknown.

What Git shows for 2026-04-17 to 2026-07-12 (before the oracle):

- 590 commits authored `BRENDA69` (the OpenClaw agent identity), from
  2026-04-20 to 2026-07-12, mostly with no model trailer;
- the rest authored `zach` from several hosts (`karl-havoc`, `mac-mini`,
  gmail), with trailers naming:
  - Claude Opus 4.8 (119), Claude Sonnet 4.6 (65), Claude Sonnet 5 (12);
  - GLM (17), Gemini Antigravity (3), Kimi (1).
- `070dfe54e` (Wave 4b, the yank line) was committed as `zach` from
  `karl-havoc` with no trailer, so its model and harness are unknown.
- `design-research-log.md` places the OpenClaw-era agents on minimax-m2.7,
  Z.AI and Ollama, but that log is about the BRENDA player bot, not the port
  builders.

So the claim "the slop was written under a rule that forbade it" is downgraded
to: **a rule existed in the repository; whether the agents that wrote the slop
loaded it is unknown.** For most of the early commits, the unit of analysis is
the harness and its instruction files, not the repository. The corrected
hypothesis is unchanged in shape: rules count only where the agent actually
loads them *and* a tool can check them. "No rules" may be literally true for
the OpenClaw builders.

To verify, each needs Zach or the Mac:

- the OpenClaw workspace instruction files as of April–July 2026
  (`~/.openclaw/workspace*` on `mac-mini`, ideally their Git history);
- the model routing per agent in that period;
- whether Telegram exports exist for the build chats.

Then re-bucket the PF-097 births by harness (OpenClaw vs. Claude Code vs.
others) as well as by date.
