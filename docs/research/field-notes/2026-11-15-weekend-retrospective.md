# Weekend retrospective, 2026-11-14 to 11-15

2026-11-15, evening. Field note by Claude, written with Zach while Luna and Hydrogen were still running. It complements the method note, [`2026-11-15-human-on-the-loop.md`](2026-11-15-human-on-the-loop.md): that note describes how the work is organised, and this one records what the weekend's work actually was, most of it work no player will ever see.

## The numbers

Counted on `origin/main` at the end of 2026-11-15:

| | |
|---|---|
| PRs merged on 11-14 and 11-15 | 136: GLM 34, Hydrogen 30, DeepSeek 28, Flash 16, Claude 10, Kimi 9, Luna (`codex/` branches) 9 (`gh pr list --search "merged:2026-11-14..2026-11-15"`) |
| Commits on main | 287 |
| Census runs | 221 run directories under `~/Archives/darkpawns/oracle-runs/2026-11-14` and `2026-11-15`, about 49 GB |
| New oracle scenarios | 56 |
| Go code | +6,460 / −436 lines in 137 files |
| GitHub issues closed | 69 |
| Linear | about 12 issues filed (DP-1536 to DP-1547), and about 85 closed, cancelled or ruled on |
| Agent prompts written by Claude | 61 (`~/PROMPT-2026-11-14-*` and `-11-15-*`, outside the repo) |
| Zach's rulings | 13 |

## What players will notice

Very little, which is the point of a fidelity port. Mobs regain mana (DP-1542) and dream while asleep (DP-1543). Fighter guards berserk again (DP-1547). Coin piles have C's extra descriptions (DP-1536). Three trailing spaces are back (DP-1471, DP-1544). Switched immortals can use about twenty more commands as the mob (DP-1535). The Go-only spam filter and guest login were ruled out and are being removed (DP-1014, DP-1379).

## The unglamorous work

- **Integration:** stacking related PRs into one branch (#2678, which surfaced a duplicate lock), retargeting chains and close-and-reopen to trigger CI (the potion and scroll series), and holding a proof (#2639) for a day while a timing bug elsewhere was fixed (#2686).
- **CI reds on other agents' branches:** scenario `depth-case` labels with no manifest row (#2675, #2677, #2680, #2688), a test that pinned behaviour a ruling had just changed (#2705), and a ledgered divergence whose fingerprint changed when the normalizer did (#2710).
- **Bisecting:** four targeted censuses pinned two guard-fight reds on #2700 (DP-1547).
- **Recounting:** GLM's DP-1536 report had contradictory draw counts. Hydrogen's recount found the first misaligned draw in `create_money`, a path GLM had ruled out.
- **Archaeology:** two "live server" mysteries from 2026-09-23, the Temple boards (DP-1313) and the zone-144 skull rider (DP-1322), were both explained by one world-state reload that `d4b3298c7` removed three days after they were filed.
- **Triage:** 118 issues read (#2715, #2717). 22 GitHub issues no longer stood, and 44 Linear tickets described designs that had since been deleted (MA-013).
- **Evidence:** every ruling recorded in Zach's words on its issue, ledger rows PF-121, PF-122, MA-012 and MA-013, and this note.

## What went well

- **The census caught what review missed,** three times: the `reply` regression in the session stack, the DP-1547 capture (found by a census run for an unrelated change), and 31 trailing-space divergences the normalizer had hidden (PF-122).
- **Class audits (R5c) kept finding the next bug:** Hydrogen's trim measurement, DeepSeek's mana and dream findings, and Luna's peek gate.
- **Agents learned inside the weekend.** After a lesson was written into its prompt, DeepSeek stopped making the manifest mistake and started giving issue ids.
- **"Matches C" was an acceptable answer.** Agents reported C-parity (DP-1388, DP-1313's reset cap) instead of inventing fixes, and Zach ruled on the one that mattered.

## What went badly

- **The orchestrator approved a regression** (DP-1547). It assumed where a Go call went instead of tracing it, which is the R5e failure in the reviewer.
- **Two prompts pointed agents the wrong way:** DP-1313's prompt aimed at a case the issue had already ruled out, and the first triage prompt said `gh` when the agent had Linear access. Both were prompt errors.
- **Process gaps:** Claude broke a manifest row by skipping the gate it enforced on everyone else. DeepSeek hit its monthly limit mid-task. Luna sat idle overnight between prompts while Zach slept.
- **The tracker had rotted.** A third of the backlog no longer stood, and effort went into re-proving known cases before anyone triaged.

## Proposals

- **Candidate rule, for Zach to approve or reword:** "A change to a command registry or dispatch path runs a census that includes every scenario whose mobs call those commands directly from a special (R5e)." Incident: DP-1547.
- **Process (Claude):** reread the issue before writing a prompt for it, trace any call path before approving it, and triage before fixing when a batch of issues is more than a few weeks old.
