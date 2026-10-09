package session

import (
	"fmt"
	"net"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// CConnectionHost renders an address the way C stores it in d->host: the
// zero-padded dotted quad for IPv4 (src/comm.c:1536-1539, "%03u.%03u.%03u.%03u"),
// and the address unchanged otherwise.
//
// C's new_descriptor writes either that padded quad or a resolved name into
// d->host, so an IPv6 literal has no C form: it is a Go-only transport value
// and is passed through. Go's net.ParseIP rejects leading-zero quads, so an
// already-padded string is returned unchanged.
func CConnectionHost(address string) string {
	if ip := net.ParseIP(address).To4(); ip != nil {
		return fmt.Sprintf("%03d.%03d.%03d.%03d", ip[0], ip[1], ip[2], ip[3])
	}
	return address
}

// WildBanHosts returns C's two additional accept-time ban candidates for a
// padded IPv4 host (src/comm.c:1541-1548): "%03u.%03u.%03u.*" and
// "%03u.%03u.*.*". Non-quad input yields no candidates.
func WildBanHosts(padded string) []string {
	parts := strings.Split(padded, ".")
	if len(parts) != 4 {
		return nil
	}
	return []string{
		parts[0] + "." + parts[1] + "." + parts[2] + ".*",
		parts[0] + "." + parts[1] + ".*.*",
	}
}

// AcceptBanLevel is C's isbanned() union at accept time (src/comm.c:1567-1569):
// new_descriptor tests newd->host, wildhost and double_wild and refuses a
// connection only when one of them is BAN_ALL.
//
// resolved reports whether d->host came from a successful name lookup. C fills
// wildhost and double_wild only on the resolution-failure branch
// (src/comm.c:1541-1548); after a successful lookup both are uninitialised
// stack arrays, and C compares the garbage (R1a). The port checks the host
// alone in that branch rather than reproducing undefined behaviour.
func AcceptBanLevel(bm *game.BanManager, host string, resolved bool) int {
	if bm == nil {
		return game.BanNot
	}
	level := bm.IsBanned(host)
	if resolved {
		return level
	}
	for _, candidate := range WildBanHosts(host) {
		if got := bm.IsBanned(candidate); got > level {
			level = got
		}
	}
	return level
}

// ConnectionIdentity is C's new_descriptor view of an accepted connection
// before admission: the string it stores in d->host and the ban level its
// accept-time isbanned() calls produce (src/comm.c:1534-1569).
type ConnectionIdentity struct {
	// Host is C's d->host: the zero-padded dotted quad, or a resolved name.
	Host string
	// HostResolved reports that Host came from a successful name lookup. C
	// only fills wildhost and double_wild on the resolution-failure branch.
	HostResolved bool
	// Level is C's isbanned() union over the accept-time strings.
	Level int
}

// SetMudHost records C's d->host for this descriptor, as new_descriptor stores
// it (src/comm.c:1534-1563). It is set once, before the session is admitted.
func (s *Session) SetMudHost(host string) {
	if host == "" {
		return
	}
	s.mudHost = host
}

// MudHost returns C's d->host for this descriptor: the padded IPv4 quad, or the
// resolved name when a lookup ran and succeeded (src/comm.c:1534-1563). With
// nameserver_is_slow — C's shipped default (src/config.c:206) — no lookup runs
// and it is always the padded quad.
func (s *Session) MudHost() string {
	if s.mudHost != "" {
		return s.mudHost
	}
	return CConnectionHost(s.RemoteIP())
}
