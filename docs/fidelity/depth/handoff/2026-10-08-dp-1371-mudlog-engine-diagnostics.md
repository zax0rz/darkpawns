# DP-1371 D7: engine-diagnostic producers (PR 1a)

**Local (mudlog-only) — in progress.** Base `origin/main` `245cac5ca` (merge
#1855). The landed cases add or replace a diagnostic with `game.MudLog(...)`;
they change no command routing, Act or output path, function signature, state
mutation order or error handling. Two producers are committed and proven; the
remaining Lua work and two classification findings that reshape the rest of 1a
are below. R1/R4/R5e/R5g/R5h.

## Landed cases

| C site | Contract | Boundary and order | Locks at producer | Proof |
|---|---|---|---|---|
| `src/comm.c:2523-2526` | `"SYSERR: no valid target to act()!"`, CMP, LVL_IMMORT, file TRUE | C logs after the existing `log()` line and returns before any audience is computed. The port printed the same text with `log.Println` at both arms; both now call the producer (one call writes the file side too). The `world == nil` arm is the port's own defensive arm — C's world is a global array — and says so. | none | `TestActNoValidTargetMudlog`; control `act-no-valid-target` |
| `src/db.c:2047-2053` | `"SYSERR: error in zone file: %s"`, NRM, LVL_GOD, file TRUE, macro argument verbatim | Six C `ZONE_ERROR` branches (`2174` P, `2187` G, `2202`/`2207` E, `2250` D, `2279` default) are the port's seven arms (D is split into direction-out-of-range and no-such-exit; each fires at most once per command, so their union is C's condition). The producer replaces the port's `slog.Warn` at each arm and precedes the default arm's reset-table mutation. | `World.zoneResetMu` only | `TestZoneErrorMudlogBranches`; control `zone-error-arms` |

`TestZoneErrorMudlogBranches` covers all seven triggers; the pre-existing
`TestZoneResetUnknownCommandDisabledOnce` now counts the producer rather than the
removed slog line, which strengthens its "exactly once" intent.

## Lock audit

`MudLog` writes its file side (the existing log-writer lock) and then visits a
session snapshot: `EachSession` snapshots under the manager mutex and releases it
before recipient traversal, observer flags/level are read and released before
sending, and the existing body delivery routes through switch-aware routing,
heartbeat staging and snooping.

- **Act arm:** no lock is held. `Act`/`actDeliver` take no world lock, and the
  producer runs before `world.actChar(roomVNum)`; callers that reach `Act` have
  already released their snapshots (the death path was audited in #1854).
- **Zone-error arms:** `executeZoneResetLocked` holds `World.zoneResetMu` and
  explicitly never `World.mu` (its own comment, `pkg/game/spawner.go:209-210`).
  `zoneResetMu` is acquired only by the reset entry points
  (`world_write.go:102`, `world_zone.go:162`, `zone_clock.go:13,33`); nothing on
  MudLog's file or delivery path takes it, so no delivery -> reset reverse edge
  exists. `GetObjNum`/`resetDoor` take and release `World.mu` internally before
  each producer. No world, player, manager, lifecycle or editor lock is held.

## Findings that reshape the rest of 1a (for review)

**1. `src/spell_parser.c:534` is unreachable in C — no producer added.**
Every `MAG_MANUAL` entry in the C spell table has a case in the manual switch
(26 `spello(...MAG_MANUAL)` entries, 26 cases, `src/spell_parser.c:502-536`), so
C's `default:` is a defensive arm for a corrupt spellnum. Classification:
`excluded-valid-play`, with that bounded table/switch audit as the evidence.

**2. Live divergence found at that same arm (stop tier, separate train).**
`SpellDominate` is `76` (`pkg/spells/spells.go:154`), registered
`RoutineManual` (`pkg/spells/affect_spells.go:1543`) and granted to classes at
level 19/20 (`pkg/game/class_spells.go:137,310`) — but `ExecuteManualSpell`
(`pkg/spells/affect_spells.go:1200`) has **no** `case SpellDominate`, where C
routes `SPELL_DOMINATE` into `spell_charm` (`src/spell_parser.c:506-507`). A
live cast therefore reaches Go's `default:` arm and sends
`"Spell not yet implemented.\r\n"` — a string that appears **nowhere in C** — and
runs no charm effect. R1/R4. **No producer was added at that arm:** doing so
would emit bytes C never emits, the same class of error as `comm.c:784`. It needs
its own routing/behaviour train.

**3. `spec_procs2.c:634` (`kender_steal`) — the ruling is confirmed.**
`src/spec_procs2.c:598-599` returns for every non-NPC victim, so the `:628` arm
(`!IS_NPC(victim) && ...`) is dead code; `src/act.informative.c:487-489` does
pass any room occupant, but the callee rejects PC victims on entry. The port's
`KenderSteal(mob-only)` boundary and its comment are correct. Inventory row
corrected to `excluded-valid-play`.

**4. `lua-seams`: seven of eight sites are behaviour fixes, not log-only.**
C returns **0** from the bad-argument arm of `lua_canget` (`:216`),
`lua_direction` (`:328`, `:337`), `lua_iscorpse` (`:651`), `lua_isfighting`
(`:671`) and `lua_item_check` (`:753`); the port returns `1` with a value pushed
(`pkg/scripting/engine.go:1907-1909, 2509-2518, 2530-2533, 2582-2586`), and
`luaDirection` has no argument-type gate at all. `lua_item_check`'s "Unable to
determine shop" arm (`:739`) is not separable: the port's `ShopBuysType` bool
collapses "not a shopkeeper" and "shop does not buy this type"
(`pkg/game/world_zone.go:110`). Per the ruling these move to **1b**. The one site
whose return already matches C is `lua_skip_spaces` (`:1394`): C logs in the
`else` and still returns 1.

Also: the inventory's `go_owner` for `scripts.c:216` ("no registered canget
binding") is wrong. All eight bindings are registered — `canget`
(`engine.go:808`), `iscorpse` (`:807`), `isfighting` (`:767`), `item_check`
(`:783`), `skip_spaces` (`:793`), `direction` (`:790`) — so the group is
reachable through real Lua calls with a bad argument.

**5. `db.c:2057` needs a parse-time source line.** `parser.ZoneCommand`
(`pkg/parser/zon.go:26-32`) carries `Command`, `IfFlag`, `Arg1-3` and no `Line`,
so the ordered second line has no port representation. 1a landed the paired first
line only; the inventory row records this as the PR 1b design decision (add
`Line`, filled from the 1-based zone-file line, never serialized).

## Remaining for 1a

Everything Claude listed for 1a is landed or classified except `scripts.c:1765`:

1. `scripts.c:1394` (`lua_skip_spaces`) — landed (BRF/LVL_IMMORT/file FALSE via
   the existing `Bridge.Log`).
2. `scripts.c:1694` (reachable kinds) and `:1779` — landed (CMP/31/file TRUE and
   BRF/31/file TRUE, in C's order, through the new adapter sink).
3. `scripts.c:1800` — landed (BRF/31/file TRUE, before the write-back).
4. `scripts.c:1765` (unassigned script) — **moved to 1b, not landed.** C logs
   `"SYSERR: Attempting to call unassigned script for %s (#%d)."` and then
   returns TRUE. The port's counterpart is `MobInstance.RunScript`, which
   reaches `Engine.RunScript("")` and returns an error, so a log-only insert
   would sit on a path whose return value already differs from C's — the same
   class the lua-seams ruling sends to 1b. The inventory row carries that reason.
5. The file-TRUE sink itself (the ruled adapter passthrough) is landed, with a
   compile-time assertion in `pkg/game/scripting_adapter_mudlog_test.go`.

Left before this PR can be opened: nothing outstanding in code; the tree still
needs its combined census at this tip, run from this worktree in the single
shared census slot.


## Reproduce

```sh
go test ./pkg/game -run 'TestActNoValidTargetMudlog|TestZoneErrorMudlogBranches' -count=1
python3 docs/fidelity/depth/handoff/2026-10-08-dp-1371-mudlog-engine-diagnostics-controls.py --output /absolute/evidence/path
go test -race ./pkg/game -run 'TestActNoValidTargetMudlog|TestZoneErrorMudlogBranches' -count=1
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
```

Both controls print `1 -> 0 -> 1` and require the named test's assertion failure
(`file payload count=0 want 1`), never a build failure. Gates at this tip:
`gofumpt -l` clean, `go build ./...` 0, `go vet ./...` 0, `go test ./...` 0
(45 ok packages). No census has run for this train yet; it runs once 1a is
complete, in the single shared census slot.

