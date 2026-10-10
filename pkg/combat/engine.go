package combat

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/internal/dpclock"

	"github.com/zax0rz/darkpawns/pkg/metrics"
)

// CombatPairKey uniquely identifies a combat pair by both participants.
type CombatPairKey struct {
	Attacker Combatant
	Target   Combatant
}

// CombatPair represents two entities fighting each other
type CombatPair struct {
	// RangedRetaliation delays attacker enrollment until damage protections pass.
	RangedRetaliation bool
	Attacker          Combatant
	Defender          Combatant
	Started           time.Time
	LastAttackType    int // Track what type of attack killed the victim (spell number, skill number, or TYPE_ constant)
	// DeferDefenderEnrollment preserves hit()'s NPC-special ordering.  C's
	// damage() sets the victim's FIGHTING field after the NPC switcheroo scan;
	// ordinary command entry keeps the historical eager enrollment.
	DeferDefenderEnrollment bool
	// DefenderNeedsStand retains the pending posture half of an eager initial
	// enrollment. C performs both halves in damage(), src/fight.c:1443-1445.
	DefenderNeedsStand bool
}

// CombatEngine manages all active combat in the game
type CombatEngine struct {
	mu sync.RWMutex

	// Active combat pairs
	combatPairs map[CombatPairKey]*CombatPair // key: (attacker, target)
	combatOrder []Combatant                   // C combat_list order: most recently engaged first
	parried     map[Combatant]string          // C IS_PARRIED flag, keyed by fighter whose next turn is reduced

	// Background-loop lifecycle. Stop closes stopChan exactly once and waits
	// for every started loop to exit, including an in-flight combat round.
	lifecycleMu sync.Mutex
	background  sync.WaitGroup
	stopChan    chan struct{}
	stopped     bool
	tickEvery   time.Duration

	// Message broadcaster function (set by game)
	BroadcastFunc func(roomVNum int, message string, exclude []Combatant)

	// MessageFunc routes hit/miss messages through the game-layer message system.
	// If non-nil, sendHitMessage and sendMissMessage call it and skip the generic
	// fallback output. The callback receives the attacker, defender, damage (0 for
	// misses), and the attack type recorded on the combat pair.
	MessageFunc func(attacker, defender Combatant, dam, attackType int) bool

	// DeathFunc handles corpse creation and deferred extraction (set by game layer)
	// Called after death messages are sent.
	DeathFunc func(victim, killer Combatant, attackType int)

	// ScriptFightFunc fires the "fight" trigger on a mob after each combat round.
	// Set by the game layer. Called with (mobName, targetName, roomVNum).
	// Source: mobact.c — mobs use scripts during combat
	ScriptFightFunc func(mob, target Combatant, roomVNum int)

	// MobSpecialFunc fires a MOB_SPEC procedure after an NPC's combat turn.
	// Set by the game layer. Source: fight.c:2030-2031 — perform_violence()
	// invokes the assigned special after the ordinary attack loop.
	MobSpecialFunc func(mob Combatant) bool

	// ScriptDeathFunc fires the "death" trigger on a mob when it dies.
	// Set by the game layer. Called with (victimName, killerName, roomVNum).
	// Source: fight.c — raw_kill() calls Lua death trigger.
	ScriptDeathFunc func(victim, killer Combatant, roomVNum int)

	// DamageFunc is called after damage is applied to a combatant each round.
	// Set by the session manager to propagate health changes to agent sessions.
	// victimName is the name of the character who took damage.
	DamageFunc func(victim Combatant)

	// OnRoundEnd is called after each combat round. Used for wait state decrement.
	OnRoundEnd func()

	// Callbacks holds the game-layer bridge functions used by the legacy
	// fight_core path. Populated during engine initialization.
	Callbacks *GameCallbacks
}

// NewCombatEngine creates a new combat engine
func NewCombatEngine() *CombatEngine {
	return &CombatEngine{
		combatPairs: make(map[CombatPairKey]*CombatPair),
		parried:     make(map[Combatant]string),
		stopChan:    make(chan struct{}),
		tickEvery:   2 * time.Second,
	}
}

// SetBroadcastFunc sets the function used to broadcast messages to rooms
func (ce *CombatEngine) SetBroadcastFunc(fn func(roomVNum int, message string, exclude []Combatant)) {
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
func (ce *CombatEngine) SkillMessage(dam int, ch, vict Combatant, attackType int, roomVNum int) bool {
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
	GetFightingBody() Combatant
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
	return ce.startCombat(attacker, defender, false, false)
}

// StartCombatAfterDamage enrolls the command's attacker without undoing
// damage()'s !AWAKE victim stop (src/fight.c:1443-1445, 1630-1632).
// The ordinary opener must still defer its position check until its hit.
func (ce *CombatEngine) StartCombatAfterDamage(attacker, defender Combatant) error {
	if !ValidBody(attacker) || !ValidBody(defender) || BodyRetired(attacker) || BodyRetired(defender) {
		return fmt.Errorf("combat requires concrete bodies")
	}
	if attacker == defender || defender.GetPosition() == PosDead {
		return nil
	}
	if attacker.GetPosition() <= PosStunned || attacker.GetFightingBody() != defender {
		if defender.GetFightingBody() == attacker && defender.GetPosition() > PosSleeping {
			return ce.startCombat(defender, attacker, attacker.GetPosition() <= PosSleeping, true)
		}
		return nil
	}
	return ce.startCombat(attacker, defender, defender.GetPosition() <= PosSleeping || defender.GetFightingBody() != attacker, true)
}

// EnrollAfterDamage bridges command/world engines to the post-damage entry.
// Older test engines can use ordinary entry only for an awake victim.
func EnrollAfterDamage(engine interface {
	StartCombat(Combatant, Combatant) error
}, attacker, defender Combatant,
) error {
	if starter, ok := engine.(interface {
		StartCombatAfterDamage(Combatant, Combatant) error
	}); ok {
		return starter.StartCombatAfterDamage(attacker, defender)
	}
	if defender.GetPosition() <= PosSleeping || attacker.GetPosition() <= PosStunned || attacker == defender {
		return nil
	}
	return engine.StartCombat(attacker, defender)
}

// StartCombatFromMob starts the synchronous combat opener used by a C mobile
// special that calls hit().  The defender is enrolled inside performOneHit,
// after hit() has consumed its to-hit and damage draws and after the C
// high-level switcheroo scan, rather than at the StartCombat boundary.
func (ce *CombatEngine) StartCombatFromMob(attacker, defender Combatant) error {
	return ce.startCombat(attacker, defender, true, false)
}

// ApplyMobDamageRedirects exposes the damage() redirect seam to native mob
// specials that call damage() directly rather than entering through a
// one_hit() combat pair. C performs these redirects before applying the
// caller-supplied damage amount (fight.c:1370-1440), so the game layer must
// use this same seam for direct fighter-special damage as well.
func (ce *CombatEngine) ApplyMobDamageRedirects(attacker, defender Combatant) bool {
	return ce.applyMobCombatRedirects(attacker, defender)
}

func (ce *CombatEngine) startCombat(attacker, defender Combatant, deferDefenderEnrollment, afterDamage bool) error {
	if !ValidBody(attacker) || !ValidBody(defender) || BodyRetired(attacker) || BodyRetired(defender) {
		return fmt.Errorf("combat requires concrete bodies")
	}
	ce.mu.Lock()
	defer ce.mu.Unlock()
	if BodyRetired(attacker) || BodyRetired(defender) {
		return fmt.Errorf("combat requires live bodies")
	}

	attackerName := attacker.GetName()

	// Build composite key
	key := CombatPairKey{
		Attacker: attacker,
		Target:   defender,
	}

	// Check if already fighting
	if _, exists := ce.combatPairs[key]; exists {
		return fmt.Errorf("%s is already fighting", attackerName)
	}

	// Also check same attacker attacking different target (prevent silent overwrite)
	for k := range ce.combatPairs {
		if k.Attacker == attacker {
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
	attacker.SetFightingBody(defender)
	// The attacker is stood to POS_FIGHTING at entry: it is always standing when
	// it initiates (do_hit gates on position), so this is observably equal to
	// C's set_fighting(ch)-in-damage() and keeps do_hit's swing-branch check
	// (GET_POS == POS_STANDING) refusing a second target once fighting.
	if !afterDamage && attacker.GetPosition() > PosStunned {
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
	defenderNeedsStand := !deferDefenderEnrollment && defender.GetFightingBody() == nil
	if defenderNeedsStand {
		defender.SetFightingBody(attacker)
	}
	// The defender must be in combatOrder (C's combat_list) whenever it is
	// fighting this attacker — even when its FIGHTING field was set before
	// StartCombat ran. DoSpellDamage sets the victim's field directly
	// (damage_stubs.go) without enrolling it, so gating the prepend on an
	// empty field left positive-damage skill victims out of combatOrder and
	// they never got a turn (DP-1213). prependFighterLocked is idempotent.
	if !deferDefenderEnrollment && defender.GetFightingBody() == attacker {
		ce.prependFighterLocked(defender)
	}

	// Start combat
	ce.combatPairs[key] = &CombatPair{
		Attacker:                attacker,
		Defender:                defender,
		Started:                 time.Now(),
		DeferDefenderEnrollment: deferDefenderEnrollment,
		DefenderNeedsStand:      defenderNeedsStand,
	}

	return nil
}

// PerformInitialAttack resolves the single synchronous hit made by do_hit.
// It deliberately bypasses perform_violence's round-only attack-count,
// parry/dodge, wait-state, redirect, and fight-trigger work. Subsequent attacks
// continue through PerformRound on the normal combat pulse.
func (ce *CombatEngine) PerformInitialAttack(attacker, defender Combatant) error {
	if !ValidBody(attacker) || !ValidBody(defender) || BodyRetired(attacker) || BodyRetired(defender) {
		return fmt.Errorf("combat requires concrete bodies")
	}
	key := CombatPairKey{Attacker: attacker, Target: defender}

	ce.mu.RLock()
	pair, ok := ce.combatPairs[key]
	ce.mu.RUnlock()
	if !ok {
		return fmt.Errorf("combat pair %s -> %s is not active", key.Attacker.GetName(), key.Target.GetName())
	}

	ce.performOneHit(pair)
	return nil
}

// PerformRangedRetaliation executes shoot's synchronous hit(), without eager
// enrollment. C damage enrolls the attacker after protections and the victim
// after redirects (src/act.offensive.c:955; src/fight.c:1314-1458).
// Existing initial attack paths retain their behavior.
func (ce *CombatEngine) PerformRangedRetaliation(attacker, defender Combatant) error {
	if !ValidBody(attacker) || !ValidBody(defender) || BodyRetired(attacker) || BodyRetired(defender) {
		return fmt.Errorf("combat requires concrete live bodies")
	}
	ce.performOneHit(&CombatPair{Attacker: attacker, Defender: defender, RangedRetaliation: true, DeferDefenderEnrollment: true})
	return nil
}

// PerformUnenrolledInitialAttack resolves a direct hit() call made by a mob
// special without creating the command-engine combat pair. In C, the aware
// backstab branch calls hit(vict, ch) directly: damage() leaves the mob facing
// the player, but the player is not enrolled as a reciprocal fighter and the
// hit is not added as a new engine pair. The mob therefore does not receive a
// later perform_violence turn from this one-off retaliation.
func (ce *CombatEngine) PerformUnenrolledInitialAttack(attacker, defender Combatant) error {
	if !ValidBody(attacker) || !ValidBody(defender) || BodyRetired(attacker) || BodyRetired(defender) {
		return fmt.Errorf("combat requires concrete bodies")
	}
	ce.performOneHit(&CombatPair{Attacker: attacker, Defender: defender})
	ce.mu.Lock()
	defer ce.mu.Unlock()
	if !BodyRetired(attacker) && !BodyRetired(defender) && defender.GetPosition() != PosDead && attacker.GetFightingBody() == nil {
		attacker.SetFightingBody(defender)
		if attacker.GetPosition() > PosStunned {
			attacker.SetPosition(PosFighting)
		}
	}
	return nil
}

// StopCombat ports stop_fighting for one body, including C's ordered retarget.
func (ce *CombatEngine) StopCombat(body Combatant) {
	if !ValidBody(body) {
		return
	}
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.stopCombatLocked(body, false)
}

// stopCombatLocked never calls world/session hooks. Engine -> one body lock.
func (ce *CombatEngine) stopCombatLocked(body Combatant, force bool) {
	target := body.GetFightingBody()
	if !force && target != nil && (target.GetPosition() == PosDead || target.GetRoom() != body.GetRoom()) {
		for _, candidate := range ce.combatOrder {
			if candidate != body && candidate.GetFightingBody() == body && candidate.GetPosition() > PosDead && !BodyRetired(candidate) {
				body.SetFightingBody(candidate)
				for key, pair := range ce.combatPairs {
					if key.Attacker == body {
						delete(ce.combatPairs, key)
						ce.combatPairs[CombatPairKey{Attacker: body, Target: candidate}] = &CombatPair{Attacker: body, Defender: candidate, Started: pair.Started}
					}
				}
				return
			}
		}
	}
	body.StopFighting()
	body.SetPosition(GetPositionFromHP(body.GetHP(), PosStanding))
	for key := range ce.combatPairs {
		if key.Attacker == body {
			delete(ce.combatPairs, key)
		}
	}
	delete(ce.parried, body)
	ce.removeFighterLocked(body)
}

// RetireCombatant removes the retired body and only references to that body.
// Survivors may retarget through C's ordinary dead-target stop path.
func (ce *CombatEngine) RetireCombatant(body Combatant) {
	if !ValidBody(body) {
		return
	}
	ce.mu.Lock()
	defer ce.mu.Unlock()
	if marker, ok := body.(interface{ SetCombatRetired(bool) }); ok {
		marker.SetCombatRetired(true)
	}
	ce.stopCombatLocked(body, true)
	fighters := append([]Combatant(nil), ce.combatOrder...)
	for _, fighter := range fighters {
		if fighter.GetFightingBody() == body {
			ce.stopCombatLocked(fighter, false)
		}
	}
	for key := range ce.combatPairs {
		if key.Attacker == body || key.Target == body {
			delete(ce.combatPairs, key)
		}
	}
}

// prependFighterLocked mirrors C's set_fighting(), which inserts a newly
// fighting character at the head of combat_list. A fighter may appear once.
// ce.mu must be held for writing.
func (ce *CombatEngine) prependFighterLocked(fighter Combatant) {
	if fighter == nil {
		return
	}
	for _, existing := range ce.combatOrder {
		if existing != nil && existing == fighter {
			return
		}
	}
	ce.combatOrder = append(ce.combatOrder, nil)
	copy(ce.combatOrder[1:], ce.combatOrder[:len(ce.combatOrder)-1])
	ce.combatOrder[0] = fighter
}

// removeFighterLocked removes one character from the combat_list analogue.
// ce.mu must be held for writing.
func (ce *CombatEngine) removeFighterLocked(body Combatant) {
	for i, fighter := range ce.combatOrder {
		if fighter != nil && fighter == body {
			copy(ce.combatOrder[i:], ce.combatOrder[i+1:])
			ce.combatOrder[len(ce.combatOrder)-1] = nil
			ce.combatOrder = ce.combatOrder[:len(ce.combatOrder)-1]
			return
		}
	}
}

// IsFighting checks if a character is in combat
func (ce *CombatEngine) IsFighting(body Combatant) bool {
	if !ValidBody(body) {
		return false
	}
	return !BodyRetired(body) && body.GetFightingBody() != nil
}

// PerformRound executes one round of combat for all active fighters.
//
// Source: fight.c perform_violence() walks combat_list head-first. set_fighting
// prepends newly engaged characters, so the most recently engaged fighter acts
// first. Each fighter attacks its own current FIGHTING target.
func (ce *CombatEngine) PerformRound() {
	metrics.CombatRound()
	ce.mu.RLock()
	fighters := append([]Combatant(nil), ce.combatOrder...)
	ce.mu.RUnlock()

	seen := make(map[Combatant]bool, len(fighters))
	for _, fighter := range fighters {
		if fighter == nil || BodyRetired(fighter) || fighter.GetPosition() == PosDead || fighter.GetFightingBody() == nil {
			continue
		}
		if seen[fighter] {
			continue
		}
		seen[fighter] = true

		// Combat can mutate while a round is executing (death, flee, redirect),
		// so resolve the fighter's live target immediately before its turn.
		ce.mu.RLock()
		target := ce.findFightingTarget(fighter)
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
// attacking. Returns nil if the fighter has no stored body reference.
//
// Must be called with ce.mu held (at least RLock).
func (ce *CombatEngine) findFightingTarget(fighter Combatant) Combatant {
	return fighter.GetFightingBody()
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
	hasHaste := cbHasAffect(attacker, AFF_HASTE)
	hasSlow := cbHasAffect(attacker, AFF_SLOW)
	numAttacks := GetAttacksPerRound(attacker, hasHaste, hasSlow)

	// 2. Parry/dodge pre-check (fight.c:1949-1973). PC draws Number(0,10000)
	// every round; NPC dodge draws Number(0,100) if AFF_DODGE.
	ce.prepareRoundDefense(attacker, defender)

	// 3. NPC stand-up (fight.c:1975-1988). C zeros attacks only for
	// GET_MOB_WAIT > 0, which only the Lua bridge writes (scripts.c:2017) —
	// never WAIT_STATE (utils.h:462-464 writes ch->wait, a different field).
	// Go's scripting layer writes no mob wait, so NPC attacks are NEVER zeroed
	// by wait. A downed mob stands and swings in the same round (DP-1215).
	if attacker.IsNPC() && attacker.GetPosition() < PosFighting {
		attacker.SetPosition(PosFighting)
		ce.scrambleBroadcast(attacker)
	}

	// 4. PC stand-up (fight.c:1990-1998). C: !IS_NPC && GET_POS < POS_FIGHTING
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

	// 5. IS_PARRIED adjustment (fight.c:1999-2007).
	defenseAction := ce.consumeParried(attacker)
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

	// 6. Attack loop (fight.c:2009-2025). Gated ONLY on AWAKE (GET_POS >
	// POS_SLEEPING) and same-room — a sitting/resting attacker (downed but
	// awake) still swings. NOT awake or different room → stop_fighting.
	if attacker.GetPosition() <= PosSleeping {
		ce.StopCombat(attacker)
		return
	}
	if defender.GetPosition() == PosDead {
		return // defender extracted; nothing to hit
	}
	if attacker.GetRoom() != defender.GetRoom() {
		ce.StopCombat(attacker)
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
		ce.ScriptFightFunc(attacker, defender, attacker.GetRoom())
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
			[]Combatant{c})
	}
	sendCombatMessage(c, "You drag yourself to your feet.\r\n")
}

func (ce *CombatEngine) prepareRoundDefense(fighter, opponent Combatant) {
	defenseAction := ""
	switch {
	case CheckParry(fighter, opponent) == ParrySuccess:
		defenseAction = "parry"
		sendCombatMessage(fighter, "With a dazzling show of swordplay, you move into defensive position...\r\n")
		// C act() → CAP uppercases the first byte (comm.c:2477). The fighter's
		// name may be lowercase (e.g. "a guard trainee").
		sendCombatMessage(opponent, capitalizeFightMessage(fmt.Sprintf("%s displays a dazzling show of swordplay, fending off your every blow!\r\n", fighter.GetName())))
		ce.sendDefenseObserverMessage(
			fighter,
			opponent,
			capitalizeFightMessage(fmt.Sprintf("%s displays a dazzling show of swordplay, fending off %s's every blow!\r\n", fighter.GetName(), opponent.GetName())),
		)
	case CheckDodge(fighter, opponent) == DodgeSuccess:
		defenseAction = "dodge"
		sendCombatMessage(fighter, "You dodge!\r\n")
		sendCombatMessage(opponent, capitalizeFightMessage(fmt.Sprintf("%s dodges your attack!\r\n", fighter.GetName())))
		ce.sendDefenseObserverMessage(
			fighter,
			opponent,
			capitalizeFightMessage(fmt.Sprintf("%s dodges %s's attack!\r\n", fighter.GetName(), opponent.GetName())),
		)
	}
	if defenseAction != "" {
		ce.setParried(opponent, defenseAction)
	}
}

func (ce *CombatEngine) sendDefenseObserverMessage(fighter, opponent Combatant, message string) {
	observed := false
	for _, observer := range cbGetRoomCombatants(fighter.GetRoom()) {
		if observer == nil || observer == fighter || observer == opponent || observer.GetPosition() <= PosSleeping {
			continue
		}
		sendCombatMessage(observer, message)
		observed = true
	}
	if !observed && ce.BroadcastFunc != nil {
		// Test and embedding fallback when the game-layer room lookup is absent.
		ce.BroadcastFunc(fighter.GetRoom(), message, []Combatant{fighter, opponent})
	}
}

func (ce *CombatEngine) setParried(body Combatant, defenseAction string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.parried[body] = defenseAction
}

// MarkParried records a C IS_PARRIED result for the actual fighter. Native
// mob specials use this same one-round defense state as perform_violence's
// built-in parry/dodge path.
func (ce *CombatEngine) MarkParried(body Combatant, defenseAction string) {
	if !ValidBody(body) {
		return
	}
	ce.setParried(body, defenseAction)
}

func (ce *CombatEngine) consumeParried(body Combatant) string {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	defenseAction := ce.parried[body]
	delete(ce.parried, body)
	return defenseAction
}

// performOneHit is the shared C hit() path used by both do_hit's synchronous
// first strike and each attack selected by perform_violence.
// It returns true when death or a successful flee ends the exchange.
func (ce *CombatEngine) performOneHit(pair *CombatPair) bool {
	attacker := pair.Attacker
	defender := pair.Defender
	if BodyRetired(attacker) || BodyRetired(defender) {
		return true
	}

	// fight.c:1792-1806 one_hit w_type derivation: the message attack-type is
	// derived from the wielded weapon (offset val3), NOT from the damage type.
	// Compute it fresh every round so the miss branch (below) doesn't read a
	// stale pair.LastAttackType. This offset is handed ONLY to the message
	// senders — CalculateDamage keeps AttackNormal so AC reduction still applies
	// (R3: damage math is unchanged).
	msgAttackType := cbWeaponInfo(attacker)

	hit := CalculateHitChance(attacker, defender, HitModifiers{
		WeaponBlessed: cbWeaponBlessed(attacker),
		DrunkLevel:    cbDrunk(attacker),
	})
	var damage int
	if hit {
		weaponDamage := attacker.GetDamageRoll()
		damage = CalculateDamage(attacker, defender, weaponDamage, AttackNormal)
	}
	// hit() still consumes its to-hit and damage draws before damage() rejects
	// a POS_DEAD victim (fight.c:1319-1326). Keep that boundary here so a
	// second mobile special cannot print a duplicate death transcript during
	// the heartbeat in which the first attacker killed the target.
	if defender.GetPosition() <= PosDead {
		return false
	}

	// damage()'s protection block runs after hit() has consumed its to-hit
	// and damage draws, before combat enrollment or messages (fight.c:1318-
	// 1368). Shopkeeper protection stops both sides, which ends C's attack
	// loop; the peaceful and level refusals leave FIGHTING set, so C's loop
	// swings (and refuses) again on the next attack.
	if cbDamageRefused(attacker, defender) {
		if cbIsShopkeeper(defender) {
			ce.StopCombat(attacker)
			ce.StopCombat(defender)
			return true
		}
		return false
	}

	// Only ranged entry delays attacker enrollment past the jail redirect.
	var afterJail func() bool
	ranged := pair.RangedRetaliation
	if ranged {
		afterJail = func() bool {
			if attacker.GetPosition() > PosStunned && attacker.GetFightingBody() == nil {
				if err := ce.startCombat(attacker, defender, true, true); err != nil {
					return false
				}
				attacker.SetPosition(PosFighting)
				key := CombatPairKey{Attacker: attacker, Target: defender}
				ce.mu.RLock()
				pair = ce.combatPairs[key]
				ce.mu.RUnlock()
				if pair == nil {
					return false
				}
			}
			return true
		}
	}
	if ce.applyMobCombatRedirectsAfterJail(attacker, defender, afterJail) {
		return true
	}

	// C damage() enrolls the victim after the NPC switcheroo scan.  Do this
	// before the remaining damage-side effects, including stop_follower and
	// the hit/miss message path.
	ce.mu.Lock()
	if BodyRetired(attacker) || BodyRetired(defender) {
		ce.mu.Unlock()
		return true
	}
	if defender.GetFightingBody() == nil && defender.GetPosition() > PosStunned {
		defender.SetFightingBody(attacker)
		ce.prependFighterLocked(defender)
		pair.DeferDefenderEnrollment = false
		pair.DefenderNeedsStand = true
		if ranged && attacker.GetFightingBody() == nil {
			// C enrolls the awake victim even when the attacking mob is wounded.
			ce.combatPairs[CombatPairKey{Attacker: defender, Target: attacker}] = &CombatPair{Attacker: defender, Defender: attacker}
		}
	}
	ce.mu.Unlock()

	if ranged && attacker.IsNPC() && !defender.IsNPC() && defender.GetLevel() < LVL_IMMORT {
		if cb := GetCallbacks(); cb != nil && cb.RangedHunt != nil {
			cb.RangedHunt(attacker, defender)
		}
	}

	// C set_fighting(victim) — which sets POS_FIGHTING — runs INSIDE damage(),
	// after the to-hit decision AND after one_hit has finished computing damage
	// (which itself reads the victim's pre-fight position for the prone-victim
	// multiplier, fight.c:1857 dam *= 1 + (POS_FIGHTING - GET_POS(victim))/3).
	// So the victim must keep its pre-combat position through BOTH the to-hit
	// AWAKE check and the damage multiplier, and only stand at the damage point.
	// Miss path: C still calls damage(ch,victim,0,...), so the victim stands on
	// a miss too. Gate on > POS_STUNNED and !FIGHTING (src/fight.c:1443):
	// a dying victim stays prone, and an existing fighter keeps its posture.
	// DefenderNeedsStand is the pending posture half of initial Go enrollment.
	standVictim := func() {
		if defender.GetPosition() > PosStunned && (pair.DefenderNeedsStand || defender.GetFightingBody() == nil) {
			defender.SetPosition(PosFighting)
			pair.DefenderNeedsStand = false
		}
	}

	// C damage() breaks the victim's following when the attacker IS the
	// master (fight.c:1457-1458): stop_follower's charm branch announces
	// "$n hates your guts!" to the master BEFORE the swing's message, on
	// both the miss (damage 0) and hit paths.
	if cbGetFollowing(defender) == attacker {
		cbStopFollowerOfMaster(defender, attacker)
	}

	if !hit {
		standVictim()
		if DamageBeforeMessage(attacker, defender, 0, callbacks) {
			return true
		}
		ce.sendMissMessage(attacker, defender, msgAttackType)
		return false
	}

	ce.mu.Lock()
	pair.LastAttackType = int(AttackNormal)
	ce.mu.Unlock()
	damage = ApplyDamageModifiers(attacker, defender, damage)

	// Stand the victim only now — after the prone-victim damage multiplier has
	// been read against its pre-fight position (mirrors C set_fighting-in-damage).
	standVictim()

	defender.TakeDamage(damage)
	rescued := DamageBeforeMessage(attacker, defender, damage, callbacks)
	if ce.DamageFunc != nil {
		ce.DamageFunc(defender)
	}

	if BodyRetired(attacker) || BodyRetired(defender) {
		return true
	}
	if rescued {
		return true
	}
	ce.sendHitMessage(attacker, defender, damage, msgAttackType)
	if BodyRetired(attacker) || BodyRetired(defender) {
		return true
	}

	newPos := UpdatePositionAfterDamage(defender, ce.BroadcastFunc)
	if newPos != PosDead {
		return ce.handleSurvivingVictimState(attacker, defender, damage, newPos)
	}

	// damage() emits its POS_DEAD bytes, then the death callback owns XP,
	// autogold, death_cry and the silent corpse (src/fight.c:1644-1691, 573-578).
	EmitDeathPositionMessage(defender, ce.BroadcastFunc)
	// The death callback owns XP/autogold before raw_kill's cry/corpse.
	ce.handleDeath(defender, attacker)
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
		sendCombatMessage(defender, "That really did HURT!\r\n")
		if GetRoller().Number(0, 2) == 0 {
			ce.sendDefenseObserverMessage(
				defender,
				attacker,
				fmt.Sprintf("%s screams in pain!\r\n", defender.GetName()),
			)
		}
	}

	if defender.GetHP() < defender.GetMaxHP()/4 {
		sendCombatMessage(defender, "You wish that your wounds would stop BLEEDING so much!\r\n")
		if cbHasMobFlag(defender, "MOB_WIMPY") && attacker != defender {
			cbDoFlee(defender)
		}
	}

	wimpLevel := cbGetWimpyLev(defender)
	if !defender.IsNPC() && wimpLevel > 0 &&
		attacker != defender &&
		newPos >= PosFighting && defender.GetHP() < wimpLevel {
		sendCombatMessage(defender, "You wimp out, and attempt to flee!\r\n")
		if cbGetSkill(defender, SKILL_RETREAT) > 0 ||
			cbGetSkill(defender, SKILL_ESCAPE) > 0 {
			cbDoRetreat(defender)
		} else {
			cbDoFlee(defender)
		}
	}

	return defender.GetRoom() != attacker.GetRoom() || defender.GetFightingBody() == nil
}

// applyMobCombatRedirects ports the mob-initiated damage() redirects from
// src/fight.c:1370-1440. These run before damage lands and may move the victim
// or retarget the attacker, causing this combat exchange to abort.
func (ce *CombatEngine) applyMobCombatRedirects(attacker, defender Combatant) bool {
	return ce.applyMobCombatRedirectsAfterJail(attacker, defender, nil)
}

// afterJail is exclusive to direct ranged hit entry; ordinary readers pass nil.
func (ce *CombatEngine) applyMobCombatRedirectsAfterJail(attacker, defender Combatant, afterJail func() bool) bool {
	if !attacker.IsNPC() {
		return false
	}

	// Jail guard intercept: city jail guards subdue eligible PCs instead of
	// killing them, then cart them to jail.
	// TODO(DP-1054): C also gates on CAN_SEE(ch, victim); no CanSee callback exists yet.
	if !defender.IsNPC() &&
		cbMobHasJailGuardSpec(attacker) &&
		attacker.GetHP() > attacker.GetMaxHP()/2 &&
		!cbHasAffectStr(defender, AFF_STR_VAMPIRE) &&
		!cbHasAffectStr(defender, AFF_STR_WEREWOLF) {
		if cbJailGuardSubdue(attacker, defender) {
			ce.StopCombat(defender)
			ce.StopCombat(attacker)
			return true
		}
	}

	if afterJail != nil && !afterJail() {
		return true
	}

	// Charmed-pet retarget: an NPC about to damage a charmed NPC follower may
	// switch to the follower's master if the master is in the same room.
	if defender.IsNPC() &&
		cbHasAffect(defender, AFF_CHARM) &&
		GetRoller().Number(0, 10) == 0 {
		master := cbGetFollowing(defender)
		if master != nil && master.GetRoom() == attacker.GetRoom() && ce.redirectAttacker(attacker, master) {
			return true
		}
	}

	// High-level NPC switcheroo: high-level mobs sometimes switch to another
	// character in the room that is currently fighting them.
	if attacker.GetLevel() > 20 {
		for _, vict := range cbGetRoomCombatants(attacker.GetRoom()) {
			if vict == nil {
				continue
			}
			if vict.GetFightingBody() == attacker && GetRoller().Number(0, 80) == 0 {
				if ce.redirectAttacker(attacker, vict) {
					return true
				}
			}
		}
	}

	return false
}

func (ce *CombatEngine) redirectAttacker(attacker, target Combatant) bool {
	ce.StopCombat(attacker)
	if err := ce.StartCombat(attacker, target); err != nil {
		// Only reachable via a concurrent re-engagement race: another
		// goroutine started this attacker between StopCombat and StartCombat.
		// The redirect did not happen, so let the original attack proceed
		// instead of silently dropping the round (DP-1239).
		slog.Warn("combat redirect failed, original attack proceeds",
			"attacker", attacker.GetName(),
			"target", target.GetName(),
			"error", err)
		return false
	}
	// C's redirect branches call hit(), not only set_fighting(). Preserve
	// that synchronous opener before the normal combat round resumes.
	if initial, ok := interface{}(ce).(interface {
		PerformInitialAttack(Combatant, Combatant) error
	}); ok {
		_ = initial.PerformInitialAttack(attacker, target)
	}
	return true
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
	sendCombatMessage(attacker, fmt.Sprintf("You hit %s for %d damage!\r\n", defenderName, damage))

	// Message to defender
	sendCombatMessage(defender, fmt.Sprintf("%s hits you for %d damage!\r\n", attackerName, damage))

	// Message to room
	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s hits %s!\r\n", attackerName, defenderName), []Combatant{attacker})
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

	sendCombatMessage(attacker, fmt.Sprintf("You miss %s!\r\n", defenderName))
	sendCombatMessage(defender, fmt.Sprintf("%s misses you!\r\n", attackerName))

	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s misses %s!\r\n", attackerName, defenderName), []Combatant{attacker})
	}
}

// handleDeath processes character death.
//
// Attack-type lookup takes engine RLock alone. Retirement takes engine Lock
// then one body mutex at a time. Both release before DeathFunc acquires World
// or session locks; no world/session callback runs under the engine lock.
//
// Faithful to Dark Pawns die()/raw_kill() in fight.c:
//   - Player: lose EXP/3, create corpse with inventory+equipment+gold, send to room 8004
//   - Mob: create corpse, remove from world
//   - Corpse creation is delegated via DeathFunc callback (set by game layer)
func (ce *CombatEngine) handleDeath(victim, killer Combatant) {
	roomVNum := victim.GetRoom()

	// Fire death script trigger on victim mob (before DeathFunc removes it)
	if ce.ScriptDeathFunc != nil && victim.IsNPC() {
		ce.ScriptDeathFunc(victim, killer, roomVNum)
	}

	// Snapshot attack type before retiring related pairs; callbacks run unlocked.
	attackType := -1
	ce.mu.RLock()
	for key, pair := range ce.combatPairs {
		if key.Attacker == killer {
			attackType = pair.LastAttackType
			break
		}
	}
	ce.mu.RUnlock()
	ce.RetireCombatant(victim)
	if ce.DeathFunc != nil {
		ce.DeathFunc(victim, killer, attackType)
	}
}

// GetCombatTarget returns who a character is fighting
func (ce *CombatEngine) GetCombatTarget(body Combatant) (Combatant, bool) {
	if !ValidBody(body) {
		return nil, false
	}
	target := body.GetFightingBody()
	return target, target != nil
}
