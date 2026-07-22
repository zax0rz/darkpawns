# BRIEF 2026-07-13 — Domain 3 Chunk 2: Unified `do_gen_door` (containers + doors) — THE ANCHOR (GPT)

**Executor:** ChatGPT/GPT. **Branch:** `refactor/domain-object-inventory` (chunk 1 already merged to main via
PR #303; `git fetch && git checkout main && git pull && git checkout -b <keep-or-reuse>` — reuse the domain
branch, base off current `main`). **Reviewer/merger:** Claude, against `origin/main` + `src/act.movement.c`,
gated by the oracle. This is the biggest chunk of Domain 3 — the architectural one. It's fine to land it as
its own PR.

**Read first:** the parent brief `BRIEF-2026-07-13-domain3-object-inventory.md` (§2, §3, §5 especially) and the
chunk-1 result (PR #303) for the established patterns (instance `SetValue`, `Act` messaging, oracle-proof gate,
session→game delegation).

---

## 0. The one-paragraph why

`open`/`close`/`lock`/`unlock`/`pick` only operate on directional doors, never containers — this is why the
consumables and object-inventory scenarios can't `open pack` and had to spawn items directly. C has ONE handler,
`do_gen_door`, that tries the argument as a **container object first** (`generic_find`), then as a **door exit**,
and drives both through shared macros. Go instead has **three** divergent door code paths and a broken exit
state model. This chunk consolidates everything onto a single C-faithful `do_gen_door(obj|door)` in the game
layer, retires the invented door subsystem, and adds container support. When it lands, `open`/`close`/`lock`/
`unlock`/`pick` work on containers AND doors with C-exact wording, and DP-1092's `$P` line finally goes oracle-green.

**Oracle-proof gate (non-negotiable):** green build/test is NOT sign-off. Each behavior is done only when the C
oracle shows no normalized divergence (RNG paths → seeded Tier-2 unit test, §6). Paste before/after in the PR.
Player-facing wording is first-class fidelity.

---

## 1. The three-way fork you are consolidating (do NOT discover this mid-flight)

| # | Path | State model | Wired to | Verdict |
|---|------|-------------|----------|---------|
| 1 | **`pkg/game/act_movement.go` `doGenDoor`/`findDoor`/`doDoorcmd`** (lines 410, 458, 608) | `room.Exits[dir].DoorState` int, reciprocal-toggled | ONLY `pick` (via `skill_stealth.go:217 DoPickLock`) | **KEEP & EXTEND — this is the base.** Closest to C (exit-based, reciprocal). |
| 2 | **`pkg/session/door_cmds.go` `(s *Session) doGenDoor`** + `doDoorOpen/Close/…` | `systems.DoorManager` bool `Door` struct | the **registered** `open/close/lock/unlock/pick/bash` (commands.go:180-183, 174) | **RETIRE.** Invented strings + mechanics. |
| 3 | **`pkg/game/systems/door.go` + `door_manager.go`** | bool `Door{Closed,Locked,Pickproof,Bashable,Hp,MaxHp,Difficulty}` | path 2 + movement `CanPass` (movement_cmds.go:232, cmd_movement.go:13) | **RETIRE the invented parts** (HP/bash/difficulty pick/strings). |

`pick` is currently **double-registered** (session path 2 AND game path 1 via DoPickLock) — they race/conflict.
Consolidation resolves this: all five subcommands route to ONE game-layer handler.

---

## 2. THE LATENT BUG you must fix as part of this: `DoorState` semantic collision

`pkg/parser/wld.go:59` `Exit.DoorState` is **loaded from the .wld as a capability code**:
`0=open(no door), 1=EX_ISDOOR, 2=EX_ISDOOR|EX_PICKPROOF`. But `act_movement.go` **reuses the same field as
runtime state**: `doorOpen=0, doorClosed=1, doorLocked=2` (act_movement.go:48-50). These are incompatible
encodings on one int:
- Pickproof (capability bit) is **lost** — the runtime can never see EX_PICKPROOF, so `pick` can't honor it.
- A door loaded as capability `1` (ISDOOR, initially open) is misread by the runtime as `doorClosed`.
- C keeps these as **independent bits** in `exit_info`: `EX_ISDOOR`, `EX_CLOSED`, `EX_LOCKED`, `EX_PICKPROOF`.

**Fix:** give the runtime exit a proper bitfield mirroring C's `exit_info`. Recommended: add
`ExitInfo int` (bit flags) to the runtime `Exit`, keep the .wld capability load feeding the ISDOOR/PICKPROOF
bits, and set the initial `EX_CLOSED`/`EX_LOCKED` from **zone-reset `D` commands** (the C door-init path), not
from the capability code. If touching zone-reset door init, note the related HH-314 / DP-850 (L/D-command Arg3
door-state semantics) — reconcile, don't re-break. Define the C bit values exactly (src/structs.h):
`EX_ISDOOR`, `EX_CLOSED`, `EX_LOCKED`, `EX_PICKPROOF` — grep the C header for the numeric bits and port verbatim.

If a full `exit_info` refactor proves too large, the minimum viable fix is a runtime state field **separate**
from the capability field so pickproof survives and open/closed/locked are independent — but the bitfield is the
correct C-faithful end state and what review expects.

---

## 3. Target: one `do_gen_door(ch, obj, door, scmd)` — C, verbatim (src/act.movement.c:598)

Extend the game-layer `doGenDoor` (act_movement.go:609). Full C control flow:

```c
skip_spaces(&argument);
if (!*argument) { sprintf(buf,"%s what?\r\n", cmd_door[subcmd]); send_to_char(CAP(buf), ch); return; }
two_arguments(argument, type, dir);
if (!generic_find(type, FIND_OBJ_INV | FIND_OBJ_ROOM, ch, &victim, &obj))   // CONTAINER FIRST (new for Go)
    door = find_door(ch, type, dir, cmd_door[subcmd]);                       // then door exit (Go has this)
if ((obj) || (door >= 0)) {
    keynum = DOOR_KEY(ch, obj, door);
    if (!DOOR_IS_OPENABLE(ch,obj,door))          act("You can't $F that!", ...TO_CHAR);   // $F = cmd_door[subcmd]
    else if (!DOOR_IS_OPEN   && NEED_OPEN)        stc("But it's already closed!\r\n");
    else if (!DOOR_IS_CLOSED && NEED_CLOSED)      stc("But it's currently open!\r\n");
    else if (!DOOR_IS_LOCKED && NEED_LOCKED)      stc("Oh.. it wasn't locked, after all..\r\n");
    else if (!DOOR_IS_UNLOCKED && NEED_UNLOCKED)  stc("It seems to be locked.\r\n");
    else if (!has_key(ch,keynum) && GET_LEVEL(ch)<LVL_GOD && (subcmd==LOCK||subcmd==UNLOCK))
                                                  stc("You don't seem to have the proper key.\r\n");
    else if (ok_pick(ch, keynum, DOOR_IS_PICKPROOF, subcmd))  do_doorcmd(ch, obj, door, subcmd);
}
```

**`cmd_door[]` / `flags_door[]` (port verbatim — Go has `cmdDoor`/`flagsDoor`, verify they match):**

| scmd | verb | flags_door |
|------|------|------------|
| 0 | open | `NEED_CLOSED\|NEED_UNLOCKED` |
| 1 | close | `NEED_OPEN` |
| 2 | unlock | `NEED_CLOSED\|NEED_LOCKED` |
| 3 | lock | `NEED_CLOSED\|NEED_UNLOCKED` |
| 4 | pick | `NEED_CLOSED\|NEED_LOCKED` |

`NEED_OPEN=1, NEED_CLOSED=2, NEED_UNLOCKED=4, NEED_LOCKED=8`.

**`DOOR_IS_*` — implement as helpers taking (obj OR door), reading exit_info for doors, contFlags for containers:**
- OPENABLE: obj→ type==CONTAINER && (contFlags&contCloseable); door→ (exit_info&EX_ISDOOR)
- OPEN: obj→ !(contFlags&contClosed); door→ !(exit_info&EX_CLOSED). CLOSED=!OPEN.
- LOCKED: obj→ (contFlags&contLocked); door→ (exit_info&EX_LOCKED). UNLOCKED=!LOCKED.
- PICKPROOF: obj→ (contFlags&contPickproofBit); door→ (exit_info&EX_PICKPROOF)
- DOOR_KEY: obj→ GetValue(contKey); door→ exit.Key
- **Toggle** (do_doorcmd): obj→ instance `SetValue(contFlags, contFlags ^ contClosed/contLocked)` — NEVER
  `Prototype.Values`; door→ toggle exit_info bit on ch's exit AND the reciprocal exit (Go already does the
  reciprocal for doors — keep it; containers have NO reciprocal, matching C `if (!obj)`).

**Go's current check-ladder is WRONG — replace it.** act_movement.go:645-660 emits invented "It's not closed.\r\n"
/ "It's not open.\r\n" / "It's not locked.\r\n" and an invented "The lock seems to be magical.\r\n" (line 664).
Replace all with the exact C strings above. Add the missing `"You can't $F that!"` (DOOR_IS_OPENABLE) check.
Remove the invented `ext.Key > 0 → magic lock` branch — C gates pickability on EX_PICKPROOF via `ok_pick`, not
key presence.

**`do_doorcmd` output (src/act.movement.c:477):**
- OPEN/CLOSE: toggle CLOSED (+ reciprocal if door); `send_to_char(OK, ch)`. **The `OK` macro is `"Ok.\r\n"`**
  (verify utils.h) — Go currently sends `"OK.\r\n"` (act_movement.go:499,514); **fix the casing.** Room:
  `"$n opens the door."` / `"$n opens $p."` (container). Reciprocal room (doors only, OPEN/CLOSE only):
  `"The <door> is opened/closed from the other side.\r\n"`. Go currently sends NO room broadcasts for
  open/close — add them.
- UNLOCK/LOCK: toggle LOCKED (+ reciprocal if door); `"*Click*\r\n"` (Go has this ✓). Room `"$n unlocks $p."` etc.
- PICK: toggle LOCKED; `"The lock quickly yields to your skills.\r\n"`; room `"$n skillfully picks the lock on ..."`.

**`find_door` (src/act.movement.c:370):** Go's `findDoor` (act_movement.go:410) mostly matches — verify its
strings against C: "That's not a direction.\r\n", "I see no %s there.\r\n", "I really don't see how you can do
anything there.\r\n", "What is it you want to %s?\r\n", "There doesn't seem to be %s %s here.\r\n". Secret-door
skip (keyword contains "secret" unless ROOM_SECRET_MARK) — confirm present.

**`has_key` (src/act.movement.c:428):** key in `ch->carrying` OR `WEAR_HOLD`. Go's `hasKey` — verify it checks
both inventory and held equipment.

---

## 4. Container branch (the new capability) + the `$P` payoff

Add the `generic_find(FIND_OBJ_INV|FIND_OBJ_ROOM)` container resolution BEFORE `find_door`, exactly as C. When
`obj != nil`, the whole ladder runs against the container (contFlags bits). This makes `open pack` / `close
chest` / `lock`/`unlock`/`pick <container>` work.

**Verify obj 8038 (starter backpack) starts CLOSED** (cont-flags value 5 = CLOSEABLE|CLOSED). A fresh newbie's
first `open pack` must succeed; a second must say "But it's currently open!". This is the DP-1091 acceptance.

Once containers open, **extend the chunk-1 oracle scenario** `object-inventory.txt` (its comment already says
"Container probes join this scenario in the next chunk") to prove:
`open pack`, `open pack` (already open), `get bread pack` → **"You get a loaf of bread from a leather backpack."**
(this is the DP-1092 `$P` oracle proof — chunk 1 only unit-tested it), `put bread pack`, `close pack`,
`get bread pack` (closed → "$p is closed."). Add a lockable container + its key if 8038 isn't lockable — pick a
real lockable chest vnum from `lib/world/obj/*.obj` and cite it in a scenario comment. Also add door probes at a
real door exit (find one in `lib/world/wld`, cite room/dir) covering the wrong-precondition messages.

---

## 5. Retire the invented door subsystem (DELETE — no C basis)

Per parent brief §5:
- **Door HP / bashing** (`Door.Hp/MaxHp/Bash()`, `damage=strength/10`, session `doDoorBash`, `doorSCMDBash`,
  `cmdBashDoor`). C's `do_bash` is a `POS_FIGHTING` **combat** skill (interpreter.c:347) — NOT a door command,
  and `do_gen_door` has no bash. Remove door-bash from the door family. If a `bash` command must remain wired,
  it points at combat; file a follow-up rather than keeping the invented door-HP model. (Confirm nothing else
  depends on door HP before deleting.)
- **`Difficulty` deterministic pick** (`skill >= difficulty`). Use C `ok_pick` (act.movement.c:536):
  `percent = number(1,101); if (percent > GET_SKILL(ch, SKILL_PICK_LOCK)) fail`. Go already has `okPick`
  (act_movement.go) wired for pick via DoPickLock — verify it matches C `ok_pick` (see §6) and route it through
  the unified handler.
- **Invented strings** in `systems/door.go` (`Open/Close/Lock/Unlock/Pick/Bash` return strings like "You open
  the door.", "It's locked.", "You must close it first.", "You don't have the right key.", "You fail to pick the
  lock.") and session `door_cmds.go` ("(Try: open door north)", "There is no door %s of here.", etc.). All gone.
- **`systems.DoorManager`** as the runtime door-state store: movement `CanPass` (movement_cmds.go:232,
  cmd_movement.go:13) must read the new exit_info instead. Either delete DoorManager entirely or reduce it to a
  thin read over exit_info — no invented state, no invented messages. Ensure `look`/room-desc door rendering
  (look.go references DoorManager) reads the unified state too.
- Session `door_cmds.go`: the `cmdOpen/Close/Lock/Unlock/Pick` handlers become thin — resolve args and call the
  game-layer `doGenDoor`. Delete the session `doGenDoor`/`doDoor*` methods.

**Preserve door reset semantics:** doors must return to their zone-reset initial closed/locked state on reset;
drive that from exit_info + the zone `D` reset, not the retired struct's `initialClosed/initialLocked`.

---

## 6. Tier-1 vs Tier-2 (same discipline as consumables/chunk 1)

**Tier-1 — must be oracle-green:** container open/close/lock/unlock and door open/close/lock/unlock; every
deterministic gate ("You can't $F that!", "But it's already closed!", "But it's currently open!", "Oh.. it
wasn't locked, after all..", "It seems to be locked.", "You don't seem to have the proper key."); the pick
deterministic gates (`keynum<0` → "Odd - you can't seem to find a keyhole.\r\n"; not holding lockpicks (obj
**8027** in WEAR_HOLD) & level<IMMORT → "You'll need to hold a set of lockpicks before you can pick a lock!\r\n";
pickproof → "It resists your attempts to pick it.\r\n"); DP-1091 (pack starts closed); DP-1092 ($P line).

**Tier-2 — seeded unit test only, `// Tier-2` note, NOT faked in the oracle:** the `ok_pick` success roll
(`number(1,101) > GET_SKILL`) and the lockpick-breakage roll (`number(0,30)+can_break` → swap obj 8027→8028 in
WEAR_HOLD + "$n curses as $e bends some of $s lockpicks." / "You ruin your lockpicks in the process.\r\n"). The C
RNG isn't ported, so pick *success* can't be byte-matched — prove the roll logic in a seeded test, prove the
gates in the oracle. Verify Go's existing `okPick` against C `ok_pick` and fix divergences (esp. the lockpick
vnum check and breakage).

---

## 7. Invariants (don't regress)

- **Prototype-mutation ban:** container closed/locked toggles via instance `SetValue(contFlags, …)` only. Add a
  two-instance isolation test: opening one instance of a container vnum must not open another.
- **Faithful messaging via `Act`** for all room/char/reciprocal broadcasts (`$n opens $p.`, `$N`, `$F`, `$p`).
- **Session→game delegation:** session door commands resolve args and call the game handler; no session-side
  door logic remains.
- Keep the reciprocal-door toggling that act_movement.go already does correctly.

---

## 8. Oracle proof — how

```
DP_ORACLE_BIN=$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff -scenario object-inventory     # extended with container+door probes (§4)
```
Add door probes (real door exit, cite vnums) and, if useful, a dedicated `object-doors.txt`. Green = the
container/door probes show no normalized divergence. Do NOT diff a pick *success* (RNG). Paste before→after.

---

## 9. Success criteria (done when ALL hold)

1. `open/close/lock/unlock/pick <container>` work, match C wording — DP-1091 oracle-green incl. pack-starts-closed.
2. `do_gen_door` is a **single** game-layer handler over obj|door; the exact C check-ladder + `do_doorcmd`
   messages ("Ok.\r\n", "*Click*", room broadcasts, reciprocal "from the other side") are faithful.
3. The `DoorState` collision is fixed — runtime exit_info bitfield with independent ISDOOR/CLOSED/LOCKED/PICKPROOF;
   pickproof honored.
4. `systems.DoorManager` invented HP/bash/difficulty/strings **removed**; movement `CanPass` + `look` read the
   unified exit_info; session `door_cmds.go` invented logic deleted; `pick` no longer double-registered.
5. DP-1092 `$P` line is oracle-green (extended object-inventory scenario).
6. Tier-2 RNG (pick success, lockpick breakage) in seeded unit tests with `// Tier-2` notes; not faked in oracle.
7. Prototype-mutation ban held + two-instance container isolation test.
8. `go build ./... && go vet ./... && go test ./...` + gofumpt + lint green; no orphaned dead code.

---

## 10. Wrap-up / handoff

Commit onto the domain branch; open/update a PR with the **green** oracle report (container + door probes) inline
+ the seeded Tier-2 tests. STOP for Claude's review — Claude QAs vs `origin/main` + `src/act.movement.c`, runs the
oracle independently, and merges. Do NOT close Linear (Claude does it). **Closes DP-1091 and proves/closes
DP-1092.** After this, only chunk 3 remains (inventory/equipment views, DP-1102).
