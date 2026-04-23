// Package game manages the game world state and player interactions.
package game

import (
	"log/slog"

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
