package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// characterCommandContext keeps the acting body separate from the descriptor
// that may control it. A mobile dispatched by force has no Session; a switched
// body keeps the real Session that owns its descriptor. This foundation is not
// yet wired to NPC command callers because native specials still accept
// *Player and cannot safely receive a MobInstance.
type characterCommandContext struct {
	actor   combat.Combatant
	session *Session
}

// characterCommandContextForSession selects the body C's command_interpreter
// sees while retaining the descriptor-owning session. It is deliberately
// separate from ExecuteCommand's existing entry path until the full
// actor-aware special sequence is ready.
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
			return characterCommandContext{actor: s.switchedMob, session: s}
		}
		if s.switchedPlayer != nil {
			return characterCommandContext{actor: s.switchedPlayer, session: s}
		}
	}
	return characterCommandContext{actor: s.player, session: s}
}

// characterCommandResolution uses the existing ordered C command table and
// registry metadata. It performs no output, random draws, special calls, or
// handler invocation so the NPC path can remain unwired until those actor
// hooks are safe. Name is the resolved canonical command when found, or the
// normalized typed command otherwise.
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
// Its caller supplies the draw source so tests can prove ordering and count;
// transport input must continue through executeCommandRaw, whose existing
// empty-line filter intentionally remains outside this foundation.
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
