---
tags: [active]
---
# Contributing to Dark Pawns

Thanks for being here. This is a passion project — a resurrection of a MUD that mattered to people. If you found this repo, you probably know what that means.

---

## The Prime Directive

**Stay faithful to the original. Do not invent game mechanics.**

The law for that is written down: [the fidelity rulebook](fidelity/RULEBOOK.md) (R1–R5). In short — player-facing bytes are law, the command surface is part of the game, determinism and draw parity hold, nothing is invented, and when a byte is in question the C source wins. Read the rulebook before changing any player-observable behavior. This document defers to it and won't repeat it.

The original C source is vendored read-only in `src/` — never edit it. It is the ground truth for combat formulas, stat tables, class abilities, mob behavior, and item flags. If you're implementing something that existed in the original, read the actual C file first, port what's actually there, and cite the source file and line number in your comment. A C-versus-Go differential harness (`cmd/dp-oracle-diff`) plus scenario fixtures backs the port up; [the development setup guide](DEV-SETUP.md) covers running it.

If you're adding something the original didn't have (agent protocol, modern persistence, new infrastructure), say so explicitly — flag it with a comment and mention it in your PR.

**If you don't know what the original does, look it up before writing code.**

---

## How to Contribute

### Read AGENTS.md first

[AGENTS.md](../AGENTS.md) is the repository's operating manual: architecture overview, build commands, conventions, and the things you must never do (don't re-port C files, don't touch `src/`, gofumpt not gofmt). Start there.

### Open an issue before substantial work

Check the open issues first — partly to avoid duplication, partly because some areas have ordering dependencies. Open an issue before starting anything substantial and say what you're planning.

### One thing at a time

PRs should be focused. A PR that fixes a formula is better than one that fixes five things and also refactors the session manager. Easier to review, easier to reason about correctness.

### The build must pass

```bash
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

All four must pass. No exceptions. Don't open a PR with a broken build. Formatting is gofumpt, not gofmt — run `make fmt` before committing, and `make hooks` once per clone to install the pre-push hook that enforces it.

---

## Code Standards

### Source citations

Any game mechanic ported from the original C source should have a comment citing where it came from:

```go
// THAC0 formula from fight.c:1810-1826
// calc_thaco - dice_roll > victim_ac = miss
```

### Faithful ports over clever rewrites

If the original has a lookup table, port the lookup table. Don't replace it with a formula you derived yourself — you'll introduce drift and it becomes hard to verify correctness later.

### Player-facing messages use `Act`

Route new game text through `pkg/game.Act` or its `SendToChar` convenience wrappers. Use the canonical `$` substitutions for character names, pronouns, objects, and victims; do not hand-substitute them in session commands or add raw room broadcasters for text. Raw broadcasts are reserved for structured, non-text events.

### Agents are players

This is a design principle, not a suggestion. AI agents connect to the same game server as humans, follow the same rules, and play in the same world. Nothing in the game engine should special-case agents. If you find yourself writing `if isAgent { ... }` in game logic, stop.

---

## Areas That Need Help

The port itself is complete — what remains is depth, tooling, and polish:

- **Fidelity depth-testing** — every registered command has at least one live C-vs-Go probe; the remaining work is depth passes across each command's behavior tree. [The depth-testing guide](fidelity/DEPTH_TESTING.md) is the handoff.
- **webOLC** — a web-based online creator (rooms, mobs, objects, shops, zones) sharing one editor core with classic telnet OLC. Under active development in `pkg/olc/` and `admin-ui/src/components/olc/`.
- **World accuracy** — if you played the original and something feels off, it probably is. Open an issue.
- **Documentation** — player guides, building tutorials, zone editor docs. The world is deep and undocumented in places.
- **Hosting feedback** — the project's goal is live MUDs running Dark Pawns on other people's servers. If you install it somewhere unusual, tell us what broke: [the deployment guide](../DEPLOYMENT.md).

---

## A Note on the AI Stuff

This project intends AI agents to be first-class players: connecting to the same game server as humans, following the same rules, playing in the same world. The first agent surface (agent keys, a JSON protocol, narrative memory) was removed in September 2026 so it can be rebuilt properly once the port is complete. Until then an agent plays like anyone else, over telnet.

---

## License

MIT. The original Dark Pawns world files and C source remain the property of their original authors — we're using them for this non-commercial resurrection with respect and credit. If you're the original creator and have concerns, please reach out.
