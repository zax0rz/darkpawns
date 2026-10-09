# DP-1371 D7: mudlog DP-1404 — the stored rent code and Crash_load's four live entry producers (tier: stop)

Base `origin/main` `a37bbd123`. R1/R2/R4/R5e/R5g/R5h.

Zach approved the save-format change on 2026-10-09 (DP-1404). The approval is
narrow: Go stores C's rent code, which in practice is a **last-exit code**, and
this PR ports only the four `Crash_load` entry arms reachable under
`free_rent = YES` (`src/config.c:106`). The rent-receptionist path
(`objsave.c:1186`), the corrupt-cost arm (`:504`), the force/timeout arm
(`:534`), the default arm (`:539`) and DP-1419 (the saved-equipment
`check_for_bad_stats` pass) are out of scope.

## Recon answer: the representation already existed

The DP-1404 recon asked whether `Player.RentedOut` or the object snapshot's
`kind` already carries the information. It does — **`object_saves.kind`**, not
`RentedOut`:

| kind | C code | written by (Go) | C path |
|---|---|---|---|
| 1 | `RENT_CRASH` | `pkg/game/persistence_seam.go:65` (guarded by `SaveCrash` + `SaveSucceeded`) | `Crash_crashsave` (`src/objsave.c:785`, `:1217`) |
| 2 | `RENT_RENTED` | `pkg/session/cmd_inventory.go` (quit rent), `pkg/game/limits_misc.go:105` (idle force-rent) | `Crash_rentsave` (`src/objsave.c:912`), `ch, 0` from `src/act.other.c:167` and `src/limits.c:446` |
| 3 | `RENT_CRYO` | `pkg/session/cmd_inventory.go` (PLR_NODELETE quit) — **added here** | `Crash_cryosave` (`src/objsave.c:959`), `ch, 0` from `src/act.other.c:164-165` |
| — | (no file) | no row | `fopen` fails: every new character's first entry (`src/objsave.c:474-489`) |

So this is a 1:1 carry on the existing column, **not a new column**, and no
migration default is needed: the two values Go already wrote coincide with C's
`RENT_CRASH` and `RENT_RENTED`, and `db.ObjectSaveLoaded` already performs
`Crash_load`'s header rewrite to `RENT_CRASH` (`src/objsave.c:659-663`).
`Player.RentedOut` is a bool and cannot carry the code, so it is left alone.

## The C selection

`Crash_load` reads the header, emits the selected arm, loads the objects, then
rewrites the header (`src/objsave.c:459-663`):

- `:488-489` `"%s entering game with no equipment."` — NRM, `MAX(LVL_IMMORT, invis)`, TRUE (failed `fopen`).
- `:519` `"%s un-renting and entering game."` — NRM, same level, TRUE (`RENT_RENTED`).
- `:524` `"%s retrieving crash-saved items and entering game."` — NRM, same level, TRUE (`RENT_CRASH`).
- `:528` `"%s un-cryo'ing and entering game."` — NRM, same level, TRUE (`RENT_CRYO`).

All four name `GET_NAME(ch)` and log inside `CON_MENU` (`src/interpreter.c:2184`),
so the entering descriptor is not itself an audience member (`mudlog` delivers
only to `CON_PLAYING` descriptors, `src/utils.c:212-242`).

## Every Go exit path mapped to C's code

| Go exit path | C path | code |
|---|---|---|
| `quit` in a temple/home/owned room, or any immortal | `do_quit` → `Crash_rentsave(ch, 0)` (`src/act.other.c:155-167`) | RENT_RENTED |
| `reallyquit` in a save room, or immortal | same block (its condition is `isokquit || level >= LVL_IMMORT`) | RENT_RENTED |
| `quit` with `PLR_NODELETE` in a save room | `do_quit` → `Crash_cryosave(ch, 0)` (`src/act.other.c:164-165`) | RENT_CRYO |
| idle force-rent | `check_idling` → `Crash_rentsave(ch, 0)` (`src/limits.c:445-446`) | RENT_RENTED |
| link loss (linkdead) | no save; the idle path rents on the later threshold | — |
| `save` | `do_save` → `Crash_crashsave` (`src/act.other.c:199`) | RENT_CRASH |
| autosave | `Crash_save_all` (`src/objsave.c:1213-1220`) | RENT_CRASH |
| shutdown / reboot | `Crash_save_all` (`src/objsave.c:1217`) | RENT_CRASH |
| idle void pull | `save_char` + `Crash_crashsave` (`src/limits.c:434-435`) | RENT_CRASH |
| `reallyquit` outside a save room (LOSTEQ) | no `Crash_rentsave`; extraction then deletes a file still holding `RENT_CRASH` (`src/handler.c:1163`) | none → the no-file arm |
| new character's first entry | no file | — (no-file arm) |

`RENT_FORCED` (4) and `RENT_TIMEDOUT` (5) are never written under free rent:
`RENT_TIMEDOUT` comes only from `Crash_idlesave` (`src/objsave.c:883`), which
runs only when rent is not free (`src/limits.c:447-448`), and `RENT_FORCED` is
never written at all. Those arms stay `excluded-valid-play` in the inventory.

## Go changes

- `pkg/session/crash_load.go` (new)
  - `rentCrash`/`rentRented`/`rentCryo` name C's codes (`src/structs.h:587-591`)
    on the existing `object_saves.kind`.
  - `Session.crashLoadEntry` reads the stored code, emits the selected entry
    line, then calls `db.ObjectSaveLoaded` (the header rewrite). A store that
    does not implement `GetObjectSave` (a `db.GameStore` wrapper) keeps the old
    behaviour: rewrite only.
  - `crashLoadEntryLine` is the switch; an absent row is the no-file arm. The
    force/timeout and default arms emit nothing (excluded-valid-play).
- `pkg/session/menu.go` — `enterReturningPlayer` calls `s.crashLoadEntry()` at
  the old `db.ObjectSaveLoaded` site, which is C's position: after
  `prepareWorldEntry` (reset_char + the INVSTART gate, `src/interpreter.c:2174-2184`)
  and before the entry save (`:2186`).
- `pkg/session/char_creation.go` — `completeCharCreation` calls the same helper
  before its entry save; a brand-new character has no row, so this is the
  no-file arm.
- `pkg/session/cmd_inventory.go` — the quit-rent snapshot now passes
  `rentCryo` for a `PLR_NODELETE` quitter and `rentRented` otherwise
  (`src/act.other.c:164-165`). The two C functions' object passes are identical
  under free rent, so only the code differs.
- `pkg/db/object_save.go` + `pkg/session/manager.go` — `DeleteCrashObjectSave`
  is `Crash_delete_crashfile` (`src/objsave.c:177-201`), called at PC extraction
  (`src/handler.c:1163`) from the site that already reproduces the adjacent
  `save_char`. See the section below for why it lands in this PR.

### Order

C emits the entry line before loading the objects and before `do_start`'s
`advance_level` producer. Go's object restore happens earlier (the store record
is built at login), which is invisible to players: the producer is an
independent immortal-facing channel. What this PR pins is the order that is
observable — the entry line precedes the header rewrite, the entry save, and
`advance_level` (the first-entry proof asserts
`entering game with no equipment.` immediately before `advanced to level 1`).

## Lock audit

| Producer | Locks held at `MudLog` |
|---|---|
| `crashLoadEntry` (all four arms) | None. `crashLoadEntry` takes no lock: `GetObjectSave` and `ObjectSaveLoaded` run on the SQLite handle, and `MudLog` runs after both. `enterReturningPlayer` holds no manager/world lock at that point (`menuActive` is cleared later), and `completeCharCreation` holds none. |

## Reachability

- **No file** — every new character's first entry (the creation proof), and any
  returning character who never produced an object save.
- **RENT_RENTED** — every ordinary re-login after a legal quit or an idle
  force-rent.
- **RENT_CRASH** — any login after `save`, an autosave, a shutdown/reboot save
  or an idle void pull.
- **RENT_CRYO** — a `PLR_NODELETE` character's quit, then a later login. The
  write side is proved with the real `quit` command.

## The PLR_CRYO finding (reported, not ported)

C's `Crash_cryosave` also sets `PLR_CRYO` (`src/objsave.c:1005`), and
`nanny` clears it at the password prompt (`src/interpreter.c:1791`). Nothing in
C **reads** it — the flag is write-only in C (the only other write is the
receptionist's cryo arm, `src/objsave.c:1183`). It therefore has no
player-facing byte to reproduce, and Go does not set it (R4: no invented
mechanism). This is recorded for the next sweep rather than acted on.

## Reproduce

```sh
go test ./pkg/session -run 'TestCrashLoad|TestQuitStoresLastExitRentCode|TestExtractionDeletesOnlyCrashFiles' -count=1
python3 docs/fidelity/depth/handoff/2026-10-09-dp-1404-controls.py --output /absolute/evidence/path
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
DP_ORACLE_BIN=$HOME/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario mudlog-crash-load-entry --seed 1 --show-oracle
```

Controls (seven cases, each `1 -> 0 -> 1` on its named assertion): one per arm's
payload bytes (`no-equipment`, `unrenting`, `crash-saved`, `cryo`), the cryo
**write** (`rent-code`), the producer itself (`producer-off`), and the extraction
**delete** (`crash-file-delete`). Each mutates a source literal, condition or
call so the package still compiles and the named test fails on its assertion,
never on a build error.

## The extraction delete (ported here, flagged as a scope call)

Reviewing the exit paths turned up a C call the port never made:
`extract_char_final` deletes the crash file at **every PC extraction**, but only
when the header still holds `RENT_CRASH` (`src/handler.c:1163`;
`Crash_delete_crashfile`, `src/objsave.c:177-201`). Go's extraction had no such
delete, so the stored code outlived the file C would have removed:

- a legal quit rewrites the header to `RENT_RENTED` first, so C keeps the file → the next entry logs `:519`;
- a **LOSTEQ** quit skips `Crash_rentsave`, so the file still holds `RENT_CRASH` and C **deletes** it → the next entry is the **no-file arm (`:489`)**, not `:524`.

Without this the producer I am adding emits a wrong byte in a reachable case
(save, walk out of a save room, `reallyquit`, log back in), which is an R1
defect regardless of who introduced it, so it is ported here rather than left as
a known-wrong line. It is one call site, it changes no schema, and it is a
separate commit so it can be dropped if you rule it belongs in its own train.
`pkg/session/manager.go` now calls `db.DeleteCrashObjectSave` at the extraction
site that already reproduces the adjacent `save_char` (`:1162`), on both the
ordinary and the switched early-return path.

This is also an **R5e correction** to the review suggestion that "save, then a
quit outside a save room (LOSTEQ), then `1`" would reach `:524`: on C's actual
call path that sequence reaches `:489`, because extraction deleted the file. The
vehicle's first arm is exactly that sequence and asserts `:489`.

## Oracle

`cmd/dp-oracle-diff/scenarios/mudlog-crash-load-entry.txt`, claimed at seeds
**1, 2, 3, 5, 8**. A first-player God (the primary client) sets `syslog normal`
and watches two mortals enter from the menu:

| arm | probe | observed |
|---|---|---|
| `:489` | `save`, `reallyquit` (LOSTEQ, still in the newbie room), menu `1` | `[ Ren entering game with no equipment. ]` — and it proves the extraction delete, since a crash file existed |
| `:519` | `quit` in the temple, menu `1` | `[ Ren un-renting and entering game. ]` |
| `:528` | `set Cryo nodelete on`, `recall`, `quit`, menu `1` | `[ Cryo un-cryo'ing and entering game. ]` |

Two harness details worth recording: the probe is `[probe:ren]`, so the God is
the `primary` audience observed via `send:primary ~dpclock pulse 20`; and the
scenario needs inert `[relogin:*]` sections because that is how a scenario
declares the port's real SQLite store (`needsPlayerStore`) — without one the port
boots with no database and `crashLoadEntry` correctly emits nothing
(`lifecycle-quit-reenter` is the model vehicle).

**`:524` stays unit-green, deliberately.** A retained `RENT_CRASH` file requires
an unclean or unextracted stop, and the harness's `<CRASH>` and `<RESTART>`
vehicles both forbid passive peers, so no in-game immortal can witness that
relogin; `TestCrashLoadEntryProducers/crash-saved` proves the arm's bytes
instead. Every route to `:524` that a witness *could* watch is closed by the
file's lifetime.

## Stop-and-report history

Zach approved the save-format change (DP-1404, 2026-10-09) and Claude narrowed
it to the rent code after the `free_rent = YES` finding. Recon then showed the
representation already existed (`object_saves.kind`), so the approved format
change landed as one new code value (RENT_CRYO) rather than a new column or a
migration. DP-1419 (the saved-equipment `check_for_bad_stats` pass) runs inside
the same `Crash_load` and is deliberately sequenced **after** this PR.

**Two items for the reviewer, recorded rather than improvised.** (1) The
extraction-time crash-file delete (above) is the one piece of scope beyond the
rent code and its four producers; it is in its own commit and is there because
the producers are wrong without it (R1). If you rule it belongs to its own
train, drop that commit and this PR's `:489` oracle arm becomes the no-file case
only. (2) Review suggested that `save` → LOSTEQ → `1` reaches `:524`; on C's
call path it reaches `:489`, because extraction deleted the file — the vehicle
asserts `:489` and the `:524` row records the correction and why it stays
unit-green.
