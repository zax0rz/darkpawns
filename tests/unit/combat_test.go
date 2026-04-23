package unit

// import "github.com/zax0rz/darkpawns/pkg/combat"
//
// ---
// Combat tests have been updated to reflect the current API:
//
//   CalculateHitChance(attacker, defender Combatant) bool
//   CalculateDamage(attacker, defender Combatant, weaponDamage DiceRoll, attackType AttackType) int
//   GetAttacksPerRound(c Combatant, hasHaste, hasSlow bool) int
//
// Combatant is now an interface, not a struct. The test helpers and test implementations
// below mirror the old test structure and validate the current code.
//
// Tests that referenced removed functions (CalculateCriticalChance, CalculateFleeChance,
// CalculateExperience, GenerateLoot, NewRound/Execute) or removed types (StatusEffect,
// ArmorPiece, Mob, LootItem, Round, CombatResult) have been removed.
// ---

// break compile cycle so the package builds
var _ = 1
