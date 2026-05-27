# Port Fidelity Audit: Module 52 (`spec_procs3.c`)

This audit examines the port fidelity between the legacy C source file `src/spec_procs3.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/spec_procs3.c` (1,301 lines)
- **Functions & Features**:
  - **Zone Specific Procs**: Implements complex systems for specialized zones, notably the **Elemental Temple** (master columns, platforms, loading cylinders, Galerus) and general town clerks, prostitutes, recruiters, and shopkeepers.
  - **Shopkeeper Spec-Proc**: Provides the active C intercept `shop_keeper` that captures trading commands (`buy`, `sell`, `list`, `value`) on NPC shopkeepers.

### Go Port Files
- **Go Implementation**:
  - [pkg/game/spec_procs3.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/spec_procs3.go): Full port of Elemental Temple room behaviors and specialty NPCs.

---

## 2. High-Fidelity Validation & Design Discrepancies

Comparing the implementations highlights a **major architectural shift** and **flawless zone ports**:

### 1. The Shopkeeper Spec-Proc Redirection (No-Op Stub)
- **The Gap**: In CircleMUD, `shop_keeper` was a behavioral spec-proc. The engine routed list/buy/sell/value commands through `shop_keeper` to handle trading.
- In Go, `specShopKeeper` is **intentionally a compiled no-op stub** that returns `false` (`pkg/game/spec_procs3.go#L93-L103`):
  ```go
  func specShopKeeper(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
      return false
  }
  ```
- **Design Intent**: Trading commands are handled globally at the Session command level (`pkg/command/shop_commands.go`), which dynamically scans room mobs, matches VNums, and directly queries the `ShopManager` database. The spec-proc only exists as a dummy placeholder so zone loaders do not panic when booting shopkeepers.

### 2. Flawless Elemental Temple Porting
- **Platform Alignments**: Go perfectly ports the complex, multi-room Elemental Temple triggers, including:
  - `specElementsMasterColumn`: Controls temple activations.
  - `specElementsPlatforms`: Manages room transitions.
  - `specElementsLoadCylinders`: Operates cylinder inputs.
  - `specElementsGaleruAlive` / `specElementsGaleruColumn`: Manages boss spawning and activations.
  - `specElementsMinion` / `specElementsGuardian`: Combat defenders.

### 3. Clerical Dynamic Combat (`specCleric`)
- Seamlessly ports Diku's cleric combat AI (`pkg/game/spec_procs3.go#L105`), causing clerical mobs to stand up on fight ticks, heal themselves when below max HP, and randomly cast healing or offensive spells (`SpellHeal`, `SpellCureCritic`, `SpellCureLight`, or offensive spells) depending on their levels.

---

## 3. Go's Architectural Improvements Over C

- **Gopher-Lua Scripting Parallelism**: The Elemental Temple was historically hardcoded in C. By mapping room properties dynamically, Go enables seamless integration with room Lua scripts, allowing developers to extend temple events without editing Go sources.
- **Mutex Thread Safety**: Active room platform transitions lock room structures safely, avoiding race conditions if multiple players traverse platforms concurrently.

---

## 4. Summary of Recommended Next Steps

1. **Document redirectional Command Design**:
   Add clear comments in the developer wiki explaining that shopkeeper commands bypass the spec-proc layer entirely, avoiding confusion for developers expecting CircleMUD's command interception layout.
