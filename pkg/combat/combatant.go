// ARCHITECTURAL NOTE [M-05]: Combatant interface — 25 methods, name-based lookups
//
// This interface is large and mixes several unrelated responsibility domains.
// Prefer smaller, focused interfaces when adding new consumers:
//
//   Damageable:  TakeDamage, Heal, GetHP, GetMaxHP, GetAC
//   Positionable: GetRoom, GetPosition, SetFighting, StopFighting, GetFighting
//   Skilled:      GetLevel, GetClass, GetHitroll, GetDamroll, GetStr, GetStrAdd,
//                GetDex, GetInt, GetWis, GetTHAC0, GetDamageRoll, GetSex
//   Identifiable: GetName, IsNPC, SendMessage
//
// Name-based lookup concern:
//   SetFighting(target string) and GetFighting() string use a name string
//   to identify combat targets instead of an ID or reference. This couples
//   combat to the naming layer and makes renames, duplicates, or
//   cross-referencing fragile. A future refactor should use stable entity
//   IDs (e.g., entity.ID) for target resolution.
//
// Deferred to future refactor. See RESEARCH-LOG.md [DESIGN].

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
	SetFighting(target string)
	StopFighting()
	GetFighting() string

	// Messaging
	SendMessage(msg string)
}
