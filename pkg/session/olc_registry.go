package session

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

var olcSaveList = olc.NewSaveList()

const (
	olcKindRoom   = olc.KindRoom
	olcKindMob    = olc.KindMob
	olcKindObject = olc.KindObject
	olcKindShop   = olc.KindShop
	olcKindZone   = olc.KindZone
)

// Identity implements olc.Owner. sessionID is stable for one connection and
// lets the registry distinguish a reconnect from a stale owner pointer.
func (s *Session) Identity() string {
	return s.sessionID()
}

// DisplayName implements olc.Owner.
func (s *Session) DisplayName() string {
	if s.playerName != "" {
		return s.playerName
	}
	if s.player != nil {
		return s.player.GetName()
	}
	return ""
}

// Frontend implements olc.Owner. Telnet is the only live OLC client today;
// the owner boundary keeps that transport detail out of pkg/olc.
func (s *Session) Frontend() olc.Frontend {
	return olc.FrontendTelnet
}

func (m *Manager) olcClaims() *olc.Registry {
	if m.olcRegistry == nil {
		m.olcRegistry = olc.NewRegistry()
	}
	return m.olcRegistry
}

func (m *Manager) claimOLC(kind olc.Kind, number int, owner *Session) (olc.Owner, bool) {
	return m.olcClaims().Claim(kind, number, owner)
}

func (m *Manager) releaseOLC(kind olc.Kind, number int, owner *Session) {
	m.olcClaims().Release(kind, number, owner)
}

func (m *Manager) claimRoomEdit(number int, owner *Session) (string, bool) {
	holder, ok := m.claimOLC(olc.KindRoom, number, owner)
	if ok {
		return "", true
	}
	return m.olcConflictDescription(olc.KindRoom, number, holder), false
}

func (m *Manager) releaseRoomEdit(number int, owner *Session) {
	m.releaseOLC(olc.KindRoom, number, owner)
}

func (m *Manager) claimMobEdit(number int, owner *Session) (string, bool) {
	holder, ok := m.claimOLC(olc.KindMob, number, owner)
	if ok {
		return "", true
	}
	return m.olcConflictDescription(olc.KindMob, number, holder), false
}

func (m *Manager) releaseMobEdit(number int, owner *Session) {
	m.releaseOLC(olc.KindMob, number, owner)
}

func (m *Manager) claimObjEdit(number int, owner *Session) (string, bool) {
	holder, ok := m.claimOLC(olc.KindObject, number, owner)
	if ok {
		return "", true
	}
	return m.olcConflictDescription(olc.KindObject, number, holder), false
}

func (m *Manager) releaseObjEdit(number int, owner *Session) {
	m.releaseOLC(olc.KindObject, number, owner)
}

func (m *Manager) claimShopEdit(number int, owner *Session) (string, bool) {
	holder, ok := m.claimOLC(olc.KindShop, number, owner)
	if ok {
		return "", true
	}
	return m.olcConflictDescription(olc.KindShop, number, holder), false
}

func (m *Manager) releaseShopEdit(number int, owner *Session) {
	m.releaseOLC(olc.KindShop, number, owner)
}

func (m *Manager) claimZoneEdit(number int, owner *Session) (string, bool) {
	holder, ok := m.claimOLC(olc.KindZone, number, owner)
	if ok {
		return "", true
	}
	return m.olcConflictDescription(olc.KindZone, number, holder), false
}

func (m *Manager) releaseZoneEdit(number int, owner *Session) {
	m.releaseOLC(olc.KindZone, number, owner)
}

func (m *Manager) olcHolder(kind olc.Kind, number int) string {
	holder, ok := m.olcClaims().Holder(kind, number)
	if !ok {
		return ""
	}
	return olc.HolderDescription(holder)
}

func (m *Manager) olcConflictDescription(kind olc.Kind, number int, holder olc.Owner) string {
	if entry, ok := m.olcClaims().Entry(kind, number); ok {
		return olc.ClaimConflictDescription(entry, time.Now())
	}
	return olc.HolderDescription(holder)
}

// ClaimOLC exposes only the ownership-checked operations needed by the Huma
// webOLC surface. The registry remains owned and locked by Manager.
func (m *Manager) ClaimOLC(kind olc.Kind, number int, owner olc.Owner, ttl time.Duration) (olc.Owner, bool) {
	return m.olcClaims().Claim(kind, number, owner, ttl)
}

func (m *Manager) RenewOLC(kind olc.Kind, number int, owner olc.Owner, ttl time.Duration) bool {
	return m.olcClaims().Renew(kind, number, owner, ttl)
}

func (m *Manager) ReleaseOLC(kind olc.Kind, number int, owner olc.Owner) {
	m.olcClaims().Release(kind, number, owner)
}

func (m *Manager) MarkOLCDirty(kind olc.Kind, zone int) {
	markOLCDirty(kind, zone)
}

// SaveOLCZone serializes all five OLC files while holding the one zone lock.
// The member-claim check is deliberately across kinds: a pending editor must
// not be able to commit new in-memory bytes while the save snapshot is being
// assembled.
func (m *Manager) SaveOLCZone(number int) error {
	if m.world == nil {
		return fmt.Errorf("world unavailable")
	}
	zone, ok := m.world.SnapshotZone(number)
	if !ok {
		return fmt.Errorf("zone %d not found", number)
	}
	start := number * 100
	for _, entry := range m.olcClaims().List() {
		if entry.Number >= start && entry.Number <= zone.TopRoom {
			return &olc.ZoneSaveConflict{Entry: entry}
		}
	}

	saveMu := olc.ZoneSaveLock(number)
	saveMu.Lock()
	defer saveMu.Unlock()
	if err := saveReditZoneLocked(m.world, &zone); err != nil {
		return err
	}
	if err := saveMeditZoneLocked(m.world, &zone); err != nil {
		return err
	}
	if err := saveOeditZoneLocked(m.world, &zone); err != nil {
		return err
	}
	if err := saveSeditZoneLocked(m.world, &zone); err != nil {
		return err
	}
	if err := saveZeditZoneLocked(m.world, &zone); err != nil {
		return err
	}
	return nil
}

// EmitOLCPresence emits the existing room emote for an online player. It
// intentionally does not touch PlrWriting: that flag is descriptor state and
// web editing must not suppress a telnet player's prompt.
func (m *Manager) EmitOLCPresence(playerName string, start bool) {
	s, ok := m.GetSession(playerName)
	if !ok || s == nil || s.player == nil || m.world == nil {
		return
	}
	message := "$n stops using OLC."
	if start {
		message = "$n starts using OLC."
	}
	game.Act(m.world, true, s.player, nil, nil, nil, message, "", game.ToRoom)
}

// These named accessors retain the existing session package call sites while
// routing every editor through the one typed registry.
func (m *Manager) roomEditHolder(number int) string {
	return m.olcHolder(olc.KindRoom, number)
}

func (m *Manager) mobEditHolder(number int) string {
	return m.olcHolder(olc.KindMob, number)
}

func (m *Manager) objEditHolder(number int) string {
	return m.olcHolder(olc.KindObject, number)
}

func (m *Manager) shopEditHolder(number int) string {
	return m.olcHolder(olc.KindShop, number)
}

func (m *Manager) zoneEditHolder(number int) string {
	return m.olcHolder(olc.KindZone, number)
}

func markOLCDirty(kind olc.Kind, zone int) {
	olcSaveList.Mark(kind, zone)
}

func clearOLCDirty(kind olc.Kind, zone int) {
	olcSaveList.Remove(kind, zone)
}

func applyOLC(op olc.Operation) {
	if err := olc.Apply(op); err != nil {
		slog.Error("OLC operation failed", "operation", op.Kind, "error", err)
	}
}

func applyOLCClamp(value, low, high int) int {
	result := value
	applyOLC(olc.Operation{
		Kind:   olc.OpClampInt,
		Value:  value,
		Low:    low,
		High:   high,
		Result: &result,
	})
	return result
}

// sessionID returns a stable identifier for one connection. The connection
// time keeps a reconnect distinct from the session it replaces.
func (s *Session) sessionID() string {
	return fmt.Sprintf("%s-%d", s.playerName, s.connectedAt.UnixNano())
}
