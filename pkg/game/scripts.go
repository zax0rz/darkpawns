// Package game manages the game world state and player interactions.
package game

import (
	"log/slog"
	"math/rand"

	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// ScriptContext is an alias for scripting.ScriptContext
type ScriptContext = scripting.ScriptContext

// ScriptEngine is set by the server at startup.
var ScriptEngine interface {
	RunScript(ctx *ScriptContext, fname string, trigger string) (bool, error)
}

// MobInstance methods for script handling

// HasScript checks if a mob has a script for the given trigger.
// Based on the bitmask values in structs.h lines 659-690.
func (m *MobInstance) HasScript(trigger string) bool {
	if m.Prototype == nil || m.Prototype.ScriptName == "" {
		return false
	}

	// Check bitmask based on trigger type
	// From structs.h lines 659-690: MS_BRIBE=1, MS_GREET=2, MS_ONGIVE=4, MS_SOUND=8,
	// MS_DEATH=16, MS_ONPULSE_ALL=32, MS_ONPULSE_PC=64, MS_FIGHTING=128, MS_ONCMD=256
	var bitmask int
	switch trigger {
	case "bribe":
		bitmask = 1 // MS_BRIBE
	case "greet":
		bitmask = 2 // MS_GREET
	case "ongive":
		bitmask = 4 // MS_ONGIVE
	case "sound":
		bitmask = 8 // MS_SOUND
	case "death":
		bitmask = 16 // MS_DEATH
	case "onpulse_all":
		bitmask = 32 // MS_ONPULSE_ALL
	case "onpulse_pc":
		bitmask = 64 // MS_ONPULSE_PC
	case "fight":
		bitmask = 128 // MS_FIGHTING
	case "oncmd":
		bitmask = 256 // MS_ONCMD
	default:
		return false
	}

	return (m.Prototype.LuaFunctions & bitmask) != 0
}

// RunScript executes a mob's script for the given trigger.
func (m *MobInstance) RunScript(trigger string, ctx *ScriptContext) (bool, error) {
	if ScriptEngine == nil || !m.HasScript(trigger) {
		return false, nil
	}

	// Set me in context if not already set
	if ctx.Me == nil {
		ctx.Me = m
	}

	// Run the script
	handled, err := ScriptEngine.RunScript(ctx, m.Prototype.ScriptName, trigger)

	// Handle assembler.lua silent return issue
	// If ongive returns false/nil, send default message
	if trigger == "ongive" && !handled && err == nil && ctx.Ch != nil {
		// Send default "You can't give that here." message
		// In real implementation: ctx.Ch.SendMessage("You can't give that here.\r\n")
		slog.Debug("ongive returned false", "mob_vnum", m.GetVNum(), "player", ctx.Ch.GetName())
	}

	return handled, err
}

// Helper to create script context for mob events
func (m *MobInstance) CreateScriptContext(ch *Player, obj *ObjectInstance, argument string) *ScriptContext {
	ctx := &ScriptContext{
		Me:       m,
		RoomVNum: m.RoomVNum,
		Argument: argument,
		World:    nil, // Would need world reference
	}
	// Only set Ch if ch is not nil
	if ch != nil {
		ctx.Ch = ch
	}
	// Only set Obj if obj is not nil
	if obj != nil {
		ctx.Obj = obj
	}
	return ctx
}

// CounterProcsRewards faithfully reproduces the C counter_procs() kill milestone logic
// from src/fight.c lines 1252-1312.
//
// C source has a deliberate switch fall-through bug in the major milestone case:
//
//	switch(number(1,3)) {
//	    case 1: GET_MAX_HIT(ch)++;
//	    case 2: GET_MAX_MANA(ch)++;
//	    case 3: GET_MAX_MOVE(ch)++;
//	    default: GET_MAX_HIT(ch)++;
//	    break;
//	}
//
// Since number(1,3) returns 1-3 and ALL cases lack break:
//
//	roll 1 → case 1: HP++, fall→case 2: MANA++, fall→case 3: MOVE++, fall→default: HP++
//	        = HP+2, MANA+1, MOVE+1
//	roll 2 → case 2: MANA++, fall→case 3: MOVE++, fall→default: HP++
//	        = HP+1, MANA+1, MOVE+1
//	roll 3 → case 3: MOVE++, fall→default: HP++
//	        = HP+1, MOVE+1
//
// The previous Go implementation (pkg/combat/fight_core.go CounterProcs) gave
// HP+1, MANA+1, MOVE+1 unconditionally — only matching the roll=2 path.
//
// Returns true if a reward milestone was hit.
func CounterProcsRewards(p *Player) bool {
	if p == nil {
		return false
	}

	kills := int64(p.Kills)

	switch kills {
	case 5000, 15000, 25000, 35000, 45000:
		// Minor milestones: full heal + global blessing
		p.SendMessage("The gods reward your glory in battle!\r\n")
		p.Heal(p.GetMaxHP() - p.GetHP())
		return true

	case 1000, 2000, 10000, 20000, 30000, 40000, 50000:
		// Major milestones: random stat boost with C fall-through bug
		p.SendMessage("The gods reward your many victories!\r\n")

		// #nosec G404 — game RNG, not cryptographic
		roll := rand.Intn(3) + 1 // number(1,3) returns 1-3

		switch roll {
		case 1:
			p.MaxHealth += 2 // case 1 (HP++) + default (HP++) = HP+2
			p.MaxMana += 1
			p.MaxMove += 1
		case 2:
			p.MaxHealth += 1 // default only = HP+1
			p.MaxMana += 1
			p.MaxMove += 1
		case 3:
			p.MaxHealth += 1 // default only = HP+1
			p.MaxMove += 1
		}

		p.Heal(p.GetMaxHP() - p.GetHP())
		return true

	default:
		return false
	}
}
