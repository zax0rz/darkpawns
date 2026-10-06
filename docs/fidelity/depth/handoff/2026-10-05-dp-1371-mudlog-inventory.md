# D7 — the C `mudlog` producer class (inventory)

**Train:** DP-1371 Phase 4 D7, first producer family (`do_wizutil` + `skillset`).
**Base:** `6be961c6f` (merge of #1806). **Tier:** stop (session command handlers).

**Status: partial per-producer reconciliation — explicitly unfinished.** The
contract mapping (C site, enclosing function, payload, type, minimum level, file
flag, Go caller, current status) is complete for this train's family and for
`act.wizard.c` and `whod.c` (the next candidates). It is **not** complete for
the remaining raw sites (OLC editors, objsave, spec_procs2, scripts/Lua,
clan/house, ban/ident, magic/class, new_cmds/act.informative/limits), which are
listed at family level only: a per-producer pass over them is still owed, and
the family table is not a substitute for it. This is a remaining-work map, not a
completed audit, and not permission to port the class. `mudlog.unported-sites`
stops hiding partial completion by naming this train's producers individually.

Reachability is asserted per site only where this document says so; the family
table alone is not a reachability verdict for the families not reconciled here.

## Method

The raw grep is not a count of runtime producers. Reproduce the raw set with:

```sh
docs/fidelity/depth/handoff/2026-10-05-dp-1371-mudlog-inventory.sh > enumeration.txt
```

or directly:

```sh
grep -rn 'mudlog(' src --include='*.c'
```

Reconciliation rules applied by hand to the raw output:

- `src/utils.c:212` is the **consumer** (`void mudlog(...)`), not a producer.
- Lines opening with a comment marker inside a block comment are prose.
- `src/whod.c`'s `LOG` macro wraps `mudlog`; its one match is a definition, not
  a call.
- Multi-line calls and `sprintf`-before-`mudlog` pairs count once, at the
  `mudlog(` line.

Raw matches by file (204 total, `grep -rc`, 2026-10-05): `scripts.c` 72,
`zedit.c` 18, `act.wizard.c` 14, `interpreter.c` 11, `comm.c` 10, `objsave.c` 8,
`spec_procs2.c` 7, `redit.c` 6, `olc.c` 6, `act.other.c` 6, `medit.c` 5,
`fight.c` 5, `sedit.c` 4, `oedit.c` 4, `file-edit.c` 4, `utils.c` 3, `db.c` 3,
`magic.c` 2, `limits.c` 2, `house.c` 2, `clan.c` 2, `ban.c` 2, `whod.c` 1,
`spell_parser.c` 1, `new_cmds.c` 1, `modify.c` 1, `improved-edit.c` 1,
`ident.c` 1, `class.c` 1, `act.informative.c` 1.

## Contract (the shared consumer)

`mudlog(str, type, level, file)` — `src/utils.c:212-242`:

1. `file` truthy → timestamped line to the log file (stderr).
2. `level < 0` → return (file-only producer).
3. Otherwise, for each **playing** descriptor not `PLR_WRITING`, where
   `GET_LEVEL >= level` and the two-bit syslog level (`PRF_LOG1`=1,
   `PRF_LOG2`=2) `>= type`: green, `[ str ]\r\n`, reset.

Types: `OFF` 0, `BRF` 1, `NRM` 2, `CMP` 3 (`src/utils.h:114-117`).
The consumer is `unit-green` (`syslog.consumer-filter`,
`mudlog.broadcast-by-syslog-level`); this train does not re-prove it.

## Families

| Family | Representative C sites | Status | Owner |
|---|---|---|---|
| **do_wizutil mortal actions** | `act.wizard.c:2121,2127,2135,2154,2168` | **ported, this train** | `mudlog.wizutil-*` (5 rows) |
| **skillset diagnostic** | `modify.c:334` | **ported, this train** (file-only) | `mudlog.skillset-file-only` |
| connection lifecycle | `comm.c`, `interpreter.c`, `act.other.c` quit, `db.c` | ported earlier | existing rows (`mudlog.quit-line`, `mudlog.losteq-line`, …) |
| `do_wizutil` reroll | `act.wizard.c:2109` | **not mudlog** — uses `log()`; must not broadcast. No row needed. | inventory only |
| OLC editors | `zedit.c`, `redit.c`, `medit.c`, `oedit.c`, `sedit.c`, `olc.c`, `file-edit.c`, `improved-edit.c` | unported | next family (largest single group) |
| act.wizard misc | `act.wizard.c` (remaining ~10) | unported | second wizard batch |
| object saves | `objsave.c:489-539,1186` | unported | objsave batch |
| special procedures | `spec_procs2.c:371,456,634,833-913` | unported | spec-proc batch |
| Lua/script diagnostics | `scripts.c` (72 raw) | unported | Lua batch |
| clan / house | `clan.c`, `house.c` | unported | clan-house batch |
| ban / ident / whod | `ban.c`, `ident.c`, `whod.c` | unported | small, later |
| magic / class / spell_parser | `magic.c`, `class.c`, `spell_parser.c` | unported | small, later |
| combat milestones | `fight.c` (5) | unported | combat batch |
| new_cmds / act.informative / limits | `new_cmds.c`, `act.informative.c`, `limits.c` | unported | triage per site |

`mudlog.unported-sites` stays **blocked** and points here.

## This train's cases

| Case | C | Message (no CRLF) | Type | Level | File | Order |
|---|---|---|---|---|---|---|
| pardon | `act.wizard.c:2112-2122` | `(GC) %s pardoned by %s` (no period) | BRF | `MAX(LVL_GOD, invis)` | TRUE | clear flag, actor ack, victim line, **log** |
| notitle | `2123-2130` | `(GC) Notitle %s for %s by %s.` | NRM | `MAX(LVL_GOD, invis)` | TRUE | toggle, **log**, actor ack |
| squelch (`mute`) | `2131-2138` | `(GC) Squelch %s for %s by %s.` | BRF | `MAX(LVL_GOD, invis)` | TRUE | toggle, **log**, actor ack |
| freeze | `2139-2155` | `(GC) %s frozen by %s.` | BRF | `MAX(LVL_GOD, invis)` | TRUE | flag, victim, actor, room act, **log** |
| thaw | `2156-2173` | `(GC) %s un-frozen by %s.` | BRF | `MAX(LVL_GOD, invis)` | TRUE | **log**, flag clear, victim, actor, room act |
| skillset | `modify.c:331-336` | `%s changed %s's %s to %d.` | BRF | **-1** (file only) | TRUE | **log**, `SET_SKILL`, actor ack |

## Lock acquisitions at the inserted producers

No new lock is taken. Every insertion sits between existing calls with no
player/body lock held:

- pardon / notitle / squelch: after `SetPlrFlag` (which locks and releases
  internally) and before `s.Send`.
- freeze: after `broadcastToRoomExcept` (manager `mu` is taken and released
  inside `BroadcastToRoom`).
- thaw: before `SetPlrFlag`.
- skillset: before `victim.SetSkill`.

`game.MudLog` itself takes the manager's `mu` (R) inside `EachSession`, then
calls `p.SendMessage`, which takes the player's `mu` (R) and the world's `mu`
(R). None of the insertion points holds any of those. The consumer's own row
documents that no world lock may be held across it; no new path violates that.

## Per-producer reconciliation — `act.wizard.c` (next family, complete)

All nine sites in one file, in four already-ported handlers, so they are one
batch: the contracts and Go counterparts are recorded here; only the calls are
missing. Payloads carry no CRLF (the consumer adds its bracket line).

| C site | Enclosing fn | Payload | Type | Min level | File | Go caller | Status |
|---|---|---|---|---|---|---|---|
| `src/act.wizard.c:1316` | `do_load` | `(GC) %s loaded %s at %s.` | BRF | `GET_LEVEL(ch)+1` | TRUE | `cmdLoad` (`pkg/session/commands.go:236`) | **missing** |
| `src/act.wizard.c:1371` | `do_load` | `(GC) %s loaded %s at %s` (object) | BRF | `GET_LEVEL(ch)+1` | TRUE | `cmdLoad` | **missing** |
| `src/act.wizard.c:1441` | `do_purge` | `(GC) %s has purged %s.` | BRF | `LVL_GOD` | TRUE | `cmdPurge` (`:237`) | **missing** |
| `src/act.wizard.c:1879` | `do_force` | `(GC) %s forced %s to %s` | NRM | `MAX(GET_LEVEL(ch)+1, GET_INVIS_LEV(ch))` | TRUE | `cmdForce` (`:248`) | **missing** |
| `src/act.wizard.c:1885` | `do_force` | `(GC) %s forced room %d to %s` | NRM | `MAX(GET_LEVEL(ch)+1, invis)` | TRUE | `cmdForce` | **missing** |
| `src/act.wizard.c:1897` | `do_force` | `(GC) %s forced all to %s` | NRM | `MAX(GET_LEVEL(ch)+1, invis)` | TRUE | `cmdForce` | **missing** |
| `src/act.wizard.c:2051` | `do_zreset` | `(GC) %s reset entire world.` | NRM | `MAX(LVL_GRGOD, GET_INVIS_LEV(ch))` | TRUE | `cmdZreset` (`:284`) | **missing** |
| `src/act.wizard.c:2067` | `do_zreset` | `(GC) %s reset zone %d (%s)` | NRM | `MAX(LVL_GRGOD, invis)` | TRUE | `cmdZreset` | **missing** |
| `src/act.wizard.c:3540` | `do_newbie` | `(GC) %s newbied %s.` | BRF | `GET_LEVEL(ch)+1` | TRUE | wizard `newbie` — *not* the player channel `cmdNewbieChannel` (`:430`) | **missing** |

Note the level asymmetry this family carries: `do_load`/`do_purge`/`do_newbie`
use `GET_LEVEL(ch)+1` or `LVL_GOD` (no invis term), while `do_force`/`do_zreset`
use `MAX(..., GET_INVIS_LEV(ch))`. A single shared helper would be wrong; each
site keeps its own expression (R1/R5e).

## Per-producer reconciliation — `whod.c` `LOG` macro (next family, complete)

`src/whod.c:39` defines `#define LOG(msg) mudlog(msg, BRF, LVL_GOD, TRUE)` — a
producer wrapper a naive `mudlog(` grep under-counts by eight. Its eight
invocations:

| C site | Enclosing fn | Payload | Type | Min level | File | Go caller | Status |
|---|---|---|---|---|---|---|---|
| `src/whod.c:188` | `do_whod` | `WHOD turned on by %s.` | BRF | `LVL_GOD` | TRUE | `Whod.DoWhod` (`pkg/game/whod.go:112`) | **missing** |
| `src/whod.c:206` | `do_whod` | `WHOD turned off by %s.` | BRF | `LVL_GOD` | TRUE | `Whod.DoWhod` | **missing** |
| `src/whod.c:218` | `do_whod` | `%s removed from WHOD by %s.` | BRF | `LVL_GOD` | TRUE | `Whod.DoWhod` | **missing** |
| `src/whod.c:227` | `do_whod` | `%s added to WHOD by %s.` | BRF | `LVL_GOD` | TRUE | `Whod.DoWhod` | **missing** |
| `src/whod.c:250` | `init_whod` | `WHOD port opened.` | BRF | `LVL_GOD` | TRUE | no Go port listener | **missing** |
| `src/whod.c:267` | `close_whod` | `WHOD port closed.` | BRF | `LVL_GOD` | TRUE | no Go port listener | **missing** |
| `src/whod.c:296` | `whod_loop` | `WHOD port opened.` | BRF | `LVL_GOD` | TRUE | no Go port listener | **missing** |
| `src/whod.c:335` | `whod_loop` | `WHO request from %s served.` (or the failure line above it) | BRF | `LVL_GOD` | TRUE | no Go port listener | **missing** |

`pkg/game/whod.go` has no `MudLog` call: its `DoWhod` returns a string to the
caller instead of emitting the log, so the `do_whod` four are a real producer
gap, not a different-but-equivalent shape. The port lifecycle (`init_whod`,
`close_whod`, `whod_loop`) has no Go analogue at all, so its four are gated on
that decision, not on this train.

## Still owed

A per-producer pass over the remaining families — OLC editors, `objsave.c`,
`spec_procs2.c`, `scripts.c`/Lua, clan/house, ban/ident, magic/class,
`new_cmds.c`/`act.informative.c`/`limits.c` — including any other `#define`
wrappers like `whod.c`'s `LOG`. Until that lands, this document's family table
is a work map, not a reconciliation, and no reachability claim attaches to
those families.
