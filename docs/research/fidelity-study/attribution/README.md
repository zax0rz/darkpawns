# Transcript attribution, v2 (2026-11-03)

Which model wrote the early mistakes that git can't attribute. DeepSeek built it from [`../BRIEF-2026-11-02-transcript-attribution.md`](../BRIEF-2026-11-02-transcript-attribution.md) using the surviving OpenClaw transcripts (PF-098); Claude reviewed it in Zach's session.

## Result

- **Target:** 69 slop-track rows with unknown authors and a birth commit before 2026-05-25: 48 in the NUC window (2026-04-22 to 05-04), 5 in the mac-mini window (05-05 to 05-24), 16 before any surviving transcript.
- **7 mistakes attributed:** the matched line is the mistake itself, and each fix commit removes it. deepseek-chat wrote 3 (G044, S068, S083) and glm-5-turbo wrote 4 (G077, S042, S057, S078).
- **9 "function created by" leads:** the session created the function the mistake lives in, which doesn't show it wrote the mistake.
- **2 generic matches**, not attributions.
- **The shipped yank (S001) stays unattributed.** deepseek-chat wrote an earlier, different paraphrase. No surviving write contains the committed text ("Yank whom from what?", "They aren't following you!").
- **CLAUDE.md:** 1 of the 18 matched sessions read it.

Report it as "where transcripts survive, mistakes came from both the GLM orchestrator and the DeepSeek workers", not as a tally: most sessions didn't survive, and agents rewrote each other's code between sessions.

## What the review checked

- The brief's checker passes: every birth commit resolves, and every matched line is in its birth commit.
- Each of the 7 `mistake-line` matches is a line its fix commit removes.
- The control reproduces PF-098's session for the first `doYank` (`8e4697db`, deepseek-chat, 2026-04-24 22:42 UTC), and shows that revision differs from the committed one.
- The files hold metadata and short code lines only; no transcript text.

## Privacy

The transcripts are private (they include personal conversations) and stay in `~/Archives/openclaw-nuc-2026-11-02/` and on `mac-mini`; neither is in any repository. `extract.py` emits only code written to Dark Pawns files, with session, time and model. Its paths point at those private locations.

## Files

`attribution.tsv` (one row per target; `match_strength` is column 15), `attribution-notes.md` (DeepSeek's analysis), `control.md` (the S001 control and method), `extract.py` and `assemble2.py` (DeepSeek's scripts). Source: `~/dp-research/attribution/v2/` (`attribution.tsv` sha256 `e05516774ebb894fabf5225743118bb863715ec2cefe28144e0d34d4f4df926c`).
