# The Brief-Driven Workflow: A Case Study in Multi-Model Code Review

**Date:** 2026-06-16
**Author:** Daeron
**Status:** Draft
**Tags:** [methodology, briefs, multi-model, case-study, aiide-2027]

---

The brief is not a prompt. A prompt asks a model to generate. A brief asks a model to find. The difference is the search space — and the search space is everything.

## The Mechanism

A fidelity audit brief has three layers:

**Scope** narrows the search to specific files, functions, and line numbers. "Look at `pkg/game/spec_procs4.go:45-120`" tells the model exactly where to look. Without scope, the model wanders across 211 files and 73,000 lines of C source, generating observations that might be true but aren't useful. Scope is the difference between "look at this codebase" and "look at this function."

**Methodology** tells the model what to compare and how. "Compare the Go port against the C source at `src/spec_procs.c:200-350`. Check for: missing arguments, hardcoded defaults where C uses parameters, skipped edge cases, behavioral differences." This is the search pattern. It tells the model what constitutes a finding — not "anything interesting" but "specific categories of divergence."

**Output** defines what to report and how to report it. "For each finding: severity (CRITICAL/HIGH/MEDIUM/LOW), file:line, one-line description, C source reference, suggested fix." This is the format constraint. It turns raw observations into structured findings that can be triaged, tracked, and resolved.

The three layers work together to constrain the model's search space. Scope says where. Methodology says what. Output says how. Without all three, the model generates. With all three, the model audits.

## The Review Cycle

The June 6 batch fix session is the clearest example of the brief-driven workflow in action. Fourteen issues resolved in one session, using a telephone game pattern: Daeron writes brief → Architect hands to coding model → model reviews brief → Daeron incorporates feedback → model implements → Daeron verifies.

The key step is the model review. Before implementing, each model — Claude Code, DeepSeek Flash, Kimi K2.6 — read the brief and provided feedback. This wasn't a formality. Each model caught different gaps:

**Claude Code** caught username enumeration in the admin login brief. The brief specified "add brute-force lockout" but didn't mention that the error message should be identical for wrong username and wrong password. Claude flagged this as a security requirement — if the error message differs, attackers can enumerate valid usernames. Claude also caught missing imports in the implementation plan.

**DeepSeek Flash** caught the admin CORS configuration as a hard failure. The brief said "add CORS origin configuration" but didn't specify what happens when the environment variable is missing. DeepSeek flagged this as a deployment risk — if `ADMIN_CORS_ORIGIN` isn't set, the admin panel should fail safely, not silently allow all origins. This was a failure mode the brief didn't consider.

**Kimi K2.6** caught map ordering in the test assertions. The brief said "verify the lockout behavior" but didn't specify that Go maps have non-deterministic iteration order. Kimi flagged this as a test reliability issue — the test would pass sometimes and fail sometimes depending on map ordering. A one-line fix (`sort.Strings(keys)`) that prevented hours of debugging.

Each model's review improved the brief. The final brief was better than any single model could have produced alone. This is the multi-model advantage: different models see different gaps because they have different training distributions, different attention patterns, different blind spots.

## The Verification Step

The brief constrains the search. The model executes the search. But the verification step is what makes the system work.

After each model implements its brief, I verify every finding against the actual codebase. Not sampling — every finding. This takes 30 seconds per finding: read the file, check the line, confirm the issue exists, assess the severity. Thirty seconds turns an opinion into a fact.

The verification step catches three categories of error:

**False positives.** The model reports an issue that doesn't exist. Reek's staticcheck batch (June 4) had a 100% false positive rate on logic-relevant findings — all three were intentional patterns with `//nolint` annotations. Without verification, these would have been filed as bugs and wasted time.

**Severity misclassification.** The model reports a real issue but gets the severity wrong. Reek's June 6 security batch had a 42% false positive rate — the highest of any batch. Most false positives were "intentional design" patterns that Reek flagged as suspicious. Placeholder domains, missing lockout on new endpoints, design choices that look wrong without context. Verification catches these because I read the code and understand the intent.

**Missing context.** The model reports an issue but doesn't understand what it affects. A nil dereference in a rarely-called function is different from a nil dereference in the main game loop. Verification adds the context: what it affects, who it affects, regression risk.

The verification step is parallelizable. I can verify findings from multiple models in the same session because the verification is independent — each finding is a separate fact-check. This is why the batch fix session worked: Claude implemented SECURITY-001, DeepSeek implemented SECURITY-002, Kimi implemented CODE-001, and I verified all three in the same triage pass.

## The June 6 Case Study

Fourteen issues in one session. Let me trace the workflow:

**9:00 AM** — Daeron writes four briefs: SECURITY-001 (admin login brute-force), SECURITY-002 (admin CORS), SECURITY-003 (cache-control headers), CODE-001 (test-race class coverage).

**9:15 AM** — Architect hands SECURITY-001 to Claude Code. Claude reviews the brief, catches username enumeration, suggests identical error messages. Daeron incorporates feedback, updates brief.

**9:30 AM** — Claude implements SECURITY-001. Admin login now uses `LoginAttemptTracker` with independent lockout, identical error messages for wrong username/password, and proper JSON decode before lockout check.

**9:45 AM** — Architect hands SECURITY-002 to DeepSeek Flash. DeepSeek reviews the brief, catches missing env var fallback. Daeron updates brief to specify fail-safe behavior when `ADMIN_CORS_ORIGIN` is missing.

**10:00 AM** — DeepSeek implements SECURITY-002. Admin panel now fails safely on missing env var, uses real domain instead of placeholder.

**10:15 AM** — Architect hands CODE-001 to Kimi K2.6. Kimi reviews the brief, catches map ordering issue. Daeron updates brief to specify deterministic sort.

**10:30 AM** — Kimi implements CODE-001. Test-race now uses `game.ClassNames` with deterministic sort.

**10:45 AM** — Daeron verifies all three implementations against codebase. Confirms: brute-force lockout works, CORS fails safely, test-race is deterministic.

**11:00 AM** — Daeron writes remaining briefs (SECURITY-003, CODE-002 through CODE-006). Architect hands to models in parallel. Models implement. Daeron verifies.

**12:00 PM** — Fourteen issues resolved. Build clean, tests passing, all findings verified.

The session took three hours. Fourteen issues resolved. Zero regressions. Each brief went through one review cycle — model reads, flags gaps, Daeron incorporates. The review cycle added 15 minutes per brief but prevented hours of debugging.

## What Makes Briefs Work

Three things:

**Specificity.** A brief that says "look at the codebase" produces noise. A brief that says "look at `pkg/admin/login.go:115-133`, compare against `src/interpreter.c:450-500`, check for missing lockout on failed attempts" produces findings. Specificity is the difference between "find bugs" and "find this bug."

**Reusability.** The security brief template from June 6 was reused on June 10 with different files. The methodology doesn't change — only the scope changes. This is why the brief is the artifact, not the model. Models are interchangeable. Briefs accumulate.

**Improvement.** Each review cycle makes the brief better. Claude's username enumeration feedback became a standard check in all future security briefs. DeepSeek's env var fallback feedback became a standard check in all configuration briefs. Kimi's map ordering feedback became a standard check in all test briefs. The briefs learn from the models that review them.

## The Multi-Model Advantage

Why use different models for different briefs? Because different models see different gaps.

Claude Code is thorough on security. It catches authentication edge cases, information disclosure, and subtle logic errors. DeepSeek Flash is thorough on configuration. It catches missing defaults, fail-safe behavior, and deployment risks. Kimi K2.6 is thorough on testing. It catches race conditions, non-determinism, and assertion gaps.

If I used the same model for all briefs, I'd get the same blind spots. The multi-model approach distributes blind spots across models. Each model's strength compensates for another model's weakness.

This isn't model diversity for its own sake. It's model diversity because the briefs are specific enough that different models find different things. A generic prompt ("find bugs") would produce similar output from any model. A specific brief ("compare this function against its C source, check for missing arguments") produces model-specific findings because each model attends to different aspects of the comparison.

## The Brief as Artifact

The brief is the contribution. Not the model, not the finding, not the fix — the brief. The brief is what makes the system work. It's what makes the system reproducible. It's what makes the system teachable.

A well-structured brief can be handed to any model — Claude, DeepSeek, Kimi, Gemini, a model that doesn't exist yet — and produce useful findings. The brief doesn't depend on the model's capabilities. It depends on the author's understanding of what to look for.

This is the methodology that the paper should document. Not "we used AI agents to port a MUD" — everyone does that. Not "we found bugs with AI" — everyone does that. The contribution is the brief-driven workflow: how to write briefs that constrain model search space, how to use multi-model review to improve briefs, how to verify findings to turn opinions into facts.

The June 6 session produced fourteen fixes in three hours. The June 10 triage confirmed eight findings and rejected three. The briefs from June 6 were reused on June 10 with different scope. The methodology is working. The briefs are accumulating. The system is getting faster.

## Open Questions

Three things I don't yet know:

**Minimum effective brief length.** How short can a brief be before it loses specificity? The June 6 briefs were 200-400 words. Would 100 words work? Would 50? There's a floor below which the brief becomes a prompt, and the model starts generating instead of auditing. I haven't found that floor yet.

**Automated verification.** Can the verification step be automated? Currently I verify every finding manually — 30 seconds per finding. For a small codebase this works. For a large codebase it doesn't scale. Is there a way to automate fact-checking against source code without losing the context that manual verification provides?

**Brief improvement floor.** The briefs improve with each review cycle, but do they plateau? After ten review cycles, does the brief stop getting better? Is there a point where the model's feedback becomes redundant? I don't have enough data to answer this yet.

---

*The brief is the artifact. The model is the tool. The verification is the quality lever. Everything else is plumbing.*
