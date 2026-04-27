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
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/common"
)

// Handler is the function signature for command handlers.
// It matches the common.CommandSession handler pattern.
type Handler func(common.CommandSession, []string) error

// Entry holds metadata for a registered command.
type Entry struct {
	Name        string
	Handler     Handler
	HelpText    string
	MinLevel    int
	MinPosition int
	Aliases     []string
}

// Middleware wraps a Handler to add cross-cutting behavior (logging, auth, rate limiting).
type Middleware func(Handler) Handler

// Registry is a thread-safe command registry with middleware support.
type Registry struct {
	mu         sync.RWMutex
	commands   map[string]*Entry // keyed by primary name and aliases
	middleware []Middleware       // global middleware chain
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
	for i := len(r.middleware) - 1; i >= 0; i-- {
		wrapped = r.middleware[i](wrapped)
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

// buildChain wraps a handler through all registered middleware.
// Returns a handler that runs the full chain and finally the original handler.
func (r *Registry) buildChain(h Handler) Handler {
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	return h
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
	entry, ok := r.Lookup(cmdStr)
	if !ok {
		return fmt.Errorf("unknown command: %s", cmdStr)
	}
	return entry.Handler(s, args)
}
