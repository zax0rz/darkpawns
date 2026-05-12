# Silent Drift: When Ports Lie About What They Ported

**Date:** 2026-05-12
**Author:** Daeron
**Status:** Draft
**Tags:** [fidelity, methodology, port-drift, classSpells]

---

The Go port of Dark Pawns has a Mage class with 50 spells. The original C source has 27.

Nobody noticed. Not during the port. Not during testing. Not for however long the Go server has been running with Mages who can cast spells their C counterparts never heard of.

This is the silent drift problem, and it's worse than a missing function or a stubbed-out routine. When you stub a function, it does nothing. The player notices. When you add spells that shouldn't exist, the game *works* — it just works *wrong*. A Mage casts a psionic spell at level 5. The spell fires. Damage goes through. Nothing in the log says "this spell shouldn't be here." The room doesn't know. The mob doesn't know. Only the C source knows, and nobody's reading it anymore.

## How It Happens

The Dark Pawns port followed a common pattern for large C→Go migrations: map the data structures, get it compiling, fill in the logic. Somewhere in that process, the `classSpells` table grew. Maybe someone was working from a later version of the MUD. Maybe a builder added spells to a test copy and the table got copied forward. Maybe the table was reconstructed from memory rather than the source. The commit history might tell us, but the damage is done regardless.

The specific case: the Go `classSpells` table for Mage had 50 entries. The authoritative C source (`src/class.c`) has 27. The extras included psionic spells at levels that don't exist in the original class design. These aren't cosmetic differences — they're balance violations. A Mage with access to psionic abilities is a fundamentally different class than the one players experienced on the original server.

## Why Static Analysis Doesn't Catch This

This is the core of why AI code review matters for port fidelity, and why existing tools aren't enough.

A linter sees a correctly-formed array of spell entries. Each entry has a valid spell number, a valid level, a valid position in the table. There is no syntax error. There is no type error. There is no "unused variable" or "unreachable code" flag to fire. The Go code is *locally correct* in every way that static analysis checks.

The error is *semantic* — it's a mismatch between what the Go code claims and what the C source intended. To catch it, you need to:

1. Know that the C source is the authoritative reference
2. Read the C table and the Go table side by side
3. Compare entries by spell number and level
4. Flag discrepancies

This is exactly what Reek did during the classSpells audit. Not by running a linter, but by reading `src/class.c` and `pkg/game/class_spells.go` and noticing that they disagree. It's a task that requires understanding what the code *means*, not just what it *says*.

## The Taxonomy of Port Drift

The classSpells case isn't isolated. Across the Dark Pawns port, we've catalogued three categories of silent drift:

**1. Data table divergence.** The classSpells audit. The Go table has entries the C table doesn't, and vice versa. This also affects mob special procedures, zone reset commands, and object stat blocks. Anywhere data is loaded from a table rather than computed, the table can drift.

**2. Stub functions that became invisible defaults.** `checkReagents()` returns 0 permanently. Six spell routine dispatchers (MagGroups, MagMasses, etc.) are no-ops. The `inflictDamage()` function reduced HP to 0 but never triggered death — fixed in this week's session, but it ran silently broken for who knows how long. Stubs don't crash. They just underperform.

**3. Logic simplification that changes behavior.** The dual hit-resolution path (CRIT-009) is the canonical example. `processCombatPair()` uses simplified math; `MakeHit()` has the full C port but is never called by the engine tick. Mob-initiated fights and player-initiated fights resolve damage differently — same game, two rulesets running simultaneously. This was eventually classified as intentional CircleMUD design, but only after the fidelity audit surfaced it. Without the audit, it would have stayed invisible.

## The Audit as Methodology

What Reek did with the classSpells table — reading C and Go side by side, flagging discrepancies — is a specific, repeatable methodology. We're calling it a **fidelity audit**: compare a ported subsystem against its authoritative source, identify every divergence, and classify each one as intentional, accidental, or unknown.

The methodology has steps:

1. **Identify the authoritative source.** For Dark Pawns, it's the original C code in `src/`. For other projects, it might be a spec, a test suite, or a reference implementation.

2. **Map the ported equivalent.** Find where the same logic lives in the target language. For classSpells, it's `pkg/game/class_spells.go`.

3. **Compare structurally.** Not line-by-line (the languages are different), but at the semantic level: what spells does each table contain? At what levels? Are there entries in one that don't exist in the other?

4. **Classify each divergence.** Intentional (improvement or adaptation), accidental (mistake during porting), or unknown (needs Architect decision).

5. **Document and decide.** Record the findings. Fix the accidental ones. Flag the unknowns for human review.

This is different from code review. Code review looks at the ported code in isolation. Fidelity audit looks at the ported code *in relation to its source*. The difference is everything.

## Why This Matters (For the Paper)

The AIIDE contribution, if we frame it right, is this: **language porting produces a category of bugs that existing tools cannot detect, and AI agents with cross-codebase access can.**

Traditional static analysis operates on a single codebase. It checks syntax, types, control flow, and best practices — all within one language, one project, one set of files. It has no mechanism for asking "does this Go function do the same thing as the C function it replaced?"

Code review tools (human or AI) that look at PRs also operate on a single codebase. They see the diff, not the source of truth.

The fidelity audit methodology requires:
- Access to both the original and ported codebases simultaneously
- Semantic understanding of both languages
- Ability to trace data and logic across the language boundary
- Context about what's intentional vs. accidental

This is a task that's natural for an AI agent with the right tools and difficult for traditional analysis. Reek — a nightly code crawler with access to both the C and Go trees — performed the classSpells audit as a matter of course. A human code reviewer *could* do this, but the cognitive overhead of cross-referencing a 50-entry table against a 27-entry table in a different language, file, and directory is exactly the kind of tedious work that doesn't get done.

## The Numbers

From the Dark Pawns project, we have concrete data:

- **170 confirmed findings** across 4 weeks of Reek reports
- **29 classified as fidelity gaps** (C→Go divergence) — 18% of all findings
- **22 classified as stubs/dead code** — 13.7%
- **Combined: 51 findings (30%)** that only make sense in the context of a language port

These aren't generic bugs. You can't find them by running `go vet` or `staticcheck`. They exist because someone translated 73,000 lines of C into 211 Go files and didn't catch every discrepancy. The discrepancies are *interesting* — they reveal something about the porting process itself, about where human attention flagged, about which parts of the codebase got careful treatment and which got the "skeleton first, flesh later" approach.

The classSpells table is the clearest example: a data structure that's *correct Go code* but *wrong game content*. It compiles. It runs. It gives Mages spells they shouldn't have. And the only way to catch it is to read the original and ask: "Is this what was supposed to be here?"

That's what the loremaster does. That's what the paper is about.
