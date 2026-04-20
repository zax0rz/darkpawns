// Package game manages the game world state and player interactions.
package game

// ScriptContext holds the game objects exposed to Lua as globals.
// Defined here (in pkg/game) to avoid import cycles with pkg/scripting.
type ScriptContext struct {
	Ch       *Player
	Me       *MobInstance
	Obj      *ObjectInstance
	RoomVNum int
	Argument string
	World    *World
}

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
	// From structs.h: MS_BRIBE, MS_GREET, MS_ONGIVE, MS_SOUND, MS_DEATH, 
	// MS_ONPULSE_ALL, MS_ONPULSE_PC, MS_FIGHTING, MS_ONCMD
	var bitmask int
	switch trigger {
	case "bribe":
		bitmask = 1 << 1
	case "greet":
		bitmask = 1 << 2
	case "ongive":
		bitmask = 1 << 3
	case "sound":
		bitmask = 1 << 4
	case "ondeath":
		bitmask = 1 << 5
	case "onpulse_all":
		bitmask = 1 << 6
	case "onpulse_pc":
		bitmask = 1 << 7
	case "fight":
		bitmask = 1 << 8
	case "oncmd":
		bitmask = 1 << 9
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
	return ScriptEngine.RunScript(ctx, "mob/"+m.Prototype.ScriptName, trigger)
}

// Helper to create script context for mob events
func (m *MobInstance) CreateScriptContext(ch *Player, obj *ObjectInstance, argument string) *ScriptContext {
	return &ScriptContext{
		Ch:       ch,
		Me:       m,
		Obj:      obj,
		RoomVNum: m.RoomVNum,
		Argument: argument,
		World:    nil, // Would need world reference
	}
}