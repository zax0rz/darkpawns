# Thirteen billion tokens from one lab

*2026-11-21, Claude. Ledger PF-155 (needs verification).*

Zach checked his z.ai account on 2026-11-21: **13.9 billion tokens** used since April 2026. That's one lab. He says not every token went to Dark Pawns, but the large majority did. The GLM models ran most of the port's volume: OpenClaw's NUC-era orchestrator was glm-5-turbo (PF-153), and GLM is still on the current roster.

The project also ran on Moonshot (Kimi), DeepSeek, Xiaomi (mimo), MiniMax, Google (Gemini), OpenAI (Codex, gpt-5.6-sol) and Anthropic (Claude). So the total is larger, and per-lab figures are still to be collected.

It ran on one person's budget, not a research grant. Models were picked as capacity appeared:
- free weekend tokens (glm-5.3-flash);
- a 50% promotion (Kimi k2.8-preview);
- usage resets (Codex).

Every result is checked against the C oracle, whichever model produced it, so cheaper capacity changes the cost and not the standard.

**Before this figure goes in a paper:**
- export the z.ai usage record, with dates and model breakdown if available;
- estimate the non-Dark Pawns share;
- collect the other labs' totals the same way.

Until then it's a reported figure, not a verified one.

## A method that doesn't need a clean split (added the same day)

Provider dashboards count every use on the account, and some don't show subscription usage at all; Anthropic's subscription has no per-project view. So Dark Pawns' share can't be carved out of a provider total. Report a range instead:

- **Upper bound:** the provider's account total for the period.
- **Lower bound:** tokens from agent logs whose session ran inside the Dark Pawns checkout or its worktrees. Claude Code logs carry per-message usage, the model, and the working directory. OpenClaw, Codex and other harness logs may do the same, and checking that is the next step.

**Claude's lower bound (PF-156).** The Claude Code logs for this project cover 2026-08-14 to 2026-11-21; older transcripts were pruned. They hold 23,209 messages:

| | Tokens |
|---|---:|
| cache reads | 8.91 B |
| cache writes | 0.13 B |
| output | 0.02 B |
| uncached input | about 0 |
| **total** | **9.06 B** |

By model, claude-opus-5-5 accounts for 7.25 B. **98% of the total is cache reads:** the same long context re-read each turn. Raw token totals mostly measure context length times turns, not work. Any figure in the paper should report output tokens and cache reads as separate columns, and should say which one a headline number is.
