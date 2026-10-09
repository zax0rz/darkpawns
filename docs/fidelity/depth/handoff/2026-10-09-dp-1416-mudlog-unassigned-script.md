# DP-1371 D7: mudlog DP-1416 — the unassigned-script producer (tier: stop)

Base `origin/main` `c46dbf89a` (PR 5 merged). R1/R2/R4/R5e/R5g/R5h.

Ports C's `run_script` empty-name arm (`src/scripts.c:1763-1767`) at the three
owner types, plus the object-onpulse divergence Zach approved on 2026-10-09.

## The C producer and its gate

```c
if (!*script_name) {
  sprintf(buf, "SYSERR: Attempting to call unassigned script for %s (#%d).",
          GET_NAME(me), GET_MOB_VNUM(me));
  mudlog(buf, BRF, LVL_IMMORT, TRUE);
  FREE(script_name);
  return(TRUE);
}
```

- **Payload:** `SYSERR: Attempting to call unassigned script for %s (#%d).` at
  **BRF / LVL_IMMORT / file TRUE**, no trailing CRLF; then `return TRUE`.
- **Gate:** every caller tests "the script record exists **and** the trigger bit
  is set", never the name (`src/mobact.c:148`, `:161`, `:180`;
  `src/interpreter.c:1419`, `:1430`, `:1443`, `:1459`, `:1470`;
  `src/act.movement.c:282`, `:305`; `src/act.item.c:212`, `:284`, `:506`, `:704`,
  `:760`; `src/fight.c:597`, `:1891`; `src/comm.c:708`, `:790`).
- **Position:** after the name is read, before any script file is loaded.

## Reachability (R5e) — the ruling's route, and its correction

The state is "the owner's script record carries a flag but no name". C creates
**every** record with `name = NULL` at load: mobs `src/db.c:1278-1279`, rooms
`:803-804`, objects `:1374-1375`. So a builder who sets a trigger flag in the OLC
script menu without ever setting a name reaches the arm.

The DP-1416 ruling named "clear the name" via medit. That route is **gated in C**:
`medit_parse` rejects empty input for every mode above `MEDIT_NUMERICAL_RESPONSE`
(`src/medit.c:718-722`; `MEDIT_SCRIPT_NAME` is 28 > 10, `src/olc.h:218`, `:236`),
so medit's own `!strcmp(arg, "")` branch (`src/medit.c:996`) is unreachable. The
clear-an-existing-name route is real for **rooms** (`src/redit.c:1032-1037`, no
numeric gate in `redit_parse`) and **objects** (`src/oedit.c:1447-1451`, none in
`oedit_parse`), but not for mobiles. The state, and therefore the producer, is the
same either way; only the mobile route is flag-without-name.

Go already reproduces all three: `pkg/session/medit.go:650` applies C's gate and
`:739` notes the branch is unreachable; `pkg/session/redit.go:403-406` and
`pkg/session/oedit.go:1250` (`setOeditScriptNameLocked`, `:1371-1376`) write the
line through, including an empty one. **No OLC change is needed.**

## `me` per owner (the check the ruling asked for)

From every `run_script` caller (`src/scripts.c:51`):

| owner | caller `me` | payload |
|---|---|---|
| `LT_MOB` | the mobile | `GET_NAME` = short description, `GET_MOB_VNUM` = prototype vnum |
| `LT_ROOM` | `ch`, the actor (always a PC: `interpreter.c:1421` gates `!IS_NPC`) | player name, `-1` |
| `LT_OBJ` oncmd (worn/carried) | `ch` (PC) | player name, `-1` |
| `LT_OBJ` oncmd (room contents, `interpreter.c:1470-1476`, no `IS_NPC` gate) | `ch`, which may be a switched mobile | player name `-1`, or mobile short description + vnum |
| `LT_OBJ` onpulse | **NULL** (`src/comm.c:791`, `:793`) | C faults — divergence |

`GET_MOB_VNUM(mob)` is `IS_MOB(mob) ? mob_index[GET_MOB_RNUM(mob)].virtual : -1`
and `IS_MOB` is false for a PC (`src/utils.h:230-231`, `:431-432`), which is the
`-1`.

## The object-onpulse divergence (DP-1416, approved)

`object_activity` calls `run_script(NULL, NULL, obj, …)` (`src/comm.c:789-793`),
so `me` is NULL and the empty-name arm formats `GET_NAME(NULL)` / `GET_MOB_VNUM(NULL)`
— `IS_NPC(NULL)` faults before a byte is emitted (undefined behaviour, R1a). Go
cannot reproduce the fault, emits nothing, and does not panic; C ignores this
trigger's return value, so there is no consume difference. Recorded as case
`mudlog.lua-unassigned-script-object-onpulse` (`divergent-approved`, DP-1416).

## Go changes

- `pkg/game/scripts.go`
  - `MobInstance.HasScript` drops the `ScriptName != ""` test: C's gate is
    flag-only, and a set flag implies the record exists (`:36-60`).
  - `MobInstance.RunScript` gains the empty-name branch: emit and `return true, nil`
    (`:78-85`). The `true` consumes the trigger at the `onpulse_all` and `oncmd`
    callers, as C's RETURN does.
  - `scriptUnassignedProducer(name, mobVNum)` is the single producer (`:97-116`).
- `pkg/game/room_obj_scripts.go`
  - `objHasScriptTrigger` drops the name test (`:34-40`).
  - `runRoomScript`, `RunObjOnCmdScript` gain the empty-name branch and return
    `true` (consume) (`:85-92`, `:124-133`).
  - `objScriptActorPayload` resolves `me`'s name and vnum for the object oncmd
    actor (`:43-60`).
  - `RunObjPulseScript` gains the divergence guard: no producer (`:160-169`).

## Lock audit

| Producer | Locks held at `MudLog` |
|---|---|
| `scriptUnassignedProducer` (all owners) | None. Every call site is where `ScriptEngine.RunScript` would otherwise run, after the owner lookup's `World.mu.RLock` has been released (`GetRoomInWorld`, `GetMobByID`), and the mob path takes no lock. The tests assert this with `w.mu.TryLock()` inside the delivery probe. |

## Reproduce

```sh
go test ./pkg/game -run 'TestUnassignedScript' -count=1
python3 docs/fidelity/depth/handoff/2026-10-09-dp-1416-mudlog-controls.py --output /absolute/evidence/path
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
```

Controls (five cases, each `1 -> 0 -> 1` on its named assertion):
`producer-bytes` (the payload mutated), `mob-arm`, `room-arm`, `obj-oncmd-arm`
and `onpulse-divergence` (each owner's empty-name branch disabled, so the engine
runs on an empty name and the proof's "the engine ran on an unassigned script"
assertion fires).

## Stop-and-report history

The `me` check stopped the port on 2026-10-09: the object-onpulse call passes
`me = NULL`, so C faults. Zach ruled **(a)** — port the three defined
owner/trigger combinations and record object-onpulse as an approved divergence
under DP-1416 — and this PR implements that. The R5e correction above (medit is
gated, redit/oedit are not) is recorded rather than acted on: Go already matches
C on all three OLC surfaces.

