package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

var characterCommandNumber = dprng.Number

// characterCommandContext keeps the acting body separate from the descriptor
// that may control it. A mobile dispatched by force has no Session; a switched
// body keeps the real Session that owns its descriptor.
type characterCommandContext struct {
	actor   combat.Combatant
	session *Session
	manager *Manager
}

// characterCommandContextForSession selects the body C's command_interpreter
// sees while retaining the descriptor-owning session.
func characterCommandContextForSession(s *Session) characterCommandContext {
	if s == nil {
		return characterCommandContext{}
	}
	// Switch ownership fields and the attached player are published under
	// manager.mu. Snapshot them together; do not call helpers that reacquire
	// that lock while it is held. Lifecycle operations also serialize through
	// playerLifecycleMu, and switch attach/detach update these fields under
	// manager.mu while holding it. A detached, test-created Session has no
	// concurrent lifecycle and can be read directly.
	if s.manager != nil {
		s.manager.mu.RLock()
		defer s.manager.mu.RUnlock()
	}
	if s.isSwitched {
		if s.switchedMob != nil {
			return characterCommandContext{actor: s.switchedMob, session: s, manager: s.manager}
		}
		if s.switchedPlayer != nil {
			return characterCommandContext{actor: s.switchedPlayer, session: s, manager: s.manager}
		}
	}
	return characterCommandContext{actor: s.player, session: s, manager: s.manager}
}

// characterCommandResolution uses the existing ordered C command table and
// registry metadata. Name is the resolved canonical command when found, or
// the normalized typed command otherwise.
type characterCommandResolution struct {
	Name  string
	Entry *command.Entry
	Found bool
}

type characterCommandPreparation struct {
	Args    []string
	RawArgs string
	Empty   bool
	Command characterCommandResolution
}

// prepareCharacterCommand is the direct-interpreter prelude for a real body.
// Transport input continues through executeCommandRaw, whose existing
// empty-line behavior is a separate legacy boundary.
func prepareCharacterCommand(ctx characterCommandContext, line string, draw func(int, int) int) characterCommandPreparation {
	clearCharacterHideOnDraw(ctx.actor, draw)
	name, args := splitCommandInput(line)
	if name == "" {
		return characterCommandPreparation{Empty: true}
	}
	level := 0
	if ctx.actor != nil {
		level = ctx.actor.GetLevel()
	}
	return characterCommandPreparation{
		Args:    args,
		RawArgs: cRawCommandRemainder(line),
		Command: resolveCharacterCommand(name, level),
	}
}

func (m *Manager) executeCharacterCommand(actor combat.Combatant, sess *Session, line string) game.CharacterCommandResult {
	return dispatchCharacterCommand(characterCommandContext{actor: actor, session: sess, manager: m}, line, characterCommandNumber)
}

func dispatchCharacterCommand(ctx characterCommandContext, line string, draw func(int, int) int) game.CharacterCommandResult {
	prepared := prepareCharacterCommand(ctx, line, draw)
	if ctx.actor == nil {
		return game.CharacterCommandUnsupported
	}
	if prepared.Empty {
		return game.CharacterCommandHandled
	}

	if !prepared.Command.Found {
		if _, social := game.SocialMinPosition(prepared.Command.Name); social {
			if _, exists := game.Socials[prepared.Command.Name]; exists && isCSocial(prepared.Command.Name) {
				return dispatchCharacterSocial(ctx, prepared.Command.Name, prepared.RawArgs)
			}
		}
		characterPrivate(ctx, "Huh?!?\r\n")
		return game.CharacterCommandHandled
	}
	entry := prepared.Command.Entry
	if entry == nil {
		return game.CharacterCommandUnsupported
	}
	if actorCommandGateRejected(ctx, entry.MinLevel, entry.MinPosition) {
		return game.CharacterCommandHandled
	}

	special := runCharacterSpecials(ctx, prepared.Command.Name, prepared.RawArgs)
	switch special {
	case game.SpecialHandled:
		return game.CharacterCommandHandled
	case game.SpecialUnsupported:
		return game.CharacterCommandUnsupported
	case game.SpecialDeclined:
	default:
		return game.CharacterCommandUnsupported
	}
	if entry.ActorHandler == nil {
		return game.CharacterCommandUnsupported
	}
	result := entry.ActorHandler(game.CharacterCommandContext{
		Actor: ctx.actor, World: characterWorld(ctx), PrivateOutput: characterPrivateFunc(ctx), RawArgs: prepared.RawArgs,
	}, prepared.Command.Name, prepared.Args)
	if result == game.CharacterCommandHandled || result == game.CharacterCommandUnsupported {
		return result
	}
	return game.CharacterCommandUnsupported
}

func dispatchCharacterSocial(ctx characterCommandContext, name, rawArgs string) game.CharacterCommandResult {
	minimum, ok := game.SocialMinPosition(name)
	if !ok {
		return game.CharacterCommandUnsupported
	}
	if actorCommandGateRejected(ctx, 0, minimum) {
		return game.CharacterCommandHandled
	}
	special := runCharacterSpecials(ctx, name, rawArgs)
	if special == game.SpecialHandled {
		return game.CharacterCommandHandled
	}
	if special == game.SpecialUnsupported {
		return game.CharacterCommandUnsupported
	}
	if mob, ok := ctx.actor.(*game.MobInstance); ok {
		if w := characterWorld(ctx); w != nil {
			w.NPCSocialCommand(mob, name, rawArgs)
			return game.CharacterCommandHandled
		}
	}
	return game.CharacterCommandUnsupported
}

func actorCommandGateRejected(ctx characterCommandContext, minLevel, minPosition int) bool {
	if ctx.actor == nil {
		return true
	}
	if ctx.actor.GetLevel() < minLevel {
		characterPrivate(ctx, "Huh?!?\r\n")
		return true
	}
	if player, ok := ctx.actor.(*game.Player); ok && player.GetFlags()&(1<<uint(game.PlrFrozen)) != 0 && ctx.actor.GetLevel() < LVL_IMPL-1 {
		characterPrivate(ctx, "You try, but the mind-numbing cold prevents you...\r\n")
		return true
	}
	if ctx.actor.IsNPC() && minLevel >= LVL_IMMORT {
		characterPrivate(ctx, "You can't use immortal commands while switched.\r\n")
		return true
	}
	if ctx.actor.GetPosition() < minPosition {
		characterPrivate(ctx, positionFailMessage(ctx.actor.GetPosition()))
		return true
	}
	return false
}

func characterWorld(ctx characterCommandContext) *game.World {
	if ctx.manager != nil {
		return ctx.manager.world
	}
	if ctx.session != nil && ctx.session.manager != nil {
		return ctx.session.manager.world
	}
	return nil
}

func characterPrivateFunc(ctx characterCommandContext) func(string) {
	if ctx.session == nil {
		return nil
	}
	return func(text string) { ctx.session.Send(text) }
}

func characterPrivate(ctx characterCommandContext, text string) {
	if send := characterPrivateFunc(ctx); send != nil {
		send(text)
	}
}

func runCharacterSpecials(ctx characterCommandContext, name, rawArgs string) game.SpecialResult {
	w := characterWorld(ctx)
	actor, ok := ctx.actor.(*game.MobInstance)
	if w == nil || !ok || actor == nil {
		return game.SpecialUnsupported
	}
	roomVNum := actor.GetRoomVNum()
	arg := rawArgs
	if roomVNum >= 0 {
		if specName, assigned := game.GetRoomSpecName(roomVNum); assigned {
			result := game.DispatchCharacterSpecial(specName, w, actorSpecialContext(ctx), nil, name, arg)
			if result != game.SpecialDeclined {
				return result
			}
		}
	}

	for _, item := range game.COrderedMobWorn(actor) {
		if item == nil {
			continue
		}
		if specName, assigned := game.GetObjSpecName(item.VNum); assigned {
			result := game.DispatchCharacterObjectSpecial(specName, w, actorSpecialContext(ctx), item, name, arg)
			if result != game.SpecialDeclined {
				return result
			}
		}
	}
	for _, item := range actor.InventorySnapshot() {
		if item == nil {
			continue
		}
		if specName, assigned := game.GetObjSpecName(item.VNum); assigned {
			result := game.DispatchCharacterObjectSpecial(specName, w, actorSpecialContext(ctx), item, name, arg)
			if result != game.SpecialDeclined {
				return result
			}
		}
	}
	for _, mob := range w.COrderedRoomMobs(roomVNum) {
		if mob == nil || mob.HasMobFlag(game.MobFlagExtract) {
			continue
		}
		if specName, assigned := game.GetMobSpecName(mob.VNum); assigned {
			result := game.DispatchCharacterSpecial(specName, w, actorSpecialContext(ctx), mob, name, arg)
			if result != game.SpecialDeclined {
				return result
			}
		}
	}
	actorRef := &scripting.CharRef{NPC: true, ID: actor.GetID()}
	for _, item := range w.GetItemsInRoom(roomVNum) {
		if item == nil {
			continue
		}
		if specName, assigned := game.GetObjSpecName(item.VNum); assigned {
			result := game.DispatchCharacterObjectSpecial(specName, w, actorSpecialContext(ctx), item, name, arg)
			if result != game.SpecialDeclined {
				return result
			}
		}
		// C's room-object oncmd arm appends the raw argument pointer directly
		// after CMD_NAME (src/interpreter.c:1470-1476). rawArgs already starts
		// with any delimiter whitespace, so inserting another space changes it.
		if w.RunObjOnCmdScript(actorRef, nil, item, roomVNum, name+rawArgs) {
			return game.SpecialHandled
		}
	}
	return game.SpecialDeclined
}

func actorSpecialContext(ctx characterCommandContext) game.CharacterSpecialContext {
	return game.CharacterSpecialContext{Actor: ctx.actor, PrivateOutput: characterPrivateFunc(ctx)}
}

// cRawCommandRemainder mirrors the line pointer returned by C's any_one_arg
// or the punctuation branch in command_interpreter. The pointer begins at
// the whitespace that ended a word command, and punctuation consumes only
// its first byte. Unlike CommandArgumentText, this intentionally preserves
// leading and trailing whitespace for direct C-style actor preparation.
func cRawCommandRemainder(line string) string {
	line = strings.TrimLeft(line, cCommandWhitespace)
	if line == "" {
		return ""
	}
	first := line[0]
	if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') {
		return line[1:]
	}
	if idx := strings.IndexAny(line, cCommandWhitespace); idx >= 0 {
		return line[idx:]
	}
	return ""
}

func resolveCharacterCommand(typed string, level int) characterCommandResolution {
	name := strings.ToLower(typed)
	if canonical, ok := resolveCommandPrefix(name, level); ok {
		name = canonical
	}
	entry, found := cmdRegistry.Lookup(name)
	return characterCommandResolution{Name: name, Entry: entry, Found: found}
}

// clearCharacterHide mirrors command_interpreter's leading draw for the
// actual actor. Callers own invocation ordering; this helper is not called by
// the player transport path in this slice, which preserves its current empty
// input filter and draw count.
func clearCharacterHideOnDraw(actor combat.Combatant, draw func(int, int) int) {
	if actor == nil || draw == nil {
		return
	}
	if draw(0, 3) != 0 {
		return
	}
	switch body := actor.(type) {
	case *game.Player:
		body.SetAffect(game.AffHide, false)
	case *game.MobInstance:
		body.RemoveAffected(game.AffHide)
	}
}
