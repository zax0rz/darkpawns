# Shop System for Dark Pawns

## Overview

The shop system allows NPCs to function as shopkeepers that can buy, sell, repair, and identify items. Shops are integrated with the existing NPC, item, and player systems.

## Architecture

### Core Components

1. **Shop** (`pkg/world/shop.go`): Represents a shopkeeper NPC with inventory, price multipliers, and services.
2. **ShopManager** (`pkg/world/shop_manager.go`): Manages all shops in the game world.
3. **Shop Commands** (`pkg/command/shop_commands.go`): Player commands for interacting with shops.
4. **World Integration** (`pkg/game/world.go`): Shop manager integrated into the game world.

### Key Features

- **Buy/Sell Transactions**: Players can buy items from shops and sell items to shops.
- **Price Multipliers**: Shops have configurable buy/sell price multipliers (e.g., buy at 50% of value, sell at 150%).
- **Item Type Filtering**: Shops can be configured to only deal in specific item types.
- **Repair Service**: Shops can repair damaged equipment (requires repair skill).
- **Identification Service**: Shops can identify magical or unknown items.
- **Inventory Management**: Shops have limited inventory space and can restock.
- **Business Hours**: Shops can have opening and closing hours (future feature).

## Usage

### Player Commands

| Command | Description | Example |
|---------|-------------|---------|
| `list` | Show shop inventory | `list` |
| `buy <item>` | Buy an item from shop | `buy sword` or `buy 1` |
| `sell <item>` | Sell an item to shop | `sell sword` |
| `repair <item>` | Repair an item | `repair sword` |
| `identify <item>` | Identify an item | `identify potion` |
| `value <item>` | Check item's buy/sell price | `value sword` |

### Creating Shops Programmatically

```go
// Create a shop manager
shopManager := world.NewShopManager()

// Create a shop (associated with NPC VNum 1001 in room 3001)
shop := shopManager.CreateShop(1001, "Blacksmith", 3001)

// Configure what types of items the shop deals in
shop.ItemTypes = []int{2, 3} // Weapons (2) and armor (3)

// Configure price multipliers
shop.BuyMultiplier = 40  // Pays 40% of item value
shop.SellMultiplier = 200 // Sells at 200% of item value

// Configure services
shop.RepairSkill = 80    // 80% repair success chance
shop.IdentifySkill = 95  // 95% identify success chance
shop.RepairCost = 15     // 15 gold per damage point
shop.IdentifyCost = 10   // 10 gold per identification

// Add items to shop inventory
item := game.NewObjectInstance(weaponProto, -1)
shop.AddItem(item)
```

### Shop Configuration

#### Item Types
Shops can be configured to only deal in specific item types. Common type flags:
- `1`: Container
- `2`: Weapon
- `3`: Armor
- `4`: Potion
- `5`: Scroll
- `6`: Wand
- `7`: Staff
- `8`: Food
- `9`: Money
- `10`: Light source

#### Price Multipliers
- `BuyMultiplier`: Percentage of base cost the shop pays (default: 50)
- `SellMultiplier`: Percentage of base cost the shop charges (default: 150)

#### Services
- `RepairSkill`: 0-100 skill level for repairing items
- `IdentifySkill`: 0-100 skill level for identifying items
- `RepairCost`: Base cost per damage point
- `IdentifyCost`: Base cost per identification

#### Inventory
- `MaxItems`: Maximum items shop can stock (default: 50)
- `RestockInterval`: How often to restock (game ticks)
- `RestockPercent`: Chance to restock each item

## Integration with Existing Systems

### NPC Integration
Shops are associated with NPCs by VNum. When a player interacts with an NPC that has a shop, the shop system handles the transaction.

### Item System
Shops use the existing `ObjectInstance` class for items. All item properties (cost, type, weight, etc.) are preserved.

### Player System
Shops interact with player inventory and gold. Transactions validate:
- Player has enough gold (for buying)
- Player has inventory space (for buying)
- Shop has inventory space (for selling)
- Shop has enough gold (for buying - unlimited by default)

### Command System
Shop commands are registered with the session manager and follow the same pattern as other game commands.

## Database Persistence

Shop state (inventory, restock timers, etc.) can be saved to and loaded from the database using the `SaveShops()` and `LoadShops()` methods.

## Testing

Run shop system tests:
```bash
go test ./pkg/world/... -v
```

Tests cover:
- Shop creation and configuration
- Inventory management
- Price calculations
- Buy/sell transactions
- Shop manager operations

## Future Enhancements

1. **Shop Dialogues**: NPC-specific dialogues for shop interactions.
2. **Haggling System**: Players can negotiate prices based on charisma.
3. **Shop Reputation**: Shops remember players and offer better prices to regular customers.
4. **Limited Stock**: Rare items have limited quantities.
5. **Dynamic Pricing**: Prices change based on supply and demand.
6. **Shop Events**: Special sales, discounts, or rare item appearances.
7. **Player-owned Shops**: Players can own and run shops.

## Example Shop Configuration

Here's an example of a complete shop setup:

```go
// Create a blacksmith shop
blacksmith := shopManager.CreateShop(2001, "Gorak's Forge", 3050)
blacksmith.ItemTypes = []int{2, 3} // Weapons and armor
blacksmith.BuyMultiplier = 30      // Pays 30% - he's a tough negotiator
blacksmith.SellMultiplier = 250    // Sells at 250% - quality costs!
blacksmith.RepairSkill = 90        // Excellent repair skill
blacksmith.RepairCost = 20         // Expensive but worth it
blacksmith.MaxItems = 30           // Limited inventory

// Create a general store
generalStore := shopManager.CreateShop(2002, "Bazaar", 3020)
generalStore.ItemTypes = []int{1, 4, 5, 8} // Containers, potions, scrolls, food
generalStore.BuyMultiplier = 60            // Fair prices
generalStore.SellMultiplier = 140          // Reasonable markup
generalStore.MaxItems = 100                // Large inventory

// Create a magic shop
magicShop := shopManager.CreateShop(2003, "Arcane Emporium", 3080)
magicShop.ItemTypes = []int{4, 5, 6, 7} // Potions, scrolls, wands, staves
magicShop.IdentifySkill = 100           // Perfect identification
magicShop.IdentifyCost = 50             // Expensive but reliable
magicShop.SellMultiplier = 300          // Magic items are expensive!
```

## Troubleshooting

### Common Issues

1. **"Shop has nothing for sale"**: The shop's inventory is empty or the shop doesn't deal in the item type.
2. **"You don't have enough gold"**: Player doesn't have enough gold for the transaction.
3. **"Your inventory is full"**: Player's inventory is at capacity.
4. **"Shop inventory is full"**: Shop can't buy more items.
5. **"Shop isn't interested in that"**: Shop doesn't buy/sell that item type.

### Debugging

Check shop state:
```go
fmt.Printf("Shop: %v\n", shop)
fmt.Printf("Inventory: %d items\n", len(shop.GetInventory()))
fmt.Printf("Buy types: %v\n", shop.BuyTypes)
fmt.Printf("Item types: %v\n", shop.ItemTypes)
```