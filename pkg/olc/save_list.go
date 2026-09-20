package olc

import "sync"

// DirtyEntry identifies an editor kind with a zone whose source file needs a
// save. The list is deliberately keyed by zone, not by individual VNUM: one
// zone save writes the complete file for that editor kind.
type DirtyEntry struct {
	Kind Kind
	Zone int
}

// SaveList is the single typed dirty-zone list shared by all OLC editors.
type SaveList struct {
	mu    sync.Mutex
	dirty map[DirtyEntry]struct{}
}

// NewSaveList creates an empty dirty-zone save list.
func NewSaveList() *SaveList {
	return &SaveList{dirty: make(map[DirtyEntry]struct{})}
}

// Mark records that the editor kind has changed in zone.
func (s *SaveList) Mark(kind Kind, zone int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty[DirtyEntry{Kind: kind, Zone: zone}] = struct{}{}
}

// Remove clears the dirty marker after a successful zone save.
func (s *SaveList) Remove(kind Kind, zone int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dirty, DirtyEntry{Kind: kind, Zone: zone})
}

// Dirty reports whether kind has an unsaved change in zone.
func (s *SaveList) Dirty(kind Kind, zone int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.dirty[DirtyEntry{Kind: kind, Zone: zone}]
	return ok
}
