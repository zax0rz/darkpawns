# Marathon Code Audit — pkg/game/ & pkg/session/

**Date:** 2026-05-15
**Auditor:** Reek (Daeron subagent)
**Scope:** Data races, nil dereferences, logic bugs, port fidelity, error handling, resource leaks
**Files audited:** 40+ files across pkg/game/ and pkg/session/

---

## CRITICAL

### CRIT-012 — Data Race: Player.mu lock ordering inversion in performGiveGold (DEADLOCK)

**File:** `pkg/game/item_transfer.go:192-210`
**What:** `performGiveGold()` acquires `ch.mu.Lock()` then `vict.mu.Lock()` when `ch != vict`. If player A gives gold to player B while player B simultaneously gives gold to player A, goroutine 1 locks A→B while goroutine 2 locks B→A. Classic ABBA deadlock.
**Why it matters:** Two players exchanging gold simultaneously will deadlock the server. All other goroutines blocked on those player mutexes will also stall. Server-wide freeze.
**Suggested fix:** Acquire both locks in consistent order (e.g., always lock by pointer address: `if uintptr(unsafe.Pointer(&ch.mu)) < uintptr(unsafe.Pointer(&vict.mu))`). Or use a single global transaction lock for gold transfers.

---

### CRIT-013 — Data Race: Inventory.Items slice accessed without lock across 20+ files

**Files:** `pkg/game/item_equipment.go:222,247,266,297`, `pkg/game/act_movement.go:183,206`, `pkg/game/item_container.go:62,94`, `pkg/game/item_transfer.go:78,247`, `pkg/game/look.go:252`, `pkg/game/info_commands.go:68`, `pkg/game/combat_ranged.go:60`, `pkg/game/skills.go:302`, `pkg/game/item_door.go:25,75,121,163,194,236`
**What:** `ch.Inventory.Items` is accessed directly (iterating, reading, checking length) without acquiring `Inventory.mu`. The `Inventory.mu` is only acquired in the exported `AddItem`/`RemoveItem`/`FindItem` methods, but the vast majority of code reads `ch.Inventory.Items` directly.
**Why it matters:** While `Inventory.mu` protects write paths, any concurrent read of the `Items` slice while a write is happening (e.g., from a command handler in a different goroutine) can cause a concurrent map/slice read/write panic or corrupted iteration. The combat engine ticker goroutine triggers player inventory mutations (autoloot on kill) while command handlers also iterate inventory.
**Suggested fix:** Either (a) add `Inventory.mu.RLock()` in all direct `Items` access paths, or (b) add a `GetItems()` method that returns a snapshot copy under RLock, and use it everywhere.

---

### CRIT-014 — Data Race: Player fields accessed directly (RoomVNum, Level, Gold, Health) bypassing mutex

**Files and lines:**
- `pkg/game/char_mgmt.go:120` — `p.RoomVNum = roomNowhere` (bypasses `SetRoom()`)
- `pkg/game/combat_ranged.go:174` — `target.RoomVNum = ch.RoomVNum` (bypasses `SetRoom()`)
- `pkg/game/combat_basic.go:388` — `ch.Level * 2` (bypasses `GetLevel()`)
- `pkg/game/combat_basic.go:397` — `ch.Level < 2` (bypasses `GetLevel()`)
- `pkg/game/info_commands.go:34` — `ch.AC, ch.Exp, ch.Gold` (bypasses getters)
- `pkg/game/death.go:195` — `mobGold = deadMob.GetGold()` OK, but `mobGold := deadMob.GetGold()` in death handler accesses mob fields without mob.mu
- `pkg/game/act_movement.go:283,384` — `ch.RoomVNum` read without lock (bypasses `GetRoom()`)
- `pkg/game/spec_procs4.go:431,446` — `ch.Health /= 2` (bypasses `Heal`/`TakeDamage`)
- `pkg/game/world_scriptable.go:449-454` — `p.Inventory.Items` slice mutation without any lock

**What:** Player fields like `RoomVNum`, `Level`, `Health`, `Gold` are accessed directly without acquiring `p.mu`. The Player struct has proper `GetRoom()`/`SetRoom()`/`GetLevel()`/`GetHP()` methods that acquire locks, but many callers bypass them.
**Why it matters:** The combat engine runs on a separate goroutine. When the combat engine reads `target.RoomVNum` or `ch.Level` while a player goroutine writes to the same field, this is a data race. Go's race detector will flag this; in production it can cause stale reads or torn values.
**Suggested fix:** Replace all direct field accesses with their corresponding getter/setter methods. Add a linter rule to catch bare field access on Player.

---

## HIGH

### HIGH-021 — Data Race: Equipment.Slots accessed without Equipment.mu in GetHitroll/GetDamroll

**File:** `pkg/game/player_combat.go:66,97`
**What:** `GetHitroll()` and `GetDamroll()` iterate `p.Equipment.Slots` directly without acquiring `Equipment.mu`. The `Equipment` struct has its own `sync.RWMutex` that protects `Slots`, but these methods skip it.
**Why it matters:** Combat calculations run on the combat engine goroutine while equipment changes happen on player goroutines. Concurrent map iteration + map write = panic.
**Suggested fix:** Add `p.Equipment.mu.RLock()` / `defer p.Equipment.mu.RUnlock()` at the start of both methods. Or use `p.Equipment.GetEquippedItems()` which already acquires the lock.

---

### HIGH-022 — Data Race: GetAC() releases player.mu before accessing Equipment

**File:** `pkg/game/player_stats.go:78-88`
**What:** `GetAC()` acquires `p.mu.RLock()`, reads `p.AC`, then **releases** `p.mu.RUnlock()`, then reads `p.Equipment.GetArmorClass()`. Between the RUnlock and the Equipment access, another goroutine could modify the player's equipment, making the returned AC inconsistent with the equipment state.
**Why it matters:** A player could appear to have different AC than their equipment actually provides, leading to incorrect combat calculations.
**Suggested fix:** Hold `p.mu.RLock()` for the entire method, or calculate both values under a single lock acquisition.

---

### HIGH-023 — Weight check in canTakeObj uses wrong capacity formula

**File:** `pkg/game/item_transfer.go:28`
**What:** `ch.Inventory.GetWeight() + obj.GetWeight() > ch.Inventory.Capacity * 10`. The weight limit is `Capacity * 10`, but `Capacity` is item count (from `SetCapacity(dex, level) = 5 + dex/2 + level/2`). The C source uses `CAN_CARRY_W(ch)` which is based on `str_app[GET_STR(ch)].carry_w`, not item count * 10.
**Why it matters:** Weight limits are incorrect — a level 40/DEX 20 character gets Capacity=35, weight limit=350. The original C formula would give a different (likely higher) limit based on strength. Players may be incorrectly blocked from picking up items, or allowed to carry too much.
**Suggested fix:** Implement proper `CAN_CARRY_W` based on the `str_app` table, matching the C source. The current formula is a rough approximation that diverges from original behavior.

---

### HIGH-024 — Dead code: `vict.IsNPC() && false` condition in multiple combat functions

**Files:**
- `pkg/game/combat_melee.go:136` (doBash)
- `pkg/game/combat_advanced.go:193` (doSubdue)
- `pkg/game/combat_advanced.go:289` (doSleeper)
- `pkg/game/combat_advanced.go:384` (doAmbush)

**What:** The condition `vict.IsNPC() && false` always evaluates to false, making the `prob = 0` line unreachable. This appears to be debug scaffolding left from development (possibly testing "always miss NPCs").
**Why it matters:** Not a bug per se, but it's dead code that could confuse future maintainers. If someone removes the `&& false` thinking it's a mistake, it would change combat behavior (all NPC attacks would always miss).
**Suggested fix:** Either remove the dead block entirely, or document why it's intentionally disabled with a comment like `// NPC immunity disabled — was used during testing`.

---

### HIGH-025 — lookAtChar always reports "excellent condition" regardless of actual health

**File:** `pkg/game/look.go:289-298`
**What:** `lookAtChar()` hardcodes the string `"is in excellent condition."` for all players and mobs, regardless of their actual HP.
**Why it matters:** Players cannot assess enemy health through `look <player>`, breaking a core MUD mechanic. The C source has a tiered condition report based on HP percentage (excellent/good/fair/wounded/bad/terrible/awful).
**Suggested fix:** Implement the tiered condition check:
```go
pct := (target.GetHP() * 100) / target.GetMaxHP()
switch {
case pct >= 100: "is in excellent condition."
case pct >= 80:  "is in good condition."
case pct >= 60:  "is in fair condition."
case pct >= 40:  "is slightly wounded."
case pct >= 20:  "is in bad condition."
case pct > 0:    "is in awful condition."
default:         "is in terrible condition."
}
```

---

### HIGH-026 — Missing up/down arrival messages in doSimpleMove

**File:** `pkg/game/act_movement.go:310-348`
**What:** The arrival message switch only handles directions 0-3 (north/east/south/west) in the first case. Directions 4 (up) and 5 (down) are only handled in the outer switch, but the `direct` variable (used in the inner switch) is only set for directions 0-3. For up/down movement, the code falls through correctly, but if a flying character moves up/down, the message construction is correct — however, the `direct` variable is never set for up/down, which means it's unused for those cases. This is a minor inconsistency but not a bug.
**Actually:** On closer review, the code is structured as an outer switch with case 0,1,2,3 in one block and case 4,5 in separate blocks. This is correct. The `direct` variable is only used in the case 0-3 block. **This finding is REJECTED** — the code is structurally correct.

---

### HIGH-027 — Sector check in doAmbush uses wrong operators

**File:** `pkg/game/combat_advanced.go:343-345`
**What:** `sector != 3 && sector != 4 && sector != 5 && sector != 1` should probably use `||` instead of `&&`, or be structured as a whitelist. The current logic requires ALL four conditions to be true simultaneously to enter the block, but `&&` means ALL must be true — so only sectors that are simultaneously 3 AND 4 AND 5 AND 1 would pass (impossible). Wait — actually `!=` with `&&` means "is not 3 AND is not 4 AND is not 5 AND is not 1" — so this rejects sectors 1, 3, 4, 5 and allows everything else. The C source allows ambush in FOREST(3), HILLS(4), MOUNTAIN(5), and possibly FIELD(2). The Go code allows only sectors 0, 2, 6-15 (inside, city, water, desert, etc.) — the **opposite** of what's intended.
**Why it matters:** Ambush is allowed in cities, water, and fire sectors but blocked in forests, hills, and mountains. This is inverted from the C source which allows ambush in natural terrain.
**Suggested fix:** Change to whitelist: `if sector != 3 && sector != 4 && sector != 5 && sector != 2` (forests, hills, mountains, fields allowed; everything else blocked).

---

## MEDIUM

### MED-028 — Item transfer weight check uses Capacity*10 instead of proper carry_w

**File:** `pkg/game/item_transfer.go:28,337`
**What:** `canTakeObj()` and `performGive()` both check `ch.Inventory.GetWeight() + obj.GetWeight() > ch.Inventory.Capacity * 10`. The multiplier of 10 is arbitrary and doesn't match the C source's `CAN_CARRY_W(ch)` formula which uses `str_app[GET_STR(ch)].carry_w`.
**Why it matters:** Weight limits are wrong in both directions — some items that should be droppable aren't, and some that should be blocked aren't. Affects gameplay balance.
**Suggested fix:** Implement proper carry_w based on strength table from C source.

---

### MED-029 — Missing error return on JWT generation failure in completeCharCreation

**File:** `pkg/session/char_creation.go:164-166`
**What:** If `auth.GenerateJWT()` fails, the error is logged but `token` is empty string. The player is still registered and added to the world without a valid token. The `sendWelcome(token)` call sends an empty token to the client.
**Why it matters:** The player can play, but the client has no JWT for reconnection. If they disconnect, they can't resume their session.
**Suggested fix:** Return the error from `completeCharCreation()` and handle it at the caller (either retry JWT generation or abort creation).

---

### MED-030 — doLook accesses roomItems map without any lock

**File:** `pkg/game/look.go:132`
**What:** `items := w.roomItems[room.VNum]` reads the `roomItems` map directly without acquiring `w.mu`. The `roomItems` map is protected by `w.mu` in other code paths (e.g., `MoveObjectToRoom`).
**Why it matters:** If a player types `look` while another player drops an item in the same room, concurrent map read/write can cause a panic.
**Suggested fix:** Acquire `w.mu.RLock()` before accessing `w.roomItems`, or use a dedicated `GetRoomItems(roomVNum)` method.

---

### MED-031 — heal/2 in spec_procs4.go bypasses player.mu

**File:** `pkg/game/spec_procs4.go:431,446`
**What:** `ch.Health /= 2` directly mutates the Health field without acquiring `ch.mu`. The `Player` struct has `Heal()` and `TakeDamage()` methods that properly lock.
**Why it matters:** Data race with combat engine goroutine that reads/writes Health.
**Suggested fix:** Use `ch.TakeDamage(ch.GetHP() / 2)` or add a locking wrapper.

---

### MED-032 — ItemPrototype nil dereference risk in lookAtChar's listCharToChar

**File:** `pkg/game/look.go:147`
**What:** `m.VNum == ch.ID` — comparing mob VNum with player ID. These are different namespaces (mob VNums are world file numbers, player IDs are database IDs). This check is intended to prevent the player from seeing themselves listed as a mob, but the comparison is meaningless.
**Why it matters:** Not a crash risk, but the self-exclusion check is broken — a player will never match a mob VNum, so this line is dead code. If a player somehow had the same ID as a mob VNum (unlikely but possible), they'd be incorrectly hidden from the room listing.
**Suggested fix:** Use `m.GetName() == ch.GetName()` or remove the check.

---

### MED-033 — performWear removes item from inventory before equip confirmation

**File:** `pkg/game/item_equipment.go:143`
**What:** `ch.Inventory.removeItem(obj)` is called before `w.EquipItem(ch, obj, where)`. If `EquipItem` fails, the item is already removed from inventory. The rollback at line 148-151 tries to restore it, but this is a TOCTOU issue — the item could be GC'd or referenced elsewhere between removal and rollback.
**Why it matters:** Low probability but possible item duplication or loss if the rollback fails (which the code logs as an error).
**Suggested fix:** Try equip first without removing from inventory, then remove on success. Or use a transactional pattern.

---

## LOW

### LOW-008 — `randRange` uses global `math/rand` without seeding

**File:** `pkg/game/combat_helpers.go` (and multiple other files)
**What:** `randRange()` and `rand.Intn()` use the global `math/rand` source. In Go 1.20+, the global source is automatically seeded, so this is not a bug. However, for reproducible testing, having a seeded source would be better.
**Why it matters:** No gameplay impact. Testing could benefit from deterministic RNG.
**Suggested fix:** Acceptable as-is. Consider injecting a `*rand.Rand` for testability.

---

### LOW-009 — Missing `\r\n` in some error messages

**Files:**
- `pkg/game/item_transfer.go:41` — `"You can't carry that much.\n"` (missing `\r`)
- `pkg/game/item_transfer.go:106` — `"You can't drop that right now.\n"` (missing `\r`)
- `pkg/game/item_transfer.go:160` — `"$E can't carry that much weight.\n"` (missing `\r`)
- `pkg/game/item_transfer.go:175` — `"$E can't carry any more.\n"` (missing `\r`)

**What:** Some messages use `\n` instead of `\r\n`. WebSocket clients may not render the line break correctly without the carriage return.
**Why it matters:** Cosmetic — messages may appear on the same line as subsequent output.
**Suggested fix:** Replace `\n` with `\r\n` in all player-facing messages.

---

### LOW-010 — equip() returns after first successful slot instead of trying all wearFlags

**File:** `pkg/game/equipment.go:120-165`
**What:** The `equip()` method iterates `wearFlags` but returns after the first successful slot assignment. If an item has multiple wear flags (e.g., both finger and wrist), only the first one is tried. This matches C behavior (first valid slot wins).
**Why it matters:** Not a bug — matches C source. Noted for port fidelity documentation.

---

## Summary

| Severity | New | Already Tracked |
|----------|-----|-----------------|
| CRITICAL | 3 (CRIT-012, CRIT-013, CRIT-014) | 11 (all fixed) |
| HIGH | 5 (HIGH-021 through HIGH-027, excluding rejected) | 20 (all fixed) |
| MEDIUM | 6 (MED-028 through MED-033) | 27 (all fixed) |
| LOW | 3 (LOW-008 through LOW-010) | 7 (all fixed) |

**Top priority fixes:**
1. **CRIT-012** — Deadlock risk in performGiveGold lock ordering (immediate fix needed)
2. **CRIT-013** — Inventory.Items data race (affects all inventory-using commands)
3. **CRIT-014** — Player field direct access (systemic issue, ~30 call sites)
4. **HIGH-027** — Ambush sector check inverted (gameplay-breaking, trivial fix)
5. **HIGH-025** — lookAtChar always shows "excellent condition" (core MUD mechanic missing)
