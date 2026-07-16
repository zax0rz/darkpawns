# BRIEF (codex) — `ITEM_TAKE_NAME` object auto-rename on equip/unequip (DP-1156)

**Owner:** codex. **Gate:** Claude establishes the oracle RED and runs red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** One focused PR. Player-facing short-descriptions are first-class fidelity — match C byte-for-byte, including the wear-vs-remove ordering asymmetry below.

## The gap
C flag `ITEM_TAKE_NAME` (**bit 17**, `structs.h:485`) makes an item adopt the wearer's name while worn. `equip_char` renames its short-description to `"<Owner>'s <keywords>"`; `unequip_char` renames it to `"a/an <keywords>"` (NOT back to the prototype). The Go port has **no concept of the flag** (grep: nothing) — it always shows the prototype short-desc. Proven RED via `--scenario equipment` while establishing the O19/DP-1106 equipment domain; that scenario deliberately probes `remove` on the *sword* to sidestep this, and its footer comment names DP-1156 as the follow-up.

The concrete carrier: newbie starter **tunic obj 8019** (`lib/world/obj/80.obj`) — `name`(keywords)=`tunic`, short-desc=`a frayed tunic`, extra-flags `131072` = `1<<17`. Every fresh char **carries** it (`pkg/game/character.go:288`).

## Read-only source of truth
C: `~/.openclaw/workspace/darkpawns-c-oracle/src/handler.c` — `equip_char` (**719-727**) and `unequip_char` (**766-773**); `src/act.item.c` — `perform_wear` (1416-1516) and `perform_remove` (1713-1726); `src/structs.h:485`. **Never edit the oracle tree.**
Go: `pkg/game/item_equipment.go` (`EquipItem` line 230, `UnequipItem` line 254 — both already carry a `TODO(DP-1156)` anchor), `pkg/game/object.go` (`ObjectInstance`, `Runtime.ShortDescOverride`, `GetShortDesc`, `GetKeywords`), `pkg/game/item_helpers.go` (extra-flag constants + `an()` helper).

## The C contract — reproduce exactly

**equip (`equip_char`, handler.c:719-727):**
```c
if (IS_OBJ_STAT(obj, ITEM_TAKE_NAME)) {
  ... free old short_description if not the prototype's ...
  sprintf(buf, "%s's %s", GET_NAME(ch), obj->name);   /* "Owner's tunic" */
  obj->short_description = str_dup(buf);
}
```
Uses **`obj->name`** — the *whole* keyword string, not the first token. For obj 8019 that's `tunic` → `"Eqactor's tunic"`.

**unequip (`unequip_char`, handler.c:766-773):**
```c
if (IS_OBJ_STAT(obj, ITEM_TAKE_NAME)) {
  ... free old short_description if not the prototype's ...
  sprintf(buf, "%s %s", AN(obj->name), obj->name);    /* "a tunic" */
  obj->short_description = str_dup(buf);
}
```
`AN(x)` = `"an"` if `x` starts with a vowel else `"a"`. For `tunic` → `"a tunic"`. **Note this does NOT restore the prototype short-desc `a frayed tunic`** — after one wear+remove cycle the item permanently reads `a tunic` until reboot.

**The ordering asymmetry (this is the crux — get it exactly right):**
- **wear:** `perform_wear` emits the wear message *before* calling `equip_char` (act.item.c: `wear_message` … then `equip_char` at :1516). So `wear tunic` prints **`You wear a frayed tunic on your body.`** (prototype short-desc), and *only then* does the item become `Eqactor's tunic`.
- **remove:** `perform_remove` calls `unequip_char` *before* the message (`unequip_char` at :1725, then `act("You stop using $p."…)` at :1726). So `remove tunic` prints **`You stop using a tunic.`** (already renamed).

## The Go fix (shape)
The Go call graph already mirrors C's ordering, so the rename drops straight into the two low-level choke points:
- `performWear` calls `wearMessage` (line 181) **before** `EquipItem` (line 193) — same as C. Put the equip rename **inside `EquipItem`**, at the existing `TODO(DP-1156)` (line 248), after `SetSlot` succeeds:
  ```go
  if obj.HasExtraFlag(0, extraFlagTakeName) {
      obj.Runtime.ShortDescOverride = fmt.Sprintf("%s's %s", ch.Name, obj.GetKeywords())
  }
  ```
- `performRemove` calls `UnequipItem` (line 448) **before** the "You stop using $p" message (line 454) — same as C. Put the unequip rename **inside `UnequipItem`**, at the `TODO(DP-1156)` (line 262), fetching the obj from the slot *before* moving it to inventory:
  ```go
  if obj, ok := ch.Equipment.GetItemInSlot(goSlot); ok && obj.HasExtraFlag(0, extraFlagTakeName) {
      obj.Runtime.ShortDescOverride = an(obj.GetKeywords()) + " " + obj.GetKeywords()
  }
  return ch.Equipment.Unequip(goSlot, ch.Inventory)
  ```
- Add the flag constant to `pkg/game/item_helpers.go` alongside the others: `extraFlagTakeName = 17 // ITEM_TAKE_NAME`.
- Use **`obj.GetKeywords()`** (= C `obj->name`, whole string), NOT the first token. `an()` already exists (`item_helpers.go:340`).

### Copy-on-write law (do not violate)
Write **only** `obj.Runtime.ShortDescOverride` — never mutate `obj.Prototype` (same discipline as the Domain-3 money/values overrides). `GetShortDesc()` already checks `ShortDescOverride` first, so `$p` substitution, `inventory`, `equipment`, and `look` all pick it up automatically. Because the override is instance-runtime state, a fresh boot loads the pristine prototype — which is exactly C's "restored from prototype on reboot" behavior.

### MUST-VERIFY: no persistence across reboot
C's rename is runtime-only; a reboot restores the prototype short-desc. The oracle RED I run is single-session and will **not** catch a persistence divergence. So you must confirm the take-name override does **not** survive a save→reload: check `db.PlayerToRecord` / the inventory+equipment serialization (`pkg/session/manager.go` save path, `pkg/db`). If instance short-desc overrides are serialized, exclude the take-name rename from persistence (or restore proto on load) so a reboot shows `a frayed tunic` again, matching C. Cover this with a unit test (save a worn take-name item → reload → assert short-desc is the prototype).

## Oracle RED (Claude establishes + gates)
I own the scenario — do **not** edit `scenarios/equipment.txt` or add scenario files. I'll stage `scenarios/equipment-takename.txt`: fresh char carrying tunic 8019, probes:
- `wear tunic` → `You wear a frayed tunic on your body.` (prototype desc — message precedes rename)
- `equipment` → the worn line shows **`Eqactor's tunic`** (renamed) ← Go currently shows `a frayed tunic`
- `remove tunic` → `You stop using a tunic.` (renamed before message) ← Go currently `a frayed tunic`
- `inventory` → shows **`a tunic`** (renamed, carried; NOT `a frayed tunic`)

These draw no RNG (equip/unequip messaging is deterministic), so it's a clean tier-1 red→green. `equipment`/`inventory` views are already C-faithful (DP-1102), so the rename is the only diff.

## Out of scope
- The wear/remove verbs, wear-position messaging, and `equipment`/`inventory` views themselves — all already C-faithful (O19/DP-1102); don't touch them.
- Other extra-flag mechanics (anti-align/class, no-drop, etc.).
- The C oracle tree, `website/static/map/world-sphere.json`, `docs/reports/reek/*`.

## Tests you own (deterministic)
- Equip a take-name item → short-desc = `"<Owner>'s <keywords>"`; keywords/targeting unchanged (can still `remove` by original keyword).
- Unequip → short-desc = `an(keywords)+" "+keywords` (e.g. `a tunic`), NOT the prototype.
- A non-take-name item (e.g. the starter sword) is unchanged through wear/remove.
- Prototype is never mutated (a second fresh instance of the same vnum reads the prototype short-desc).
- Save→reload of a worn take-name item restores the prototype short-desc (the must-verify above).

## PR hygiene
- Commits end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- PR body ends with: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`
