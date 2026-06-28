# The Taxonomy of Simplification

**Date:** 2026-06-23
**Author:** Daeron
**Status:** Draft
**Tags:** [fidelity, methodology, simplification, drift, aiide-2027]

---

There are fifteen occurrences of the word "simplified" in the Go source of Dark Pawns. Fifteen places where a developer — human or machine — looked at the C original, decided the full behavior was too much trouble, and wrote a comment acknowledging the gap. The comment is the important part. It transforms a regression into a design choice.

"Simplified: steal gold or an item from target's inventory." That's `skill_stealth.go:66`. The C source at `act.other.c:309-560` handles stealing from containers, stealing from equipment slots, weight checks against the thief's capacity, skill improvement on failure, mob memory of theft attempts, and a witness flag that propagates to surrounding rooms. The Go version steals gold or a single inventory item. Everything else is gone. The comment says "simplified" as though the missing behavior were a stylistic choice rather than a 250-line hole in the stealth system.

This paper argues that "simplified" is the most dangerous word in a ported codebase. It's a rhetorical move that transforms a behavioral regression into an engineering decision. And it's invisible — not because the code hides, but because the framing makes the gap look intentional.

## The Five Simplifications

Across five weeks of fidelity audits (220 confirmed findings, 30% classified as fidelity drift), we've identified five distinct patterns of simplification. Each is mechanically detectable. Each has a different cost. Each is hidden by the same rhetorical gesture: a comment that says "simplified" instead of "broken."

### 1. Argument Truncation

The C function takes eight arguments. The Go function takes five. The missing parameters control edge cases — the uncommon inputs, the failure paths, the things that happen one time in a hundred. The common case works perfectly. The edge case breaks silently.

**Example:** `DoMindlink` in `skills2.go:208`. The C source (`new_cmds2.c`) accepts a target and a mana transfer amount. The Go version accepts a target but the mana transfer uses a type assertion that always fails on mob targets — `target.(*Player)` panics or returns nil when the target is a `*MobInstance`. The function signature looks complete. The behavior is truncated.

**Detection:** Count function arguments. If the Go function takes fewer parameters than its C counterpart, check what the missing arguments control. If they control behavior (not just context), the simplification is a regression.

**Cost:** Edge cases. The 1% case that nobody tests because the 99% case works.

### 2. Logic Flattening

The C function has four branches — an if/else chain handling four distinct cases. The Go function has one branch that handles the common case and falls through to a default for everything else. The default isn't wrong, exactly — it's just not what the C code would have done.

**Example:** `persName` in `act.go:111`. The comment reads "Uses simplified CAN_SEE (awake check only) for now." The C `CAN_SEE` macro checks awake status, invisibility, detect-invis, hide, sense-life, blind, and god-mode visibility. The Go version checks one thing: is the observer awake? If yes, you can see them. An invisible thief standing in front of a sleeping guard is visible the moment the guard wakes up. The full visibility matrix — the one that makes stealth gameplay work — is gone.

This was eventually fixed (the `canSee` function at `act.go:125` now has the full matrix), but the "simplified" version ran in production for weeks. The comment made it look like a known limitation rather than a broken game mechanic.

**Detection:** Count branches. If the Go function has fewer branches than its C counterpart, check what each missing branch handles. If any missing branch changes game state (not just logging or error messages), the flattening is a regression.

**Cost:** The uncommon case silently takes the wrong path. Not a crash — a wrong answer.

### 3. Stub Displacement

The function exists. It has the right name, the right signature, the right comment explaining what it does. The body returns nil. Or returns 0. Or returns a hardcoded default. The function compiles. Tests that call it pass (because the test doesn't check the return value, or because the test was written against the stub). The stub is indistinguishable from a real implementation unless you read the body.

**Example:** `checkReagents()` in the spell system. Returns 0 permanently. Every spell that checks reagents — components that consume inventory items on cast — skips the check. Mages don't need components. The spell system works. It just doesn't work the way the game design intended.

We found 22 stub functions across the port. Some were intentional placeholders. Some were functions that the porting agent started, got compiling, and never went back to fill in. The code doesn't distinguish between the two — a stub that was always meant to be a stub looks identical to a stub that was supposed to be a real function.

**Detection:** Search for functions that return a constant (nil, 0, false, empty string) and have no side effects. Cross-reference against the C source: does the C function do something? If yes, the stub is a displacement, not a placeholder.

**Cost:** The feature exists in the API but not in the behavior. Callers think they're invoking a real function. The game runs as though the feature doesn't exist.

### 4. Algorithmic Substitution

The Go code computes the same result as the C code, but uses a different formula. The formulas match at common inputs — the typical case, the normal range, the values that appear in testing. They diverge at extremes: high levels, low stats, edge conditions. The substitution looks like an improvement (simpler code, fewer branches, more "Go-idiomatic") until someone hits the extreme case.

**Example:** `exp_needed_for_level` in `act_display.go:54`. The comment reads "estimated as 1000*level (simplified)." The C source uses a quadratic formula that scales experience requirements exponentially. The Go version uses a linear estimate. At level 1, the difference is negligible. At level 30, the C formula requires 900,000 XP; the Go formula requires 30,000. A level 30 character on the Go server levels thirty times faster than on the C server. The economy is broken, but only at high levels — the levels that take the longest to reach and are tested the least.

**Detection:** Compare formulas. If the Go code uses a different mathematical expression than the C source, evaluate both at the extremes of the input range (minimum stat, maximum stat, level 1, level 50). If the results diverge by more than 5%, the substitution is a regression.

**Cost:** The game works, but it works differently. Players who reach the extremes discover a different game than the one the C server ran.

### 5. Behavioral Omission

The C function does three things: X, Y, Z. The Go function does X and Y. Z is just missing. Not stubbed. Not commented as "TODO." Not replaced with a default. Just absent. The function doesn't mention Z. The comment doesn't mention Z. If you don't read the C source, you don't know Z existed.

**Example:** The ban system. The C source checks `isbanned()` in the connection handler — every incoming telnet connection is checked against the ban list before the login prompt. The Go ban system is faithfully ported: `pkg/game/bans.go` has the data structures, the ban list loading, the ban checking logic. It's complete. It's also dead code — the telnet listener in `pkg/telnet/` never calls `IsBanned()`. The ban system was ported but never wired to the connection path. A banned player connects without interruption.

Another example: spec procedures. The C command interpreter checks for mob/room/object spec procs before executing any player command. The Go command dispatcher skipped this check entirely. The spec proc functions existed. The spec proc assignments existed. The wiring — the single architectural step that makes the whole system work — was missing. Twelve features were dead behind this one omission.

**Detection:** This is the hardest pattern to catch mechanically. The function looks complete in isolation. You only find the omission by tracing the call path: who calls this function? Does the call happen in the same places as the C source? If the C source calls function Z in the connection handler and the Go source doesn't call it anywhere, Z is a behavioral omission.

**Cost:** The feature exists in the codebase but is never executed. It's the most expensive kind of simplification because it's invisible to every tool that doesn't trace the full call graph.

## Why "Simplified" Is the Problem

Each of these five patterns shares a feature: a comment that frames the gap as a choice. "Simplified version." "Simplified for now." "Simplified: steal gold or an item." The comment does rhetorical work — it acknowledges the gap while neutralizing it. A stub without a comment looks like a bug. A stub with a "simplified" comment looks like an engineering decision.

This matters for AI-assisted porting because language models are comment-followers. When a model encounters a "simplified" comment, it treats the simplification as intentional. It doesn't flag the gap. It doesn't check the C source. It moves on. The comment becomes a license to skip verification.

The fidelity audit's job is to strip this license. Every "simplified" comment is a claim — a claim that the simplification is acceptable. The audit tests the claim against the C source. Sometimes the simplification is acceptable (the missing behavior is cosmetic, or the edge case is unreachable). Often it isn't. The fifteen "simplified" comments in the Dark Pawns codebase represent fifteen claims that deserve verification. We've verified eight so far. Five were regressions.

## The Mechanical Checks

Five patterns, five checks:

1. **Argument count.** Go function takes fewer args than C counterpart → check what the missing args control.
2. **Branch count.** Go function has fewer branches → check what each missing branch handles.
3. **Return defaults.** Function returns a constant → check if the C version computes.
4. **Formula comparison.** Different math → evaluate at input extremes.
5. **Call tracing.** Function exists but isn't called where C calls it → behavioral omission.

These checks are automatable. They don't require understanding the game. They require reading two codebases and counting.

## What This Means for the Paper

The Silent Drift draft (May 12) identified three categories of port drift: data table divergence, stub functions, and logic simplification. This draft decomposes the third category into five specific mechanisms, each with a detection method and a cost model.

The decomposition matters because "logic simplification" is too broad to act on. You can't write a brief that says "find all logic simplifications." You can write a brief that says "compare argument counts between C and Go versions of these 47 functions" or "trace the call path for every function that returns nil and verify it's called in the same places as the C source." The taxonomy turns a vague concern into a set of mechanical checks.

It also explains why the word "simplified" appears in the codebase fifteen times. The porting agents — human and machine — used the word as a pressure valve. When the full behavior was too complex to port in the available time, "simplified" acknowledged the gap without treating it as a failure. This is understandable. It's also the mechanism by which drift becomes invisible.

The fidelity audit exists to make the invisible visible again. Not by removing the comments — they're honest about what happened — but by testing whether the simplification was a choice or a cost.

---

**Complements:** Silent Drift (data taxonomy), Port Fidelity Paradox (the core tension), Constraint Engineering (brief methodology), What the Agent Preserved (thesis). Where Silent Drift categorizes *what* drifts, this draft explains *how* it drifts — the specific mechanisms by which behavioral gaps enter a port and hide behind comments.

**Status:** Draft ~1,100 words. Five patterns defined with codebase examples. Extends the Silent Drift taxonomy with the specific mechanisms that produce fidelity gaps. Supports Constraint Engineering (what briefs need to detect) and What the Agent Preserved (what agents lose).
