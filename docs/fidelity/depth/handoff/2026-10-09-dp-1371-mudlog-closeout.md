# DP-1371 D7: mudlog PR 3 — closeout (tier: local, mudlog-only)

**Local, mudlog-only.** Base `origin/main` `3e771df2a`. R1/R2/R4/R5e/R5h.

The code in this PR is three producer changes (A1, A2, C1) plus C2's audit and
its bounded caller proof; everything else is inventory, a survey and
`2026-10-09-dp-1371-mudlog-decisions.md`, which is one table for Zach.

## A1. `limits.c:281` "Initiating autowiz."

C: `do_advance` promotes through one `gain_exp_regardless` per level
(`src/act.wizard.c:1569-1571`); inside it, after the rise message,
`check_autowiz(ch)` (`src/limits.c:357-360`) logs `Initiating autowiz.` at
**CMP / LVL_IMMORT / file FALSE** when `!mini_mud && use_autowiz &&
GET_LEVEL(ch) >= LVL_IMMORT` and then runs `system("nice ../bin/autowiz …")`.

- The producer is emitted at exactly that position
  (`pkg/game/limits_exp.go:CheckAutowiz`, called from `gainExpRegardless` after
  the rise message). Nothing is spawned: the reference oracle has no autowiz
  binary and `src/util/` no source, so C's `system()` fails quietly there too.
  Go's `wizlist` and `immlist` are read as static files, so nothing needs
  regenerating (reported, not changed).
- **Gate:** `use_autowiz` is YES (`src/config.c:273`) and the port has no such
  flag, so YES is written directly rather than inventing one (R4). `mini_mud` is
  set only by C's `-m` startup option (`src/comm.c:194-196`); the port has no
  mini-mud mode, so `!mini_mud` always holds. Level ≥ `LVL_IMMORT` is the same
  test in both.
- **Position and per-call framing (review round).** C's `check_autowiz` is per
  *call* of `gain_exp_regardless`, and do_advance's promotion loop calls it once
  per level, so the promoted character's own stream interleaves
  `You rise a level!` with `[ Initiating autowiz. ]` per level. The port batched
  do_advance's announcements in the caller and used a `announce` flag to
  suppress them, which put every autowiz line ahead of every rise line.
  `gainExpRegardless` now owns both lines exactly as C does
  (`src/limits.c:350-360`) and do_advance's loop calls
  `GainExpRegardless(victim, gain)` per level; the `…Silent` variant is gone.
  Two byte facts came out of that: the rise message must go through
  `p.SendMessage` (it already ends in CRLF; `sendToChar` appends a second one,
  which the pre-fix `announce` path would have exposed), and each line is now
  its own queued frame, so the concatenated wire bytes are C's while a
  WebSocket client sees one message per line instead of one batched message
  (content unchanged; no prompt is added — `outputSincePrompt` only asks
  whether output was queued).
- **Oracle vehicle:** `advance <peer> 31` by an implementor watched by an
  independent complete-syslog immortal. No corpus scenario stages it, so the
  claim is unit-green; the seeds are named in `mudlog.tsv` only when a scenario
  covers it.
- **Lock audit:** none held. `gainExpRegardless` runs in the game loop with no
  world/player/manager/zone lock at the call, and `MudLog` takes its own read
  locks after.

## A2. `scripts.c:791` Lua `log` success writes the file

C (`lua_log`, `src/scripts.c:780-793`): the bad-argument arm logs
`"[Lua] Invalid argument passed to lua_log."` at BRF / LVL_IMMORT / **FALSE** and
returns 1; the success arm logs the text at BRF / LVL_IMMORT / **TRUE**. The
port sent both arms through the file-FALSE sink.

`bridgeLog` now writes the success arm through the existing file-TRUE helper
`scriptMudLogFile` (`pkg/scripting/engine.go:403`); the bad arm is untouched.

Test note worth keeping: **C's `lua_isstring` accepts numbers**, so
`log(42)` takes the *success* arm and logs "42" at file TRUE, and the
bad-argument arm needs a table. `TestLuaLogProducerFileFlag` covers all three
(success, number-coerced success, table) through real Lua runs.

**Unit-green only:** no shipped non-archive script calls `log()` —
`lib/world/scripts/globals.lua` defines an `_ALERT` helper that nothing calls —
so there is no reachable oracle vehicle. The single non-archive call is an
error seam (`mob/assembler.lua:68`, "Could not return object.").

## C1. `utils.c:724` "%s started hunting %s"

C's `set_hunting` (`src/utils.c:708-729`) clears the old target, then: a mobile
prey is stored silently (`IS_MOB`); a prey with `GET_IDNUM(vict) > 0` logs
`"%s started hunting %s"` at **CMP / LVL_IMMORT / file FALSE** *before* it stores
the id. Anything angers a `MOB_HUNTER` mob, so this is ordinary play and cannot
be excluded.

`LogHuntingStart(hunter, prey *Player)` (`pkg/game/hunting_mudlog.go`) is called
at each site that holds a **typed player prey**, before the assignment:

| Go site | C caller |
|---|---|
| `pkg/game/combat_wire.go` `RangedHunt` (via `combat/engine.go`) | `src/fight.c:1453-1455`, `damage()`'s hunter arm |
| `pkg/game/spec_procs2.go` assassin hire | `src/spec_procs2.c:893-907` (`get_player_vis`) |
| `pkg/command/skill_commands.go` training spawn | `src/new_cmds2.c:609-613` (`create_mobile`, also gated `!mini_mud` in C) |
| `pkg/session/wiz_zone.go` `sethunt` | `src/act.wizard.c:3443-3472` (`get_char_vis`, player or mobile) |
| `pkg/game/world_bridge.go` `SetHunt` (Lua `set_hunt`) | `src/scripts.c:1343-1360` |

A mobile prey is silent at every site: a `*Player` is never `IS_MOB`, and the
combat and `sethunt` sites assert the type before logging.
`TestHuntingMudlogProducer` drives the real `sethunt` command (player quiet/log,
mobile silent, id-less silent, level and brief-syslog filters, file FALSE) and
`TestLogHuntingStartGates` covers the helper's gates and `damage()`'s arm.

**Name-only sites are listed, not guessed** (decisions table §4):
`World.SetHunting(hunterName, preyName, …)` (its single caller logs at the call
site), the `luaSetHunt` no-bridge fallback (engine-test-only path), the mobile
equipment pestilence site (`IS_MOB` → silent by C), and the `NULL` clears.
**How hunting is stored is unchanged** — Go still stores a prey name, and
`HuntingID`/`HuntingMobID` keep their existing behaviour.

## C2. `fight.c:513` death_cry NOWHERE — audit finished

Full audit, the two guards, the mount arm, the Lua surface and the residual are
in `2026-10-09-dp-1371-mudlog-decisions.md` §3. Summary:

- `raw_kill` re-checks `ch->in_room == NOWHERE` (`src/fight.c:536`) and never
  reaches `death_cry` with it; `do_move` re-checks the **rider**
  (`src/act.movement.c:279-281`) but **not the retained mount pointer**
  (`:195` … `:298`).
- The Lua surface can clear a character's room (`extchar` + `inworld`), but
  `tport` refuses an invalid destination before moving anything
  (`src/scripts.c:1531-1534`), and **no non-archive shipped script extracts a
  character** — the extractors are archive content.
- Go is structurally stronger: `deathTrapMount` re-resolves the mount in the
  rider's room and returns early when it is gone, so Go never passes a cleared
  room to `deathCry`.
- `TestDeathCryBoundedCallers` (`pkg/game`) is the fail-capable bounded caller
  proof: the rider's cry lands in the real trap room, and a rider whose mount
  has already left produces no mount cry (removing the re-resolution fails it).

Recommendation: `excluded-valid-play`, with the residual stated (the exclusion
rests on the guards plus the absence of shipped content, not on a mechanism that
makes the arm impossible). No producer was added and no body placement was
designed, as instructed.

## B. Relabels applied (the two Claude settled)

- `src/spell_parser.c:534` → **`excluded-valid-play`** (26 `MAG_MANUAL` spellos,
  26 cases; `src/spell_parser.c:502-536`). The DP-1405 dominate bug beside it
  was fixed in #1859.
- `src/comm.c:2235` → the note now says **SIGUSR1** (`src/comm.c:2303`); the
  current inventory already had SIGUSR1 in `go_owner`, so this is a note
  correction plus the bounded audit, not a status change.

Everything the brief marks "recommend" still has its old state
(`comm.c:2245`, `spec_procs2.c:371`, `zedit.c:264`/`:270`, `medit.c:364`,
`sedit.c:486`, `oedit.c:422`, `ident.c:235`, `scripts.c:1765`, the eight rent
rows and the four `whod.c` rows). They wait for Zach in the decision table.

## D. The 57 `implemented-needs-proof` Lua rows

`2026-10-09-dp-1371-lua-needs-proof-survey.tsv` — one line per row with
`c_bad_arg_return`, `go_binding`, `go_bad_arg_return`, `matches`,
`payload_matches`, `reachable_via_real_lua_call` and notes. Read-only; nothing
was fixed.

**Result: 56 of 57 match C.** Every bridge binding returns C's value from the
bad-argument arm and logs C's exact payload; the flag families
(`mob_flags`, `plr_flags`) build the string from the global's name, which
reproduces C's text byte for byte.

The one mismatch is C's own bug: `src/scripts.c:1532` (`lua_tport`, invalid
room) ends `return -1`. A C function's return value is its number of Lua
results, so C hands Lua a **negative result count** and the caller reads below
its frame — undefined behaviour (R1a). The bridge returns 0 (no results), which
is the only defined reading. PR 4 should classify this `divergent-approved`/R1a
rather than reproduce it.

**Class note for PR 4 (not a row defect):** the *no-bridge* fallbacks in
`pkg/scripting/engine.go` are a second implementation of the same globals, and
by the file's own comment they run only for "engine tests that construct a world
without one". Twelve of them neither emit the producer nor return C's value
(`luaAffFlagged`, `luaCanSee`, `luaExitFlagged`, `luaInworld`, `luaIshunt`,
`luaIsNPC`, `luaLoadRoom`, `luaMobFlagged`, `luaOLoad`, `luaObjFlagged`,
`luaObjList`, `luaPlrFlagged` — each returns 1 with a pushed `nil`/`false` where
C logs and returns 0). The survey's `notes` column names each one. That is the
same shape as the #1859 seam finding, one layer down; whether to align or retire
those fallbacks is PR 4's call.

## Lock audit

| Producer | Locks held at `MudLog` |
|---|---|
| `Initiating autowiz.` | None. `gainExpRegardless`/`CheckAutowiz` run in the game loop with no world, player, manager or zone lock held; delivery takes its own read locks. |
| Lua `log` (file TRUE) | None beyond the scripting engine's own `e.mu`, which `RunScript` already holds and which delivery never takes (same as the 1b/1c script producers). The sink is a stateless forward to `game.MudLog`. |
| `started hunting` | None. The five call sites emit before their assignment, outside `World.mu` and outside any player lock; `LogHuntingStart` itself takes nothing. |
| `Losing player:` / `Losing descriptor` (PR 2) | None (PR-2 handoff §4). |

No producer in this PR needs a world, player, manager or zone lock.

## Reproduce

```sh
go test ./pkg/session -run 'TestAutowizMudlogProducer|TestHuntingMudlogProducer' -count=1
go test ./pkg/scripting -run 'TestLuaLogProducerFileFlag|TestBridgeTables|TestLua4StringFunctions' -count=1
go test ./pkg/game -run 'TestLogHuntingStartGates|TestDeathCryBoundedCallers' -count=1
python3 docs/fidelity/depth/handoff/2026-10-09-dp-1371-mudlog-closeout-controls.py --output /absolute/evidence/path
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
```

Controls (six cases, each `1 -> 0 -> 1` on its named assertion):
`autowiz-producer` (the producer removed), `autowiz-interleave` (the caller
batches the announcements again — the ordering defect this round fixed),
`autowiz-rise-bytes` (`sendToChar` instead of `SendMessage`, which doubles the
rise line's CRLF), `lua-log-file-flag`, `hunting-producer` and
`deathcry-mount-guard`.

## Stop-and-report

1. **A hunting call site carrying only a name** — see the decisions table §4
   (`World.SetHunting`, the `luaSetHunt` fallback). Reported, not guessed.
2. **`death_cry` at NOWHERE** — no legal path found, but the exclusion rests on
   guards plus the absence of shipped content; the residual is stated in §3 of
   the decisions table rather than papered over.
3. **The `log(42)` coercion** is not a bug: C's `lua_isstring` accepts numbers,
  so the port's success arm is correct there.
4. **Three behaviour gaps found while sweeping C's `set_hunting` callers**, out
   of this PR and filed by Claude: `hunt_items` is dead in Go
   (`engine.GameLoopCallbacks.OnHuntItems` is never assigned, so the hourly
   hook never runs), `specMallory` only barks where C's mickey takes revenge
   with `set_hunting(mickey, FIGHTING(ch))`, and `check_for_bad_stats` has only
   its mobile path ported (C's player path is missing). Each is a *feature*
   gap, not a log, so none of them becomes a producer here.

