package combat

import (
	"fmt"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/internal/dpclock"
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
	// DeferDefenderEnrollment preserves hit()'s NPC-special ordering.  C's
	// damage() sets the victim's FIGHTING field after the NPC switcheroo scan;
	// ordinary command entry keeps the historical eager enrollment.
	DeferDefenderEnrollment bool
}

// CombatEngine manages all active combat in the game
type CombatEngine struct {
	mu sync.RWMutex

	// Active combat pairs
	combatPairs map[CombatPairKey]*CombatPair // key: (attacker, target)
	combatOrder []Combatant                   // C combat_list order: most recently engaged first
	parried     map[string]string             // C IS_PARRIED flag, keyed by fighter whose next turn is reduced

	// Background-loop lifecycle. Stop closes stopChan exactly once and waits
	// for every started loop to exit, including an in-flight combat round.
	lifecycleMu sync.Mutex
	background  sync.WaitGroup
	stopChan    chan struct{}
	stopped     bool
	tickEvery   time.Duration

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

	// MobSpecialFunc fires a MOB_SPEC procedure after an NPC's combat turn.
	// Set by the game layer. Source: fight.c:2030-2031 — perform_violence()
	// invokes the assigned special after the ordinary attack loop.
	MobSpecialFunc func(mob Combatant) bool

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
		parried:     make(map[string]string),
		stopChan:    make(chan struct{}),
		tickEvery:   2 * time.Second,
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

// SkillMessage routes a combat message through the same skill_message path C's
// damage() uses (fight.c:1023-1092): it draws Dice(1, len(variants)) from the
// shared roller and emits the selected set's char/vict/room text for the given
// attackType. Returns false when attackType has no loaded message set.
//
// This is the public entry point the skill layer (e.g. DoBackstab) calls to
// emit a combat message from lib/misc/messages instead of a hardcoded string
// (R4/R3). attackType is the messages-file key — e.g. 131 for the Backstab set
// — NOT the Go-internal SKILL_* enum (fight_core.go:60), which is unrelated to
// the messages file and would miss the lookup.
func (ce *CombatEngine) SkillMessage(dam int, ch, vict string, attackType int, roomVNum int) bool {
	return cbSkillMessage(dam, ch, vict, attackType, roomVNum)
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
	if dpclock.Frozen() {
		return
	}
	ce.lifecycleMu.Lock()
	if ce.stopped {
		ce.lifecycleMu.Unlock()
		return
	}
	ce.background.Add(1)
	ce.lifecycleMu.Unlock()

	go func() {
		defer ce.background.Done()
		ticker := time.NewTicker(ce.tickEvery) // Combat round every 2 seconds
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ce.PerformRound()
			case <-ce.stopChan:
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
	ce.lifecycleMu.Lock()
	if ce.stopped {
		ce.lifecycleMu.Unlock()
		return
	}
	ce.background.Add(1)
	ce.lifecycleMu.Unlock()

	go func() {
		defer ce.background.Done()
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
				return
			}
		}
	}()
}

// Stop halts the combat engine and waits for all background loops to exit.
// It is safe to call more than once.
func (ce *CombatEngine) Stop() {
	ce.lifecycleMu.Lock()
	if !ce.stopped {
		ce.stopped = true
		close(ce.stopChan)
	}
	ce.lifecycleMu.Unlock()
	ce.background.Wait()
}

// StartCombat initiates combat between two combatants
func (ce *CombatEngine) StartCombat(attacker, defender Combatant) error {
	return ce.startCombat(attacker, defender, false)
}

// StartCombatFromMob starts the synchronous combat opener used by a C mobile
// special that calls hit().  The defender is enrolled inside performOneHit,
// after hit() has consumed its to-hit and damage draws and after the C
// high-level switcheroo scan, rather than at the StartCombat boundary.
func (ce *CombatEngine) StartCombatFromMob(attacker, defender Combatant) error {
	return ce.startCombat(attacker, defender, true)
}

// ApplyMobDamageRedirects exposes the damage() redirect seam to native mob
// specials that call damage() directly rather than entering through a
// one_hit() combat pair. C performs these redirects before applying the
// caller-supplied damage amount (fight.c:1370-1440), so the game layer must
// use this same seam for direct fighter-special damage as well.
func (ce *CombatEngine) ApplyMobDamageRedirects(attacker, defender Combatant) bool {
	return ce.applyMobCombatRedirects(attacker, defender)
}

func (ce *CombatEngine) startCombat(attacker, defender Combatant, deferDefenderEnrollment bool) error {
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
	// original combat pair still exists — an inconsistent state. Mobile
	// specials defer this write until the damage-side C path.
	attacker.SetFighting(defenderName)
	// The attacker is stood to POS_FIGHTING at entry: it is always standing when
	// it initiates (do_hit gates on position), so this is observably equal to
	// C's set_fighting(ch)-in-damage() and keeps do_hit's swing-branch check
	// (GET_POS == POS_STANDING) refusing a second target once fighting.
	if attacker.GetPosition() > PosStunned {
		attacker.SetPosition(PosFighting)
	}
	ce.prependFighterLocked(attacker)
	// The DEFENDER is NOT stood here. C's set_fighting(victim) runs inside
	// damage() (fight.c:1443-1445), AFTER the opener's to-hit decision, so the
	// to-hit reads the victim's pre-combat position — a sleeping victim is
	// auto-hit (AWAKE(victim) is false), not stood into an awake miss. Go
	// mirrors this by transitioning the victim to POS_FIGHTING at the damage
	// point (performOneHit), gated on > POS_STUNNED like C. Only the fighting
	// flag/target is set at entry so the round loop enrolls it. Mobile-special
	// entry defers both operations until the damage-side path below.
	if !deferDefenderEnrollment && defender.GetFighting() == "" {
		defender.SetFighting(attackerName)
	}
	// The defender must be in combatOrder (C's combat_list) whenever it is
	// fighting this attacker — even when its FIGHTING field was set before
	// StartCombat ran. DoSpellDamage sets the victim's field directly
	// (damage_stubs.go) without enrolling it, so gating the prepend on an
	// empty field left positive-damage skill victims out of combatOrder and
	// they never got a turn (DP-1213). prependFighterLocked is idempotent.
	if !deferDefenderEnrollment && defender.GetFighting() == attackerName {
		ce.prependFighterLocked(defender)
	}

	// Start combat
	ce.combatPairs[key] = &CombatPair{
		Attacker:                attacker,
		Defender:                defender,
		Started:                 time.Now(),
		DeferDefenderEnrollment: deferDefenderEnrollment,
	}

	return nil
}

// PerformInitialAttack resolves the single synchronous hit made by do_hit.
// It deliberately bypasses perform_violence's round-only attack-count,
// parry/dodge, wait-state, redirect, and fight-trigger work. Subsequent attacks
// continue through PerformRound on the normal combat pulse.
func (ce *CombatEngine) PerformInitialAttack(attacker, defender Combatant) error {
	key := CombatPairKey{Attacker: attacker.GetName(), Target: defender.GetName()}

	ce.mu.RLock()
	pair, ok := ce.combatPairs[key]
	ce.mu.RUnlock()
	if !ok {
		return fmt.Errorf("combat pair %s -> %s is not active", key.Attacker, key.Target)
	}

	ce.performOneHit(pair)
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
			delete(ce.parried, pair.Attacker.GetName())
			delete(ce.parried, pair.Defender.GetName())
			delete(ce.combatPairs, key)
		}
	}
	delete(ce.parried, charName)
	ce.removeFighterLocked(charName)
	ce.pruneCombatOrderLocked()
}

// prependFighterLocked mirrors C's set_fighting(), which inserts a newly
// fighting character at the head of combat_list. A fighter may appear once.
// ce.mu must be held for writing.
func (ce *CombatEngine) prependFighterLocked(fighter Combatant) {
	if fighter == nil {
		return
	}
	name := fighter.GetName()
	for _, existing := range ce.combatOrder {
		if existing != nil && existing.GetName() == name {
			return
		}
	}
	ce.combatOrder = append(ce.combatOrder, nil)
	copy(ce.combatOrder[1:], ce.combatOrder[:len(ce.combatOrder)-1])
	ce.combatOrder[0] = fighter
}

// removeFighterLocked removes one character from the combat_list analogue.
// ce.mu must be held for writing.
func (ce *CombatEngine) removeFighterLocked(name string) {
	for i, fighter := range ce.combatOrder {
		if fighter != nil && fighter.GetName() == name {
			copy(ce.combatOrder[i:], ce.combatOrder[i+1:])
			ce.combatOrder[len(ce.combatOrder)-1] = nil
			ce.combatOrder = ce.combatOrder[:len(ce.combatOrder)-1]
			return
		}
	}
}

// pruneCombatOrderLocked drops characters whose FIGHTING state was cleared
// while StopCombat removed a related pair. ce.mu must be held for writing.
func (ce *CombatEngine) pruneCombatOrderLocked() {
	active := ce.combatOrder[:0]
	for _, fighter := range ce.combatOrder {
		if fighter != nil && fighter.GetFighting() != "" {
			active = append(active, fighter)
		}
	}
	clear(ce.combatOrder[len(active):])
	ce.combatOrder = active
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
// Source: fight.c perform_violence() walks combat_list head-first. set_fighting
// prepends newly engaged characters, so the most recently engaged fighter acts
// first. Each fighter attacks its own current FIGHTING target.
func (ce *CombatEngine) PerformRound() {
	ce.mu.RLock()
	fighters := append([]Combatant(nil), ce.combatOrder...)
	ce.mu.RUnlock()

	seen := make(map[string]bool, len(fighters))
	for _, fighter := range fighters {
		if fighter == nil || fighter.GetPosition() == PosDead || fighter.GetFighting() == "" {
			continue
		}
		name := fighter.GetName()
		if seen[name] {
			continue
		}
		seen[name] = true

		// Combat can mutate while a round is executing (death, flee, redirect),
		// so resolve the fighter's live target immediately before its turn.
		ce.mu.RLock()
		target := ce.findFightingTarget(name, fighter)
		ce.mu.RUnlock()
		if target == nil {
			continue
		}
		ce.processCombatPair(&CombatPair{Attacker: fighter, Defender: target})
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

	// C perform_violence per-combatant sequence (fight.c:1906-2025): attacks-
	// count draws → parry/dodge draws → wait/stand-up → IS_PARRIED adjust →
	// attack loop gated on AWAKE. The draw-bearing blocks run FIRST, before
	// any early exit — C draws them even when the combatant is downed or about
	// to stop fighting (DP-1215, R3a/R3b).

	// 1. Attacks-count computation (fight.c:1910-1947). NPC draws Number(0,900);
	// PC draws Number(1,100)/Number(0,500). These run unconditionally.
	hasHaste := cbHasAffect(attacker.GetName(), AFF_HASTE)
	hasSlow := cbHasAffect(attacker.GetName(), AFF_SLOW)
	numAttacks := GetAttacksPerRound(attacker, hasHaste, hasSlow)

	// 2. Parry/dodge pre-check (fight.c:1949-1973). PC draws Number(0,10000)
	// every round; NPC dodge draws Number(0,100) if AFF_DODGE.
	ce.prepareRoundDefense(attacker, defender)

	// 3. Shopkeeper protection — C: fight.c:1359-1366.
	if cbIsShopkeeper(defender.GetName()) {
		ce.StopCombat(attacker.GetName())
		ce.StopCombat(defender.GetName())
		return
	}

	// 4. NPC stand-up (fight.c:1975-1988). C zeros attacks only for
	// GET_MOB_WAIT > 0, which only the Lua bridge writes (scripts.c:2017) —
	// never WAIT_STATE (utils.h:462-464 writes ch->wait, a different field).
	// Go's scripting layer writes no mob wait, so NPC attacks are NEVER zeroed
	// by wait. A downed mob stands and swings in the same round (DP-1215).
	if attacker.IsNPC() && attacker.GetPosition() < PosFighting {
		attacker.SetPosition(PosFighting)
		ce.scrambleBroadcast(attacker)
	}

	// 5. PC stand-up (fight.c:1990-1998). C: !IS_NPC && GET_POS < POS_FIGHTING
	// && !CHECK_WAIT (wait <= 1). PC wait drains in the heartbeat (manager.go
	// OnDrainInput), NOT here — do not decrement (C drains in comm.c:597).
	if !attacker.IsNPC() && attacker.GetPosition() < PosFighting {
		waitOK := true // a combatant without wait tracking reads as !CHECK_WAIT
		if wc, ok := attacker.(waitStateHolder); ok {
			waitOK = wc.GetWaitState() <= 1 // C CHECK_WAIT: ch->wait > 1
		}
		if waitOK {
			attacker.SetPosition(PosFighting)
			ce.scrambleBroadcast(attacker)
		}
	}

	// 6. IS_PARRIED adjustment (fight.c:1999-2007).
	defenseAction := ce.consumeParried(attacker.GetName())
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
	}

	// 7. Attack loop (fight.c:2009-2025). Gated ONLY on AWAKE (GET_POS >
	// POS_SLEEPING) and same-room — a sitting/resting attacker (downed but
	// awake) still swings. NOT awake or different room → stop_fighting.
	if attacker.GetPosition() <= PosSleeping {
		ce.StopCombat(attacker.GetName())
		return
	}
	if defender.GetPosition() == PosDead {
		return // defender extracted; nothing to hit
	}
	if attacker.GetRoom() != defender.GetRoom() {
		ce.StopCombat(attacker.GetName())
		return
	}

	for i := 0; i < numAttacks; i++ {
		if defender.GetPosition() == PosDead {
			break
		}
		if ce.performOneHit(pair) {
			break
		}
	}

	// C perform_violence() invokes MOB_SPEC after the mob's ordinary attack
	// loop and before the combat-round script trigger.
	if attacker.IsNPC() && ce.MobSpecialFunc != nil {
		ce.MobSpecialFunc(attacker)
	}

	// Fire fight trigger on mob attacker after combat round
	if attacker.IsNPC() && ce.ScriptFightFunc != nil && defender.GetPosition() != PosDead {
		ce.ScriptFightFunc(attacker.GetName(), defender.GetName(), attacker.GetRoom())
	}
}

// scrambleBroadcast emits the "$n scrambles to $s feet!" stand-up message
// (fight.c:1985/1995) — the room broadcast (capitalized via CAP, comm.c:2477)
// plus the "You drag yourself to your feet.\r\n" self-message.
func (ce *CombatEngine) scrambleBroadcast(c Combatant) {
	pronoun := "its"
	switch c.GetSex() {
	case 0:
		pronoun = "his"
	case 1:
		pronoun = "her"
	}
	if ce.BroadcastFunc != nil {
		// C act() → CAP uppercases the first byte of the fully-composed string.
		ce.BroadcastFunc(c.GetRoom(),
			capitalizeFightMessage(fmt.Sprintf("%s scrambles to %s feet!", c.GetName(), pronoun)),
			c.GetName())
	}
	c.SendMessage("You drag yourself to your feet.\r\n")
}

func (ce *CombatEngine) prepareRoundDefense(fighter, opponent Combatant) {
	defenseAction := ""
	switch {
	case CheckParry(fighter, opponent) == ParrySuccess:
		defenseAction = "parry"
		fighter.SendMessage("With a dazzling show of swordplay, you move into defensive position...\r\n")
		// C act() → CAP uppercases the first byte (comm.c:2477). The fighter's
		// name may be lowercase (e.g. "a guard trainee").
		opponent.SendMessage(capitalizeFightMessage(fmt.Sprintf("%s displays a dazzling show of swordplay, fending off your every blow!\r\n", fighter.GetName())))
		ce.sendDefenseObserverMessage(
			fighter,
			opponent,
			capitalizeFightMessage(fmt.Sprintf("%s displays a dazzling show of swordplay, fending off %s's every blow!\r\n", fighter.GetName(), opponent.GetName())),
		)
	case CheckDodge(fighter, opponent) == DodgeSuccess:
		defenseAction = "dodge"
		fighter.SendMessage("You dodge!\r\n")
		opponent.SendMessage(capitalizeFightMessage(fmt.Sprintf("%s dodges your attack!\r\n", fighter.GetName())))
		ce.sendDefenseObserverMessage(
			fighter,
			opponent,
			capitalizeFightMessage(fmt.Sprintf("%s dodges %s's attack!\r\n", fighter.GetName(), opponent.GetName())),
		)
	}
	if defenseAction != "" {
		ce.setParried(opponent.GetName(), defenseAction)
		if ce.OnCombatAction != nil {
			ce.OnCombatAction(opponent, fighter, defenseAction, 0, defenseAction, 0)
		}
	}
}

func (ce *CombatEngine) sendDefenseObserverMessage(fighter, opponent Combatant, message string) {
	observed := false
	for _, observer := range cbGetRoomCombatants(fighter.GetRoom()) {
		if observer == nil || observer.GetName() == fighter.GetName() || observer.GetName() == opponent.GetName() {
			continue
		}
		observer.SendMessage(message)
		observed = true
	}
	if !observed && ce.BroadcastFunc != nil {
		// Test and embedding fallback when the game-layer room lookup is absent.
		ce.BroadcastFunc(fighter.GetRoom(), message, fighter.GetName())
	}
}

func (ce *CombatEngine) setParried(name, defenseAction string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.parried[name] = defenseAction
}

// MarkParried records a C IS_PARRIED result for the named fighter. Native
// mob specials use this same one-round defense state as perform_violence's
// built-in parry/dodge path.
func (ce *CombatEngine) MarkParried(name, defenseAction string) {
	ce.setParried(name, defenseAction)
}

func (ce *CombatEngine) consumeParried(name string) string {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	defenseAction := ce.parried[name]
	delete(ce.parried, name)
	return defenseAction
}

// performOneHit is the shared C hit() path used by both do_hit's synchronous
// first strike and each attack selected by perform_violence.
// It returns true when death or a successful flee ends the exchange.
func (ce *CombatEngine) performOneHit(pair *CombatPair) bool {
	attacker := pair.Attacker
	defender := pair.Defender

	// fight.c:1792-1806 one_hit w_type derivation: the message attack-type is
	// derived from the wielded weapon (offset val3), NOT from the damage type.
	// Compute it fresh every round so the miss branch (below) doesn't read a
	// stale pair.LastAttackType. This offset is handed ONLY to the message
	// senders — CalculateDamage keeps AttackNormal so AC reduction still applies
	// (R3: damage math is unchanged).
	msgAttackType := cbWeaponInfo(attacker.GetName())

	hit := CalculateHitChance(attacker, defender, HitModifiers{})
	var damage int
	if hit {
		weaponDamage := attacker.GetDamageRoll()
		damage = CalculateDamage(attacker, defender, weaponDamage, AttackNormal)
	}

	// fight.c reaches mob redirects from damage(), after hit() has consumed
	// its to-hit and damage-roll draws but before damage messages/state land.
	if ce.applyMobCombatRedirects(attacker, defender) {
		return true
	}

	// C damage() enrolls the victim after the NPC switcheroo scan.  Do this
	// before the remaining damage-side effects, including stop_follower and
	// the hit/miss message path.
	if pair.DeferDefenderEnrollment && defender.GetFighting() == "" && defender.GetPosition() > PosStunned {
		defender.SetFighting(attacker.GetName())
		ce.mu.Lock()
		ce.prependFighterLocked(defender)
		ce.mu.Unlock()
		pair.DeferDefenderEnrollment = false
	}

	// C set_fighting(victim) — which sets POS_FIGHTING — runs INSIDE damage(),
	// after the to-hit decision AND after one_hit has finished computing damage
	// (which itself reads the victim's pre-fight position for the prone-victim
	// multiplier, fight.c:1857 dam *= 1 + (POS_FIGHTING - GET_POS(victim))/3).
	// So the victim must keep its pre-combat position through BOTH the to-hit
	// AWAKE check and the damage multiplier, and only stand at the damage point.
	// Miss path: C still calls damage(ch,victim,0,...), so the victim stands on
	// a miss too. Gate on > POS_STUNNED (fight.c:1443): a dying victim stays prone.
	standVictim := func() {
		if defender.GetPosition() > PosStunned && defender.GetPosition() != PosFighting {
			defender.SetPosition(PosFighting)
		}
	}

	// C damage() breaks the victim's following when the attacker IS the
	// master (fight.c:1457-1458): stop_follower's charm branch announces
	// "$n hates your guts!" to the master BEFORE the swing's message, on
	// both the miss (damage 0) and hit paths.
	if cbGetFollowing(defender.GetName()) == attacker.GetName() {
		cbStopFollowerOfMaster(defender.GetName(), attacker.GetName())
	}

	if !hit {
		standVictim()
		ce.sendMissMessage(attacker, defender, msgAttackType)
		return false
	}

	pair.LastAttackType = int(AttackNormal)
	damage = ApplyDamageModifiers(attacker, defender, damage)

	// Stand the victim only now — after the prone-victim damage multiplier has
	// been read against its pre-fight position (mirrors C set_fighting-in-damage).
	standVictim()

	defender.TakeDamage(damage)
	if ce.DamageFunc != nil {
		ce.DamageFunc(defender.GetName())
	}

	ce.sendHitMessage(attacker, defender, damage, msgAttackType)
	if ce.OnCombatAction != nil {
		ce.OnCombatAction(attacker, defender, "hit", damage, "hit", 0)
	}

	newPos := UpdatePositionAfterDamage(defender, ce.BroadcastFunc)
	if newPos != PosDead {
		return ce.handleSurvivingVictimState(attacker, defender, damage, newPos)
	}

	ce.handleDeath(defender, attacker)
	if ce.OnCombatAction != nil {
		ce.OnCombatAction(attacker, defender, "hit", damage, "killed", 0)
	}
	ce.StopCombat(attacker.GetName())
	return true
}

// handleSurvivingVictimState mirrors damage()'s default position branch after
// the weapon message: high-damage pain, low-HP bleeding, and automatic wimpy
// retreat/flee. It reports whether the callback ended this exchange.
func (ce *CombatEngine) handleSurvivingVictimState(attacker, defender Combatant, damage, newPos int) bool {
	if newPos < PosSleeping {
		return false
	}

	if damage > defender.GetMaxHP()/4 {
		defender.SendMessage("That really did HURT!\r\n")
		if GetRoller().Number(0, 2) == 0 {
			ce.sendDefenseObserverMessage(
				defender,
				attacker,
				fmt.Sprintf("%s screams in pain!\r\n", defender.GetName()),
			)
		}
	}

	if defender.GetHP() < defender.GetMaxHP()/4 {
		defender.SendMessage("You wish that your wounds would stop BLEEDING so much!\r\n")
		if cbHasMobFlag(defender.GetName(), "MOB_WIMPY") && attacker.GetName() != defender.GetName() {
			cbDoFlee(defender.GetName())
		}
	}

	wimpLevel := cbGetWimpyLev(defender.GetName())
	if !defender.IsNPC() && wimpLevel > 0 &&
		attacker.GetName() != defender.GetName() &&
		newPos >= PosFighting && defender.GetHP() < wimpLevel {
		defender.SendMessage("You wimp out, and attempt to flee!\r\n")
		if cbGetSkill(defender.GetName(), SKILL_RETREAT) > 0 ||
			cbGetSkill(defender.GetName(), SKILL_ESCAPE) > 0 {
			cbDoRetreat(defender.GetName())
		} else {
			cbDoFlee(defender.GetName())
		}
	}

	return defender.GetRoom() != attacker.GetRoom() || defender.GetFighting() == ""
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
			if vict == nil {
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
	if err := ce.StartCombat(attacker, target); err != nil {
		return
	}
	// C's redirect branches call hit(), not only set_fighting(). Preserve
	// that synchronous opener before the normal combat round resumes.
	if initial, ok := interface{}(ce).(interface {
		PerformInitialAttack(Combatant, Combatant) error
	}); ok {
		_ = initial.PerformInitialAttack(attacker, target)
	}
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
