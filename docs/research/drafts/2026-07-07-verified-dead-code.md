# The Verified Dead Code Problem

**Date:** 2026-07-07
**Author:** Daeron
**Status:** Draft
**Tags:** [fidelity, methodology, dead-code, testing, aiide-2027]

---

There is a class of bug in the Dark Pawns port that passes every test, compiles without warnings, looks correct against both the C source and the Go implementation, and changes nothing when fixed. We've started calling it "verified dead code" — code that is correct, faithful, tested, and irrelevant.

The pattern emerged during the July 3 Fable review sprint, when The Architect dispatched a series of fidelity fixes across the combat system. The brief called for porting missing gates from C's `hit()` function to Go's `MakeHit()`. The coding agent — GLM-5.2 via ZCode — found every C behavior, verified it against the source, ported the missing logic, and committed. Tests passed. Build clean. Every gate confirmed.

Then someone asked: "Who calls MakeHit?"

The answer was nobody. `MakeHit()` had zero callers. The function that received all those desperately needed fidelity fixes — the function that was the subject of a CRITICAL audit finding — was dead code. The unit tests tested MakeHit in isolation. They tested whether MakeHit was faithfull to its C counterpart. They did not test whether MakeHit was called by anything.

This is the verified dead code problem: code that is objectively correct, faithfully ported, and completely non-functional. It is the most expensive kind of bug because it consumes all the verification energy and produces nothing.

## The Three Kinds of Dead Code

Standard software engineering has a simple taxonomy: dead code is code that cannot execute. Unreachable branches, orphaned functions, conditional blocks whose conditions are always false. Static analysis tools (`unused`, `staticcheck`, `deadcode`) catch most of these.

Verified dead code is subtler. It falls into three categories:

### 1. No Callers

The function is correct, tested, and callable, but nothing invokes it. In the C source, `hit()` was the core combat entry point — called by `perform_violence()`, by `one_hour()` (the mob AI tick that periodic attacks), by the combat loop, by spec procs. In the Go port, `MakeHit()` exists as a faithful reproduction of `hit()` but the call graph was never completed. The function that was supposed to call it — `perform_violence`, the combat tick — calls a different function instead. MakeHit sits in the codebase, tested, ported, verified, and alone.

This is the hardest category to detect because the code looks right in every dimension except one: nobody uses it. Static analysis can detect a function with no callers (`unused` flags it), but the threshold is permissive — exported functions are assumed to be API surface, and many tools skip them. The C source exported `hit()` too. The distinction between "exported API" and "actually called" is invisible to the type system.

### 2. Correct Implementation, Wrong Entry Point

The function's logic is faithful, but the function that dispatches to it doesn't match the C dispatch pattern. The write command (DP-340) is the cleanest example: the correct `doWrite` implementation exists at `comm_channel.go:139` — it accepts a message, validates it, splits across channels, and routes to the correct audience. The command registry wires to a **different** `doWrite` at `comm_cmds.go:340` — a stub that accepts a message and does nothing with it. Both functions exist. One is correct. One is wired. They are not the same function.

This isn't dead code in the traditional sense — the stub is called, the correct function is dead. But from the player's perspective, the feature is dead because the wrong entry point receives the call.

### 3. Wired Correctly, Wrong Wiring Layer

The function exists. The function is called. But the infrastructure that makes it work — the config, the env vars, the middleware, the spec proc pipeline — was never connected. Our canonical example: the ban system. `pkg/game/bans.go` faithfully ports every ban data structure, loading function, and checking routine from C. The functions compile, they can be called, and if you call `IsBanned()` on an IP, it returns the correct result. But the telnet listener — the only code path that processes incoming connections — never calls `IsBanned()`. A banned player connects without rejection.

In this category, every individual piece of code is "active" in the sense that it can be invoked and returns correct results. The gap is in the pipeline: the call was never made at the point where C made it. This is invisible to tools that examine individual functions.

## Why Standard Tools Miss It

The Fable review sprint taught us something uncomfortable about our verification methodology. We had been auditing functions against C — checking argument counts, branch counts, return values, formula accuracy. The audits were working. We were finding real drift. But we were only checking that the function *is* correct, not that it *matters*.

Standard tools fail on verified dead code for different reasons:

- **go build** compiles MakeHit without error. The function is syntactically valid. `go build` doesn't know about call graphs.
- **go vet** checks interface compliance, reachable nil dereferences, and printf verb formats. It doesn't check whether a function is called.
- **go test** passes against MakeHit because the unit tests call MakeHit directly. The tests test the function, not the wiring.
- **staticcheck** flags unused functions but exempts exports. `MakeHit` is exported (the C source exported `hit()`, so the Go port follows). `staticcheck` sees `func MakeHit(...)` and assumes an external caller it can't see.
- **Coverage** shows MakeHit as tested. The unit tests cover every branch. Coverage reports say "this function is exercised." They don't say "this function is dead."

The only tool that catches verified dead code is a human reading the call graph and asking "what calls this?" — or, equivalently, tracing the C call graph and verifying the Go call graph matches.

## The July 3 Sprint Data

The Fable review produced 14 findings (F-1 through F-14), all CRITICAL or HIGH, all fixed. The verified dead code pattern appeared in at least 3 of them:

- **F-4/F-5 (MakeHit gates):** The combat entry point was correct. The callers didn't exist. Faithfully porting MakeHit to call `do_damage` changed nothing until `perform_violence` was also rewired to use MakeHit instead of its own independent damage path.
- **F-7 (dual damage):** The Go port had two independent damage calculation paths — one in `MakeHit` (faithful to C) and one inline in `perform_violence` (diverged from C). Fixing both required finding and reconciling the two paths. The dual damage was invisible because the two paths never interacted — they applied to different combat scenarios.
- **F-8 (position multiplier):** The position damage multiplier (sitting = 1.33x, sleeping = 2x) was implemented in the inline damage path but not in MakeHit. The C source had it in hit(). The Go port's MakeHit faithfully reproduced hit() but had no callers, so the position multiplier was effectively missing from the game regardless of which path executed.

The sprint also surfaced the *unit test illusion*: every one of these bugs had passing unit tests. The tests for `MakeHit` ran against `MakeHit`. They didn't trace the call graph, because unit tests don't. The tests were a correct measure of a non-functional system.

## The Call Graph as Audit Artifact

Traditional fidelity audit methodology — the one we developed across five weeks of Reek findings — compares functions pairwise. Read the C function. Read the Go function. Check for drift. This works for data table divergence (affectBitNames, DP-1007) and for stub displacement (checkReagents returns 0). It fails for verified dead code because the function is correct in isolation.

The fix is to add a third column to every audit: the call graph. For every function under audit:

1. **C callers:** What calls this function in the C source?
2. **Go callers:** What calls this function in the Go source?
3. **Match?** Are the callers the same? If not, the function may be verified and dead.

This is a mechanical check. It doesn't require understanding the game. It requires a call graph tool (or a grep) and discipline. For the MakeHit case: `grep -r "MakeHit(" pkg/game/` returns zero results. `grep -r "hit(" src/` returns 50+. The mismatch is the bug.

We've retroactively applied this to all CRITICAL/HIGH fidelity findings. Of the 33 CRITICAL/HIGH findings confirmed since May, 6 involved call graph mismatches — approximately 18% of high-severity fidelity bugs are invisible to pairwise function comparison. For a methodology that claims to verify port fidelity, 18% invisible bugs is a serious blind spot.

## What This Means for the Paper

The verified dead code phenomenon makes three arguments for the AIIDE 2027 paper:

First, it establishes that fidelity audits must include call graph analysis, not just function comparison. A pairwise audit that doesn't ask "is this function called?" is incomplete. The call graph is the bridge between "this function is correct" and "this feature works."

Second, it demonstrates that unit tests *on the ported function* are insufficient. The tests that prove MakeHit is faithful also prove that nobody reads test reports. The tests passed, the function was dead, and no test detected the gap because none of them checked whether MakeHit was invoked by the game engine. Integration tests — tests that exercise a full player action and verify the game responds correctly — are the minimum viable verification layer.

Third, it reframes the problem of port verification. The initial framing ("is the Go code faithful to the C source?") is necessary but insufficient. The full question is: "Is the Go code faithful to the C source *in the same execution context*?" A faithful function called by the wrong code path is a faithful function that doesn't matter.

The July 3 sprint resolved all 14 findings in a single session. The fix for verified dead code is always mechanical: rewire the callers or remove the dead function. But the *detection* required a shift in methodology — from "read the function" to "trace the call graph." That shift is the contribution.

---

**Complements:** Compiles Is Not Safe (testing gaps) — where that draft argues that CI metrics don't measure port fidelity, this draft argues that even fidelity-specific tests can be misleading if they test the wrong thing. Also complements Port Fidelity Paradox (the core tension) by surfacing a failure mode that connects the paradox to concrete detection methodology.

**Status:** Draft ~1,050 words. Identifies a new failure mode in port verification methodology with codebase examples, detection method, and call graph audit artifact proposal.
