# DP-1456 Unwitnessed Scenarios Diagnosis (`blocked-no-witness.tsv`)

Linear: DP-1456 (follow-up) · Rules: R1, R3, R5e, R5g, R5h, R5i, R5j

## Question

Why did the 22 scenarios in `docs/fidelity/evidence/dp-1456/blocked-no-witness.tsv` pass `cmd/dp-oracle-diff` after DP-1456's setup-refusal repair while the C transcript lacked the claimed witness event?

## Root Cause Summary

Contrary to the initial hypothesis that `spawn-obj`, `force-load`, or `no-settle` failed to place objects during boot, **`cmd/dp-oracle-diff` applies `spawn-obj` and `force-load` to the disposable `lib/world/` tree before either server starts** (`cmd/dp-oracle-diff/main.go:349-388`, `applyObjectSpawnFixtures` at `main.go:846-912`, `applyForceLoadFixtures` at `main.go:960-1035`), and C's `boot_db` executes `reset_zone` on every indexed zone at startup (`src/db.c:398-401`) regardless of `no-settle` (which only skips post-boot `~dpclock pulse` settling at `cmd/dp-oracle-diff/main.go:714-725`).

Instead, the 22 scenarios break down into **five distinct C call-path causes** ( plus 1 already-witnessed false positive on `blocked-no-witness.tsv`):

### 1. `wear <item>` vs `hold <item>` on `ITEM_WAND` / `ITEM_STAFF` (17 scenarios)

- **C call path:** `do_use` (`src/act.other.c:895-974`) requires a wand or staff to be **equipped**: it checks `GET_EQ(ch, WEAR_HOLD)` (`src/act.other.c:907`) and scans `GET_EQ(ch, i)` across `0..NUM_WEARS-1` (`src/act.other.c:926-936`). If the item is only in `ch->carrying` (inventory), `do_use` refuses with `"You don't seem to have a %s.\r\n"` (`src/act.other.c:933-935`).
- In `[warmup]`, these scenarios ran `get <item>` followed by `wear <item>`. `get <item>` succeeded (placing the item in inventory), but `do_wear` (`src/act.item.c:1584-1658`) dispatches through `find_eq_pos(ch, obj, 0)` (`src/act.item.c:1521-1581`), which checks `ITEM_WEAR_FINGER` through `ITEM_WEAR_WIELD` (`src/act.item.c:1553-1568`) and **never checks `ITEM_WEAR_HOLD` (`src/structs.h:460`)**.
- Because these prototypes have `wear_flags = 16385` (`ITEM_WEAR_TAKE | ITEM_WEAR_HOLD`) or `1` (`ITEM_WEAR_TAKE`, #4813), `do_wear` prints `"You can't wear $p."` (`src/act.item.c:1655`). Equipping in `WEAR_HOLD` requires `hold` / `grab` (`do_grab`, `src/interpreter.c:472,497`; `src/act.item.c:1685-1709`), which accepts `ITEM_WEAR_HOLD` or any `ITEM_WAND`/`ITEM_STAFF`/`ITEM_SCROLL`/`ITEM_POTION` (`src/act.item.c:1700-1704`) and calls `perform_wear(ch, obj, WEAR_HOLD)` (`src/act.item.c:1707`).
- Because `"You can't wear ..."` is object-parameterized (`$p`), it is not in `setupRefusalLines` (`internal/oraclediff/setuprefusal.go:37-140`), so warmup refusal checking passed while `[probe]` `equipment` printed `Nothing.` and `use` printed `You don't seem to have a ...`.
- **Affected (15 pure `wear`→`hold` + 2 compound):** `spellmatrix-world-staff-{acid-blast,dispel-evil,earthquake-live,sense-life-live,sleep,smokescreen}-depth`, `spellmatrix-world-wand-{charm-person,create-water,earthquake,energy-drain,fireball,group-invisibility-live,heal,lightning-bolt,soul-leech-live}-depth`, plus `spellmatrix-world-staff-cure-critic-depth` (also wrong keyword: prototype `#8055` at `lib/world/obj/80.obj:446-448` has namelist `staff~` and s-desc `a staff of healing~`, so `get healing` failed with `"You don't see a healing here."` at `src/act.item.c:236`) and `spellmatrix-world-wand-group-invisibility-live-depth` (also `group ItemPeer` without `group all`: `do_group` at `src/act.other.c:726-727` sets `AFF_GROUP` on `vict` only, so `mag_groups` at `src/magic.c:1474` returned early on `!IS_AFFECTED(ch, AFF_GROUP)`).

### 2. Unindexed `.obj` / `.zon` files in `lib/world/*/index` (2 scenarios)

- **C call path:** `boot_db` loads only files listed in `lib/world/obj/index` and `lib/world/zon/index` (`src/db.c:361-368,452-502`). Neither `131.obj`/`131.zon` nor `162.obj`/`162.zon` is in the shipped index files.
- `spellmatrix-world-scroll-harm-live-depth` (`#13107` in `131.obj`) and `spellmatrix-world-wand-smokescreen-live-depth` (`#16250` in `162.obj`) omitted `add-obj-index` and targeted unindexed zones `131` and `162` in `spawn-obj`. Adding `add-obj-index {131,162}.obj` and spawning in indexed zone `80` delivers both items.
- In addition, `#16250` (`lib/world/obj/162.obj:274-279`) has `extra_flags = 41472` (`ITEM_ANTI_GOOD | ITEM_ANTI_CLERIC | ITEM_ANTI_WARRIOR`, `src/structs.h:477-483`), so `ItemMage` with `align 1000` was zapped on `wear cloak` (`equip_char`, `src/handler.c:701-711`); setting `align -1000` allows `wear cloak` (into `WEAR_ABOUT`) to succeed.

### 3. Prototype `#7018` keyword, wield weight, and `ITEM_TWO_HANDED` (1 scenario)

- In `spellmatrix-world-staff-ray-of-disruption-depth`, `#7018` (`lib/world/obj/70.obj:201-208`) has namelist `staff~` (s-desc `a twisted staff~`), `wear_flags = 8193` (`ITEM_WEAR_TAKE | ITEM_WEAR_WIELD`), `extra_flags = 80863297` (including `ITEM_TWO_HANDED`, bit 16), and `weight = 15`.
- `get twisted` failed because `twisted` is not in `obj->name`; `wear staff` routes to `WEAR_WIELD` (`src/act.item.c:1568`), where `perform_wear` checks `GET_OBJ_WEIGHT(obj) > str_app[STRENGTH_APPLY_INDEX(ch)].wield_w` (`src/act.item.c:1492-1496`). A default Mage (STR ≤ 14, `wield_w <= 14`, `src/constants.c:366-386`) prints `"It is too heavy for you to use.\r\n"`. Setting `str 18` before self-demotion (`set ... str` requires `LVL_GOD` at `src/act.wizard.c:2624`) and using `get staff` / `wear staff` / `use staff` equips and fires `#7018`.

### 4. Prototype `#1211` missing `ITEM_WEAR_TAKE` (1 scenario)

- In `spellmatrix-world-wand-blindness-depth`, `#1211` (`lib/world/obj/12.obj:25-32`) has `wear_flags = 8192` (`ITEM_WEAR_WIELD` without `ITEM_WEAR_TAKE`). `can_take_obj` (`src/act.item.c:170`) refuses `get massager` with `"Mandelbrot's marvelous hand-held massager: you can't take that!"` (`src/act.item.c:186`). Loading `#1211` into inventory via `load obj 1211` (`do_load` calls `obj_to_char(obj, ch)` at `src/act.wizard.c:1368`) before self-demotion allows `wear massager` to wield and fire it.

### 5. `spellmatrix-scroll-olc-02-depth` (multi-block OLC + `group all` + recall/invisibility) and `set-name-cleanup`

- **`spellmatrix-scroll-olc-02-depth`:**
  1. Warmup ran `group ItemPeer` instead of `group all`, leaving `ItemMage` without `AFF_GROUP` (`src/act.other.c:726-727`), so `mag_groups` (`src/magic.c:1474`) skipped `group recall` (49) and `group invisibility` (57) on the first scroll.
  2. When DP-1456 hoisted the first `oedit 4404` + `load obj 4404` to `[warmup]`, `do_load` (`src/act.wizard.c:1368`) placed the first scroll directly in `ItemMage`'s inventory (`obj_to_char`), so the first `recite book ItemPeer` in `[probe]` fired `sobriety` on `ItemPeer` (`"You are splashed in the face with HOT coffee, but feel much more sober."`, `src/spells.c:696-697`). However, the 3 subsequent `[probe]` blocks ran `load obj 4404` as `ItemMage` (demoted to level 30 at the end of `[warmup]`), which printed `"Huh?!?"` (`src/interpreter.c:544,916`). Furthermore, once `group all` enables `group recall` (49) and `group invisibility` (57) on block 1, `ItemMage` and `ItemPeer` relocate to room `8003` (`mortal_start_room`, `src/spells.c:155`) with `AFF_INVISIBLE`. Giving `ItemBuilder` `holylight` (`PRF_HOLYLIGHT`) and `ItemMage` `detect invisibility` (`AFF_DETECT_INVIS`) in `[warmup]`, teleporting `ItemMage` and `ItemPeer` back to `8162` after block 1, and running `send:builder load obj 4404` + `send:builder drop book` on blocks 2–4 delivers all four scrolls and exposes two real C vs Go divergences (`mag_groups` target order in `recite book ItemPeer [actor]` and `stop_follower` timing in `damage()` at `src/fight.c:1457-1458`).
- **`set-name-cleanup`:** Already has its C witness in `[probe]` block 1 (`show rent Archivepeer` prints `plrobjs/A-E/archivepeer.objs` and the 5 starter items rented by `do_quit` in room `8004` via `Crash_rentsave(ch, 0)` at `src/act.other.c:88-90,167`, followed by `set file Archivepeer name Archiveother` deleting the old rent file at `src/act.wizard.c:2893` so block 3 prints `archivepeer has no rent file.`). It was included on `blocked-no-witness.tsv` by mistake during batch classification and is not `blocked` in any manifest.
