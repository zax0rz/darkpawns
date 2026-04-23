// Package command provides a command registry for Dark Pawns.
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

// Registry is a thread-safe command registry.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*Entry // keyed by primary name and aliases
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]*Entry),
	}
}

// Register adds a command to the registry.
// Aliases are registered pointing to the same entry.
func (r *Registry) Register(name string, handler Handler, helpText string, minLevel, minPosition int, aliases ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := &Entry{
		Name:        name,
		Handler:     handler,
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
