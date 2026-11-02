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

## Evidence from the OpenClaw workspace, same day

Checked read-only on `mac-mini` (`~/.openclaw/workspace`, a Git repository
with 766 commits since 2026-03-08). Zach: OpenClaw first ran on a 2017 NUC named
`karl-havoc`, which is no longer on the network.

**The `karl-havoc` commits are the OpenClaw NUC era.** In `darkpawns`, the
`zach@karl-havoc.labz0rz.com` identity spans 2026-04-17 to 2026-05-04 (284
commits). `zach@mac-mini.local` starts on 2026-05-24, after the migration
planned in the workspace's `KARL-HAVOC-MIGRATION.md` (dated 2026-04-28). So the
earlier count, which treated `karl-havoc` commits as Zach's, understates the
agent share. Most of the "zach" commits before May are OpenClaw.

**The OpenClaw agents wrote the rule themselves.**
- `CLAUDE.md` was added from `karl-havoc` (`f725fd017`, 2026-04-18).
- It was updated from there and by `brenda@labz0rz.com` on 2026-04-20
  (`2e7d42bac`, `1b0ec63fe`).
- The agents knew it existed and edited it.
- The workspace's own `AGENTS.md` and `SOUL.md` at the 2026-04-23 commit
  (`2f42129e6`) contain no Dark Pawns or fidelity rule.
- The daily notes for 2026-04-23 to 2026-05-08 never mention `CLAUDE.md`.
- Whether a building session loaded it is still unknown.

**The models.**
- The 2026-04-23 note: GLM-5-turbo was the main orchestrator ("barely
  distinguishable from Sonnet for orchestration"), the fallback when Anthropic
  was rate-limited.
- The 2026-04-24 note delegates Wave 8 to "deepseek-v4-pro subagent (auth failed
  for Anthropic API, fell back to deepseek-chat via litellm)" and Wave 9 to
  deepseek-v4-flash.
- `DARK_PAWNS_SUBAGENT_SWARM_PLAN.md` (added 2026-04-28) allocates GLM-5.1,
  DeepSeek Chat, MiniMax M2.7 and Claude Sonnet.
- Model routing changed mid-session on auth failures, so the model behind any
  one commit is often not recoverable from Git.

**The definition of done was a compile.**
- The 2026-04-24 note records Wave 6.5 (`2098e2cce`): 22 act.other.c commands,
  `yank` among them, "now registered and usable", "All registered at level 0
  (any player can use)".
- Its check was "`go build ./... && go vet ./...` clean".
- The same day reports Wave 7 as "~4,843 C lines → 1,846 Go lines" and Wave 9
  as "4,203 C lines → 559 Go lines", each "complete".

**What this changes.** The rule was present, written by the same agents, and
sometimes edited by them; the "done" check they actually ran was build and vet.
That fits the corrected hypothesis better than "no rules": an aspirational rule
coexisted with an operational definition of done that could not fail on a
paraphrase. A 4:1 to 7:1 line compression reported as "complete" was the visible
symptom, and nothing in the loop could question it.

**Still open.**
- Which model ran each wave; the workspace logs and transcripts may hold it.
- Whether any session transcript shows `CLAUDE.md` being read.
- What remains on the NUC itself.

## The surviving transcripts, same day

Scanned read-only on `mac-mini`: `~/.openclaw/agents/*/sessions` (including
`.bak` and `.deleted` copies) and `~/.openclaw/backups/orphan-transcripts-20260711`.

**Coverage.**
- The earliest surviving Dark Pawns build transcript is from 2026-04-27.
- Nothing survives for 2026-04-17 to 2026-04-26, the window that produced
  `CLAUDE.md`, Waves 1–9 and the `yank` port.
- Those sessions ran on the NUC (`karl-havoc`) and were not migrated. So
  waves cannot yet be tied to models.

**Models, 2026-04-27 to 2026-05-24.** 88 sessions mention `darkpawns` at least
five times (largest copy per session id; the `brenda` persona sessions excluded
by that threshold).
- **Orchestrator:** `glm-5-turbo` in nearly all of them.
- **Subagents:** `deepseek-v4-flash` (and `deepseek-chat`).
- **From 2026-05-07:** `minimax-m2.7` sessions.
- **Occasional:** `claude-sonnet-4-6`/`sonnet-payg` (2026-04-29, -04-30, -05-06);
  `mimo-v2.5-base` (2026-05-04, -05-13, -05-19).

**Did they read the rule?**
- Of the 88 sessions, **2** read `CLAUDE.md` (a file-read tool call or `cat`;
  2026-04-27 and 2026-04-28).
- **2** spawned the Claude Code CLI (2026-04-28 and -29), which loads
  `CLAUDE.md` automatically.
- Most other mentions are directory listings or doc reviews. Those reviews filed
  it as "AI agent context, separate concern" and under "Files to NOT touch".
- The rule was an artifact the agents catalogued, not one they consulted.

This supports the corrected hypothesis directly: for the surviving OpenClaw
period, the fidelity rule was effectively unloaded, and "no rules" is close to
literally true for the builders, as Zach said. The finding stops at
2026-04-27. The decisive window is on the NUC.

**To close it:** recover the NUC's `~/.openclaw` (agents' sessions and
workspace) for 2026-04-17 to 2026-04-26, and repeat this scan.

## The NUC, same day: the session that wrote `yank`

Zach reconnected `karl-havoc` (192.168.1.106). Its OpenClaw gateway
(v2026.5.2) had exited on a config error at boot (status 78, no restarts),
so no old bot or cron came back. The build-agent transcripts (`main`,
`claude-sonnet`, `gemini`, `opus-reviewer`, `janitor`; 1,928 files, 1.1 GB)
were copied read-only to a private local archive,
`~/Archives/openclaw-nuc-2026-11-02/` (not in any repository; it holds
unrelated personal material).

**Coverage.** Dark Pawns build transcripts on the NUC start 2026-04-22.
2026-04-17 to 2026-04-21 (CLAUDE.md, Phases 0–2) were not found.

**Models, 2026-04-22 to 2026-05-04** (unique sessions mentioning `darkpawns`
at least five times, dated by start in UTC):
- **Orchestrator:** `glm-5-turbo`.
- **Workers:**
  - `deepseek-chat` (via litellm), `deepseek-v4-flash` and `deepseek-v4-pro`;
  - `glm-5.1` from 2026-04-26;
  - `kimi-k2.6` from 2026-04-26, heavy on 2026-05-01 and -03;
  - `mimo-v2.5-pro` (2026-05-02, -03) and `mimo-v2.5-base` (2026-05-04);
  - `claude-opus-4-6` (2026-04-23 to -27), `claude-opus-4-7` (2026-05-03);
  - Sonnet 4.6 (2026-04-29 to -30).
- **How often sessions read `CLAUDE.md`** (file-read tool or `cat`): 2 of 4
  sessions on 2026-04-24, and 0 or 1 a day after that. Sessions with hundreds
  of mentions never opened it.

**The `yank` port.**
- `doYank` first appears in OpenClaw `main` session `8e4697db`, written by
  **`deepseek-chat`** in an assistant tool call at 2026-04-24 22:42 UTC (18:42
  ET).
- It was committed as `070dfe54e` at 19:53 ET.
- Wave 6.5 (`2098e2cce`, 21:15 ET, session `6d338c07`, also deepseek-chat)
  only wired it to the command table.
- The session began with OpenClaw's stock `/new` startup prompt (read the
  workspace files, greet the user in persona); it contains no fidelity
  instruction.
- It never opened `CLAUDE.md`, though the name appears 19 times.
- **It did open the C:** 38 file-read calls on `src/act.other.c`.

The port, against `src/act.other.c` (`do_yank` at :1620):

| C | deepseek-chat's Go |
|---|---|
| `"That's wierd."` (:1633) | `"That's weird."` |
| `"$N is is no position to be yanked around!"` (:1652) | `"%s is in no position to be yanked around!"` |
| `"That probably wouldn't be appreciated."` branch (:1638) | absent |
| `NOPERSON` | a literal `"I see no one here by that name."` |

The model had the source in context and still produced a corrected,
plausible paraphrase. The typos are the tell (see the AI-slop-squash notes).

**What this does to the hypothesis.** It sharpens it. For the session that
wrote the canonical slop specimen:
- **The rule was not loaded:** "no rules" is literally true.
- **The source was loaded:** reading the C is not enough on its own.
- **The definition of done was a compile.**

The behaviour changed only once something could fail a paraphrase: the oracle
(R1) and a rule naming bytes, not intent.
