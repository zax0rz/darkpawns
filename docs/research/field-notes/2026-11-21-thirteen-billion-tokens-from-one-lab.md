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
