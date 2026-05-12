# Session Handoff — 2026-05-12

Paste this at the start of the next session to pick up where we left off.

---

## Context

We're building Dark Pawns (Go MUD) for an AIIDE 2027 paper. Novel contribution: **server-hosted emotionally valenced autobiographical memory for game agents**. Memory is baked into the game engine, not managed client-side. The paper argues this is a genuine research gap.

We did a full session today despite a context-limit crash at the start. Gemini Deep Research Max produced 55K chars of proper academic research. Daeron organized it into `docs/research/`. Related work v2 is ~80% done.

## Key Documents

- `docs/research/evaluation-methodology.md` — 4 metrics, 4 experimental phases, Postgres-backed data pipeline
- `docs/research/related-work-draft-v2.md` — publication-ready Related Work, needs citations merged from resolved-citations.json
- `docs/research/foundations/resolved-citations.json` — 61 citations from the deep research
- `docs/research/design-research-log.md` — living build spec, hypothesis tracker, every architecture decision
- `docs/research/foundations/deep-research-max-2026-05-12.md` — raw deep research output

## Built This Session

- **`cmd/dp-agent/`** — Go CLI binary (9MB, zero new deps). Subcommands: play, session, config, keygen, whoami, exec. WebSocket connect → structured state → FSM override → LLM call → command → log. Session logging currently in-memory.
- **`pkg/agentcli/`** — config, client, LLM client (LiteLLM proxy), combat FSM, session logger, prompt builder, output parser.
- **`scripts/gemini_deep_research.py`** — REST-based Gemini Deep Research Max wrapper.
- **`static/skill.md`** — agent onboarding doc (5 min to playing).

## Experiment Design (per Daeron)

- **Co-player problem solved.** Zach plays multiple characters (Zakarr, Brenn — possibly Aiko, Misteryuck later). Same player, different conditions. Blind scoring by Daeron.
- **Centerpiece experiment:** Zakarr + Brenn co-op (thief/assassin/cleric party). Train for a boss fight for 2-3 weeks. Boss wrecks them. Social memory of joint trauma is measured by subsequent behavior.
- **Multi-character design proves entity-specific memory:** BRENDA's memory of Zakarr doesn't transfer to Aiko.

## Critical Path — Next Session

1. ~~Wire Postgres session logging~~ — **Resolved.** JSONL export is the eval methodology's primary data format. Postgres is optional.
2. **Merge citations into related-work-v2** — resolved-citations.json has all 61. Flagged items: ReasonPlanner arXiv ID, Memoria venue, TALES author list.
3. ~~Build dreaming layer (Phase 5d)~~ — **Built.** 607 lines across 3 files. Core contribution is implemented.
4. ~~Valence toggle~~ — **Built.** `--valence true/false` flag on session, config, and WebSocket login.
5. **Run baseline sessions** — Zakarr + Brenn co-op, 10 sessions with memory off. Need a running game server first.
6. **Wire dreaming into server** — Server needs to serve memory summaries from dreaming output. Currently `MemorySummary` field exists in GameState but nothing populates it.
7. **Dreaming valence heuristics** — Current extract.go assigns basic valence (combat=+1, flee=-2, damage=-2). Needs content-aware valence based on entity relationships, context.
8. **Human dp-agent client** — Zach mentioned this. A TUI/web UI for direct play alongside the agent. Out of scope for paper but worth scoping early.

## Git Changes

- `cmd/dp-agent/` and `pkg/agentcli/` — NEW, not yet committed
- `docs/research/evaluation-methodology.md` — NEW
- `docs/research/foundations/` — deep research + resolved citations
- `static/skill.md` — NEW
- `brenda-site/` — CI fix committed and pushed
- `.learnings/subagent-learnings.md` — updated with Deep Research + dp-agent patterns
- `MEMORY.md` — updated
