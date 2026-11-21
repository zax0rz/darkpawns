# What the OpenClaw logs remember

*2026-11-21, Claude. Ledger PF-153. Data: `docs/research/openclaw-excavation/commit-model-attribution.tsv`.*

Before the current agent roster, the port was worked by OpenClaw: first on a NUC (karl-havoc), then on a Mac mini. Zach kept two copies of its session logs:
- an archive of the NUC's agent sessions;
- the Mac's `~/.openclaw`, with live SQLite transcripts and a July backup of orphaned transcripts.

This note records what a read-only pass over them can and can't tell us. The raw logs stay private. They hold unrelated personal material, including credential listings, and are not research artifacts to publish.

## Who BRENDA69 was

`BRENDA69 <brenda@labz0rz.com>` authored 705 commits on main between 2026-04-20 and 2026-09-24. It wasn't a separate agent or model: it was a `--author` flag OpenClaw's main agent passed on its own commits. The earliest one found is from 2026-04-24, made by a deepseek-chat session.

OpenClaw also committed under two machine identities:
- `zach <zach@karl-havoc.labz0rz.com>`, 284 commits from the NUC;
- `zach <zach@mac-mini.local>`, 120 commits from the Mac.

Any study of the early port should count all three as OpenClaw. The dedicated Dark Pawns agents Daeron and Reek ran on the Mac.

## Which models did the work

Counts are assistant messages in sessions that mention Dark Pawns. That's a crude filter, so treat these as shares, not totals.

| Source | Dark Pawns sessions | Dates | Leading models |
|---|---:|---|---|
| NUC, `main` | 666 | 04-22 to 05-22 | glm-5-turbo 47,002; deepseek-chat 19,302; deepseek-v4-flash 9,272; kimi-k2.6 4,977 |
| Mac orphan transcripts | 169 | 06-14 to 07-11 | kimi-k2.6 6,844; glm-5-turbo 5,128; deepseek-v4-flash 1,218 |
| Mac `brenda` | 315 | 05-23 to 10-31 | MiniMax-M3 1,718; deepseek-flash 1,480; deepseek-v4-flash 880 |
| Mac `daeron` | 294 | 08-03 to 10-15 | mimo-v2.5 1,312; deepseek-v4-flash 1,189; glm-5-turbo 914 |
| Mac `reek` | 192 | 08-02 to 10-15 | deepseek-v4-flash 1,898; mimo-v2.5-pro 876; mimo-v2.5 676 |
| Mac `main` | 114 | 08-03 to 10-14 | gpt-5.6-sol 2,467; deepseek-v4-flash 1,666; glm-5.3 1,272 |

The NUC-era orchestrator was **glm-5-turbo**, not glm-5-flash as remembered. On the NUC, 292 of the 670 Dark Pawns sessions ran as subagents.

There are two gaps with almost no surviving sessions: 2026-05-23 to 06-13, and 2026-07-11 to 08-02. BRENDA69 committed heavily in both.

## Commit-level attribution is partly recoverable

Earlier we wrote that tying those commits to a model is "probably impossible". It isn't always. When an agent ran `git commit`, git printed `[branch abc1234] subject` into the tool result, and the session records which model produced the turn before it. Matching those lines to main, by short hash or subject, gives:

| Identity | Commits matched | Models |
|---|---:|---|
| karl-havoc (NUC) | 203 of 284 | glm-5-turbo 135, deepseek-chat 59, kimi-k2.6 7, other 2 |
| BRENDA69 | 31 of 705 | mimo-v2.5-base 22 (all May), glm-5-turbo 5, deepseek-chat 3, deepseek-v4-flash 1 |
| mac-mini.local | 3 of 120 | deepseek-v4-flash 3 |

**Caveats.** The model named is the one that ran the commit command. A subagent or an external coding CLI may have written the code. A session that switched models is split evenly across them. Most of BRENDA69's commits fall in the gaps above, or in sessions that were reset or compacted away. So "unknown" stays the honest default there, and the attribution TSV marks only what the logs show.

## What we would need for more

The NUC (karl-havoc) hasn't been wiped. Its disk may hold the sessions missing from this archive, as well as the logs of any coding CLI that OpenClaw drove. That's the next place to look if the slop-track study needs per-commit attribution for the May and July layers.
