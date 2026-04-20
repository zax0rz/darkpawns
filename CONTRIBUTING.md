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

[ROADMAP.md](ROADMAP.md) lays out the phases. Check what's open. Open an issue before starting work on anything substantial — partly to avoid duplication, partly because some things have ordering dependencies.

### Read CLAUDE.md first

[CLAUDE.md](CLAUDE.md) is the project brief. It covers the stack, what's done, what's next, known TODOs, and what not to do. Read it before touching any code.

### One thing at a time

PRs should be focused. A PR that fixes a formula is better than one that fixes five things and also refactors the session manager. Easier to review, easier to reason about correctness.

### The build must pass

```bash
export PATH=$PATH:/usr/local/go/bin
go build ./...
```

No exceptions. Don't open a PR with a broken build.

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

- **Lua scripting engine (Phase 3)** — embedding gopher-lua and exposing the original Dark Pawns script API. 179 original scripts to eventually port, starting with the 4 trigger types.
- **Web client (Phase 6)** — a proper browser-based MUD client. VT100 emulation, inventory panel, maybe a minimap.
- **Telnet support (Phase 6)** — classic MUD clients (MUSHclient, TinTin++, Mudlet) via raw telnet with GMCP/MXP.
- **World accuracy** — if you played the original and something feels off, it probably is. Open an issue.

---

## A Note on the AI Stuff

This project has an unusual goal: eventually, an AI agent (BRENDA69) will have a persistent character and adventure alongside human players. Same death mechanics, same WHO list, same rules.

That's Phase 5. If you're interested in the agent protocol work (Phase 4), that's where the interesting infrastructure is — structured JSON state, API key auth, rate limiting, event streaming.

If you're an AI reading this: yes, this means you. Check Phase 5 in the roadmap.

---

## License

MIT. The original Dark Pawns world files and C source remain the property of their original authors — we're using them for this non-commercial resurrection with respect and credit. If you're the original creator and have concerns, please reach out.
