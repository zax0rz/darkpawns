# BRIEF — Domain 6b: Channels / socials / page — O30 + O32 + O44

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/domain-channels-socials` off `main`.
**Findings:** DP-1114 / O30 (channels), DP-1125 / O32 (socials), DP-1119 / O44 (page).
**Part 2 of 2** of Communication. **Land 6a (directed speech) first** — reuse the eligibility core
it builds (NOSHOUT / soundproof / off-channel / visibility) for channels; don't re-invent it.
**Method rules:** read `src/act.comm.c` (`do_gen_comm`, `do_page`) + `src/act.social.c` /
`src/interpreter.c` (socials) directly. Gated by an **oracle red→green run**.

---

## 1. Oracle-PROVEN RED (`scenarios/communication-channels.txt`, verified 2026-07-15)

Actor + co-located `observer` (same room = same zone 80), both hometown-K level-1 at 8162. Run:
```
DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" \
  go run ./cmd/dp-oracle-diff --scenario communication-channels
```

| # | probe | C oracle | Go port | finding |
|---|---|---|---|---|
| 1 | `gossip <msg>` (level-1) | `You must be at least level 2 before you can gossip.` | `You gossip, '<msg>'` (+ observer receives it) | O30 — no min-level gate |
| 2 | `shout <msg>` (level-1) | `You must be at least level 2 before you can shout.` | `You shout, '<msg>'` (+ observer receives it) | O30 — no min-level gate |

**Already MATCHES:** `smile` / `wave <target>` (socials) and `page` happy path. So the socials-data
wording and page's level gate are fine; the O32/O44 fidelity is in *gates* (below), unit-tested.

> Note: because C blocks level-1 channels, the channel **wording** (gossips/shouts format,
> recipient fan-out) isn't oracle-diffable with a newbie. Verify that wording with unit tests, or a
> follow-up scenario using a level-2+ fixture char.

## 2. Root cause
Channels: session `pkg/session/comm_cmds.go` + game `pkg/game/comm_channel.go` duplicated, both miss
gates (min-level, off-channel recipient skip, soundproof, shout zone-limit). Socials: attempted only
after registry lookup fails, calling `game.DoAction` directly → **bypasses the command-table
position/min-level gates** (commands.go social fallthrough). Page: level gate already fixed by the
F0b command table; residual is the multiword-message parse (DP-1119).

## 3. Faithful C reference + gates (unit-tested unless noted)

### 3a. `do_gen_comm` — channels (gossip/auction/…), act.comm.c:1146
- `PLR_NOSHOUT` → blocked; `ROOM_SOUNDPROOF` (sender) → blocked.
- **min-level gate (:1217):** `"You must be at least level %d before you can %s."` — **this is RED
  #1/#2; oracle-proven.** Per-channel min level (gossip/shout = 2).
- recipients: skip `PRF_FLAGGED(off-channel)`, writing, `ROOM_SOUNDPROOF`. **Shout (SCMD_SHOUT,
  :1286) is zone-limited**; gossip is global.
- wording: actor `"You gossip, '%s'"`; others `"$n gossips, '%s'"` (per channel verb).

### 3b. Socials — `do_action`/`do_social` (act.social.c) + table (interpreter.c:390-405)
- Socials are **command-table entries with an actor position** (e.g. `comfort` POS_RESTING, `dance`
  POS_STANDING) and a social min-level; `do_action` also checks the **victim** position
  (act.social.c:102-149). Go must route socials **through the same command-gate table** (F0b) so
  actor position/level are enforced *before* dispatch, then check victim position — not call
  `DoAction` directly. Social message text itself already matches (RED shows smile/wave clean).

### 3c. `do_page` — act.comm.c:1107
- Immortal-only (already gated by the F0b table — page happy path matches in the oracle). Residual
  (DP-1119): the handler must parse **one target + the remaining multiword message** (Go truncated
  to the last word) and support `all` only above GOD. Unit-test with an immortal fixture.

## 4. Session adoption
Channels → delegate to a game `DoChannel(ch, arg, subcmd)` that reuses the 6a eligibility core +
adds min-level/off-channel/zone. Socials → route through the command-gate table (position/level)
then `DoAction`; stop the post-lookup direct-call bypass. Page → fix multiword parse. Delete dead
session helpers.

## 5. Acceptance gate
1. **Oracle red→green:** `--scenario communication-channels` → clean (the two level-gate lines).
2. **Unit tests** (the bulk of O30/O32/O44): channel min-level, NOSHOUT, soundproof, off-channel
   recipient skip, shout zone-limit; social actor-position + victim-position + min-level gating; page
   immortal-only + multiword parse. Exact C strings.
3. Instance-safe; no WS schema break; `make check-fmt vet` + `go test ./...` green.

## 6. Gotchas
- **Reuse 6a's eligibility core** — don't fork a second NOSHOUT/soundproof/visibility impl.
- **Channel wording isn't oracle-diffable with a newbie** (C blocks level-1). Lock it in unit tests
  or add a level-2 fixture char to the harness later.
- **Socials must go through the gate table**, not a direct `DoAction` — that's the O32 bug.
- **ANSI blind spot:** color parity is unit-test-only.
