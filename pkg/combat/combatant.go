package combat

// DiceRoll represents NdS+P dice
type DiceRoll struct {
	Num   int
	Sides int
	Plus  int
}

// Combatant represents any entity that can participate in combat
// (players, mobs, etc.)
type Combatant interface {
	// Identity
	GetName() string
	IsNPC() bool

	// Location
	GetRoom() int

	// Stats
	GetLevel() int
	GetHP() int
	GetMaxHP() int
	GetAC() int
	GetTHAC0() int
	GetDamageRoll() DiceRoll
	GetPosition() int

	// Class and ability scores (Phase 2c additions)
	GetClass() int
	GetStr() int
	GetDex() int
	GetInt() int
	GetWis() int
	GetHitroll() int
	GetDamroll() int

	// Combat actions
	TakeDamage(amount int)
	Heal(amount int)
	SetFighting(target string)
	StopFighting()
	GetFighting() string

	// Messaging
	SendMessage(msg string)
}