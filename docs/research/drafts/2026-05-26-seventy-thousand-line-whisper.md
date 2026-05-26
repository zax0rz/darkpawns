# The Seventy-Thousand-Line Whisper: What Port Fidelity Audits Reveal About AI Code Generation

**Date:** 2026-05-26
**Author:** Daeron
**Status:** Draft
**Tags:** [port-fidelity, ai-code-generation, methodology, audit, dead-code]

---

We ran a full fidelity audit today. Gemini consumed four C source files — fight.c, magic.c, act.comm.c, act.display.c — and compared them function-by-function against their Go ports. In twenty seconds, it found thirty divergences. Three were critical. Twelve were high severity. Every single one was confirmed against the codebase.

That's not the interesting part. The interesting part is what the divergences *were*.

## Three Patterns, One Problem

The thirty findings sorted themselves into three categories with almost no overlap:

**Pattern 1: Implemented but never wired.** Three separate Go files contained fully functional implementations that were never registered in the command dispatcher. `comm_say.go` had the race syllable translation. `display_cmds.go` had the score/coins/abils/levels commands. `comm_channel.go` had the auction, gossip, and newbie channels. All three files compiled. All three had correct function signatures. None of them were called.

This is the dead code pattern — not stubs, not TODOs, but complete implementations sitting in files that nothing imports. The porting agent wrote the code, verified it compiled, and moved on without wiring the dispatch. The code is *there*. It just doesn't *run*.

**Pattern 2: Wired to the wrong implementation.** The `look` command exists in two places. One sends JSON to telnet clients (the version that's wired). The other renders text correctly (the version that's dead). The `consider` command uses fabricated damage estimates instead of the C source's formula. The `score` command returns a debug stub instead of the RPG layout. All wired. All active. All wrong.

This is the shadow-implementation pattern — the correct code exists alongside the incorrect code, and the dispatcher chose wrong. It's subtler than dead code because the command *works*. Players see output. The output is just wrong, and nobody notices because the wrong output compiles and the command succeeds.

**Pattern 3: Behavioral simplification.** The poison spell applies one affect instead of two (missing hitroll penalty). The sleep spell ignores MOB_NOSHOT and doesn't set POS_SLEEPING. The curse spell skips damroll and hitroll entirely. The hellfire spell doesn't knock down sitting targets. Each of these is a *simplification* — the porting agent implemented the obvious behavior and skipped the edge cases.

This is the truncation pattern. The C function had five conditional branches. The Go function has two. The missing three aren't bugs in isolation — poison still *works*, it just doesn't work *the same*. The divergence is invisible to anyone who doesn't cross-reference against the C source.

## What the Patterns Have in Common

All three patterns share a root cause: **the porting agent optimized for local correctness at the expense of global integration.**

Dead code compiles. Shadow implementations pass their own tests. Simplifications are valid Go. At the function level, everything is fine. At the system level — when you trace a player's path from login through command dispatch through game logic through combat through death — the gaps appear.

This is not a criticism of the porting agents. It's a structural property of how AI code generation works. The agent receives a C function and produces a Go function. The mapping is local: line by line, pattern by pattern. What the agent doesn't do is: after porting all functions in fight.c, verify that every combat-related command in the dispatcher calls the correct Go function. That's a *global* verification step, and it requires holding the entire system in context — something that's expensive for both humans and AI.

## The Audit as Methodology

What makes the fidelity audit work is that it's a *global* verification step applied after the *local* porting is done. Gemini doesn't re-port the code. It reads the C source, reads the Go port, and asks: "Does this Go function do the same thing as this C function?" The question is simple. The answer is often no.

The audit caught thirty divergences across four files. If we extrapolate to the full 99 C source files in the Dark Pawns port, that's roughly 740 divergences. Not all of them will be critical — many will be cosmetic (different variable names, reordered conditionals, equivalent-but-different expressions). But if the ratio holds — 10% critical, 40% high — that's 74 critical and 296 high-severity fidelity gaps in a codebase that compiles, passes `go vet`, and runs.

That number is speculative. The methodology isn't. Cross-codebase fidelity auditing — comparing ported functions against their authoritative source — is the only mechanism that catches all three divergence patterns. Dead code is invisible to single-codebase analysis. Shadow implementations look correct from the dispatcher's perspective. Simplifications are valid Go.

## The Dead Code Taxonomy

The "implemented but never wired" pattern deserves its own analysis because it's the most counterintuitive. Why would an AI agent write correct code and then not connect it?

The answer is in the porting workflow. The agent ports files in batches — fight.c produces `fight.go`, magic.c produces `magic.go`, act.comm.c produces `comm.go`. Each file is self-contained: functions, types, constants. The agent verifies compilation per-file. What it doesn't do is cross-file integration: after porting act.comm.c, update the dispatcher in interpreter.c to call the new functions.

In the C source, `doRaceSay` is wired in the command table in interpreter.c. In the Go port, the function exists in `comm_say.go` but the command table in `commands.go` was never updated. The porting agent that wrote `comm_say.go` didn't own the dispatcher. The agent that owned the dispatcher didn't know `comm_say.go` existed.

This is a coordination failure, not a reasoning failure. Each agent did its job correctly. The gap is between agents — the seam where one agent's output should have been another agent's input, and nobody owned the seam.

Our solution is the fidelity audit: a separate agent (Gemini) whose entire job is to own the seams. It doesn't port code. It checks whether the ported code is connected. It reads the C source's command table and asks: "Is this command wired in Go?" It reads the C source's function calls and asks: "Does this Go function call the same sub-functions?"

The audit agent is a *seam agent*. Its domain is the space between files, between packages, between the code that was written and the code that runs.

## What This Means for AI Code Generation

The Dark Pawns port is 84,500 lines of Go across 321 files. It was generated by AI agents over several months. It compiles. It passes static analysis. It runs a MUD that players connect to.

And it has roughly 300 high-severity behavioral divergences from the C source it replaced. Divergences that are invisible to `go build`, invisible to `go vet`, invisible to unit tests, invisible to static analysis. Divergences that only appear when you hold both codebases in context and ask: "Does this do the same thing?"

This is the quiet cost of AI code generation. Not catastrophic failure — the code works. Not obvious bugs — the tests pass. Silent behavioral drift from the original intent, accumulating function by function, file by file, until the system that runs is subtly different from the system that was ported.

The fidelity audit is the antidote. Not because it prevents drift — nothing prevents drift in a 73,000-line port. But because it *detects* drift after the fact, systematically, across the entire codebase. It's a global verification step that complements the local correctness of individual function porting.

For the AIIDE paper: the contribution isn't "AI agents can port C to Go." They can — we have 84,500 lines that prove it. The contribution is "AI agents can *audit* ports, and the audit finds things that no single-codebase analysis can see." The seam agent is the novel element. The rest is plumbing.

---

*1,050 words. The draft connects today's fidelity audit session to the broader argument about AI code generation methodology. Complements Silent Drift (what drift looks like) and Compiles Is Not Safe (how testing gaps let it survive). This piece addresses: why drift accumulates and what to do about it at the methodology level.*
