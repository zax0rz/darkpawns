# Port Fidelity Audit: Module 48 (`shop.c`)

This audit examines the port fidelity between the legacy C source file `src/shop.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/shop.c` (1,445 lines)
- **Functions & Features**:
  - **Shopkeeper Transactions**: Implements shopkeeper commands (`buy`, `sell`, `list`, `value`) allowing players to trade items.
  - **Charisma-Based Dynamic Pricing**: Calculates item prices using float profit margins (`SHOP_BUYPROFIT`, `SHOP_SELLPROFIT`) modified by the player's Charisma attribute.
  - **Keeper Wealth & Trade Constraints**: Restricts trades based on shopkeeper gold limits (`bankAccount`), business hours, and player race/flag constraints (`with_who` trade limitations).
  - **Temper Behaviors**: Implements keeper verbal abuse or refusal reactions (`temper`) when players try to buy/sell unapproved items or lack cash.

### Go Port Files
- **Go Implementation**:
  - **SPLIT-BRAIN REDUNDANT IMPLEMENTATIONS**: The Go port features **two completely separate, duplicate implementations** of the entire Shop system:
    1. **UNUSED**: [pkg/game/shop.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/shop.go): A CircleMUD-faithful lightweight implementation utilizing float multipliers and including full Charisma dynamic pricing calculations. (Unused in production, referenced only in one admin test file).
    2. **ACTIVE**: [pkg/game/systems/shop.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/systems/shop.go) and [pkg/game/systems/shop_manager.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/systems/shop_manager.go): The active shop engine instantiated by the Session Manager, using integer-percentage multipliers, managing dynamic slices for inventories, and handling restock loops.
  - **Shop Commands Parser**:
    - [pkg/command/shop_commands.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/command/shop_commands.go): Processes commands (`CmdBuy`, `CmdSell`, `CmdRepair`, `CmdIdentify`, `CmdValue`).

---

## 2. High-Fidelity Validation & Critical Gaps

Comparing the active `systems.Shop` implementation against legacy `src/shop.c` reveals several **critical game balance gaps** and **redundancies**:

### 1. Split-Brain Architectural Duplication
- **The Gap**: Having two separate `Shop` and `ShopManager` types across `pkg/game/` and `pkg/game/systems/` causes substantial code confusion. The faithful `game.Shop` was fully coded but bypassed, while `systems.Shop` was hooked up to the production Session Manager.
- **Impact**: Increased codebase bloat, difficulty maintaining shop logic, and high risk of developers editing the wrong file.

### 2. Player Charisma Completely Ignored in Active Pricing
- **The Gap**: In `src/shop.c#L111` and the unused Go `game.Shop.BuyPrice` / `SellPrice` routines, player Charisma provides dynamic trade discounts/markups:
  ```go
  if cha > 0 {
      price -= price * (float64(cha) * 0.005)
  }
  ```
- In the active `systems.Shop.CalculateBuyPrice` and `CalculateSellPrice` methods (`pkg/game/systems/shop.go#L109-L140`), the player's Charisma attribute is **completely ignored**, using raw static integer percentages instead:
  ```go
  price := (baseCost * s.BuyMultiplier) / 100
  ```
- **Impact**: Breaks the MUD economic progression. Players with high Charisma (such as specialized merchants or bards) get zero trade benefits, severely hurting character build utility and game balance.

### 3. Infinite Keeper Gold (No Shopkeeper Cash Limits)
- **The Gap**: Legacy C shopkeepers have a `bankAccount` limit. If the shopkeeper runs out of money, they refuse to buy any more items from players (`sedit` S_NOCASH1: `I can't afford that!`).
- The active Go `systems.Shop` has **no concept of keeper gold limits**. Shopkeepers have infinite cash and will purchase infinite high-level items from players.
- **Impact**: Under high player farming, characters can generate infinite currency by dumping junk items onto shopkeepers, leading to runaway gold inflation.

### 4. Trade Constraints & Outlaw/Race Blockages Omitted
- **The Gap**: In `src/shop.c`, the `with_who` bitvector prevents shopkeepers from trading with specific classes, races, outlaws, or players of opposing alignments.
- The active Go `systems.Shop` has **no trade blockages**. Dark/Evil shopkeepers will trade with Light/Good holy players, and reputable town blacksmiths will happily trade with wanted Outlaws and Werewolves, destroying MUD roleplay constraints.

---

## 3. Go's Architectural Improvements Over C

Despite the pricing gaps, Go's active `systems.Shop` introduces exceptional engineering upgrades:
1. **Dynamic Restock Loops**: Go implements an automated `Restock` routine (`pkg/game/systems/shop.go#L280`) driven by game tick intervals, replenishing empty shop stocks safely.
2. **Built-in Repair & Identify Services**: Shopkeepers have typed `RepairSkill` and `IdentifySkill` values, implementing robust, scaled repair and identification algorithms based on item values.
3. **Dynamic Slice Inventories**: C shops utilized static size limits. Go shops manage dynamic slices (`[]common.ObjectInstance`), safely tracking items with full custom state support.

---

## 5. Summary of Recommended Next Steps

1. **Unify Split-Brain Implementations**:
   Delete the unused `pkg/game/shop.go` file. Consolidate all shop structures into the active `pkg/game/systems/` package, renaming the tests accordingly.
2. **Restore Charisma Dynamic Pricing**:
   Update `CalculateBuyPrice` and `CalculateSellPrice` in `pkg/game/systems/shop.go` to accept the player's Charisma attribute and apply the Diku-faithful 0.5% discount/markup per Charisma point.
3. **Implement Shopkeeper Bank Account Limits**:
   Add a `Gold` attribute to `systems.Shop` to track shopkeeper cash, preventing transactions when the keeper runs out of funds, and allowing them to replenish gold slowly over restock ticks.
4. **Enforce Outlaw & Alignment Trade Constraints**:
   Read and validate `with_who` flags from `.shp` files to prevent shopkeepers from dealing with wanted criminals, opposing alignments, or select races.
