// Package command provides a command registry for Dark Pawns.
//
// NOTE (M-04): Several packages (pkg/session, pkg/game) use init() functions
// to register commands via cmdRegistry.Register(). This implicit initialization
// makes order-of-initialization hard to test and reason about. These should
// eventually migrate to explicit registration in main.go or a top-level
// RegisterCommands() function that is called at startup. See cmd/server/main.go
// and pkg/session/commands.go for the current init()-based registration sites.
package command

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// ExecutionContext carries the body a command acts on separately from the
// descriptor-owning session. A switched mob is Body while Session remains the
// immortal's descriptor; for ordinary and PC-switched play, Body is the active
// player.
type ExecutionContext struct {
	Body    combat.Combatant
	Session common.CommandSession
	World   *game.World
	RawArgs string
	// PrivateOutput is used by direct actor calls that have no descriptor. A
	// live session context leaves it nil and writes through Session.
	PrivateOutput func(string)
}

// SendPrivate writes through the descriptor currently owned by the session.
func (ctx ExecutionContext) SendPrivate(text string) {
	if ctx.PrivateOutput != nil {
		ctx.PrivateOutput(text)
	} else if ctx.Session != nil {
		ctx.Session.Send(text)
	}
}

// CharacterContext adapts the shared execution context for game-level actor
// implementations that use the narrower character command contract.
func (ctx ExecutionContext) CharacterContext() game.CharacterCommandContext {
	privateOutput := ctx.PrivateOutput
	if privateOutput == nil && ctx.Session != nil {
		privateOutput = ctx.Session.Send
	}
	var descriptorOwner *game.Player
	if ctx.Session != nil {
		descriptorOwner, _ = ctx.Session.GetPlayer().(*game.Player)
	}
	return game.CharacterCommandContext{
		Actor: ctx.Body, DescriptorOwner: descriptorOwner, World: ctx.World,
		PrivateOutput: privateOutput, RawArgs: ctx.RawArgs,
	}
}

// Handler receives the shared command context, the resolved command name, and
// tokenized arguments.
type Handler func(ExecutionContext, string, []string) error

// ActorHandler handles a command for an explicit character body. The same
// Entry retains its session handler and C-derived command metadata. It receives
// the same body and descriptor context as the ordinary handler.
type ActorHandler func(ExecutionContext, string, []string) game.CharacterCommandResult

// Entry holds metadata for a registered command.
type Entry struct {
	Name         string
	Handler      Handler
	ActorHandler ActorHandler
	HelpText     string
	MinLevel     int
	MinPosition  int
	Aliases      []string
}

// SetActorHandler adds an actor-aware lane to an existing command entry.
// Aliases already point to that same entry, so both lanes share one command
// registration and its C-derived metadata. Like Register, this is a startup
// operation; command entries are read without holding the registry lock once
// command execution begins.
func (r *Registry) SetActorHandler(name string, handler ActorHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.commands[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("cannot set actor handler for unregistered command %q", name)
	}
	entry.ActorHandler = handler
	return nil
}

// Middleware wraps a Handler to add cross-cutting behavior (logging, auth, rate limiting).
type Middleware func(Handler) Handler

// Registry is a thread-safe command registry with middleware support.
type Registry struct {
	mu         sync.RWMutex
	commands   map[string]*Entry // keyed by primary name and aliases
	middleware []Middleware      // global middleware chain
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands:   make(map[string]*Entry),
		middleware: make([]Middleware, 0),
	}
}

// Register adds a command to the registry.
// Aliases are registered pointing to the same entry.
func (r *Registry) Register(name string, handler Handler, helpText string, minLevel, minPosition int, aliases ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Wrap handler through middleware chain
	wrapped := handler
	for _, v := range slices.Backward(r.middleware) {
		wrapped = v(wrapped)
	}

	entry := &Entry{
		Name:        name,
		Handler:     wrapped,
		HelpText:    helpText,
		MinLevel:    minLevel,
		MinPosition: minPosition,
		Aliases:     aliases,
	}

	// Register primary name
	r.commands[strings.ToLower(name)] = entry

	// Register aliases
	for _, alias := range aliases {
		r.commands[strings.ToLower(alias)] = entry
	}
}

// Use adds a middleware function to the global chain.
// Middleware are executed in order, each wrapping the next.
// All future Execute calls through the registry will run through the chain.
func (r *Registry) Use(mw Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, mw)
}

// Lookup finds a command entry by name (case-insensitive).
func (r *Registry) Lookup(name string) (*Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.commands[strings.ToLower(name)]
	return entry, ok
}

// GetAll returns a slice of all unique registered entries.
func (r *Registry) GetAll() []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var result []*Entry
	for _, entry := range r.commands {
		if !seen[entry.Name] {
			seen[entry.Name] = true
			result = append(result, entry)
		}
	}
	return result
}

// Execute looks up and executes a command.
func (r *Registry) Execute(s common.CommandSession, cmdStr string, args []string) error {
	var body combat.Combatant
	if s != nil {
		if player, ok := s.GetPlayer().(*game.Player); ok && player != nil {
			body = player
		}
	}
	return r.ExecuteContext(ExecutionContext{Session: s, Body: body}, cmdStr, args)
}

// ExecuteContext looks up and executes a command with an already resolved body
// context. The actor and session lanes share this one execution context.
func (r *Registry) ExecuteContext(ctx ExecutionContext, cmdStr string, args []string) error {
	entry, ok := r.Lookup(cmdStr)
	if !ok {
		return fmt.Errorf("unknown command: %s", cmdStr)
	}
	return entry.Execute(ctx, cmdStr, args)
}

// Execute invokes the body's actor handler when available, then the session
// handler if the actor lane does not implement that case.
func (entry *Entry) Execute(ctx ExecutionContext, cmdStr string, args []string) error {
	if entry == nil {
		return fmt.Errorf("cannot execute a nil command entry")
	}
	if ctx.Body != nil && ctx.Body.IsNPC() && entry.ActorHandler != nil {
		if entry.ActorHandler(ctx, cmdStr, args) == game.CharacterCommandHandled {
			return nil
		}
	}
	if entry.Handler == nil {
		return fmt.Errorf("command %q has no handler", entry.Name)
	}
	return entry.Handler(ctx, cmdStr, args)
}
