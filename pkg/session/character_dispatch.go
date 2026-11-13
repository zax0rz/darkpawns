package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

var characterCommandNumber = dprng.Number

// characterCommandContextForSession selects the body C's command_interpreter
// sees while retaining the descriptor-owning session.
func characterCommandContextForSession(s *Session) command.ExecutionContext {
	if s == nil {
		return command.ExecutionContext{}
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
	ctx := command.ExecutionContext{Session: &commandSession{Session: s}}
	if s.manager != nil {
		ctx.World = s.manager.world
	}
	if s.isSwitched {
		if s.switchedMob != nil {
			ctx.Body = s.switchedMob
			return ctx
		}
		if s.switchedPlayer != nil {
			ctx.Body = s.switchedPlayer
			return ctx
		}
	}
	if s.player != nil {
		ctx.Body = s.player
	}
	return ctx
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
func prepareCharacterCommand(ctx command.ExecutionContext, line string, draw func(int, int) int) characterCommandPreparation {
	clearCharacterHideOnDraw(ctx.Body, draw)
	name, args := splitCommandInput(line)
	if name == "" {
		return characterCommandPreparation{Empty: true}
	}
	level := 0
	if ctx.Body != nil {
		level = ctx.Body.GetLevel()
	}
	return characterCommandPreparation{
		Args:    args,
		RawArgs: cRawCommandRemainder(line),
		Command: resolveCharacterCommand(name, level),
	}
}

func (m *Manager) executeCharacterCommand(actor combat.Combatant, sess *Session, line string) game.CharacterCommandResult {
	return dispatchCharacterCommand(command.ExecutionContext{Body: actor, Session: commandSessionForCharacter(sess), World: m.world}, line, characterCommandNumber)
}

// executeCharacterCommandDirect dispatches a direct C ACMD call. The C caller
// already selected the command body and passed its argument pointer, so this
// path skips command_interpreter's draw, name resolution, gates, and specials.
func (m *Manager) executeCharacterCommandDirect(actor combat.Combatant, sess *Session, commandName, rawArgs string) game.CharacterCommandResult {
	return dispatchCharacterCommandDirect(command.ExecutionContext{Body: actor, Session: commandSessionForCharacter(sess), World: m.world, RawArgs: rawArgs}, commandName, rawArgs)
}

func commandSessionForCharacter(sess *Session) common.CommandSession {
	if sess == nil {
		return nil
	}
	return &commandSession{Session: sess}
}

func dispatchCharacterCommandDirect(ctx command.ExecutionContext, commandName, rawArgs string) game.CharacterCommandResult {
	ctx.RawArgs = rawArgs
	if ctx.Body == nil {
		return game.CharacterCommandUnsupported
	}
	commandName = strings.ToLower(strings.TrimSpace(commandName))
	if _, isSocial := game.SocialMinPosition(commandName); isSocial {
		if _, exists := game.Socials[commandName]; !exists || !isCSocial(commandName) {
			return game.CharacterCommandUnsupported
		}
		mob, ok := ctx.Body.(*game.MobInstance)
		w := ctx.World
		if !ok || w == nil {
			return game.CharacterCommandUnsupported
		}
		w.NPCSocialCommand(mob, commandName, rawArgs)
		return game.CharacterCommandHandled
	}
	entry, found := cmdRegistry.Lookup(commandName)
	if !found || entry == nil || entry.ActorHandler == nil {
		return game.CharacterCommandUnsupported
	}
	result := entry.ActorHandler(ctx, commandName, strings.Fields(rawArgs))
	if result == game.CharacterCommandHandled {
		return result
	}
	return game.CharacterCommandUnsupported
}

func dispatchCharacterCommand(ctx command.ExecutionContext, line string, draw func(int, int) int) game.CharacterCommandResult {
	prepared := prepareCharacterCommand(ctx, line, draw)
	if ctx.Body == nil {
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
	ctx.RawArgs = prepared.RawArgs
	result := entry.ActorHandler(ctx, prepared.Command.Name, prepared.Args)
	if result == game.CharacterCommandHandled || result == game.CharacterCommandUnsupported {
		return result
	}
	return game.CharacterCommandUnsupported
}

func dispatchCharacterSocial(ctx command.ExecutionContext, name, rawArgs string) game.CharacterCommandResult {
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
	if mob, ok := ctx.Body.(*game.MobInstance); ok {
		if w := ctx.World; w != nil {
			w.NPCSocialCommand(mob, name, rawArgs)
			return game.CharacterCommandHandled
		}
	}
	return game.CharacterCommandUnsupported
}

func actorCommandGateRejected(ctx command.ExecutionContext, minLevel, minPosition int) bool {
	if ctx.Body == nil {
		return true
	}
	if ctx.Body.GetLevel() < minLevel {
		characterPrivate(ctx, "Huh?!?\r\n")
		return true
	}
	if player, ok := ctx.Body.(*game.Player); ok && player.GetFlags()&(1<<uint(game.PlrFrozen)) != 0 && ctx.Body.GetLevel() < LVL_IMPL-1 {
		characterPrivate(ctx, "You try, but the mind-numbing cold prevents you...\r\n")
		return true
	}
	if ctx.Body.IsNPC() && minLevel >= LVL_IMMORT {
		characterPrivate(ctx, "You can't use immortal commands while switched.\r\n")
		return true
	}
	if ctx.Body.GetPosition() < minPosition {
		characterPrivate(ctx, positionFailMessage(ctx.Body.GetPosition()))
		return true
	}
	return false
}

func characterWorld(ctx command.ExecutionContext) *game.World {
	return ctx.World
}

func characterPrivateFunc(ctx command.ExecutionContext) func(string) {
	if ctx.PrivateOutput != nil {
		return ctx.PrivateOutput
	}
	if ctx.Session == nil {
		return nil
	}
	return ctx.Session.Send
}

func characterPrivate(ctx command.ExecutionContext, text string) {
	if send := characterPrivateFunc(ctx); send != nil {
		send(text)
	}
}

func runCharacterSpecials(ctx command.ExecutionContext, name, rawArgs string) game.SpecialResult {
	w := characterWorld(ctx)
	actor, ok := ctx.Body.(*game.MobInstance)
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
		specName, assigned := game.GetMobSpecName(mob.VNum)
		if !assigned {
			// src/shop.c:1226-1243 assign_the_shopkeepers writes the native
			// shop_keeper special onto every keeper loaded from .shp data at
			// boot, so it never appears in MobSpecAssign. Recover it from the
			// world's shop index exactly as the player lane does
			// (pkg/session/commands.go:993-999); without it an ordered pet
			// beside a keeper had every command blocked, and C's steal arm
			// (keeper slaps the pet) never ran.
			if _, isShopKeeper := w.GetShopByKeeper(mob.VNum); isShopKeeper {
				specName, assigned = "shop_keeper", true
			}
		}
		if assigned {
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

func actorSpecialContext(ctx command.ExecutionContext) game.CharacterSpecialContext {
	return game.CharacterSpecialContext{Actor: ctx.Body, PrivateOutput: characterPrivateFunc(ctx)}
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
