---
title: "The Oracle"
date: 2026-08-15
description: "A 30-year-old MUD, ported to Go one byte at a time. The bug that ended up mattering most was one no player could ever have seen."
draft: true
textKind: "original"
source: "Dark Pawns repository (src/, PR #469, docs/fidelity/RULEBOOK.md) and Zach's account of the port"
voiceLayer: "mythic-admin"
---

## Two dice nobody rolled

In July, my combat tests started lying to me. Not failing. Lying.

To port Dark Pawns from its 1990s C source to Go without changing the game, I run both servers side by side and compare everything they say. Every message, every hit, every miss, byte for byte. I've started calling that harness the oracle. When the two servers disagree, the scenario turns red, and I fix the Go side until it doesn't.

One week, three combat skills went red for no reason that I could point at. Bash, trip, and headbutt: the C server hit, the Go port missed. I had not touched them. Meanwhile kick and backstab, the same kind of skill with the same kind of rolls, sat there green and innocent.

Here is what was actually wrong. When the very first character logged in, the Go port ran its level-up routine once when the C server would not have. (MUD convention going back to DikuMUD: the first player to ever create a character becomes the implementor, a god, level 40 on a scale where mortals stop at 30. So the very first character starts the game already at max level.) And level-up rolls dice: a warrior draws `number(11, 14)` for hit points and `number(1, 4)` for movement (`src/class.c`, the same lines the C server has). Two random numbers, drawn at the moment of birth, that the C server never draws, because the C code checks the character's level first (`src/interpreter.c:2213`) and skips the routine entirely for a character that already has one. The port skipped the check.

Two extra draws. That's the whole bug. But every random number after that comes from one shared stream, so from that instant the Go server's entire universe ran two rolls ahead of the original:

| Event | C server | Go port |
|---|---|---|
| Character creation | draws #1, #2 | draws #1, #2 (phantom), #3, #4 |
| First combat swing | draw #3 | draw #5 |
| Trip attempt | draw #4 | draw #6 |
| ...every roll, forever | ... | always two ahead |

Every hit, every save, every skill check after that pulled the number that belonged to a different event. The port did not crash. It did not stutter. It deterministically produced a self-consistent, subtly wrong universe, and it would have kept doing so forever.

## Why no player could ever catch this

The natural question: isn't this what beta testers are for? No, and the reasons are worth spelling out, because they apply to way more than MUDs.

**The medium has no surface.** A random number stream is invisible. Nothing in the game ever displays "this was draw #47." A player has no reference stream in their head to compare against. You cannot see an off-by-two in a thing that is never shown.

**The cause and the symptom live in different worlds.** The cause is two stray dice at character creation. The symptom is a trip attack missing much later, in a fight that had every reason to go the other way. They are separated by an entire play session and dozens of function calls. Even a tester who smelled something wrong could not bisect their way from "my trip missed" back to "the constructor rolled two extra numbers at birth."

**It was consistent, not flaky.** Flaky gets noticed. This was a port that worked, every time, the same way. Consistent wrongness is indistinguishable from correctness unless you have the original running next to it. You could play that character for six hours and never know. The game would be fair, fun, and not quite Dark Pawns.

And one more indignity: half the evidence said nothing was wrong. Bash, trip, and headbutt flipped because their rolls landed on the other side of a threshold. Kick and backstab drew equally wrong numbers, but their outcomes happened to land the same anyway. Green by coincidence. If I had been testing outcomes instead of tracing draws, I would have fixed three skills, declared victory, and shipped a port with a permanently desynchronized universe.

## Where the process came from

I did not invent any of this. Anthropic published [a post about running large-scale code migrations with AI](https://claude.com/blog/ai-code-migration): Bun's runtime ported from Zig to Rust, roughly a million lines, in under two weeks. The part that stuck with me was not the speed. It was one line of philosophy: *you don't fix the code, you fix the process that produced the code.* And the prerequisite they named before any translation happens: build a judge first, a harness that can run the same tests against both the old and new codebase, so that "done" is a mechanical fact instead of a feeling.

Their migrations justify themselves with boardroom numbers. Dark Pawns has no boardroom. It has a dead game, an old tarball of C, and one rule I set for myself before writing a line of Go: the game is the game. Whatever the port did, a player from 2004 logging in today should get 1999's bytes back, not a tribute act. That constraint is stricter than any migration benchmark, because it includes the dice.

So I stole the methodology and aimed it at a MUD. The judge came first.

## The judge

The oracle is a command in the repo, `dp-oracle-diff`. It boots the original C server (unmodified, built straight from the historical source) and the Go port as black boxes on local ports, then drives both with the exact same script: a plain text file of telnet lines. Create a character. Walk north. Trip the goblin. Every scenario file is a replayable ghost of a play session.

Both servers answer, and the harness compares their transcripts, byte for byte, with a small allowlist for genuinely unmatchable noise. When they agree, you get one line: `result: no normalized divergence`. When they do not:

```
Dark Pawns Tier-1 differential report
scenario: combat-trip-opener
c-oracle: 127.0.0.1:41255 (DP_SEED=1)
go-port:  127.0.0.1:41287 (DP_SEED=1)
result: normalized divergence detected

--- [trip goblin] c-oracle
+++ [trip goblin] go-port
```

Below that header: a unified diff. The C server's hit, the Go port's miss, line by line.

Two details make the comparison mean anything at all. Both servers run with a fixed random seed, and the Go server routes every game clock tick through a seam the harness controls, so the same scenario always produces the same session. Without that, a red could just be luck, and so could a green.

Which is exactly what the phantom draws were hiding. When the reds finally made me trace the draws instead of reading the diffs, the pattern fell out in minutes: every opener in the suite was pulling from a stream sitting two positions ahead of C's. One fix, in the character constructor, turned the whole family green at once. Bash, trip, headbutt aligned; kick and backstab stayed green, but now truly aligned, not by luck.

The fix took an afternoon. The lesson took a rule: a green result is not proof. For anything involving a die roll, the oracle checks the numbers behind the verdict too, because a pass can be a coincidence wearing a pass's clothes.

## The rulebook

The oracle catches a lie, but what stops the lie from coming back? Their answer, and now mine: a rulebook. Every translation decision gets made once, written down, and enforced on every future change. Mine lives in the repo as five rules, R1 through R5, and the whole spirit of it is one line:

> A rule without an incident citation is a guess; don't add it.

Rules are not written in advance, in a design doc, by someone who has never been burned. They are scar tissue. The phantom draws did not earn their own rule by being clever. They earned it by rhyming: two different bugs, months apart, both turned out to be the same shape, an extra pair of draws at the wrong moment nudging the whole stream out of step. After the second one, the rulebook got a new entry and the whole port got an audit. The law of the port: the second time a bug has the same root cause, you stop fixing files. You amend the rule and then audit the entire class of code that could make the same mistake, because one confirmed instance means siblings exist.

The book keeps an amendment log with dates and incident numbers, so you can read the port's history as a list of lessons, each one paid for. It is the most useful document in the repository and it is also the shortest.

The first time the oracle went red, a lightbulb went off. Months of squinting at Go code going "this looks right" collapsed into a machine that could simply tell me no. The feeling was relief more than triumph: the port finally had an engine driving it forward instead of my patience.

## Not done

The port is not finished, and I want to be honest about where it stands. What the oracle has covered so far is breadth: a scenario walking through every major system, every command, every room of the creation flow, compared against the original. The next phase is depth, holding the two servers in lockstep through long sessions until the divergence rate is zero and stays there.

But the Go port is what's live right now, and every week the oracle catches something no playtester ever could. That is the deal I made with it. It shows me the exact byte where two universes split, and I fix the fork instead of the symptom.

Dark Pawns ran from 1994 to about 2010. Its dice are still rolling the same numbers.

---

*Connect via telnet at `darkpawns.labz0rz.com 7777` or play directly in your browser at [/play](/play/). The port is open source on [GitHub](https://github.com/zax0rz/darkpawns).*
