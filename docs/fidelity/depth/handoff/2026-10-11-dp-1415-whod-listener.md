# DP-1415 port: the WHOD daemon listener (C whod.c)

Base `origin/main` `eae1c63d5`, worktree `/home/zach/dp-mudlog-1415`, branch
`deepseek/mudlog-1415`. Tier: stop (new listener and transport surface).
Zach merges. Ruling (Linear DP-1415): port it, prod port 7779, plaintext.

## The C site

`src/whod.c` opens a second port (game port + 1) that answers a WHO request and
closes. `init_whod(port)` is called at boot with the game port
(`src/comm.c:274`), `close_whod()` at shutdown (`:290`), and `whod_loop()` from
the game loop (`:555`). `LOG(msg)` is `mudlog(msg, BRF, LVL_GOD, TRUE)`
(`src/whod.c:39`); the four producers are:

| Site | Payload | Point |
|---|---|---|
| `src/whod.c:250` | `WHOD port opened.` | `init_whod`, **before** the bind |
| `src/whod.c:267` | `WHOD port closed.` | `close_whod`, only when state != `WHOD_CLOSED` |
| `src/whod.c:296` | `WHOD port opened.` | `whod_loop`'s `WHOD_OPENING` reopen arm |
| `src/whod.c:335` | `WHO request from <host> served.` | before any byte is written |

The response is `SHORT_GREETINGS` (`src/config.c:221-231`) then the header
`"\n\rA list of active players on Dark Pawns:\n\r\n\r"` (`:341`), one line per
character, then `"\n\rPlayers : %d     Gods : %d\n\r\n\r"` (`:422`). Per
character C keeps only non-NPCs with `ch->desc` (`DISPLAY_LINKDEAD` is 0, `:45`)
and, without `SHOW_INVIS`, only non-`IS_INVIS` characters (`:345`), where
`IS_INVIS` is `AFF_INVISIBLE || GET_INVIS_LEV || ROOM_NO_WHO_ROOM` (`:33`).

## The recovered formatter

`BuildWhoList` was deleted as unreachable by #1904 because the listener was
never wired (`git show 4176ed383^:pkg/game/whod.go`). It was **a starting
point, not proof**: it was missing C's list header entirely and had never been
oracle-tested. It is restored with the header, and every branch was re-checked
against `src/whod.c:341-425` — the `IS_SET(SHOW_LEVEL|SHOW_CLASS)` gate, the
god-token ladder (`*IMP*`/`GRGOD`/`HIGOD`/`_LEG_`/` GOD `/`TITAN`/` IMM `), the
`_GOD_` fallback when only one of level/class is set, `%2d` + space + class
abbrev, title + ` (AFK)`, ` [%s]` / ` [ ** Unknown ** ]`, and the counters.
`SHORT_GREETINGS` was extracted mechanically from `src/config.c` (11 literals,
509 bytes) rather than transcribed.

## The port

| File | Role |
|---|---|
| `pkg/game/whod.go` | `WhodGreeting`, `WhodHeader`, `BuildWhoList`, `classAbbrev`; `Whod.mode` moved behind a mutex with `GetMode`/`SetMode`; `World.WhodInvisible` ports `IS_INVIS` |
| `pkg/session/whod_listener.go` | the listener, the entry builder, the supervisor, `StartWhodListener`/`StopWhodListener`/`WhodListenerAddr` |
| `cmd/server/main.go` | `-whod-port` (default `-telnet-port` + 1, `0` disables) and the shutdown call |

* **Flag:** C's rule is game port + 1, so the `-1` default derives from
  `-telnet-port` (7777 → 7778) and a stock self-host matches C. **Prod sets
  7779**, because 7778 is `-telnet-tls-port`.
* **Order:** the open producer is emitted **before** the bind, as C does.
* **Visibility:** `World.WhodInvisible` (`IsAffected(affInvisible) ||
  GetInvisLevel() != 0 || RoomNoWho`); linkless players are C's `desc == NULL`
  and are dropped; `SHOW_SITE` uses `Session.MudHost()`, C's padded descriptor
  host, while the served line uses C's own reverse-resolve-or-unpadded-quad.
* **Order of the list:** C walks `character_list`, which prepends each new
  character. Go has no such list, so the entries are ordered by
  `connectionNumber` descending — the same newest-first order the `users`
  command models.

## Concurrency difference (approved)

**C serves one connection at a time.** `whod_loop` accepts, writes the
greeting and the list, holds the socket for `WHOD_DELAY_TIME` (3 seconds,
`src/whod.c:42,427-441`), closes it, and only then accepts the next request; a
request that arrives during the delay waits. **This port serves each accepted
connection on its own goroutine**, so overlapping requests are answered
concurrently rather than queued. The bytes each client receives are identical —
greeting, header, list, footer, same order, same hold before close — but the
concurrency model is not C's. This shape was **approved for DP-1415** (Claude
with Zach, 2026-10-10: "Go serves each request on its own goroutine"); the
approval is recorded here so the divergence is auditable rather than silent.
No player-facing byte depends on it, and C's single-connection queueing is not
observable in a one-request oracle transcript.

The on/off lifecycle is C's: `whod off` closes the port with one close
producer, `whod on` reopens it with the OPENING arm's producer. C reconciles
those bits every game loop with a 100µs select (`src/whod.c:303-304`); the
supervisor ticks every 100ms. Only the reconcile latency differs.

## Lock audit

`serve` calls `MudLog` before it takes anything; `entries()` then holds the
manager read lock while building the list, and `WhodInvisible` nests the world
read lock inside it (the same order the `users` command already uses). **No
`MudLog` call happens under the manager, world, player or zone lock**: the
reconcile path unlocks before logging, and the open producer runs before the
listener exists.

## Proof

Unit (real parser, real dispatch, real socket):

* `TestWhodGreetingMatchesConfigBytes` — 509 bytes, the escape-heavy banner
  lines literal.
* `TestWhodBuildWhoListDefaultMode`, `...GodTokenLadder` (the full ladder, the
  `_GOD_` fallback, `SHOW_WIZLEVEL` suppression), `...InvisibleAndSite`,
  `...AfkMarker`.
* `TestWhodListenerServesExactCBytes` — a real TCP connect returns greeting +
  header + `[ 3 Wa] Albe the Test\r\n` + footer byte for byte.
* `TestWhodListenerHidesInvisibleAndLinkless` — wizinvis hidden unless
  `SHOW_INVIS`, linkless dropped, counters follow the shown set.
* `TestWhodListenerProducers` — the three producers at BRF/`LVL_GOD`/file TRUE,
  a level-33 negative, and one close line despite a second stop.
* `TestWhodListenerOffClosesAndOnReopens` — `off` closes and refuses, `on`
  reopens and serves.
* `TestWhodRequestHostBranches` — resolved name, unpadded quad, nil address.

R5h controls, six cases, each 1 → 0 → 1:
`docs/fidelity/depth/handoff/2026-10-11-dp-1415-whod-controls.py`
(`greeting-bytes`, `list-header`, `invisible-gate`, `linkless-filter`,
`served-producer`, `close-producer`).

**Oracle vehicle: not yet claimed.** The issue asks for one "if the harness can
connect to the C oracle's game port + 1" and compare the raw WHOD bytes; that
has not been demonstrated, so these rows are `unit-green`, and the vehicle is
the open item for the census step rather than an assumed pass.

## Inventory

* `2026-10-06-dp-1371-mudlog-sites.tsv` `src/whod.c:250`, `:267`, `:296`,
  `:335`: `missing` → `implemented-boundary`, owner
  `pkg/session/whod_listener.go`.
* `mudlog.tsv`: `mudlog.whod-port-open`, `mudlog.whod-port-close`,
  `mudlog.whod-request-served` added; the five `do_whod` rows' notes no longer
  say the daemon is unported.
* `mudlog.unported-sites` stays `blocked` until PR 4.

## Ops

`-whod-port 7779` on prod (7778 is the TLS telnet port); the port must be opened
in the hardening plan (B3) and is listed in `DEPLOYMENT.md`. Plaintext, exactly
as C, and `SHOW_SITE` off by default so hosts are not exposed.
