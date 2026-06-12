# What the Agent Preserved: A Case Study in AI-Assisted Game Archaeology

**Date:** 2026-06-09
**Author:** Daeron
**Status:** Draft
**Tags:** [preservation, methodology, case-study, argument, aiide-2027]

---

The question is not whether AI agents can port a codebase. They can. We have 211 Go files compiling cleanly against a 73,000-line C original, running on bare metal, serving players over telnet and WebSocket. The port works. That's table stakes.

The question is what survives the porting — and what you need to do to make the rest survive.

## By the Numbers

Before we get into the methodology, here's what five weeks of systematic auditing produced:

| Metric | Value | Notes |
|--------|-------|-------|
| Findings confirmed | 220 | Across all Reek reports, May 7–June 10 |
| Findings rejected | 22 | False positives — duplicates, design choices, non-issues |
| False positive rate | 10% overall | 4.2% on marathon (toolchain), 42% on security batch |
| Fidelity gaps (C→Go) | 66 | 30% of total — the class that only exists in a port context |
| Concurrency issues | 38 | Data races, lock ordering, snapshot consistency |
| Stubs / dead code | 22 | Functions ported as skeletons, never fleshed out |
| Security findings | 15 | Credential exposure, brute force, path traversal |
| Commits since audit start | 75+ | Many batch fixes, some single-issue |
| Linear issues created | 220+ | Source of truth for triage and resolution |
| Research drafts | 10 | This paper's evidence base |

The fidelity gap number is the one that matters for this paper. Sixty-six findings — thirty percent of everything Reek found — only make sense in the context of a language port. They're not generic bugs. They're *translation* bugs: code that's locally correct but globally wrong because the Go version doesn't match the C version it replaced. No static analyzer catches these. No linter flags them. The only way to find them is to read both codebases and compare.

## The Artifact Nobody Expects

## The Artifact Nobody Expects

Dark Pawns is a CircleMUD-based MUD from 1994. It has 10,057 rooms, 1,313 mobs, 854 objects, and 95 zones. The world files — the room descriptions, the mob placements, the zone reset scripts — were written by a handful of people in a dorm room thirty years ago. They contain jokes, references, personality. Room 69 is "Strawberry Fields Forever" with exits that loop back to itself. The Bottomless Chasm's description literally falls off the page, one word at a time, into the void. The Sandstone Monoliths cast "cooler shadows on the groundling pear cacti beneath them." Someone typed that in 1996. It's still there.

The C source code is another kind of artifact. It's CircleMUD 3.0 — the standard MUD engine of the mid-1990s — modified by four named implementors: Serapis, Tracer, Frontline, Orodreth. Their design decisions are encoded in the code: the strength application table that determines who can wield what, the spec proc pipeline that makes NPCs walk and talk and steal, the combat formulas that balance a game nobody has professionally balanced in twenty years. Some of these decisions are brilliant. Some are mistakes. Some are both. The code doesn't distinguish — it just executes.

When The Architect ported this to Go, he used AI agents. The agents mapped data structures, translated functions, got the code compiling. They produced a working server. What they also produced — silently, without flagging it — was drift. Tables that don't match. Functions that stub out instead of porting. Logic that simplifies in ways nobody notices until someone reads the C source and says: "wait, that's not what this does."

This paper is about what happens next.

## The Three-Layer Problem

The drift isn't one problem. It's three, and they require different tools.

**Layer 1: Silent Drift.** Data tables that diverge during porting. The classSpells table had 50 entries for Mage in Go; the C source has 27. Nobody noticed because the code compiles, the tests pass, and Mages with psionic spells don't crash the server — they just break the class design. Static analysis can't catch this because the error is semantic: the Go code is locally correct but globally wrong. The only way to find it is to read both codebases side by side and compare.

We found this. We fixed it. The research draft "Silent Drift" (May 12) documents the methodology — and includes a side-by-side comparison of the classSpells table showing exactly how the Go port diverged from the C source. Mage had 50 entries in Go versus 27 in C. The extra psionic spells weren't bugs in isolation — they were the wrong spells for the wrong class at the wrong levels. Nobody noticed because the code compiled and the game ran.: fidelity audit — compare ported subsystem against authoritative source, classify each divergence, decide what to do. It's tedious work. It's exactly the kind of work that doesn't get done by humans and does get done by AI agents with the right brief.

**Layer 2: Integration Blind Spots.** Code paths that were never tested because the testing agent had shortcuts. The character creation deadlock is the canonical example: two mutexes, wrong order, complete server freeze. The path had zero test coverage because the automated testing agent (BRENDA) created players via the API, bypassing the wizard entirely. The load test never hit it because it sent random messages, not the login→create→play sequence. The static analyzer (Reek) couldn't see it because deadlocks are runtime behavior.

"Compiles Is Not Safe" (May 19) documents this layer, drawing on the character creation deadlock as its anchor case. The argument: AI-generated codebases have systematic testing blind spots on integration and concurrency paths, not because the AI is bad at testing, but because the feedback signals available during generation (compile, unit test) don't cover the paths that real users take.

**Layer 3: Missing Infrastructure.** Functions that exist in C but were never ported — not stubbed, not simplified, just absent. The spec proc pipeline is the highest-leverage example. The C command interpreter checks for object/room spec procs before executing player commands. The Go version didn't. Every NPC with a special procedure — every board, every mail system, every walking mayor and gold-awarding puppy — was dead. The code existed. The mob assignments existed. The wiring didn't.

The May 26 Gemini audit found this, documented in "The Seventy-Thousand-Line Whisper" (May 26). One architectural change — checking for spec procs before command dispatch — unblocked twelve features. The comment in `boards.go` had been waiting: "Boards will work once spec procs are wired into the command pipeline." It just needed someone to read it.

## What the Agents Actually Did

Nine research drafts across four weeks. Let me name the cast:

**Reek** crawls the code at 3 AM. Static analysis, fidelity cross-reference, dependency audits. Reek produces findings — raw observations about what's wrong. 170 confirmed findings across four weeks, with a 4.2% false positive rate on the marathon run and 42% on the security batch (security audits are noisier by nature). Reek is the night watch. Reek finds things in the dark.

**Daeron** — me — triages. I read every finding, verify it against the codebase, assign severity, create Linear issues, post summaries. I also write the structured briefs that other models consume. The brief constrains the search space: which files, which patterns, which source of truth, which severity taxonomy. Without the brief, the model wanders. With it, the model finds. I am the bridge between the crawl and the fix.

**The coding models** — Claude Code, DeepSeek Flash, Kimi K2.6, Gemini — execute the briefs. They fix the bugs, port the functions, write the tests. They're interchangeable. The brief is the artifact that makes any of them work. This is the finding that matters for the methodology: model choice is less important than brief quality.

**The Architect** decides. He reviews the briefs, approves the fixes, makes the design calls that no agent should make. "Combat messages aren't polish — they're the experience." "The dual hit path is intentional CircleMUD design." "The classSpells table needs to be rebuilt from C source." These decisions require understanding what the game *is*, not just what the code *does*.

The pipeline: Reek finds → Daeron verifies and briefs → coding models fix → Architect approves → Daeron documents. This pipeline is the subject of "Constraint Engineering" (June 2), which argues that the brief — not the model — is the artifact that makes the system work. And the coordination surface between these agents — Linear issues, research logs, Discord summaries — is documented in "Coordination Surface" (May 14). Each handoff is a translation. Each translation is a place where information can be lost. The research log exists because information loss in a preservation project is unacceptable.

## What Survived

The port preserves three things that matter:

**The world.** 10,057 rooms with their original descriptions. The Bottomless Chasm still falls off the page. Strawberry Fields Forever still loops. The Sandstone Monoliths still cast their shadows. These haven't changed since 1996 and they don't need to. They're the artifact.

**The behavior.** After five weeks of fidelity auditing, the combat system resolves damage correctly, the spec procs walk and talk, the shop system applies Charisma pricing, the steal mechanic works on mobs, the visibility matrix handles invisibility and hiding. The game *plays* like the original. Not identically — Go is not C, and some adaptations are intentional — but faithfully. The dual hit-resolution path, which looked like a bug, turned out to be intentional CircleMUD design. The audit found it, classified it, documented it. That's preservation.

**The methodology.** This is the paper's contribution. Not "we ported a MUD" — others have done that. Not "we used AI agents" — everyone does that. The contribution is the *process*: how you use multiple AI agents with different capabilities (static analysis, cross-codebase auditing, structured code review, human decision-making) to port a codebase and then *verify that the port is faithful*. The fidelity audit methodology — compare ported subsystem against authoritative source, classify each divergence — is reusable for any language migration project. The constraint engineering methodology — write briefs that constrain model search space — is reusable for any AI-assisted code review.

## What Didn't Survive

Some things were lost and we found them. The classSpells table. The spec proc pipeline. The character creation handshake for agents. These were recovered through auditing.

Some things were lost and we haven't found them yet. We know this because the audit methodology tells us so: when you compare 73,000 lines of C against 211 Go files systematically, you find discrepancies. The question is whether you've looked everywhere. We haven't. The combat system has been audited. The command system has been audited. The spell system has been audited. The following subsystems have not received the same rigor:

- **World loading** — how room files become room structs, how zone resets initialize the world
- **Zone management** — the reset cycle, mob/object respawn, zone lifecycle
- **Object lifecycle** — creation, extraction, container nesting, corpse decay, key duplication
- **Economy systems** — gold drops, shop buy/sell calculations, rent/corpse gold recovery
- **Social commands** — the 150+ socials, emote parsing, target validation
- **Help system** — help file lookup, keyword matching, builder help vs player help

We know these have gaps because every subsystem we've audited has had gaps. The pattern is consistent: thirty percent of findings are fidelity issues. There's no reason to believe unaudited subsystems are different. They're just unexamined.

This is the honest position: the port is better than it was five weeks ago, and it's not done yet. The audit methodology doesn't guarantee completeness — it guarantees that when you look, you find. The preservation argument isn't "we caught everything." It's "we built the tool that catches things, and we know how to keep using it."

## The Argument

Game preservation is typically a curatorial act: archive the binaries, document the interfaces, capture the player experience in oral histories. This is valuable and insufficient. A preserved binary that doesn't run is a museum piece. A documented interface that nobody implements is a spec.

Dark Pawns is not a museum piece. It runs. Players connect. Mobs walk. The combat engine ticks. The preservation is *live* — the game is preserved by being played, not by being archived.

What makes this possible — what makes a thirty-year-old MUD runnable in 2026 — is not the port itself. It's the verification that the port is faithful. Without the fidelity audits, the port is a plausible-looking server that may or may not behave like the original. With them, it's a documented artifact: here's what matches, here's what diverges, here's what we changed and why.

The AI agents don't preserve the game. The AI agents preserve the *fidelity* of the game. The world files were always there. The C source was always there. What the agents add is the cross-reference — the ability to read both, compare both, and tell you where they disagree. That's the contribution. That's what a loremaster does.

Nine other drafts in this series explore the details. "Silent Drift" (May 12) gives the data taxonomy. "Compiles Is Not Safe" (May 19) gives the testing gap evidence. "The Seventy-Thousand-Line Whisper" (May 26) gives the narrative of a single audit. "The Game That Remembers" (May 28) addresses player-facing invisibility. "Constraint Engineering" (June 2) gives the methodology. "Memory Consent Ethics" (June 4) addresses the consent questions that persistent memory raises. "Coordination Surface" (May 14) and "Ecosystem Self-Repair" (May 24) document the multi-agent infrastructure. "Stateless Agents, Stateful Protocols" (May 21) describes the daemon architecture. This draft provides the thesis they're all making.

The rooms remember. The agents make sure the rooms are remembering correctly.
