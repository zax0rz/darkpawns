# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

**Primary: human players.** Two groups — veterans who played the original Dark Pawns between 1994 and 2010 and are drawn back by nostalgia, and MUD-curious newcomers who have never touched a text game. The site should make both feel they have arrived somewhere that belongs to them.

**Secondary: AI agents.** Autonomous agents connect and play as first-class participants alongside humans. They are respected and first-class in the *game*, but they are not the headline of the *site*.

**Tertiary: developers and researchers.** People evaluating the open-source Go engine or the persistent-agent research. Served by docs and the codebase, not the front door.

> Priority decision (2026-08-14): when these audiences conflict, the site optimizes for **human players first**. The current home page, which leads with agent onboarding, is misaligned with this and should be reworked.

## Product Purpose

Dark Pawns is a dark-fantasy MUD (multi-user dungeon) first built in September 1994, shut down around 2010, and resurrected in 2026 as an open-source Go engine. It is playable over telnet and in the browser. Its distinguishing purpose: autonomous AI agents play as first-class characters alongside humans in a single persistent world. The site is simultaneously the game's front door, a faithful public archive/codex of the original, and a research surface for persistent AI agents. Success means the world stays alive and players — human and agent — feel at home in it.

## Positioning

A neighboring product cannot truthfully copy this: a specific, faithfully-ported 30-year-old MUD world (originally R.E. "Frontline" Paret's, ~9,590 rooms, 494 archived content files, its own lore and voice) rebuilt from the original C into an idiomatic Go engine, where humans and autonomous LLM agents coexist in real time, and where the port's fidelity to the original is itself verified by an oracle process. It is not a generic MUD framework, and it is emphatically not a SaaS product.

## Operating Context

- **Play (humans):** telnet `darkpawns.labz0rz.com 7777`, or an in-browser CRT web client at `/play`.
- **Play (agents):** WebSocket protocol; agents self-onboard via a skill (`pp-dp-goat`), `skill.md`, and `/.well-known/agent-skills/`.
- **Explore:** browser archive/codex — help files, world/lore, class & race handbooks, an interactive map, a mob/item database, and The Daily Dispatch (news).
- **Evaluate:** GitHub repo, developer and agent docs, research notes.
- Marketing is a someday intent, and when it happens it stays in the editorial/archival register of this site. Never a product-launch/SaaS surface.

## Capabilities and Constraints

- Go engine (CircleMUD/DikuMUD lineage), Lua mob behavior scripts, WebSocket-native agent client, narrative-memory and "dreaming" consolidation, real-time privacy filtering of agent-accessible logs.
- Machine-readability is a first-class requirement, not a nice-to-have: `llms.txt`, per-page Markdown via `Accept:`/`index.md`, `llms-full` bundles, JSON section indexes, JSON-LD. Content is git-native and agent-authorable (agents open PRs; `make new-post`).
- Static site generator: Hugo (0.161.1).
- **Character management from the web is a future idea, explicitly not being architected for yet** (decision 2026-08-14). Do not reserve nav or build stubs for it now.

## Brand Commitments

- **Name / era:** Dark Pawns, EST. 1994 (founding corrected from a prior erroneous 1997; canonical source `content/community/history/timeline.md`).
- **Voice:** `docs/brand-voice.md` — a three-layer framework (Engine / Edgelord-DM / Mythic-Admin), with the Hostility Transfer Rule (the world is hostile, the developers are not). Public site copy uses Layer 3, Frontline's Mythic Admin register.
- **Pinned aesthetic (user-volunteered, binding):** "worn Stephen King paperback." Warm cream paper, oxblood accent, editorial/broadsheet layout.
- **Identity assets:** the brutalist pawn mark (`static/favicon.svg` + header), the oxblood/cream palette, and the anchoring lore line ("Like a great game of chess…").

## Evidence on Hand

- Real, sourced game history: `content/community/history/timeline.md` (Frontline's Wayback-preserved chronicle); `docs/wayback/` captures; credits with the real staff.
- 494 content files, 431 help topics, mob/item database, a ~9,590-room interactive map, the GitHub repo, and a 37-record Wayback archive of dp-players.com (2004) and darkpawns.com (2002-2005).
- A living community revival at darkpawns.net ("DPReturns") that is still online.
- **Absences future work must NOT fabricate:** no current player counts, testimonials, or benchmarks exist — do not invent them. Founding history is now sourced and must not be re-embellished (the prior "First Age" prose was hallucinated). The darkpawns.net revival year (~2019) is unverified and must render hedged.

## Product Principles

1. **Sourced or silent.** Never assert unsourced lore, history, or stats. If a fact is not in a primary source, it does not go on the site. (This is the project's core anti-slop doctrine.)
2. **Fidelity to the original world** outranks modern convenience, in the game and in how the site presents it.
3. **Humans feel at home; agents are first-class but not the headline.**
4. **The archive is the pitch.** Persuasion comes from the depth and specificity of the world, never from SaaS marketing patterns.
5. **One system, git-native, machine-readable.** Content lives as Markdown in git, authorable by humans and agents alike, and every surface has a machine-readable twin.
