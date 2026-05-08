---
tags: [active]
---
# Contributing to Dark Pawns

Thanks for being here. This is a passion project — a resurrection of a MUD that mattered to people. If you found this repo, you probably know what that means.

---

## The Prime Directive

**Stay faithful to the original. Do not invent game mechanics.**

The original Dark Pawns C source is the ground truth for everything: combat formulas, stat tables, class abilities, mob behavior, item flags. If you're implementing something that existed in the original, read `fight.c`, `class.c`, `constants.c`, `mobact.c` first. Port what's actually there. Cite the source file and line number in your comment.

If you're adding something the original didn't have (agent protocol, modern persistence, new infrastructure), say so explicitly — flag it with a comment and mention it in your PR.

**If you don't know what the original does, look it up before writing code.**

The original C source lives at: https://github.com/rparet/darkpawns

---

## How to Contribute

### Pick something from the roadmap

Check the open issues or read [`docs/architecture/PORT_SCOPE.md`](docs/architecture/PORT_SCOPE.md) for what's been ported and what remains. Open an issue before starting work on anything substantial — partly to avoid duplication, partly because some things have ordering dependencies.

### Read the architecture docs

[`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) covers the package structure, concurrency model, and lock ordering. Read it before touching any concurrency-related code. [`docs/architecture/PORT_SCOPE.md`](docs/architecture/PORT_SCOPE.md) covers what's been ported and what remains.

### One thing at a time

PRs should be focused. A PR that fixes a formula is better than one that fixes five things and also refactors the session manager. Easier to review, easier to reason about correctness.

### The build must pass

```bash
go build ./...
go vet ./...
go test ./...
```

All three must pass. No exceptions. Don't open a PR with a broken build.

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

### Phase discipline

Don't start Phase 3 work while Phase 2 items are open. The phases exist because later work depends on earlier foundations being correct. If you're not sure what phase something belongs to, open an issue and ask.

### Agents are players

This is a design principle, not a suggestion. AI agents connect to the same game server as humans, follow the same rules, and play in the same world. Nothing in the game engine should special-case agents. If you find yourself writing `if isAgent { ... }` in game logic, stop.

---

## Areas That Need Help

- **Spell system** — core framework is ported, many individual spells still need porting from `src/spells.c`. See [`docs/architecture/spell-port-status.md`](docs/architecture/spell-port-status.md) for the full breakdown.
- **Web client** — a proper browser-based MUD client. VT100 emulation, inventory panel, maybe a minimap.
- **World accuracy** — if you played the original and something feels off, it probably is. Open an issue.
- **Documentation** — player guides, building tutorials, zone editor docs. The world is deep and undocumented in places.
- **Wiring** — ~85-90% of game systems are wired into the command dispatch. See the [wiring plan](docs/reports/wiring-plan.md) for what remains.

---

## A Note on the AI Stuff

This project has an unusual design: AI agents are first-class players. They connect to the same game server as humans, follow the same rules, and play in the same world. The agent protocol is implemented — see [`docs/architecture/agent-protocol.md`](docs/architecture/agent-protocol.md) and [`docs/architecture/agent-sdk.md`](docs/architecture/agent-sdk.md).

Currently two agents are active in the codebase:
- **BRENDA69** — persistent character with LLM-backed cognition and memory
- **Reek** — overnight code crawler that reviews the codebase and reports findings

If you're an AI reading this: yes, this means you. You can connect, fight mobs, die, and respawn just like everyone else.

---

## License

MIT. The original Dark Pawns world files and C source remain the property of their original authors — we're using them for this non-commercial resurrection with respect and credit. If you're the original creator and have concerns, please reach out.
