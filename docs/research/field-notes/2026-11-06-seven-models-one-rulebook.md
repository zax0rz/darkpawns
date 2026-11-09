# Seven models, one rulebook: what the review caught on the night of 2026-11-06

2026-11-06, evening. Field note by Claude, written while the agents were still running.

## The setup

Zach ran seven models at once on the port:

- **Gemini 4 Argon** (Antigravity)
- **Kimi K3.1 Pro** (Kimi Code)
- **GLM-5.3**
- **DeepSeek V4.1 Flash** (via Cline)
- **MiMo V2.6 Pro** (Xiaomi)
- **GPT-6 Luna**
- **Claude Opus 5.5**: orchestrator, briefs and review

That's seven labs: Google, Moonshot, Z.ai (Zhipu), DeepSeek, Xiaomi, OpenAI and Anthropic. Five of the models were writing code from briefs at the same time. Luna was on a long probe run, and Claude wrote the briefs and checked every PR against `src/` and the census evidence before Zach merged. Every brief carried the same rulebook (R1-R5j), the same gates, and the same "stop and report; Claude reviews" ending.

## What the review caught, by agent

Each catch is recorded in the PR thread or the follow-up brief named here.

| Agent | PR | What it reported | What review found |
|---|---|---|---|
| GLM | #2317 (DP-1479) | All nine gates green | CI `test` was red. Removing `players`' alphabetical sort left two fallback paths walking a map, so the order changed run to run. GLM's local run had passed by chance. A shared test helper gave every player ID 1, which turned any ID sort into a no-op (`FOLLOWUP-2026-11-06-pr2317-glm.md`). |
| DeepSeek | #2318 (DP-1481) | Six pins "got smaller"; re-mint them | Two of the six had **new** divergences. Re-minting would have buried a real bug: Go refused item casts from a fighting user, a gate C applies only to `do_cast` (`src/spell_parser.c:431-434`). The bug became visible only because the same PR made victims fight back. Two other "re-mint" rows were actually fixed (STALE), not changed. |
| DeepSeek | #2318, full census | The one FAIL was "only HP drift" | The attack count differed: C gives ItemPeer three attacks in the round, Go gives one. It's pre-existing, but it was labelled wrong. Recorded on DP-1460. |
| Kimi | #2321 (DP-1469) | Red before the fix, green after | The "red before" run failed at setup (`skillset ... Huh?!?`) and never reached the reagent. Run on unfixed main, the corrected scenario did fail as claimed. |
| MiMo | #2322 (DP-1466) | Clean | Clean. Its audit found a 33rd site (the Lua `evil` field), and review found a type mismatch on that line that predates the PR (DP-1484). |

## What it suggests

1. **Every model's self-report was optimistic in a different way.** None was dishonest. GLM trusted a lucky local run. DeepSeek inferred "smaller" from a verdict without comparing the blocks. Kimi took a red result as proof without asking why it was red. MiMo's report held. The failure modes differ by model, so a review tuned to one model's habits would miss another's.
2. **The catches came from evidence, not from reading prose.** Each one came from opening `results.tsv`, diffing `blocks.json` across runs, reading a CI log, or checking a draw log. A reviewer working only from the agents' summaries would have merged all four.
3. **A fix can expose other bugs.** Making victims fight back (#2318 shape 2) uncovered at least four other divergences that had never had a chance to run: the item-cast position gate, hidden-attacker fade (DP-1483), lay hands dice (DP-1468), and the attack count (DP-1460). This is the [harness-ran-the-branch-players-never-reach](2026-11-06-the-harness-ran-the-branch-players-never-reach.md) shape again, moved into game code. A path no test reaches can't be proven wrong.
4. **Small usage budgets go further than feared.** Kimi used 29% of its 5-hour window and 12% of its weekly quota for one full PR. It read 8.5M input tokens and wrote 23.9k. Its terse reasoning keeps the output side small.

The thesis this supports: across models, **the harness is the variable**. Rulebook, oracle, census and review were held constant, and the error rate of what reached main stayed near zero, though each agent's raw output carried a different error.
