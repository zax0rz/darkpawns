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
	SetPosition(pos int)

	// Class and ability scores (Phase 2c additions)
	GetClass() int
	GetStr() int
	GetStrAdd() int // For 18/xx exceptional strength
	GetDex() int
	GetInt() int
	GetWis() int
	GetHitroll() int
	GetDamroll() int
	GetSex() int

	// Combat actions
	TakeDamage(amount int)
	Heal(amount int)
	SetFightingBody(target Combatant)
	StopFighting()
	GetFighting() string // display only
	GetFightingBody() Combatant

	// Messaging
	SendMessage(msg string)
}

// BodyRetired reads the body's transient retirement marker, without a registry.
func BodyRetired(body Combatant) bool {
	if retired, ok := body.(interface{ CombatRetired() bool }); ok {
		return retired.CombatRetired()
	}
	return false
}
