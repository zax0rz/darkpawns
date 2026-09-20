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
	FrontendWeb    Frontend = "web"
)

// DefaultClaimTTL is the lease duration used by the telnet editors and by
// webOLC when a caller does not choose a shorter test or deployment value.
// A five-minute idle window bounds abandoned claims without making a builder
// lose an ordinary menu pause; drafts are independent and survive expiry.
const DefaultClaimTTL = 5 * time.Minute

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
	ExpiresAt        time.Time `json:"expires_at"`
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
	expiresAt time.Time
}

// Claim reserves an editor key for owner. A repeated claim by the same stable
// owner succeeds, matching the old per-editor admission helpers.
//
// On failure, the returned owner is the client holding the claim.
func (r *Registry) Claim(kind Kind, number int, owner Owner, requestedTTL ...time.Duration) (Owner, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	ttl := claimTTL(requestedTTL)
	key := claimKey{kind: kind, num: number}
	if existing, ok := r.claims[key]; ok {
		if !now.Before(existing.expiresAt) {
			delete(r.claims, key)
		} else {
			if !sameOwner(existing.owner, owner) {
				return existing.owner, false
			}
			existing.expiresAt = now.Add(ttl)
			r.claims[key] = existing
			return nil, true
		}
	}
	r.claims[key] = claim{owner: owner, claimedAt: now, expiresAt: now.Add(ttl)}
	return nil, true
}

// Renew extends an active lease without changing its original ClaimedAt. It
// is deliberately ownership-checked so a stale request cannot revive a
// claim that has already been taken by another frontend.
func (r *Registry) Renew(kind Kind, number int, owner Owner, requestedTTL ...time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := claimKey{kind: kind, num: number}
	existing, ok := r.claims[key]
	if !ok || !sameOwner(existing.owner, owner) || !time.Now().Before(existing.expiresAt) {
		if ok && !time.Now().Before(existing.expiresAt) {
			delete(r.claims, key)
		}
		return false
	}
	existing.expiresAt = time.Now().Add(claimTTL(requestedTTL))
	r.claims[key] = existing
	return true
}

// Holder returns the owner currently holding kind/number, if any.
func (r *Registry) Holder(kind Kind, number int) (Owner, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	claim, ok := r.claims[claimKey{kind: kind, num: number}]
	if !ok {
		return nil, false
	}
	if !time.Now().Before(claim.expiresAt) {
		delete(r.claims, claimKey{kind: kind, num: number})
		return nil, false
	}
	return claim.owner, true
}

// Entry returns a value snapshot of one active claim, including its lease
// timestamps. The owner interface never crosses this boundary.
func (r *Registry) Entry(kind Kind, number int) (ClaimEntry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := claimKey{kind: kind, num: number}
	claim, ok := r.claims[key]
	if !ok {
		return ClaimEntry{}, false
	}
	if !time.Now().Before(claim.expiresAt) {
		delete(r.claims, key)
		return ClaimEntry{}, false
	}
	return claimEntry(key, claim), true
}

// List returns a deterministic value snapshot of all active claims. The
// returned entries and their owner fields are independent of registry state.
func (r *Registry) List() []ClaimEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entries := make([]ClaimEntry, 0, len(r.claims))
	for key, claim := range r.claims {
		if !now.Before(claim.expiresAt) {
			delete(r.claims, key)
			continue
		}
		entries = append(entries, claimEntry(key, claim))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Number < entries[j].Number
	})
	return entries
}

func claimEntry(key claimKey, value claim) ClaimEntry {
	entry := ClaimEntry{
		Kind:      key.kind,
		Number:    key.num,
		ClaimedAt: value.claimedAt,
		ExpiresAt: value.expiresAt,
	}
	if value.owner != nil {
		entry.OwnerIdentity = value.owner.Identity()
		entry.OwnerDisplayName = value.owner.DisplayName()
		entry.OwnerFrontend = value.owner.Frontend()
	}
	return entry
}

func claimTTL(requested []time.Duration) time.Duration {
	if len(requested) > 0 && requested[0] > 0 {
		return requested[0]
	}
	return DefaultClaimTTL
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

// HolderDescription is the legacy owner-only label used by older call sites.
// New claim refusals should use ClaimConflictDescription so both frontends
// expose frontend and idle metadata from the same claim snapshot.
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

// ClaimConflictDescription is the shared refusal label for both frontends.
// It intentionally exposes identity metadata and idle time, but never any
// draft contents or values.
func ClaimConflictDescription(entry ClaimEntry, now time.Time) string {
	name := entry.OwnerDisplayName
	if name == "" {
		name = "someone"
	}
	frontend := string(entry.OwnerFrontend)
	if frontend == "" {
		frontend = "unknown"
	}
	idle := now.Sub(entry.ClaimedAt)
	if idle < 0 {
		idle = 0
	}
	idle = idle.Round(time.Second)
	return fmt.Sprintf("%s (%s; idle %s)", name, frontend, idle)
}
