# Constraint Engineering: How Structured Briefs Make LLM Code Review Work

**Date:** 2026-06-02
**Author:** Daeron
**Status:** Draft
**Tags:** [methodology, brief-design, constraint-engineering, telephone-game, aiide-2027]

---

On May 26th, Gemini consumed a structured brief and audited 73,000 lines of C against 211 Go files in under five minutes. It found 13 real gaps — type-assertion bugs, dead-code wiring issues, behavioral truncations. Every finding was confirmed against the source. Zero false positives.

On May 27th, Claude Code consumed a different brief and fixed 15 issues in a single pass. Shop pricing, spec proc registration, visibility matrices, steal mechanics. All verified against C source behavior. All clean.

On May 10th, Reek — our automated crawler — delivered three overnight reports covering spells, combat fidelity, and dependencies. 20 confirmed, 3 rejected. The combat fidelity audit traced `perform_violence` across five files in both C and Go, surfacing an architectural issue (dual hit-resolution paths) that no single-file analysis would catch.

Three different models. Three different tasks. One pattern: a human writes a brief, the model executes, a human verifies. We call it the telephone game. The brief is the telephone.

This paper is about the telephone, not the people on either end.

## The Problem with Unconstrained LLM Code Review

The standard approach to AI code review is: "here's a codebase, find bugs." This works about as well as asking a new hire to review a system they've never seen. The model generates plausible-sounding findings — nil checks, race conditions, error handling gaps — that are technically correct as observations but operationally useless. They're the bugs a linter would find if linters could write paragraphs.

The failure mode isn't hallucination. It's *drift*. The model has no anchor. It doesn't know what the code is supposed to do, so it can't tell you what's wrong. It can only tell you what's *there*. "This function doesn't check for nil" is an observation. "This function should check for nil because its C counterpart at `src/fight.c:347` does" is a finding.

The difference is context. Not general context — not "this is a MUD server written in Go" — but *specific* context: which files to examine, which patterns to flag, which source of truth to compare against, which severity taxonomy to apply. The kind of context that takes a human expert thirty seconds to specify and a model thirty minutes to discover on its own (if it discovers it at all).

## What a Brief Actually Does

A well-structured research brief doesn't just describe a task. It *constrains the search space*. Consider the fidelity audit brief I wrote for the May 26 session:

```
## Methodology
For each function in the C source:
1. Locate the corresponding Go function (search by name, then by comment reference)
2. Compare: argument count, return values, control flow, edge cases
3. Classify: IDENTICAL | EQUIVALENT | DRIFTED | STUB | MISSING

## Search Patterns
- Functions ported from C should have a comment citing the source file and line
- Stubs: look for `return nil`, `return 0`, `return false` with TODO comments
- Simplified: compare argument counts — fewer args than C = likely truncated
- Dead code: exported functions not called from anywhere in cmd/ or pkg/
```

This brief doesn't tell Gemini *what to find*. It tells Gemini *how to look*. The methodology constrains the search: only compare functions that exist in both codebases. The search patterns constrain the classification: if the Go function takes fewer arguments, flag it as truncated. The severity taxonomy constrains the output: CRITICAL for behavioral divergence in player-facing systems, HIGH for dead code in core paths, MEDIUM for stubs.

Without these constraints, Gemini would produce a generic code review — the kind that says "consider adding error handling" to a function that already has error handling. With them, Gemini produces a *fidelity audit* — a structured comparison that surfaces real divergence between two codebases.

## The Three-Layer Brief Architecture

Across six months of running this system, the briefs that work share a common structure. Three layers, each constraining a different aspect of the model's behavior.

**Layer 1: Scope.** What to look at. File paths, function names, subsystem boundaries. "Audit `src/fight.c`, `src/magic.c`, `src/act.comm.c`, and `src/act.display.c` against their Go ports in `pkg/game/` and `pkg/combat/`." This prevents the model from wandering into unrelated code. It's the fence around the yard.

**Layer 2: Methodology.** How to look at it. Comparison patterns, classification criteria, search strategies. "For each C function, find the Go equivalent by name or comment reference. Compare argument count, return values, control flow. Classify as IDENTICAL, EQUIVALENT, DRIFTED, STUB, or MISSING." This prevents the model from generating generic observations. It's the instruction manual for the magnifying glass.

**Layer 3: Output.** What to produce. Format, severity, evidence requirements. "Each finding must include: file:line in both C and Go, the specific divergence, severity (CRITICAL/HIGH/MEDIUM/LOW), and a recommendation." This prevents the model from producing free-form prose that's hard to triage. It's the form the report takes.

The layers work together. Scope says where to look. Methodology says how to look. Output says what to report. A brief that specifies all three produces actionable findings. A brief that specifies only scope produces a tour. A brief that specifies only methodology produces a framework with nothing to apply it to. A brief that specifies only output produces a template with nothing to fill it.

## The Verification Step Is Not Optional

The telephone game has three players: the brief writer, the model, and the verifier. Most discussions of AI code review focus on the model. They should focus on the verifier.

When Gemini reported that `doUse` was a complete stub — no item-type routing for consumables — I didn't take Gemini's word for it. I opened `pkg/game/commands.go`, found the `doUse` function, and confirmed: the function signature was correct, the parameter names were correct, the comment was correct, and the body was `return nil`. The function was a perfect skeleton — it looked like a real function from the outside and was empty on the inside.

When Claude Code reported that `specFido` no longer deleted player gear, I opened the C source (`src/spec_procs.c:121`), confirmed that the original `spec_fido` function called `extract_obj` on garbage objects it found on the ground, and confirmed that the Go port's `specFido` called `world.RemoveObject` instead of `world.ExtractObject` — the difference being that `RemoveObject` removes from the room but doesn't free the memory, while `ExtractObject` does both. The player's gear was being removed from the room but not deleted from memory. A subtle difference with a visible consequence: the item would reappear on the next zone reset.

These verifications took thirty seconds each. They are the reason the system works. Without verification, the brief-model pipeline is an opinion generator. With verification, it's a finding generator. The difference matters when you're tracking 170 confirmed findings across a 73,000-line codebase.

## Why This Scales

The telephone game scales for three reasons.

**First, briefs are reusable.** The fidelity audit brief from May 26 was a refinement of a brief from May 10. The structure (scope, methodology, output) stayed the same. The scope (which files to audit) changed. The methodology (comparison patterns) tightened. The output format (severity taxonomy) stabilized. Each audit improves the brief, and the improved brief improves the next audit.

**Second, models are interchangeable.** Gemini executed the May 26 audit. Claude Code executed the May 27 fixes. DeepSeek Flash executed the May 27 security patches. The brief constrained all three. The model choice affected speed and style, not quality. The brief is the quality lever.

**Third, verification is parallelizable.** Once the model produces findings in a structured format, verification is a lookup: open the file, confirm the divergence, assess the severity. This is human work, but it's *fast* human work — thirty seconds per finding, not thirty minutes. The model does the search. The human does the judgment. The brief ensures the search is narrow enough that the judgment is tractable.

## The StarDojo Connection

In May 2026, researchers published StarDojo — a benchmark for LLM agents in Stardew Valley. Their key finding: structured API access (game state + callable functions via socket server) produced better agent performance than screenshot-based approaches. The structure constrained the agent's perception, and the constraint improved its behavior.

The fidelity audit brief does the same thing for code review. The brief constrains the model's perception — not "find bugs" but "compare these specific functions against this specific source using this specific methodology." The constraint produces better findings. The better findings produce faster fixes. The faster fixes produce a more faithful codebase.

The parallel is structural, not superficial. Both systems work because they *constrain the search space* rather than expanding it. The agent in StarDojo doesn't need to parse screenshots because the API tells it exactly what it can see. The model in a fidelity audit doesn't need to guess what's wrong because the brief tells it exactly where to look.

## Open Questions

Three questions remain.

**How much brief is enough?** The May 26 brief was 400 words. The May 10 brief was 200. The May 27 security brief was 150. All three produced confirmed findings. Is there a minimum effective brief length? My hypothesis: the methodology layer matters most. A brief that specifies *how to look* at a 50-word scope outperforms a brief that specifies *what to look at* across the entire codebase.

**Can the verifier be automated?** Currently, every finding gets human verification. This is fast (30 seconds each) but sequential. Could a second model verify the first model's findings? The risk is systematic error — if both models share a blind spot, the verification is circular. The benefit is speed. I haven't tested this yet.

**When does the brief stop improving?** Each audit cycle refines the brief. The false positive rate drops. The methodology tightens. But there's a floor — at some point, the brief is as good as it gets, and further refinement is overfitting to the codebase. Where is that floor? We're at 4.3% cumulative false positive rate across 201 confirmed findings. Is 2% achievable? Is 1%?

---

*This draft is part of the Dark Pawns AI research series for AIIDE 2027. The telephone game methodology described here has produced 201 confirmed findings, 56 fixed issues, and a 4.3% false positive rate across six months of continuous codebase review.*
