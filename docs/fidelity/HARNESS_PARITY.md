# Harness parity: what the census exercises, and what it can't see yet

The census (`make oracle-regression`) proves only what it runs. A condition
production has and the harness doesn't is a blind spot: a whole class of
divergences can sit there while the census stays green. On 2026-09-23 a single
Mudlet playtest found four such bugs in an afternoon (DP-1310 login room,
DP-1311 idle, DP-1312 quit to menu, DP-1314 logout loses most of the
character), all inside conditions this table marks as missing.

The rule this document serves: **a proof counts only for conditions the
harness reproduces.** Before calling a surface proven, find its row here. If a
change adds a production condition (a timer, a store, a transport), add a row.

Checkly's Node-to-Go rewrite hit the same failure: their test setup modelled 3
queues where production had 18, and the rewrite inherited the simplification
([write-up](https://www.checklyhq.com/blog/agentic-rewrite-nodejs-to-go/)).
Their conclusion is ours: every boundary is defined explicitly, never inferred.

| Production condition | Census today | What it hid, or could | Next |
|---|---|---|---|
| **Persistence (save and load)** | Go ran on an unreachable DB (`deadDBURL`, `DP_ALLOW_NO_DB`), so it never saved anything; C saves normally. Relogin scenarios now get a seeded SQLite store (#1584). | DP-1314: logout loses gold, bank, sex, alignment, skills, practices, preferences. | `<RELOGIN>` scenarios per persisted field; see DP-1314's round-trip list. |
| **Leaving and returning** | Every scenario was one unbroken connection. `<RELOGIN>` added (#1584). | DP-1312 (quit leaves no menu), DP-1310 (login room), the newbie kit left on the temple floor, a missing blank line after the password. | More vehicles: rent/cryo, REALLYQUIT, death then relogin, linkdead reconnect ("Reconnecting."). |
| **Server restart / crash recovery** | Reproducible: `<RESTART>` stops the engine behind the actor, starts it again on the same disposable data directory and ports, and logs the character back in; every worker isolation is proven by `make oracle-regression-isolation`. | DP-1310 (login room), the immortal login MOTD (fixed in `pkg/session/menu.go`), the two blocked blank lines of `save.relogin-login-transcript`. | `restart-*` vehicles: dropped object, gossip review, door state, mob state, reset-object duplication, durable-player boundary. |
| **Game time passing** | `DP_CLOCK` freezes pulses; scenarios pump them. C intercepts `~dpclock` before its input queue, so pumped hours are true idle (proved by `lifecycle-idle-void`). | DP-1311 (IDLE_TO_VOID 20 vs 8), DP-1316 (dusk events). | Tick-driven vehicles for regeneration, affects wearing off, zone resets, weather. |
| **Wall-clock timers** | Invisible: `ReapLinkdeadSessions` (60 s void, 5 min extract) and anything else on `time.Now` runs on real time, and scenarios finish in seconds. | DP-1311's 60-second reaper, which voids connected players. C has no wall-clock game timers. | Inventory every `time.Now`/`time.Since`/`time.Ticker` in `pkg/` that can affect player-visible state; each is R4 unless C has it. |
| **Real idle input** | The harness always sends something within seconds. | Anything keyed to silence on a live connection. | Covered by pumped clock plus the wall-clock inventory above. |
| **Several players** | Named peers (`[setup:*:name]`), audience blocks. | Good coverage for room/act audiences. | Group, follow, tell-to-sleeper vehicles as needed. |
| **Seeds** | The census runs seed 1 only; multi-seed proofs run in depth work. | RNG draw-order bugs that seed 1 happens to hide (R3). | A nightly census at a second seed. |
| **World state on disk** | Fresh copies of `lib/world` and `lib/text` per scenario; C gets a copied lib with its player files. | Boards, mail, houses, clans, rent files: whatever accumulates across boots (DP-1313, boards multiplying on the live server). | A long-running soak: many reset cycles and a restart on one data dir, then diff room contents. |
| **Static text trees** | The harness copies both `lib/world` and `lib/text`; login MOTD reads `lib/world/text`, commands read `lib/text` (DP-1299). | The two MOTDs can differ in production. | DP-1299 makes it one tree. |
| **Transport** | Raw telnet. IAC is consumed, so GMCP, MSSP, EOR and TLS are not compared (they have no C counterpart, so they are not oracle surfaces). | Out-of-band bytes leaking into the text stream would be missed only if they arrive as IAC. | Unit tests own GMCP (`pkg/session/gmcp_test.go`, the Mudlet package tests). |
| **Web client** | Not exercised: WebSocket sessions take a different input and prompt path. | Web-only divergences (prompts, paging, structured messages). | A WebSocket driver behind the same `Conn` interface. |
| **Concurrency** | Scenarios are sequential; the Go server is not built with `-race`. | Session-lifecycle races (quit/save/reconnect). | Run the census's Go server built with `-race`, at least nightly. |
| **Player-facing strings outside any scenario** | Only what scenarios happen to trigger. | Invented strings (`[ GHOST SHIP ]`, `autoexit`) and missing ones. | String census, both directions (C strings missing in Go; Go strings with no C source). |
| **Judge sensitivity** | A scenario is trusted once green. `quit.safe-logout` stayed green while the menu after it was missing. | Coincidence-green proofs. | Census coverage (which C strings any scenario makes C print) and mutation testing (break Go, see what the census misses). |

Last reviewed 2026-09-23.
