# 2026-10-08: The exhaustive call-site inventory as a defect detector

## Scope

- **Tracks:** legacy-port fidelity; test-oracle validity; multi-agent software engineering
- **Window:** 2026-10-06 to 2026-10-08, DP-1371 Phase 4 case D7 (`mudlog.unported-sites`)
- **Primary artifacts:** the C producer-site inventory `docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites.tsv` and its checker (#1819); reconciliation #1855; D7 trains #1812–#1815, #1818, #1820–#1822, #1829, #1834, #1837, #1845, #1847, #1854; DeepSeek's PR 1a reconnaissance (2026-10-08); Linear DP-1398, DP-1401, DP-1404, DP-1405
- **Status:** artifact-grounded field note. It adds a third detection method to PF-053 (review) and PF-054 (playtest).

## What happened

D7 is one Phase 4 case: "the Go server emits C's immortal syslog lines". #1819 enumerated it mechanically as **every active `mudlog()` call in `src/`**, with a checker that fails if the inventory and the source disagree. That gave 207 sites across nearly every subsystem. Each site was then ported, proven unreachable, or blocked, one row at a time.

The work was meant to produce log lines. Along the way it also surfaced player-visible defects and representation gaps that no scenario or reviewer had reached, because proving a log line forces you to reach the code path that emits it:

| Found while proving | Defect | C authority |
|---|---|---|
| steal producers (#1822) | `steal-depth`'s success rows were coincidence greens: the vehicle ran in a peaceful room (DP-1398) | `src/act.other.c:340-344` |
| ban-refusal producer (PR 2 recon) | Banned telnet users never receive `Sorry, your site is banned.` | `src/comm.c:1572` |
| `call_magic` default (PR 1a recon) | `dominate` prints an invented `Spell not yet implemented.` and doesn't charm (DP-1405) | `src/spell_parser.c:506-507` |
| `Crash_load` producers | Go stores no rent code, so seven entry outcomes can't be told apart (DP-1404) | `src/objsave.c:489-539` |
| `lua_*` argument diagnostics | Seven Lua bindings return success where C returns 0 on bad arguments | `src/scripts.c:216-753` |
| house restore, during object registration | House container contents are dropped at boot (DP-1401) | `pkg/game/house_save.go:96-104` |

The inventory also corrected itself in the other direction. A reviewer's ruling and an agent's reading both treated `kender_steal`'s PC-victim log (`spec_procs2.c:634`) as reachable. A second read of the function's first guard (`:597-598`) showed it's dead code in C.

## Methodological observations

- **An exhaustive, checker-enforced inventory is a third detector**, distinct from differential testing (PF-053) and human play (PF-054). The census proves equality only on paths scenarios reach. The inventory forces someone to *reach* each C call site, and reaching it exposes what is missing nearby.
- **The cost is breadth:** D7 took about three days and roughly 20 PRs for one case whose output only immortals see. A day-two reconciliation (#1855) found 17 of 63 "missing" rows had already landed, so inventory state labels drift unless they're reconciled against code.
- **Agents applied the "don't invent" rule to the inventory itself:** one refused to attach a log line to a message C never prints (dominate), and reported the gap instead.
