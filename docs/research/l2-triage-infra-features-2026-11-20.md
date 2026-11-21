# L2 triage: DP-1601, DP-1602, DP-1604, DP-1605 — unported C infrastructure

2026-11-20, Kimi (DP-1603 sprint). Question per item: **can a player, immortals
included, see any difference?** Cites are `src/` lines read this session (R5g).
The reference oracle build (`~/darkpawns-c-oracle/src/config.c`) carries the
same shipped defaults cited below.

## Verdict summary

| issue | feature | player-visible? | recommendation |
|---|---|---|---|
| DP-1601 | POSIX signal handlers | No in-game trigger exists; every effect is operator/OS-level | Close as outside the game (2026-11-19 ruling); ops notes below |
| DP-1602 | record_usage | No — log file only | Close as outside the game |
| DP-1604 | DNS cache | No — the cache is a lookup accelerator; all observable resolution behavior is ported | Close as outside the game |
| DP-1605 | ident lookup | Yes, in one corner: immortal flips `ident` on AND a client host runs identd | Decision for Zach; described exactly below |

## DP-1601 — POSIX signals

C's signal_setup (src/comm.c:2295-2321) registers:

- **SIGUSR1 → reread_wizlists** (src/comm.c:2231-2238): mudlogs "Signal
  received - rereading wizlists." (CMP, LVL_IMMORT — reaches immortals with the
  CMP syslog class enabled, src/comm.c:2235) and reloads the wizlist. Only an
  operator with shell access can raise SIGUSR1; no command or game event does.
  Go has no handler, so a SIGUSR1 to the Go process terminates it — an ops
  hazard note, not a player-visible byte.
- **SIGUSR2 → unrestrict_game** (src/comm.c:2240-2250): mudlogs "Received
  SIGUSR2 - completely unrestricting game (emergent)" (BRF, LVL_IMMORT) and
  clears the ban list, game restriction, and invalid-login counter. Same
  operator-only trigger. The ban-clearing *effect* would be player-visible
  (banned players could reconnect) but cannot be triggered by any player.
- **SIGVTALRM → checkpointing** (src/comm.c:2215-2230): aborts the process if
  the game loop stalls 180 virtual seconds. Log/abort only; players see a
  crash either way if it ever fired.
- **SIGHUP/SIGINT/SIGTERM → hupsig** (src/comm.c:2252-2258): log + immediate
  `exit(0)` — no save, no leave line. Go instead drains gracefully (saves,
  leave broadcasts; the leave line itself was DP-1586). This asymmetry only
  materializes when an operator restarts the production server by signal, and
  which behavior is preferable is an ops decision, not a fidelity defect the
  oracle can judge (the harness's restart is SIGINT + 3s grace on both
  engines; C exits instantly either way).
- **SIGPIPE/SIGALRM ignored**: Go's runtime already treats these
  equivalently (EPIPE on socket writes, no setitimer users).

No command, toggle, or in-game event raises any of these signals, so no
player — immortal included — can produce a byte difference through gameplay.
**Recommend: close as outside the game under the 2026-11-19 ruling**, with two
ops notes for the runbook: (1) SIGUSR1/SIGUSR2 kill the Go server where C
reloaded wizlists / unrestricted the game — document that these are unhandled;
(2) a signal restart of Go is graceful where C's is an immediate exit (no
save), so post-restart state freshness differs by design (related: DP-1603's
date_record, which C writes hourly from the live heartbeat and at the
comm.c:291 shutdown position that hupsig's exit(0) never reaches).

## DP-1602 — record_usage

src/comm.c:903-925: counts connected/playing descriptors and `log()`s
"nusage: %-3d sockets connected, %-3d sockets playing" plus, when compiled
with RUSAGE, a getrusage line. Both go to the log file only; nothing is
sent_to_char, no file players read, no command consumes it. The Go heartbeat
even preserves the 5-minute callback slot (`OnRecordUsage`,
pkg/engine/gameloop.go:358) — only the logging body is unported.
**Recommend: close as outside the game.**

## DP-1604 — DNS cache

C's new_descriptor resolves the peer's PTR record only when
`nameserver_is_slow` is off; the cache (get_host_from_cache /
add_dns_host, around src/comm.c:1529-1560) merely avoids repeat lookups.
The shipped default is `nameserver_is_slow = YES` (src/config.c:206, same in
the oracle build), so the reference C server never resolves: d->host is always
the zero-padded quad (src/comm.c:1536-1539), and `users` prints
`[127.000.000.001]` (src/act.informative.c:2093-2095). Go mirrors all of this:
the `slowns` toggle is registered (pkg/session/commands.go:706, LVL_IMPL-1 as
C's src/interpreter.c:708), the flag lives at
pkg/game/other_settings.go:203-220, and the accept path performs the same
quad-vs-resolved-name branch with the same ban semantics
(pkg/telnet/listener.go:311-338, including C's DNS-failure mudlog,
src/comm.c:1554).

The only unported part is the cache itself. Its absence is not player-visible:
resolution results are identical at connection time. The single theoretical
divergence is a PTR record changing mid-run — C would keep serving the cached
name for the process lifetime, Go would re-resolve. No player can cause or
observe this through gameplay.
**Recommend: close as outside the game.**

## DP-1605 — ident lookup

The command surface has parity: `ident` is registered at LVL_IMPL-1 on both
sides (src/interpreter.c:511; pkg/session/commands.go:705) with C's toggle
messages (src/act.other.c:1180-1181; pkg/game/other_settings.go). The shipped
default is `ident = NO` (src/config.c:280), so the reference C server performs
no lookups and the census (loopback, no identd) cannot see any difference.

The visible corner, exactly:

1. An immortal at LVL_IMPL-1 types `ident` (turning lookups on) **and**
2. a player connects from a host running an identd (RFC 1413) that answers.

Then C's ident success path rewrites d->host to "user@host"
(src/ident.c:210-216), so the immortal's `users` Site column prints
`[someuser@127.000.000.001]` (src/act.informative.c:2093-2095) instead of the
bare quad, and the post-ident ban recheck (src/ident.c:233-234) matches
against the augmented string. Go never performs the lookup, so the column
stays `[127.000.000.001]`. While a lookup is pending, C also holds the new
descriptor's commands (waiting_for_ident, src/comm.c:584-597) — a login-timing
effect without output bytes. On ident failure/malformed/timeout, C leaves
d->host as the quad and logs the error (src/ident.c:197-214) — byte-identical
to Go's output.

In 2026, client hosts answering identd are rare (and the harness has none),
so this corner is production-only and immortal-only.
**Decision for Zach:** (a) close as outside the game (default off; identd rare;
the exact divergence is now recorded here), or (b) port the lookup so the
corner matches. Recommendation: (a) — porting a 1990s trust-the-client
protocol to restore a display nuance is not fidelity work the stopping rule
needs, and `users`' Site column is unproven oracle surface anyway (no depth
manifest covers it — a gap worth its own scenario either way).
