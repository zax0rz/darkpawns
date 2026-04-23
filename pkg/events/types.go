package events

import "time"

// --- Combat events ---

// MobKilledEvent is emitted when a mob is killed.
type MobKilledEvent struct {
	KillerID string
	MobVNum  int
	RoomVNum int
}

func (e MobKilledEvent) Type() string { return "combat.mob_killed" }

// PlayerKilledEvent is emitted when a player kills another player.
type PlayerKilledEvent struct {
	KillerID   string
	VictimID   string
	RoomVNum   int
	DamageType string
}

func (e PlayerKilledEvent) Type() string { return "combat.player_killed" }

// DamageDealtEvent is emitted when damage is dealt.
type DamageDealtEvent struct {
	AttackerID string
	TargetID   string
	Amount     int
	DamageType string
}

func (e DamageDealtEvent) Type() string { return "combat.damage_dealt" }

// --- Player events ---

// PlayerLeveledEvent is emitted when a player gains a level.
type PlayerLeveledEvent struct {
	PlayerID string
	NewLevel int
	RoomVNum int
}

func (e PlayerLeveledEvent) Type() string { return "player.leveled" }

// PlayerConnectedEvent is emitted when a player connects.
type PlayerConnectedEvent struct {
	PlayerID string
}

func (e PlayerConnectedEvent) Type() string { return "player.connected" }

// PlayerDisconnectedEvent is emitted when a player disconnects.
type PlayerDisconnectedEvent struct {
	PlayerID string
}

func (e PlayerDisconnectedEvent) Type() string { return "player.disconnected" }

// --- Economy events ---

// ItemBoughtEvent is emitted when a player buys an item from a shop.
type ItemBoughtEvent struct {
	PlayerID string
	ItemName string
	Price    int
	ShopVNum int
}

func (e ItemBoughtEvent) Type() string { return "economy.item_bought" }

// ItemSoldEvent is emitted when a player sells an item to a shop.
type ItemSoldEvent struct {
	PlayerID string
	ItemName string
	Price    int
	ShopVNum int
}

func (e ItemSoldEvent) Type() string { return "economy.item_sold" }

// GoldEarnedEvent is emitted when a player earns gold (loot, sell, quest).
type GoldEarnedEvent struct {
	PlayerID string
	Amount   int
	Source   string
}

func (e GoldEarnedEvent) Type() string { return "economy.gold_earned" }

// --- World events ---

// RoomEnteredEvent is emitted when a player enters a room.
type RoomEnteredEvent struct {
	PlayerID string
	RoomVNum int
	FromVNum int
}

func (e RoomEnteredEvent) Type() string { return "world.room_entered" }

// MobSpawnedEvent is emitted when a mob is spawned.
type MobSpawnedEvent struct {
	MobVNum  int
	RoomVNum int
}

func (e MobSpawnedEvent) Type() string { return "world.mob_spawned" }

// --- Game events ---

// CommandExecutedEvent is emitted when a command is executed.
type CommandExecutedEvent struct {
	PlayerID    string
	CommandName string
	Args        string
	Duration    time.Duration
	Timestamp   time.Time
}

func (e CommandExecutedEvent) Type() string { return "game.command_executed" }

// --- Admin events ---

// WizardCommandEvent is emitted when a wizard command is executed.
type WizardCommandEvent struct {
	PlayerID    string
	CommandName string
	TargetID    string
	Success     bool
}

func (e WizardCommandEvent) Type() string { return "admin.wizard_command" }
