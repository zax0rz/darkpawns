package combat

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CombatPairKey uniquely identifies a combat pair by both participants.
type CombatPairKey struct {
	Attacker string
	Target   string
}

// CombatPair represents two entities fighting each other
type CombatPair struct {
	Attacker       Combatant
	Defender       Combatant
	Started        time.Time
	LastAttackType int // Track what type of attack killed the victim (spell number, skill number, or TYPE_ constant)
}

// CombatEngine manages all active combat in the game
type CombatEngine struct {
	mu sync.RWMutex

	// Active combat pairs
	combatPairs map[CombatPairKey]*CombatPair // key: (attacker, target)

	// Combat ticker
	ticker   *time.Ticker
	stopChan chan struct{}
	stopped  atomic.Bool

	// Message broadcaster function (set by game)
	BroadcastFunc func(roomVNum int, message string, exclude string)

	// MessageFunc routes hit/miss messages through the game-layer message system.
	// If non-nil, sendHitMessage and sendMissMessage call it and skip the generic
	// fallback output. The callback receives the attacker, defender, damage (0 for
	// misses), and the attack type recorded on the combat pair.
	MessageFunc func(attacker, defender Combatant, dam, attackType int) bool

	// DeathFunc handles corpse creation and respawn (set by game layer)
	// Called after death messages are sent.
	DeathFunc func(victim, killer Combatant, attackType int)

	// ScriptFightFunc fires the "fight" trigger on a mob after each combat round.
	// Set by the game layer. Called with (mobName, targetName, roomVNum).
	// Source: mobact.c — mobs use scripts during combat
	ScriptFightFunc func(mobName string, targetName string, roomVNum int)

	// ScriptDeathFunc fires the "death" trigger on a mob when it dies.
	// Set by the game layer. Called with (victimName, killerName, roomVNum).
	// Source: fight.c — raw_kill() calls Lua death trigger.
	ScriptDeathFunc func(victimName string, killerName string, roomVNum int)

	// DamageFunc is called after damage is applied to a combatant each round.
	// Set by the session manager to propagate health changes to agent sessions.
	// victimName is the name of the character who took damage.
	DamageFunc func(victimName string)

	// OnRoundEnd is called after each combat round. Used for wait state decrement.
	OnRoundEnd func()

	// OnCombatAction is called after each attack in a combat round.
	// Captures: attacker, defender, attack_type, damage, outcome, target state.
	// Used by decision capture (DP-213) for the combat_log table.
	OnCombatAction func(attacker Combatant, defender Combatant, attackType string, damage int, outcome string, targetCount int)

	// Callbacks holds the game-layer bridge functions used by the legacy
	// fight_core path. Populated during engine initialization.
	Callbacks *GameCallbacks
}

// NewCombatEngine creates a new combat engine
func NewCombatEngine() *CombatEngine {
	return &CombatEngine{
		combatPairs: make(map[CombatPairKey]*CombatPair),
		stopChan:    make(chan struct{}),
	}
}

// SetBroadcastFunc sets the function used to broadcast messages to rooms
func (ce *CombatEngine) SetBroadcastFunc(fn func(roomVNum int, message string, exclude string)) {
	ce.BroadcastFunc = fn
}

// SetCallbacks wires the game-layer callback struct into the engine and sets
// the temporary package-level accessor used by fight_core during migration.
func (ce *CombatEngine) SetCallbacks(cb *GameCallbacks) {
	ce.Callbacks = cb
	SetCallbacks(cb)
}

// ValidateCallbacks returns an error if the callbacks struct is missing hooks
// required for the combat engine to function. Broadcast and SendToChar are the
// minimum required hooks; without them combat messages are silently dropped.
func (ce *CombatEngine) ValidateCallbacks() error {
	if ce.Callbacks == nil {
		return fmt.Errorf("combat callbacks not configured")
	}
	if ce.Callbacks.Broadcast == nil {
		return fmt.Errorf("combat callback Broadcast is required")
	}
	if ce.Callbacks.SendToChar == nil {
		return fmt.Errorf("combat callback SendToChar is required")
	}
	return nil
}

// Start begins the combat tick loop
func (ce *CombatEngine) Start() {
	ce.ticker = time.NewTicker(2 * time.Second) // Combat round every 2 seconds

	go func() {
		for {
			select {
			case <-ce.ticker.C:
				ce.PerformRound()
			case <-ce.stopChan:
				ce.ticker.Stop()
				return
			}
		}
	}()
}

// PositionedMob is an interface for entities that can be knocked down and need position recovery.
type PositionedMob interface {
	GetName() string
	GetStatus() string
	SetStatus(string)
	GetFighting() string
}

// StartMobPositionRecovery starts a goroutine that periodically checks mob positions.
// Mobs that are sitting/resting/sleeping and not in combat are stood back up.
// getMobs returns all mobs that should be checked for position recovery.
// Separate from the combat ticker so position recovery runs at its own cadence.
func (ce *CombatEngine) StartMobPositionRecovery(getMobs func() []PositionedMob) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mobs := getMobs()
				for _, mob := range mobs {
					status := mob.GetStatus()
					fighting := mob.GetFighting()
					if fighting != "" {
						continue
					}
					if status != "sleeping" && status != "resting" && status != "sitting" {
						continue
					}
					mob.SetStatus("standing")
				}
			case <-ce.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the combat engine
func (ce *CombatEngine) Stop() {
	if ce.stopped.CompareAndSwap(false, true) {
		close(ce.stopChan)
	}
}

// StartCombat initiates combat between two combatants
func (ce *CombatEngine) StartCombat(attacker, defender Combatant) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	attackerName := attacker.GetName()
	defenderName := defender.GetName()

	// Build composite key
	key := CombatPairKey{
		Attacker: attackerName,
		Target:   defenderName,
	}

	// Check if already fighting
	if _, exists := ce.combatPairs[key]; exists {
		return fmt.Errorf("%s is already fighting", attackerName)
	}

	// Also check same attacker attacking different target (prevent silent overwrite)
	for k := range ce.combatPairs {
		if k.Attacker == attackerName {
			return fmt.Errorf("%s is already fighting", attackerName)
		}
	}

	// Set fighting state. The attacker always turns to face this defender.
	// The defender, however, keeps its existing target if it is already in
	// combat: in DikuMUD a character FIGHTs one opponent at a time and only
	// retargets when that opponent dies or flees. Overwriting it here would
	// leave the defender's FIGHTING pointing at the new attacker while its
	// original combat pair still exists — an inconsistent state.
	attacker.SetFighting(defenderName)
	if defender.GetFighting() == "" {
		defender.SetFighting(attackerName)
	}

	// Start combat
	ce.combatPairs[key] = &CombatPair{
		Attacker: attacker,
		Defender: defender,
		Started:  time.Now(),
	}

	return nil
}

// StopCombat ends combat for a character
func (ce *CombatEngine) StopCombat(charName string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Find and stop combat — iterate all pairs to find matches by attacker or defender
	for key, pair := range ce.combatPairs {
		if key.Attacker == charName || pair.Defender.GetName() == charName {
			pair.Attacker.StopFighting()
			if pair.Defender.GetFighting() == key.Attacker {
				pair.Defender.StopFighting()
			}
			delete(ce.combatPairs, key)
		}
	}
}

// IsFighting checks if a character is in combat
func (ce *CombatEngine) IsFighting(charName string) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Check if attacking or being attacked — must iterate with composite keys
	for key, pair := range ce.combatPairs {
		if key.Attacker == charName || pair.Defender.GetName() == charName {
			return true
		}
	}

	return false
}

// PerformRound executes one round of combat for all active fighters.
//
// Source: fight.c perform_violence() — iterates ALL characters on the combat
// list (every character with FIGHTING set), not just the attacker side of each
// pair. In the C codebase combat is a flat linked list; in Go we store directed
// pairs (attacker→defender) but both participants may need to swing.
//
// We collect one exchange per unique combatant, resolving each fighter's actual
// FIGHTING target (which may differ from its pair edge — e.g. a defender that
// was already fighting someone else). This mirrors C semantics where each
// character attacks whoever THEY are fighting, and gives retaliatory retarget
// for free when a target dies mid-round.
func (ce *CombatEngine) PerformRound() {
	ce.mu.RLock()

	// Snapshot every directed pair edge under the read lock. Pair processing
	// mutates combatants and may stop combat; we never touch combatPairs after
	// releasing the lock here.
	type edge struct{ attacker, defender Combatant }
	edges := make([]edge, 0, len(ce.combatPairs)*2)
	seen := make(map[string]bool, len(ce.combatPairs)*2)

	for _, pair := range ce.combatPairs {
		for _, fighter := range []Combatant{pair.Attacker, pair.Defender} {
			if fighter == nil {
				continue
			}
			name := fighter.GetName()
			if seen[name] {
				continue
			}
			seen[name] = true
			target := ce.findFightingTarget(name, fighter)
			if target == nil {
				continue
			}
			edges = append(edges, edge{attacker: fighter, defender: target})
		}
	}

	ce.mu.RUnlock()

	// Process each combatant's attack against whoever they are fighting.
	for _, e := range edges {
		ce.processCombatPair(&CombatPair{Attacker: e.attacker, Defender: e.defender})
	}

	// C-10: decrement wait states each round
	if ce.OnRoundEnd != nil {
		ce.OnRoundEnd()
	}
}

// findFightingTarget resolves the Combatant that `fighter` is currently
// attacking. Returns nil if the fighter has no FIGHTING target, or if that
// target cannot be located among the active combat pairs.
//
// Must be called with ce.mu held (at least RLock).
func (ce *CombatEngine) findFightingTarget(fighterName string, fighter Combatant) Combatant {
	targetName := fighter.GetFighting()
	if targetName == "" {
		return nil
	}
	for _, pair := range ce.combatPairs {
		if pair.Attacker.GetName() == targetName {
			return pair.Attacker
		}
		if pair.Defender.GetName() == targetName {
			return pair.Defender
		}
	}
	return nil
}

// processCombatPair handles a single combat exchange
type waitStateHolder interface {
	GetWaitState() int
	SetWaitState(int)
	DecrementWaitState()
}

func (ce *CombatEngine) processCombatPair(pair *CombatPair) {
	attacker := pair.Attacker
	defender := pair.Defender

	// C: fight.c:1975-1987 — NPCs with GET_MOB_WAIT > 0 lose the round and
	// decrement their wait. When the wait expires and the mob is below
	// POS_FIGHTING, it stands back up.
	if attacker.IsNPC() {
		if wc, ok := attacker.(waitStateHolder); ok && wc.GetWaitState() > 0 {
			wc.DecrementWaitState()
			if wc.GetWaitState() == 0 && attacker.GetPosition() < PosFighting {
				attacker.SetPosition(PosFighting)
				if ce.BroadcastFunc != nil {
					pronoun := "its"
					switch attacker.GetSex() {
					case 0:
						pronoun = "his"
					case 1:
						pronoun = "her"
					}
					ce.BroadcastFunc(attacker.GetRoom(), fmt.Sprintf("%s scrambles to %s feet!", attacker.GetName(), pronoun), attacker.GetName())
				}
			}
			return
		}
	}

	// Check if both combatants are still valid and able to fight. An attacker
	// that has itself been knocked out of a fighting position (stunned/incap/
	// mortally/dead) cannot swing; a dead defender has already been extracted.
	// A merely-downed defender (wounded band, still alive) is NOT skipped —
	// the attacker keeps swinging to finish it off (fight.c).
	if attacker.GetPosition() < PosFighting || defender.GetPosition() == PosDead {
		ce.StopCombat(attacker.GetName())
		return
	}

	// Check if they're in the same room
	if attacker.GetRoom() != defender.GetRoom() {
		ce.StopCombat(attacker.GetName())
		return
	}

	// Shopkeeper protection — C: fight.c:1359-1366
	// Any attempt to damage a shopkeeper halts combat for both sides.
	if cbIsShopkeeper(defender.GetName()) {
		ce.StopCombat(attacker.GetName())
		ce.StopCombat(defender.GetName())
		return
	}

	if ce.applyMobCombatRedirects(attacker, defender) {
		return
	}

	// Calculate number of attacks for attacker
	hasHaste := cbHasAffect(attacker.GetName(), AFF_HASTE)
	hasSlow := cbHasAffect(attacker.GetName(), AFF_SLOW)
	numAttacks := GetAttacksPerRound(attacker, hasHaste, hasSlow)

	// C checks parry/dodge once per defender round and sets IS_PARRIED on the
	// attacker. When the attacker swings, IS_PARRIED reduces attack count; it
	// does not negate each individual hit. Source: src/fight.c:1949-2004.
	defenseAction := ""
	switch {
	case CheckParry(defender, attacker) == ParrySuccess:
		defenseAction = "parry"
		attacker.SendMessage(fmt.Sprintf("With a dazzling show of swordplay, %s parries your attack!\r\n", defender.GetName()))
		defender.SendMessage("With a dazzling show of swordplay, you parry the attack!\r\n")
		if ce.BroadcastFunc != nil {
			ce.BroadcastFunc(attacker.GetRoom(),
				fmt.Sprintf("%s displays a dazzling show of swordplay, fending off %s's every blow!", defender.GetName(), attacker.GetName()), "")
		}
	case CheckDodge(defender, attacker) == DodgeSuccess:
		defenseAction = "dodge"
		attacker.SendMessage(fmt.Sprintf("%s dodges your attack!\r\n", defender.GetName()))
		defender.SendMessage("You dodge!\r\n")
		if ce.BroadcastFunc != nil {
			ce.BroadcastFunc(attacker.GetRoom(),
				fmt.Sprintf("%s dodges %s's attack!", defender.GetName(), attacker.GetName()), "")
		}
	}
	if defenseAction != "" {
		defenderDexDefense := dexApp[dexIndex(defender)].Defensive
		if defenderDexDefense < 0 {
			numAttacks += defenderDexDefense
		} else {
			numAttacks--
		}
		if numAttacks < 0 {
			numAttacks = 0
		}
		if ce.OnCombatAction != nil {
			ce.OnCombatAction(attacker, defender, defenseAction, 0, defenseAction, 0)
		}
	}

	// Perform attacks
	for i := 0; i < numAttacks; i++ {
		// Stop once the defender is dead (extracted next tick). A downed but
		// still-living defender keeps getting hit — that's how it's finished.
		if defender.GetPosition() == PosDead {
			break
		}

		if !CalculateHitChance(attacker, defender, HitModifiers{}) {
			ce.sendMissMessage(attacker, defender, pair.LastAttackType)
			continue
		}

		// Calculate damage
		weaponDamage := attacker.GetDamageRoll()
		pair.LastAttackType = int(AttackNormal)
		damage := CalculateDamage(attacker, defender, weaponDamage, AttackNormal)

		// Route through the shared damage() modifier block so melee honors
		// sanctuary, protect evil/good, race-hate weapons, the 3000 cap, and
		// immortal invulnerability — same funnel as skills and spells (DP-1025).
		// Applied before the hit message so the reported damage tier reflects
		// the post-modifier figure, matching fight.c DamMessage.
		damage = ApplyDamageModifiers(attacker, defender, damage)

		// Apply damage
		defender.TakeDamage(damage)
		if ce.DamageFunc != nil {
			ce.DamageFunc(defender.GetName())
		}

		// Send combat messages
		ce.sendHitMessage(attacker, defender, damage, pair.LastAttackType)

		// Log combat action
		if ce.OnCombatAction != nil {
			ce.OnCombatAction(attacker, defender, "hit", damage, "hit", 0)
		}

		// Transition the defender into the wounded band (stunned/incap/
		// mortally) or POS_DEAD based on its new HP, and emit the wound
		// message. Death only fires at POS_DEAD (HP <= -11) — fight.c
		// update_pos. A downed-but-living defender stops fighting back but the
		// attacker keeps swinging on later iterations/rounds.
		newPos := UpdatePositionAfterDamage(defender, ce.BroadcastFunc)
		if newPos == PosDead {
			ce.handleDeath(defender, attacker)
			if ce.OnCombatAction != nil {
				ce.OnCombatAction(attacker, defender, "hit", damage, "killed", 0)
			}
			ce.StopCombat(attacker.GetName())
			break
		}
	}

	// Fire fight trigger on mob attacker after combat round
	// Source: mobact.c — mob_activity() calls mob scripts after violence
	if attacker.IsNPC() && ce.ScriptFightFunc != nil && defender.GetPosition() != PosDead {
		ce.ScriptFightFunc(attacker.GetName(), defender.GetName(), attacker.GetRoom())
	}
}

// applyMobCombatRedirects ports the mob-initiated damage() redirects from
// src/fight.c:1370-1440. These run before damage lands and may move the victim
// or retarget the attacker, causing this combat exchange to abort.
func (ce *CombatEngine) applyMobCombatRedirects(attacker, defender Combatant) bool {
	if !attacker.IsNPC() {
		return false
	}

	attackerName := attacker.GetName()
	defenderName := defender.GetName()

	// Jail guard intercept: city jail guards subdue eligible PCs instead of
	// killing them, then cart them to jail.
	// TODO(DP-1054): C also gates on CAN_SEE(ch, victim); no CanSee callback exists yet.
	if !defender.IsNPC() &&
		cbMobHasJailGuardSpec(attackerName) &&
		attacker.GetHP() > attacker.GetMaxHP()/2 &&
		!cbHasAffectStr(defenderName, AFF_STR_VAMPIRE) &&
		!cbHasAffectStr(defenderName, AFF_STR_WEREWOLF) {
		if cbJailGuardSubdue(attackerName, defenderName) {
			ce.StopCombat(defenderName)
			ce.StopCombat(attackerName)
			return true
		}
	}

	// Charmed-pet retarget: an NPC about to damage a charmed NPC follower may
	// switch to the follower's master if the master is in the same room.
	if defender.IsNPC() &&
		cbHasAffect(defenderName, AFF_CHARM) &&
		GetRoller().Number(0, 10) == 0 {
		masterName := cbGetFollowing(defenderName)
		if masterName != "" {
			if master := findRoomCombatantByName(attacker.GetRoom(), masterName); master != nil {
				ce.redirectAttacker(attacker, master)
				return true
			}
		}
	}

	// High-level NPC switcheroo: high-level mobs sometimes switch to another
	// character in the room that is currently fighting them.
	if attacker.GetLevel() > 20 {
		for _, vict := range cbGetRoomCombatants(attacker.GetRoom()) {
			if vict == nil || vict.GetName() == defenderName {
				continue
			}
			if vict.GetFighting() == attackerName && GetRoller().Number(0, 80) == 0 {
				ce.redirectAttacker(attacker, vict)
				return true
			}
		}
	}

	return false
}

func findRoomCombatantByName(roomVNum int, name string) Combatant {
	for _, ch := range cbGetRoomCombatants(roomVNum) {
		if ch != nil && ch.GetName() == name {
			return ch
		}
	}
	return nil
}

func (ce *CombatEngine) redirectAttacker(attacker, target Combatant) {
	ce.StopCombat(attacker.GetName())
	_ = ce.StartCombat(attacker, target)
}

// sendHitMessage sends hit messages to combatants and room.
// If MessageFunc is set, it is called and the generic fallback is skipped.
func (ce *CombatEngine) sendHitMessage(attacker, defender Combatant, damage, attackType int) {
	if ce.MessageFunc != nil {
		ce.MessageFunc(attacker, defender, damage, attackType)
		return
	}

	attackerName := attacker.GetName()
	defenderName := defender.GetName()
	roomVNum := attacker.GetRoom()

	// Message to attacker
	attacker.SendMessage(fmt.Sprintf("You hit %s for %d damage!\r\n", defenderName, damage))

	// Message to defender
	defender.SendMessage(fmt.Sprintf("%s hits you for %d damage!\r\n", attackerName, damage))

	// Message to room
	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s hits %s!\r\n", attackerName, defenderName),
			attackerName)
	}
}

// sendMissMessage sends miss messages.
// If MessageFunc is set, it is called and the generic fallback is skipped.
func (ce *CombatEngine) sendMissMessage(attacker, defender Combatant, attackType int) {
	if ce.MessageFunc != nil {
		ce.MessageFunc(attacker, defender, 0, attackType)
		return
	}

	attackerName := attacker.GetName()
	defenderName := defender.GetName()
	roomVNum := attacker.GetRoom()

	attacker.SendMessage(fmt.Sprintf("You miss %s!\r\n", defenderName))
	defender.SendMessage(fmt.Sprintf("%s misses you!\r\n", attackerName))

	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s misses %s!\r\n", attackerName, defenderName),
			attackerName)
	}
}

// handleDeath processes character death.
//
// Lock ordering contract (MUST be maintained to prevent deadlocks):
//  1. CombatEngine.mu (ce.mu) — guards combat pair map
//  2. World.mu — guards player/mob maps (AddPlayer, RemovePlayer)
//  3. Player.mu — guards individual player fields
//
// Acquisition order in this function:
//   - ce.mu.RLock (to read LastAttackType from combat pair)
//   - DeathFunc callback acquires World.mu → Player.mu (via HandleDeath)
//   - StopCombat acquires ce.mu.Lock (after DeathFunc returns)
//
// Why this order: combat engine state must be read before any game-layer
// mutations (corpse creation, respawn). World/player locks are never held
// when acquiring ce.mu, preventing ABBA deadlocks with PerformRound.
//
// Faithful to Dark Pawns die()/raw_kill() in fight.c:
//   - Player: lose EXP/3, create corpse with inventory+equipment+gold, send to room 8004
//   - Mob: create corpse, remove from world
//   - Corpse creation is delegated via DeathFunc callback (set by game layer)
func (ce *CombatEngine) handleDeath(victim, killer Combatant) {
	victimName := victim.GetName()
	killerName := killer.GetName()
	roomVNum := victim.GetRoom()

	// Death message to victim
	victim.SendMessage("You have been KILLED!\r\n")

	// Death message to room
	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s has been killed by %s!", victimName, killerName),
			"")
	}

	// Fire death script trigger on victim mob (before DeathFunc removes it)
	if ce.ScriptDeathFunc != nil && victim.IsNPC() {
		ce.ScriptDeathFunc(victimName, killerName, roomVNum)
	}

	// Delegate to game layer for corpse creation + respawn
	if ce.DeathFunc != nil {
		// Get attack type from combat pair if available
		attackType := -1 // TYPE_UNDEFINED
		ce.mu.RLock()
		for key, pair := range ce.combatPairs {
			if key.Attacker == killerName {
				attackType = pair.LastAttackType
				break
			}
		}
		ce.mu.RUnlock()
		ce.DeathFunc(victim, killer, attackType)
	}
}

// GetCombatTarget returns who a character is fighting
func (ce *CombatEngine) GetCombatTarget(charName string) (Combatant, bool) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Check if attacking someone or being attacked — must iterate with composite keys
	for key, pair := range ce.combatPairs {
		if key.Attacker == charName {
			return pair.Defender, true
		}
		if pair.Defender.GetName() == charName {
			return pair.Attacker, true
		}
	}

	return nil, false
}

// GetCombatStatus returns combat status for a character
func (ce *CombatEngine) GetCombatStatus(charName string) string {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	for key, pair := range ce.combatPairs {
		if key.Attacker == charName {
			return fmt.Sprintf("You are fighting %s", pair.Defender.GetName())
		}
		if pair.Defender.GetName() == charName {
			return fmt.Sprintf("You are being attacked by %s", key.Attacker)
		}
	}

	return "You are not in combat"
}
