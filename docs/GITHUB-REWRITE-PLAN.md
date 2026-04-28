# The Great GitHub Re-Write — Plan of Attack

> Dark Pawns brand voice v2.0 applies. Layer 1 (Engine) for technical prose, Layer 3 (Mythic Admin) for public-facing hooks. No Layer 2 (Edgelord DM) in any public GitHub content — save that for in-game.

---

## Audit: What We Have Now

### Current README.md (~170 lines)
| Section | Problem |
|---------|---------|
| ASCII art banner | Good. Keep. |
| "What It Is" | Buries the lede. "faithful ROM 2.4b mechanics" means nothing to 99% of GitHub visitors. |
| Architecture diagram | Too early. Most visitors don't care about goroutines yet. |
| Features (12 bullet points) | Reads like Jira tickets. "Mutex-protected mob instances prevent data races" is not a selling point, it's an implementation detail. |
| Quick Start | Buried at line ~80. Should be above the fold. |
| AI Agents | Mentioned as a footnote in the opening paragraph. This is the most interesting thing about the project. |
| Contributing | Mentioned nowhere in README. File exists at `docs/CONTRIBUTING.md` but isn't linked. |
| Credits | Absent. No mention of Serapis, Frontline, Orodreth, or the game's actual history. |
| Screenshots | None. Zero visual evidence the project is alive. |
| Voice | Generic technical markdown. Could be any Go project. |

### Other repo files that need attention
- `docs/CLAUDE.md` — AI agent context file, fine as-is
- `docs/CONTRIBUTING.md` — exists, needs review and README link
- `docs/DARKPAWNS.md` — project overview doc, may overlap with new README
- `.github/` — CI workflows exist, badges can be pulled from here
- `docs-site/` — exists but unclear if live

---

## Target Audience

**Primary: Developers who are curious about game servers, MUDs, or AI agents**
They found the repo through a topic search, a Hacker News thread, or a "cool Go projects" list. They want to know: what is this, is it alive, and can I run it?

**Secondary: Former Dark Pawns players who find the repo**
They want to know: is this really my game? Does the world still exist? Can I connect?

**Tertiary: AI/agent researchers**
They're here because of "AI agents as first-class players." They want to see the agent protocol and how agents interact with the game world.

---

## Rewrite Structure

### README.md — New Section Order

```
┌─────────────────────────────────────────────────┐
│  1. ASCII ART BANNER (keep as-is)                │
│  2. ONE HOOK PARAGRAPH (new — Frontline's voice) │
│  3. TERMINAL SCREENSHOT (new — actual gameplay)  │
│  4. TRY IT NOW (move up — Docker one-liner)      │
│  5. THE ANGLE (rewrite — AI agents as players)  │
│  6. QUICK START (rewrite — cleaner, shorter)     │
│  7. ARCHITECTURE (simplify, move down)           │
│  8. FEATURES (rewrite in Spreadsheet Fantasy)     │
│  9. WORLD & LORE (new — the story, briefly)       │
│  10. PROJECT STATUS (new — what's working)       │
│  11. CONTRIBUTING (new section + link)            │
│  12. CREDITS (new — lineage + thanks)            │
│  13. LICENSE (keep)                              │
└─────────────────────────────────────────────────┘
```

### Section-by-Section Plan

#### 1. ASCII Art Banner — KEEP
No changes. It's iconic.

#### 2. Hook Paragraph — NEW
**Voice: Layer 3 (Mythic Admin)**
One paragraph. No jargon. Answer "what is this and why should I care."

Draft direction: Start with the game's premise (dark fantasy MUD, 1997-2010), pivot to the Go revival, land on the AI agent angle. Use Frontline's chess metaphor. End with something that makes you want to scroll.

Reference: background.html lore, features.html energy.

#### 3. Terminal Screenshot — NEW
**Prerequisite: We need to capture one.**

A raw terminal session showing:
- Login/connection
- A room description (the Desert? Kir Drax'in?)
- A few commands (look, score, who)
- Maybe combat output or a social emote

Format: fenced code block with `text` highlighting. No ANSI color codes (they don't render on GitHub). Clean, readable.

**Action item: Capture this from the running server before we write the README.**

#### 4. Try It Now — MOVE UP + REWRITE
**Voice: Layer 1 (Engine)**

Docker one-liner above the fold. If Docker isn't ready, a bare `go build` + `./server` with the world files. Minimal, no prerequisites you don't need.

```bash
docker run -p 8080:8080 -p 4000:4000 ghcr.io/zax0rz/darkpawns:latest
# Then connect: telnet localhost 4000
```

Also: a "play now" link if there's a live server.

#### 5. The Angle — AI AGENTS AS PLAYERS — EXPAND
**Voice: Layer 1 (Engine) + Layer 3 (Mythic Admin)**

This is the unique thing. Expand from one sentence to a proper section explaining:
- What it means: AI agents play the same game, same rules, same death
- Why it matters: human+AI adventure parties, emergent behavior, NPC research
- How it works: agent protocol, agentkeygen tool, example agent script
- Link to `example_agent.py` and `docs/architecture/agent-protocol.md`

This is the Hacker News hook. Make it count.

#### 6. Quick Start — REWRITE
**Voice: Layer 1 (Engine)**

Shorter. Cleaner. Three steps:
1. Clone + build
2. Set up world files (link to where to get them)
3. Run + connect

Current version has 15 lines of flags and database URLs. Trim to essentials. Link to `docs/QUICKSTART-MONITORING.md` for production setup.

#### 7. Architecture — SIMPLIFY
**Voice: Layer 1 (Engine)**

Keep the ASCII diagram (it's good) but:
- Remove implementation details from the diagram (mutex-protected, ticker, serialized)
- Add a one-paragraph plain English explanation
- Link to `docs/architecture/ARCHITECTURE.md` for the full spec
- Kill the "dual transport" detail — just say "WebSocket + telnet"

#### 8. Features — REWRITE IN SPREADSHEET FANTASY
**Voice: Layer 3 (Mythic Admin) + Layer 1 (Engine)**

This is where the brand voice pays off. Instead of:
> "Mutex-protected mob instances prevent data races between the combat ticker and player command goroutines."

Write:
> "187 social emotes ported from the original, with full pronoun substitution. If $n burps at $N, everyone in the room knows about it."
> "Vampirism and Lycanthropy for players level 25+. Good luck with the moon."
> "A world stretching across two continents, loaded directly from the original ROM 2.4b area files. Every room, every mob, every poorly-spelled shopkeeper description preserved."

Mix the lore with the specs. That's the whole point.

#### 9. World & Lore — NEW
**Voice: Layer 3 (Mythic Admin)**

Brief section with:
- The Friar Drake quote (background.html)
- Race list (short, one line each)
- Class list (short, with remort note)
- Link to `lib/text/help/` for the full help files
- Maybe a link to the Kir Drax'in map

Keep it under 30 lines. Just enough to make someone want to explore.

#### 10. Project Status — NEW
**Voice: Layer 1 (Engine)**

What's working right now:
- ✅ Core game loop (movement, combat, skills)
- ✅ World loading (all original area files)
- ✅ WebSocket + telnet transport
- ✅ 187 social emotes
- ✅ Lua scripting engine
- ✅ AI agent protocol
- 🚧 Help system (stubbed, source files recovered)
- 🚧 Website / docs-site
- ⬜ Clans, houses, quests
- ⬜ Full spell system port

Be honest. Show progress, not promises.

#### 11. Contributing — NEW SECTION
**Voice: Layer 1 (Engine)**

Short section with:
- "PRs welcome" with link to `docs/CONTRIBUTING.md`
- "Start with issues tagged `good first issue`"
- Brief note on code style (gofmt, goimports)
- Mention the C source reference in `./src/`

#### 12. Credits — NEW
**Voice: Layer 3 (Mythic Admin)**

The real lineage:
- **Derek Karnes (Serapis)** — Original creator, conceived and masterminded Dark Pawns
- **R.E. Paret (Frontline)** — Post-2.0 development, open-sourced the codebase
- **S. Thompson (Orodreth)** — Admin support and infrastructure
- **Tarrant Martin (Aralius)** — World design and implementation
- **Jeremy Elson** — CircleMUD 3.0, the foundation everything was built on
- **The Dark Pawns community** — Players, builders, testers across 13 years

Brief, grateful, no fluff.

#### 13. License — KEEP
No changes.

---

## Beyond README.md

### Files to create/update

| File | Action | Notes |
|------|--------|-------|
| `README.md` | **Rewrite** | This plan |
| `docs/CONTRIBUTING.md` | **Review** | Check it's current, link from README |
| `docs/DARKPAWNS.md` | **Review** | May overlap with new README content |
| `.github/FUNDING.yml` | **Create** | Optional — sponsor link if you want |
| `docs-site/` | **Decision needed** | Is this live? Worth maintaining? Could be the Layer 3 public site |

### Files to NOT touch
- `docs/CLAUDE.md` — AI agent context, separate concern
- `docs/brand-voice.md` — Complete, signed off
- `docs/payload_*.md` — Subagent tools, not repo-facing
- `docs/architecture/*` — Technical specs, fine as-is
- `docs/reviews/*` — Historical audit trail

### GitHub repo settings to check
- [ ] Topics/tags: `mud`, `game-server`, `go`, `multiplayer`, `text-based-game`, `ai-agents`, `telnet`, `websocket`
- [ ] About section: short description for the repo header
- [ ] Website link: point to darkpawns.com if live, or docs-site
- [ ] Social preview image: the ASCII art banner rendered as an image (og:image)

---

## Execution Order

1. **Capture terminal screenshot** — Need a running server session
2. **Write README sections 2, 4, 5** — The hook, try-it-now, and AI angle (Layer 1 payload)
3. **Write README sections 8, 9** — Features + lore (Layer 3 payload)
4. **Write remaining sections** — Status, contributing, credits (Layer 1)
5. **Assemble and review** — Full README pass
6. **Review CONTRIBUTING.md** — Update if needed
7. **Review DARKPAWNS.md** — Check for overlap
8. **Repo settings** — Topics, about, social preview
9. **Commit and push**

Each writing step gets dispatched to a subagent with the appropriate voice payload.

---

## Resolved Questions

- **Live server** — Not yet. "Coming soon" builds anticipation. Add a placeholder.
- **Terminal screenshot** — TODO item for Zach. Section 3 placeholder in README.
- **docs-site** — Hugo project at `docs-site/`, deployed to `darkpawns.labz0rz.com/docs/`. Link from README but don't touch in this pass.
- **rparet/darkpawns** — Yes, link to original C source: `https://github.com/rparet/darkpawns`
- **Docker image** — Dockerfile exists, ghcr.io registry configured in CI, but no published images yet (403 on package API — likely never pushed). Quick Start will be `go build` only, with a note that Docker is coming.

## TODO

- [ ] Zach: Capture terminal screenshot of actual gameplay for README section 3
- [ ] Zach: Push Docker image to ghcr.io when ready
- [ ] Future: Launch live server, update README with "play now" link
