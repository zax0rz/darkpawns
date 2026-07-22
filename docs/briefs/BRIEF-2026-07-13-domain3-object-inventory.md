# BRIEF 2026-07-13 — Domain 3: Object / Inventory + Door Unification (GPT)

**Executor:** ChatGPT/GPT. **Branch:** `refactor/domain-object-inventory` (already created off `main`;
`git fetch && git checkout refactor/domain-object-inventory`). Commit onto it; do NOT start a new branch.
**Reviewer/merger:** Claude, against `origin/main` + `src/*.c`, gated by the oracle. This is a **2+ session**
domain — Zach has explicitly OK'd that scope. Land it in reviewable chunks (see "Sequencing").

---

## 0. The one-paragraph why

`get`/`put`/`drop`/`give` are wired and mostly work, but the item-transfer wording diverges from C, and
**`open`/`close`/`lock`/`unlock`/`pick` only handle directional doors — not containers.** That last one is
the anchor: it's why the consumables domain couldn't `open pack` and had to spawn items directly into the
room. Worse, the Go door subsystem is a **wholly invented parallel model** (`pkg/game/systems/door*.go`)
with its own `Door` struct, its own message strings, and invented mechanics (door HP/bashing, a
deterministic `skill >= difficulty` pick) that have **no basis in the C source**. C has exactly one handler,
`do_gen_door`, that operates on *either* a container object *or* a door exit through shared macros. This
domain unifies Go onto that single C-faithful model and fixes the transfer-verb fidelity.

**Oracle-proof gate (non-negotiable):** a green `go build/vet/test` is NOT sign-off. Each behavior is done
only when the C oracle shows **no normalized divergence** for it (or, for RNG paths, a targeted seeded/Tier-2
test — see §6). Paste oracle before/after in the PR. Player-facing wording is first-class fidelity, equal to
mechanical effect — "You can't $F that!" vs "It's locked." is a real bug, not a nitpick.

---

## 1. What you're closing (Linear cluster — all in project "Oracle Differential Testing")

| Ticket | Finding | One-liner |
|--------|---------|-----------|
| **DP-1091** | O7 | `open`/`close` only handle doors, not containers; starting backpack (obj **8038**, cont-flags value **5** = CLOSEABLE\|CLOSED) must START CLOSED but Go inits it open |
| **DP-1092** | O8 | `get bread pack` → "You get a loaf of bread from **a loaf of bread**." — `$P` renders as the item, not the container |
| **DP-1098** | O17 | `get`: TAKE is a bit test `CAN_WEAR(obj, ITEM_WEAR_TAKE)`; `get all <container>` / `get all.x <container>` grammar missing |
| **DP-1099** | O20 | `give <obj> <mob>` fires the script hook but **never moves the object** |
| **DP-1102** | O12 | `inventory`/`equipment` views: Go ranges a **map** (nondeterministic order), wrong headers/labels/empty-state vs C |
| **DP-1105** | O18 | `drop` grammar (coins / `all` / `all.x`) + NODROP/cursed checks incomplete; session path bypasses game path |

Confirm each against the C source before fixing (some are "cited, not yet spot-verified"). Do NOT trust the
ticket text alone — read the C.

---

## 2. Canonical model decision (this is the architecture — follow it)

**Doors are room-exit state, exactly like C. Retire the invented `systems.Door` divergence.**

C has no `Door` struct. A door is bits in the exit's `exit_info` field on the room's `dir_option[door]`:
`EX_ISDOOR`, `EX_CLOSED`, `EX_LOCKED`, `EX_PICKPROOF`, plus the exit's `key` and `keyword`. Containers use
the parallel object-value model you already have and which is **already C-faithful**:

```
pkg/game/item_helpers.go:
  contCapacity=0  contFlags=1  contKey=2  contPickproof=3   // = C GET_OBJ_VAL(obj, 0..3)
  contCloseable=1<<0  contPickproofBit=1<<1  contClosed=1<<2  contLocked=1<<3   // = C CONT_* bits
```

**Target end state:** one game-layer `do_gen_door(ch, obj, door, subcmd)` that mirrors C exactly, operating on
*either* an `*ObjectInstance` container (obj != nil) *or* a room-exit door (obj == nil, door >= 0), via a set
of Go helpers that mirror C's `DOOR_IS_*` macros. The session `door_cmds.go` becomes a thin dispatcher that
resolves args and calls the game layer; `systems.DoorManager`'s invented message/HP/difficulty logic is
retired (see §5). Door open/close/lock/unlock state must live where the room exit lives and reset via zone
resets, same as C.

**Do NOT** invent a third model. **Do NOT** keep the `systems.Door` message strings. If unifying the exit
storage is bigger than one session, land the container path + transfer verbs first (they're independent of the
door-storage refactor), then do the door-storage unification as the second chunk — but the END state is one
handler. See Sequencing (§8).

---

## 3. THE ANCHOR — `do_gen_door` container + door, C-faithful (src/act.movement.c:598)

This is the heart of the domain. C's `do_gen_door`:

```c
skip_spaces(&argument);
if (!*argument) { sprintf(buf,"%s what?\r\n", cmd_door[subcmd]); send_to_char(CAP(buf), ch); return; }
two_arguments(argument, type, dir);
if (!generic_find(type, FIND_OBJ_INV | FIND_OBJ_ROOM, ch, &victim, &obj))   // ← CONTAINER FIRST
    door = find_door(ch, type, dir, cmd_door[subcmd]);                       // ← then door exit
if ((obj) || (door >= 0)) {
    keynum = DOOR_KEY(ch, obj, door);
    if (!DOOR_IS_OPENABLE(ch, obj, door))              act("You can't $F that!", ...TO_CHAR);  // $F=cmd_door[subcmd]
    else if (!DOOR_IS_OPEN && NEED_OPEN)               stc("But it's already closed!\r\n");
    else if (!DOOR_IS_CLOSED && NEED_CLOSED)           stc("But it's currently open!\r\n");
    else if (!DOOR_IS_LOCKED && NEED_LOCKED)           stc("Oh.. it wasn't locked, after all..\r\n");
    else if (!DOOR_IS_UNLOCKED && NEED_UNLOCKED)       stc("It seems to be locked.\r\n");
    else if (!has_key(ch,keynum) && GET_LEVEL(ch)<LVL_GOD && (subcmd==LOCK||subcmd==UNLOCK))
                                                       stc("You don't seem to have the proper key.\r\n");
    else if (ok_pick(ch, keynum, DOOR_IS_PICKPROOF, subcmd))  do_doorcmd(ch, obj, door, subcmd);
}
```

**`cmd_door[]` (index = subcmd) and `flags_door[]` requirement bits — port these tables verbatim:**

| subcmd | verb | flags_door (preconditions) |
|--------|------|----------------------------|
| 0 | `open`   | `NEED_CLOSED \| NEED_UNLOCKED` |
| 1 | `close`  | `NEED_OPEN` |
| 2 | `unlock` | `NEED_CLOSED \| NEED_LOCKED` |
| 3 | `lock`   | `NEED_CLOSED \| NEED_UNLOCKED` |
| 4 | `pick`   | `NEED_CLOSED \| NEED_LOCKED` |

`NEED_OPEN=1, NEED_CLOSED=2, NEED_UNLOCKED=4, NEED_LOCKED=8`. Note the check messages read against the bit:
e.g. `open` has NEED_CLOSED, so if it's already open → "But it's currently open!".

**`DOOR_IS_*` macros — implement as Go helpers taking (obj OR door):**
- `OPENABLE`: obj → type==CONTAINER && (contFlags & contCloseable); door → (exit_info & EX_ISDOOR)
- `OPEN`: obj → !(contFlags & contClosed); door → !(exit_info & EX_CLOSED)
- `LOCKED`: obj → (contFlags & contLocked); door → (exit_info & EX_LOCKED). `UNLOCKED` = !LOCKED. `CLOSED`=!OPEN.
- `PICKPROOF`: obj → (contFlags & contPickproofBit); door → (exit_info & EX_PICKPROOF)
- `DOOR_KEY`: obj → GetValue(contKey); door → exit.key
- **Toggle** (do_doorcmd): obj → SetValue(contFlags, contFlags ^ contClosed/contLocked); door → toggle exit_info bit.
  Use **instance** `SetValue` — NEVER `Prototype.Values` (see §7).

**`do_doorcmd` output (src/act.movement.c:477) — the success messages:**
- OPEN/CLOSE: toggle CLOSED (both sides if reciprocal exit exists); `send_to_char(OK, ch)` — the **`OK`** macro
  (verify in utils.h — it is `"Ok.\r\n"`). Room: `"$n opens the door."` / `"$n opens $p."` (container). The
  reciprocal room gets `"The <door> is opened from the other side.\r\n"` (only OPEN/CLOSE, only if reciprocal).
- UNLOCK/LOCK: toggle LOCKED (both sides); `send_to_char("*Click*\r\n", ch)`; room `"$n unlocks $p."` etc.
- PICK: toggle LOCKED; `"The lock quickly yields to your skills.\r\n"`; room `"$n skillfully picks the lock on ..."`.

**Containers have no reciprocal side** — the "other room" logic only applies to door exits (`!obj`). Match C:
`do_doorcmd` computes `back` only `if (!obj && ...)`.

**`find_door` (src/act.movement.c:370)** — direction/keyword resolution + its exact messages:
"That's not a direction.\r\n", "I see no %s there.\r\n", "I really don't see how you can do anything there.\r\n",
"What is it you want to %s?\r\n", "There doesn't seem to be %s %s here.\r\n" (AN(type)). Secret-door handling:
skip exits whose keyword contains "secret" unless room has ROOM_SECRET_MARK.

**`has_key` (src/act.movement.c:428):** key is in `ch->carrying` OR held in `WEAR_HOLD`. Match both.

**Verify obj 8038 starts CLOSED (DP-1091 second half):** the backpack's cont-flags value is 5
(CLOSEABLE|CLOSED). Confirm the object loader / zone reset preserves value[1]=5 and that Go doesn't zero or
mis-init it to open. `open pack` on a fresh newbie must first succeed (it starts closed), and a second
`open pack` must say "But it's currently open!".

---

## 4. Transfer verbs — get / drop / put / give (src/act.item.c)

The handlers live in `pkg/game/item_transfer.go` (get/drop/give) and `item_container.go` (put), wired via
`pkg/session/cmd_inventory.go`. Bring them to C wording/behavior. Read the C; don't paraphrase from memory.

**`do_get` (act.item.c:346) — key divergences:**
- **Order:** C checks `IS_CARRYING_N(ch) >= CAN_CARRY_N(ch)` → "Your arms are already full!\r\n" **before**
  the empty-arg "Get what?\r\n". Go checks empty-arg first and has no top-level arms-full precheck.
- **DP-1092 ($P bug):** `performGetFromContainer` emits `"You get $p from $P."` with (obj, cont). The `$P`
  currently renders as the item. **Verify `actToChar`/`Act` supports the second-object token `$P`/`$O`
  distinctly from `$p`/`$o`.** If the formatter collapses $P→$p, that's a primitive gap — fix the formatter
  (this is F0a-adjacent and will affect every two-object act message). Target: "You get a loaf of bread from
  a leather backpack."
- **DP-1098 (TAKE flag):** `canTakeObj` loops `WearFlags` comparing `wf == 1`. C is a **bit test**:
  `CAN_WEAR(obj, ITEM_WEAR_TAKE)` where TAKE is bit 0 (value 1<<0). Confirm the Go WearFlags model — if it's a
  bit field, test the bit; if it's a slice of set bit-indices, membership of index 0. Match C semantics exactly.
- **DP-1098 (`get all <cont>` / `get all.x <cont>`):** C's else-branch (cont_dotmode != FIND_INDIV) loops all
  visible containers in inventory then room, calling `get_from_container(..., FIND_ALL/FIND_ALLDOT)`, with
  "$p is not a container." for non-containers under ALLDOT, and the not-found messages "You can't seem to find
  any containers.\r\n" / "You can't seem to find any %ss here.\r\n". Go's `doGet` only handles `get <item>
  <container>` for the INDIV case and never does all-from-container. Port the full grammar.
- Container-not-found (INDIV): C says **"You don't have %s %s.\r\n"** (AN(arg2)); Go says "You don't see a X
  here." — fix wording. Non-container: "$p is not a container." (matches). `get_from_room` messages: read
  `get_from_room` in C for the exact "get all" empty-room and single-item-not-found strings.

**`do_drop` (act.item.c:529) — DP-1105:**
- Grammar: coins (`drop <n> coins`), individual, `all`, `all.x`. Go-game has all/dot/NODROP but the **session
  path** (`cmd_inventory.go`) reimplements a weaker version — route session `drop` through the game `DoDrop`
  and delete the session reimplementation (same pattern as get/give already use).
- `perform_drop` NODROP/cursed: `"You can't %s $p, it must be CURSED!"` with the verb (`sname`) interpolated.
  Confirm the Go message uses the command name ("drop"/"junk") like C, not a hardcoded "drop".
- **Scope note:** `junk`/`donate` are **separate commands** (SCMD_JUNK/SCMD_DONATE share `do_drop`). If they're
  already handled elsewhere, leave them; if in-scope for the shared handler, port `VANISH`/`perform_drop_gold`
  faithfully. Confirm current wiring before touching. `perform_drop_gold` and junk-value
  `MAX(1,MIN(200,cost>>4))` are deterministic (portable) — but WAIT_STATE on coin-drop is a timing concern; see §6.

**`do_give` (act.item.c:767) — DP-1099:**
- **give-to-mob must MOVE the object** (C `perform_give` transfers to victim, including NPCs, THEN fires
  behavior). Go's mob branch finds the obj, runs `ongive`, but never calls the transfer. Fix: perform the
  transfer, then the script hook, matching C ordering. Preserve the existing gold-bribe path and the
  pointer-ordered lock in `performGiveGold` (that's a real concurrency fix — keep it; don't regress DP-387).
- Verify give wording: "You give $p to $N.", "$n gives you $p.", "$n gives $p to $N.", full-hands
  "$N seems to have $S hands full.", weight "$E can't carry that much weight." against C.

**`inventory` / `equipment` (DP-1102, act.informative.c:1460):**
- Equipment: Go **ranges a map** → nondeterministic slot order. C walks the fixed `where[]` slot order (0..
  NUM_WEARS) with per-slot visibility and the "Nothing." empty state. Port the fixed order + `where[]` labels.
- Inventory: C uses `list_obj_to_char` (grouped display) — align headers/labels/empty-state. This overlaps the
  Observation domain's renderer work; reuse those primitives (`list_obj_to_char` equivalent) rather than
  re-inventing. If DP-1102 turns out to be mostly display and cleanly separable, it may be the natural first
  small chunk to land.

---

## 5. Retire the invented door mechanics (no C basis — do not "port," DELETE)

`pkg/game/systems/door.go` invents these; **C has none of them**:
- **Door HP / bashing** (`Hp`, `MaxHp`, `Bash()`, `damage = strength/10`). In this C source, `do_bash` is a
  `POS_FIGHTING` **combat** skill (interpreter.c:347) — bashing an *opponent*, not a door. There is no
  door-bash in `do_gen_door`. Remove door bashing from the door-command family. If a `bash` command must exist,
  it belongs to combat, not here — file a follow-up rather than keeping the invented door-HP model.
- **`Difficulty` deterministic pick** (`skill >= difficulty`). C uses `ok_pick` (act.movement.c:536):
  `percent = number(1,101); if (percent > GET_SKILL(ch, SKILL_PICK_LOCK)) fail`. That's RNG (Tier-2, §6).
- **Invented strings** ("You open the door.", "It's locked.", "You must close it first.", "You don't have the
  right key.", "You fail to pick the lock.", "This lock is too complex to pick.", "(Try: open door north)").
  Replace ALL with the C strings from §3.
- **`ok_pick` full behavior** to port: `keynum<0` → "Odd - you can't seem to find a keyhole.\r\n"; not holding
  lockpicks (obj **8027** in WEAR_HOLD) && level<IMMORT → "You'll need to hold a set of lockpicks before you can
  pick a lock!\r\n"; pickproof → "It resists your attempts to pick it.\r\n" (can_break=2); roll fail → "You
  failed to pick the lock.\r\n" (can_break=1); on can_break with picks held, `number(0,30)+can_break` roll may
  **break** picks: swap obj 8027 → 8028 in WEAR_HOLD + "$n curses as $e bends some of $s lockpicks." /
  "You ruin your lockpicks in the process.\r\n". All RNG → Tier-2; the deterministic gates (no keyhole, no
  picks, pickproof) are Tier-1 provable.

Preserve any door **reset** semantics you need for the oracle (doors must return to their zone-reset state),
but drive them from exit_info, not the `initialClosed/initialLocked` fields of the retired struct.

---

## 6. Tier-1 (deterministic, oracle-provable now) vs Tier-2 (RNG, defer)

**Tier-1 — must be oracle-green this domain:** all of get/drop/put/give wording+behavior;
open/close/lock/unlock on containers AND doors; the deterministic pick gates (no keyhole / no lockpicks /
pickproof / not-locked / wrong-precondition); key checks; `open pack` start-closed (DP-1091).

**Tier-2 — do NOT try to oracle-prove the roll; cover with a seeded Go unit test + a `// Tier-2` note:**
`ok_pick` success roll (`number(1,101)`), lockpick breakage roll (`number(0,30)`). The C RNG isn't ported yet,
so pick-*success* can't be byte-matched. Prove the deterministic branches in the oracle; prove the roll logic
in a seeded unit test. Same discipline as the water-drink amount in consumables.

**Timing:** C uses `WAIT_STATE` on some paths (e.g. coin drop, PULSE_VIOLENCE). The normalizer masks timing;
don't chase lag-state parity — just don't *break* existing wait-state behavior.

---

## 7. Invariants you must not regress

- **Prototype-mutation ban (corruption class):** NEVER write `obj.Prototype.Values[...]` or
  `cont.Prototype.Values[...]`. Toggling a container's closed/locked bit MUST go through instance
  `SetValue(contFlags, ...)` (copy-on-write via `ValuesOverride`). Audit the two existing prototype **reads**
  and switch them to `GetValue` for consistency: `getCheckMoney` reads `obj.Prototype.Values[0]`
  (item_transfer.go:38) and `performPut` reads `cont.Prototype.Values[contCapacity]` (item_container.go:11).
  Add/keep a two-instance isolation test: opening one instance of a vnum must not open the other.
- **Faithful messaging via `Act`/`actToChar`** (pkg/game/act.go). The $P two-object fix (§4) strengthens this
  primitive — keep it centralized, don't special-case in the handler.
- **Concurrency:** keep the pointer-ordered dual-lock in `performGiveGold` (DP-387) and the `vict.mu` guard on
  recipient inventory reads. Any new cross-player mutation follows the same lock-ordering rule.
- **Session→game delegation:** the session layer resolves args and calls the game layer; it must not
  reimplement game logic (that's the O17/O18/O20 root cause). Delete session reimplementations as you route
  through the game handlers.

---

## 8. Sequencing (land in reviewable chunks — Claude reviews each)

Independent, in recommended order:
1. **Transfer-verb fidelity** (get/drop/put/give wording + DP-1092 `$P` formatter + DP-1098 grammar + DP-1099
   give-to-mob transfer + DP-1105 drop). Independent of the door refactor. Oracle scenario: transfer probes.
2. **Container ops in a unified game-layer `do_gen_door`** (DP-1091 anchor) — container branch first, wired so
   `open/close/lock/unlock/pick <container>` work; doors still routed through the existing path temporarily is
   acceptable *within* this chunk only if clearly marked, but…
3. **Door-storage unification** — move door state to exit_info, retire `systems.Door` invented mechanics/strings
   (§5), so the single `do_gen_door` drives both. This is the biggest chunk; it's fine as session 2.
4. **inventory/equipment views** (DP-1102) — can slot in early as a small standalone if convenient.

Each chunk: build/vet/test green + gofumpt + lint, AND its oracle probes green, before moving on.

---

## 9. Oracle proof — how (this was the missing piece for other workers)

The C oracle binary is on this machine. Run:
```
DP_ORACLE_BIN=$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff -scenario object-inventory
```
Author `cmd/dp-oracle-diff/scenarios/object-inventory.txt` modeled on the existing `consumables.txt` (which
uses a `[fixture]` block with `spawn-obj <vnum> <count> <room> <maxexist>` and `quiet-mobs`). Design it to
**isolate** this domain — no dependence on un-migrated commands, no mob-room noise, one room:

- **Container probes:** spawn a closeable container that starts CLOSED (obj **8038** backpack = cont-flags 5)
  and an item, both into the start room; then: `open pack`, `open pack` (already open), `get bread pack`
  (proves DP-1092 $P), `put bread pack`, `close pack`, `get bread pack` (closed → "closed"), `lock`/`unlock`
  with the right/wrong key. Use a container+key pair from the world if 8038 has no lock, or a lockable chest
  vnum — pick real vnums from `lib/world/obj/*.obj` and cite them in a scenario comment.
- **Door probes:** stand at a room with a real door exit (find one in `lib/world/wld`); `close <dir>`,
  `open <dir>`, `lock`/`unlock` with key, and the wrong-precondition messages ("But it's currently open!"
  etc.). Cite the room/dir vnums.
- **Transfer probes:** `get`/`drop`/`give` (to a quiet mob for DP-1099), `get all`, `get all.x`, `inventory`,
  `equipment`.
- **Pick:** only the deterministic gates (no lockpicks held → the keyhole/lockpick messages). Do NOT diff a
  pick *success* (RNG).

"Green" = the object/inventory/door probes show **no normalized divergence**. World/score/quit noise from
un-migrated commands is fine to leave. Paste before→after in the PR per chunk.

---

## 10. Success criteria (done when ALL hold)

1. `open/close/lock/unlock/pick <container>` work and match C wording — the anchor (DP-1091) is oracle-green,
   including obj 8038 starting closed.
2. `do_gen_door` is a **single** game-layer handler over obj|door; `systems.Door` invented HP/difficulty/pick
   mechanics and message strings are **removed**; door state lives in exit_info; doors oracle-green.
3. Transfer verbs match C: DP-1092 ($P), DP-1098 (TAKE bit + `get all <cont>`), DP-1099 (give-to-mob transfers),
   DP-1105 (drop grammar/NODROP), each oracle-green.
4. `inventory`/`equipment` deterministic order + C headers/empty-state (DP-1102).
5. Prototype-mutation ban held (instance SetValue only) + two-instance container isolation test.
6. Tier-2 RNG (pick success, lockpick breakage) covered by seeded unit tests with `// Tier-2` notes; not faked
   in the oracle.
7. `go build ./... && go vet ./... && go test ./...` + gofumpt + lint green.
8. Session layer delegates to game layer (no reimplementation); dead `systems.Door` code deleted, not orphaned.

---

## 11. Wrap-up / handoff

Commit onto `refactor/domain-object-inventory` in reviewable chunks; open (or update) a PR per §8 chunk with the
**green** oracle report inline + the seeded Tier-2 tests. STOP after each chunk for Claude's review — Claude QAs
against `origin/main` + `src/act.item.c`/`src/act.movement.c`, runs the oracle independently, and merges.
Do NOT close Linear (Claude does that at merge). Closes DP-1091, DP-1092, DP-1098, DP-1099, DP-1102, DP-1105.
