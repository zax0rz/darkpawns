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

// bodyInfoBarUpdateOutput is C's shared dummy_mob repaint for a switched
// descriptor (src/comm.c:1158-1193; src/act.display.c:226-285). It compares
// the body's vitals with the shared last-known values, records the new
// baseline, and returns the changed-field bytes for the deferred flush.
func (s *Session) bodyInfoBarUpdateOutput() (string, bool) {
	if s.switchedMob == nil || s.manager == nil || s.manager.world == nil {
		return "", false
	}
	mob := s.switchedMob
	state := s.manager.world.NPCInfoBar()
	if state.Mode != game.InfobarOn || state.ScreenSize <= 0 {
		return "", false
	}
	update := 0
	if mob.GetMove() != state.LastMove || mob.GetMaxMove() != state.LastMaxMove {
		update |= InfoMove
	}
	if mob.GetMana() != state.LastMana || mob.GetMaxMana() != state.LastMaxMana {
		update |= InfoMana
	}
	if mob.GetHealth() != state.LastHit || mob.GetMaxHealth() != state.LastMaxHit {
		update |= InfoHit
	}
	if mob.GetGold() != state.LastGold {
		update |= InfoGold
	}
	if mob.GetExp() != state.LastExp {
		update |= InfoExp
	}
	if update == 0 {
		return "", false
	}

	is := newBodyInfobarState(mob, state)
	var output strings.Builder
	if update&InfoMana != 0 {
		output.WriteString(vtCurSave + infobarClearManaPoints(is) + infobarManaPoints(is) + vtCurRest)
	}
	if update&InfoMove != 0 {
		output.WriteString(vtCurSave + infobarClearMovePoints(is) + infobarMovePoints(is) + vtCurRest)
	}
	if update&InfoHit != 0 {
		output.WriteString(vtCurSave + infobarClearHitPoints(is) + infobarHitPoints(is) + vtCurRest)
	}
	if update&InfoExp != 0 {
		output.WriteString(vtCurSave + infobarClearExpPoints(is) + infobarExpPoints(is))
		if is.level < game.LVL_IMMORT {
			output.WriteString(infobarClearLevel(is) + infobarLevel(is))
			output.WriteString(infobarClearNeededExpPoints(is) + infobarNeededExpPoints(is))
		}
		output.WriteString(vtCurRest)
	}
	if update&InfoGold != 0 {
		output.WriteString(vtCurSave + infobarClearGold(is) + infobarGold(is) + vtCurRest)
	}

	state.LastHit = mob.GetHealth()
	state.LastMaxHit = mob.GetMaxHealth()
	state.LastMana = mob.GetMana()
	state.LastMaxMana = mob.GetMaxMana()
	state.LastMove = mob.GetMove()
	state.LastMaxMove = mob.GetMaxMove()
	state.LastExp = mob.GetExp()
	state.LastGold = mob.GetGold()
	s.manager.world.SetNPCInfoBar(state)
	return output.String(), true
}
