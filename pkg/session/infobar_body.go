package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// newBodyInfobarState builds the drawing state from the switched body's vitals
// and the shared dummy_mob infobar record. C reads GET_HIT(mob) and friends;
// the shared PRF_COLOR flags are zero, so the body never gets color codes.
func newBodyInfobarState(mob *game.MobInstance, st game.NPCInfoBarState) *infobarState {
	return &infobarState{
		colorNormal:       false,
		screenSize:        st.ScreenSize,
		lastHit:           mob.GetHealth(),
		lastMaxHit:        mob.GetMaxHealth(),
		lastMana:          mob.GetMana(),
		lastMaxMana:       mob.GetMaxMana(),
		lastMove:          mob.GetMove(),
		lastMaxMove:       mob.GetMaxMove(),
		lastExp:           mob.GetExp(),
		lastGold:          mob.GetGold(),
		expNeededForLevel: game.FindExp(0, mob.GetLevel()),
		nextLevel:         mob.GetLevel() + 1,
		level:             mob.GetLevel(),
	}
}

// mobInfoBarBody is do_infobar for a switched body (src/act.display.c:112-155).
// GET_INFOBAR and GET_SCREENSIZE are the shared dummy_mob saved fields, so the
// mode and size persist across bodies.
func mobInfoBarBody(ctx game.CharacterCommandContext, _ string, args []string) game.CharacterCommandResult {
	mob, ok := ctx.Actor.(*game.MobInstance)
	if !ok || ctx.World == nil {
		return game.CharacterCommandUnsupported
	}
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	if arg == "" && ctx.RawArgs != "" {
		arg = strings.Fields(ctx.RawArgs)[0]
	}
	state := ctx.World.NPCInfoBar()

	if arg == "" {
		switch state.Mode {
		case game.InfobarOff:
			mob.SendMessage("Your infobar is off.\r\n")
		case game.InfobarOn:
			mob.SendMessage("Your infobar is on.\r\n")
		default:
			mob.SendMessage("You had an unknown infobar setting.\r\n")
			mob.SendMessage("It is being set to OFF.\r\n")
			state.Mode = game.InfobarOff
			ctx.World.SetNPCInfoBar(state)
		}
		return game.CharacterCommandHandled
	}

	switch strings.ToLower(arg) {
	case "off":
		if state.Mode == game.InfobarOn {
			state.Mode = game.InfobarOff
			ctx.World.SetNPCInfoBar(state)
			mob.SendRaw(infobarOffOutput(state.ScreenSize))
			mob.SendMessage("Your infobar is now set to off.\r\n")
		} else {
			mob.SendMessage("Your infobar is already off.\r\n")
		}
	case "on":
		if state.Mode == game.InfobarOff {
			if state.ScreenSize == 0 {
				state.ScreenSize = 25
			}
			state.Mode = game.InfobarOn
			ctx.World.SetNPCInfoBar(state)
			is := newBodyInfobarState(mob, state)
			mob.SendRaw(infobarOnOutput(is))
			state.LastHit = mob.GetHealth()
			state.LastMaxHit = mob.GetMaxHealth()
			state.LastMana = mob.GetMana()
			state.LastMaxMana = mob.GetMaxMana()
			state.LastMove = mob.GetMove()
			state.LastMaxMove = mob.GetMaxMove()
			state.LastExp = mob.GetExp()
			state.LastGold = mob.GetGold()
			ctx.World.SetNPCInfoBar(state)
			mob.SendMessage("Your infobar is now set to on.\r\n")
		} else {
			mob.SendMessage("Your infobar is already on.\r\n")
		}
	default:
		mob.SendMessage("Usage:  infobar < on | off >\r\n")
	}
	return game.CharacterCommandHandled
}
