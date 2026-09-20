// Package olc contains state shared by the OLC frontends.
package olc

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Kind identifies the OLC editor that owns a claim or dirty save entry.
type Kind string

const (
	KindRoom   Kind = "room"
	KindMob    Kind = "mob"
	KindObject Kind = "object"
	KindShop   Kind = "shop"
	KindZone   Kind = "zone"
)

// Frontend identifies the client that holds an OLC claim.
type Frontend string

const (
	FrontendTelnet Frontend = "telnet"
)

// Owner is the stable identity of a client editing through an OLC frontend.
// Identity must remain stable for the lifetime of a claim. DisplayName is
// player-facing, while Frontend lets a future web client explain a conflict
// without teaching the registry about that client's transport.
type Owner interface {
	Identity() string
	DisplayName() string
	Frontend() Frontend
}

// ClaimEntry is a value snapshot of one active OLC claim. Owner fields are
// copied at snapshot time so callers never retain a live Owner reference.
// ClaimedAt is captured once, when the claim is first acquired, using
// time.Now's wall clock and monotonic reading.
type ClaimEntry struct {
	Kind             Kind      `json:"kind"`
	Number           int       `json:"number"`
	OwnerIdentity    string    `json:"owner_identity"`
	OwnerDisplayName string    `json:"owner_display_name"`
	OwnerFrontend    Frontend  `json:"owner_frontend"`
	ClaimedAt        time.Time `json:"claimed_at"`
}

// Registry is the single admission registry for all OLC editor kinds.
type Registry struct {
	mu     sync.Mutex
	claims map[claimKey]claim
}

type claimKey struct {
	kind Kind
	num  int
}

// NewRegistry creates an empty OLC admission registry.
func NewRegistry() *Registry {
	return &Registry{claims: make(map[claimKey]claim)}
}

type claim struct {
	owner     Owner
	claimedAt time.Time
}

// Claim reserves an editor key for owner. A repeated claim by the same stable
// owner succeeds, matching the old per-editor admission helpers.
//
// On failure, the returned owner is the client holding the claim.
func (r *Registry) Claim(kind Kind, number int, owner Owner) (Owner, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := claimKey{kind: kind, num: number}
	if existing, ok := r.claims[key]; ok {
		if !sameOwner(existing.owner, owner) {
			return existing.owner, false
		}
		return nil, true
	}
	r.claims[key] = claim{owner: owner, claimedAt: time.Now()}
	return nil, true
}

// Holder returns the owner currently holding kind/number, if any.
func (r *Registry) Holder(kind Kind, number int) (Owner, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	claim, ok := r.claims[claimKey{kind: kind, num: number}]
	if !ok {
		return nil, false
	}
	return claim.owner, true
}

// List returns a deterministic value snapshot of all active claims. The
// returned entries and their owner fields are independent of registry state.
func (r *Registry) List() []ClaimEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := make([]ClaimEntry, 0, len(r.claims))
	for key, claim := range r.claims {
		entry := ClaimEntry{
			Kind:      key.kind,
			Number:    key.num,
			ClaimedAt: claim.claimedAt,
		}
		if claim.owner != nil {
			entry.OwnerIdentity = claim.owner.Identity()
			entry.OwnerDisplayName = claim.owner.DisplayName()
			entry.OwnerFrontend = claim.owner.Frontend()
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Number < entries[j].Number
	})
	return entries
}

// Release removes kind/number only when owner still owns it. This protects a
// newer claim from a stale disconnect or failed-entry cleanup path.
func (r *Registry) Release(kind Kind, number int, owner Owner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := claimKey{kind: kind, num: number}
	if holder, ok := r.claims[key]; ok && sameOwner(holder.owner, owner) {
		delete(r.claims, key)
	}
}

func sameOwner(a, b Owner) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Identity() == b.Identity() && a.Frontend() == b.Frontend()
}

// HolderDescription is the conflict label used by frontends. Telnet keeps
// its historical player-facing bytes; other frontends include their kind so
// a claim refusal identifies which client owns the edit.
func HolderDescription(owner Owner) string {
	if owner == nil {
		return "someone"
	}
	name := owner.DisplayName()
	if name == "" {
		name = "someone"
	}
	if owner.Frontend() != "" && owner.Frontend() != FrontendTelnet {
		return fmt.Sprintf("%s (%s)", name, owner.Frontend())
	}
	return name
}
