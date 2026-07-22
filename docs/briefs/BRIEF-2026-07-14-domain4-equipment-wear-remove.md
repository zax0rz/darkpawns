# BRIEF — Domain 4: Equipment (wear / wield / hold / remove) — O19 / DP-1106

**For:** codex (frontier). **Owner of gate:** Claude (runs the oracle red→green, reviews vs C).
**Branch:** `refactor/domain-equipment` off current `main`.
**Finding:** DP-1106 / Fidelity **O19** — session hand-rolls wear/wield/hold/remove with
drifted wording and bypasses the C-faithful game layer.
**Method rules:** read `src/act.item.c` + `src/handler.c` in the C oracle clone
(`~/.openclaw/workspace/darkpawns-c-oracle`) directly — do not trust Go comments. This fix is
gated by an **oracle red→green run** (see [[darkpawns-oracle-proof-gate]]): a green build is NOT
sign-off.

---

## 1. The bug (proven RED)

The RED is already captured by the scenario **`cmd/dp-oracle-diff/scenarios/equipment.txt`**
(committed as part of this work — actor wears starter tunic/sword, a passive `observer` peer
captures the TO_ROOM broadcasts). Run it:

```
DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" \
  go run ./cmd/dp-oracle-diff --scenario equipment
```

Current divergences (C oracle `-` vs Go port `+`):

| probe | audience | C oracle | Go port | defect |
|---|---|---|---|---|
| `wear tunic` | actor | `You wear a frayed tunic on your body.` | `You wear a frayed tunic.` | missing wear-position clause |
| `wear tunic` | observer | `Eqactor wears a frayed tunic on his body.` | `Eqactor wear a frayed tunic.` | no `act()` `$n` conjugation + no location |
| `wield sword` | observer | `Eqactor wields a short sword.` | `Eqactor wield a short sword.` | raw verb, no `act()` conjugation |
| `remove sword` | observer | `Eqactor stops using a short sword.` | `Eqactor remove a short sword.` | wrong verb + no conjugation |
| `hold tunic` | actor | `You can't hold that.` | `You hold a frayed tunic.` | no holdability gate (Go holds anything) |
| `hold tunic` | observer | *(nothing)* | `Eqactor hold a frayed tunic.` | bogus broadcast for a rejected action |

(`wield`/`remove` **actor** lines already match — the divergence there is only the broadcast.)

## 2. Root cause

`pkg/session/cmd_inventory.go` `cmdWear/cmdWield/cmdHold/cmdRemove` hand-roll each verb using
`Equipment.EquipForPlayer`/`Unequip` with flat `Sprintf` wording and a bespoke
`broadcastEquipmentChange` that emits `"<Name> <rawverb> <desc>."` — no `act()` substitution.

Meanwhile the game layer **already has a mostly-faithful port** in `pkg/game/item_equipment.go`
(`performWear`, `wearMessage` + the `wearMessages`/`alreadyWearing` tables in `item_helpers.go`,
`findEqPos`, `performRemove`, `doWear`) that the session simply never calls. `wearMessage` routes
through `actToRoom`/`actToChar` → the canonical `Act()` (post-F0a), so adopting it fixes the
conjugation/location/broadcast bugs.

**But that game code has a latent slot-model bug you MUST fix as part of this** (see §4).

## 3. Scope — this PR is O19 ONLY

**In scope:** faithful `wear`/`wield`/`hold`(`grab`)/`remove` command behavior + messages +
broadcasts, session delegating to the game layer, and the slot-mapping fix that makes the game
equip/remove path correct.

**Explicitly OUT of scope (each its own finding — do NOT do here):**
- **`ITEM_TAKE_NAME`** (bit 17) object auto-renaming — **DP-1156**. C `equip_char` renames a worn take-name
  item to `"Owner's <keyword>"` and `unequip_char` to `"a <keyword>"` (handler.c:718, :766). The
  starter tunic 8019 has this flag — that's why the oracle shows `remove` → "a tunic". The
  scenario deliberately probes `remove` on the **sword** (not take-name) so this PR can go green
  without it. Leave a `TODO` at the equip/unequip sites. Tracked separately.
- Full retirement of the dual slot-numbering (`EquipmentSlot` Go order vs C `WEAR_` order) —
  **DP-1157**. You bridge the two via the existing table; you do NOT rewrite storage/save/combat.

## 4. The slot-model bug (critical)

Two incompatible numbering systems coexist:
- `EquipmentSlot` (`pkg/game/equipment.go`): Go-invented order — `SlotHead=0 … SlotWield=6,
  SlotHold=7 …`.
- `eqWear*` (`pkg/game/item_helpers.go`): **C `WEAR_` order** — `eqWearBody=5, eqWearWield=16,
  eqWearHold=17 …`.

`item_equipment.go` computes a C `eqWear` position (`where`) but then does
`GetItemInSlot(EquipmentSlot(where))` — casting a C index straight to the Go enum. So
`IsEquipped(ch, eqWearWield=16)` actually inspects `EquipmentSlot(16)` = `SlotBack`, and
`performRemove(pos=16)` looks in the wrong slot and silently no-ops. It only "works" today for
wearing because `EquipItem`→`Equip` **ignores `where`** and re-derives the slot from the item's
wear-flags. Conflict checks (already-wielding, two-handed) and remove are broken.

**The bridge already exists:** `cWearToGoSlot []EquipmentSlot` in `pkg/game/item_views.go:14`
(Domain 3). Use it. Suggested helper:

```go
// cWearSlot maps a C WEAR_ position to its Go EquipmentSlot; ok=false for
// THROW/ABLEGS/FACE/HOVER (no Go slot) or out-of-range.
func cWearSlot(where int) (EquipmentSlot, bool) {
    if where < 0 || where >= len(cWearToGoSlot) { return 0, false }
    s := cWearToGoSlot[where]
    if s < 0 { return 0, false }
    return s, true
}
```

Route **every** `EquipmentSlot(where)` cast in `item_equipment.go` (`IsEquipped`, `GetEquipped`,
`performRemove`, and the equip placement) through `cWearSlot`. For placement, equip at the
**specific** slot `where` chose (add `Equipment.SetSlot(slot, obj)` — a locked direct setter —
and set `obj.Location = LocEquippedPlayer(ch.Name, slot)` + `ch.Inventory.removeItem(obj)`); do
NOT let `Equipment.equip`'s getWearFlags re-derive it, or the finger/neck/wrist secondary-slot
bump and remove-by-position desync.

## 5. Faithful C reference (act.item.c / handler.c)

- **`do_wear`** (act.item.c:1584): `two_arguments`; empty→`Wear what?`; `find_all_dots`; wear-all
  / wear-all.x / individual `wear <item> [<position>]`; not-found `You don't seem to have %s %s.`
  (`AN(arg)`); `find_eq_pos` → `perform_wear`. **The game `doWear` already implements this** —
  keep it, just make sure it flows through the fixed `performWear`.
- **`perform_wear`** (act.item.c:1416): position-valid check (`You can't wear $p there.`);
  finger/neck/wrist secondary bump; `already_wearing[where]`; wield block (can-wield,
  `AFF_FLESH_ALTER`, weight vs `str_app[...].wield_w`, two-handed vs hold/shield); hold/shield vs
  two-handed wield; then **`if (!invalid_class) wear_message(); obj_from_char; equip_char`**.
  Order matters: invalid-class SUPPRESSES the wear message (`You cannot use $p.`); anti-align
  items DO print the wear message, then `equip_char` zaps them back (`You are zapped by $p and
  instantly let go of it.` / room `$n is zapped by $p and instantly lets go of it.`).
- **`do_wield`** (act.item.c:1661): `one_argument`; empty→`Wield what?`; not-found
  `You don't seem to have %s %s.`; `AFF_FLESH_ALTER`→can't; else `perform_wear(WEAR_WIELD)`.
  NOTE C does **not** check item type here (the session's `"That's not a weapon."` is invented —
  drop it; `perform_wear`'s wear-flag check yields `You can't wear $p there.`).
- **`do_grab`** (act.item.c:1685): `one_argument`; empty→`Hold what?`; not-found; then
  `if !CAN_WEAR(HOLD) && type∉{WAND,STAFF,SCROLL,POTION}` → `You can't hold that.` else
  `perform_wear(WEAR_HOLD)`. (Item type consts in `item_helpers.go`: WAND=3 STAFF=4 SCROLL=2
  POTION=10.)
- **`perform_remove`** (act.item.c:1712): NODROP→`You can't remove $p, it must be CURSED!`;
  carry-count full→`$p: you can't carry that many items!`; else unequip→inv,
  `You stop using $p.` (char) + `$n stops using $p.` (room).
- **`do_remove`** (act.item.c:1732): `one_argument`; `find_all_dots`; `remove all` loops
  `0..NUM_WEARS` (`You're not using anything.` if none); `remove all.x` matches by keyword
  (`You don't seem to be using any %ss.`); individual via `get_object_in_equip_vis`
  (`You don't seem to be using %s %s.`).

Add game entry points `DoWear`/`DoWield`/`DoGrab`/`DoRemove(ch *Player, arg string)` plus
`getObjInInvVis`/`getObjInEquipVis` helpers (visible carried / equipped by keyword, the latter
returning the C `WEAR_` index). `an()`, `isname`, `canSeeObject`, `findAllDots`, `ch.IsAffected`,
`ch.MaxWieldWeight`, `obj.GetWeight`, `obj.GetTypeFlag` all already exist in `pkg/game`.

## 6. Session adoption

Replace the four handler bodies in `pkg/session/cmd_inventory.go` with thin delegations
(`s.manager.world.DoWear(s.player, strings.Join(args," "))`, etc.), keep
`s.markDirty(VarInventory, VarEquipment)`, and **delete `broadcastEquipmentChange`**. `grab` is
already aliased to `cmdHold` (commands.go:95) — route it to `DoGrab`. Check whether `an()` in
cmd_inventory.go has other session callers before deleting it. Preserve the anti-align/anti-class
**zap** (it now lives in `performWear` via `objAntiAlign`/`objInvalidClass` — mirror the predicate
logic already in `Equipment.EquipForPlayer`, splitting it into the two distinct C messages).

## 7. Acceptance gate (all required)

1. **Oracle red→green:** `--scenario equipment` goes from the §1 divergences to **clean**
   (only expected/masked noise). Claude will run this — but run it yourself first.
2. **Observer broadcasts** are part of the diff (peer `observer`) — they must match C, not just
   the actor lines.
3. **Unit tests** for the room-observer wording of each verb (`$n wears $p on $s body.`,
   `$n wields $p.`, `$n grabs $p.`, `$n stops using $p.`) — the oracle proves actor+observer
   here, but lock them so they can't silently regress. Also a slot-mapping test (equip at
   `eqWearWield` then `IsEquipped(eqWearWield)` true, `performRemove(eqWearWield)` removes it).
4. **Instance-safe:** never write `obj.Prototype.*`. **No WS schema break:** the
   `DoInventory`/`DoEquipment` golden (`protocol_schema_test.go`) stays green.
5. `make check-fmt vet` + `go test ./...` green.

## 8. Gotchas (learned establishing the RED)

- **Recall misaligns C and Go start rooms** (DP-1085: C `recall`→8162 Infirmary, Go→8004 Altar),
  and spawned floor items decay during the slower two-character setup. The scenario sidesteps
  both by probing the **starter-worn-not** gear (newbie carries tunic 8019 + sword 8037 per
  `class.c:522-530`); don't reintroduce a fixture/recall dependency.
- **ANSI blind spot:** the oracle normalizer strips color, so any color parity must be unit-tested
  separately (see [[darkpawns-fidelity-testing]]).
- **The starter tunic is take-name** — keep `remove` probes on the sword, per §3.
