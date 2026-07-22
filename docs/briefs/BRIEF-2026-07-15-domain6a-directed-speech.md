# BRIEF — Domain 6a: Directed speech (say / tell / reply / whisper / ask) — O29 + O31

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/domain-directed-speech` off current `main`.
**Findings:** DP-1113 / O29 (say/tell/reply), DP-1124 / O31 (whisper/ask).
**Part 1 of 2** for the Communication domain; part B (channels/page/socials, O30/O44/O32) is a
separate brief. **Build the one eligibility/delivery core here** — say/tell/whisper/ask share the
same gates (NOSHOUT / soundproof / notell / visibility / self) so this PR establishes the core that
part B extends to channels.
**Method rules:** read `src/act.comm.c` in the C oracle clone directly. Gated by an **oracle
red→green run** — a green build is NOT sign-off. This domain's happy paths already largely match, so
**most of the fidelity is in gates that unit tests must lock** (see §3).

---

## 1. Oracle-PROVEN RED (`scenarios/communication.txt`, verified 2026-07-15)

Actor + co-located `observer`, both hometown-K at 8162. Run:
```
DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" \
  go run ./cmd/dp-oracle-diff --scenario communication
```

| # | probe | audience | C oracle | Go port | finding |
|---|---|---|---|---|---|
| 1 | `ask Comobserv <msg>` | observer (victim) | `Comactor asks you, '<msg>'` | `Comactor asks, '<msg>'` | O31 — victim form drops the direct-address "you" |
| 2 | `whisper Comactor <msg>` (self) | actor | `You can't get your mouth close enough to your ear...` | `No-one by that name here.` | O31 — self-target not handled |
| 3 | `tell Nobody <msg>` | actor | `No-one by that name here.` | `There is no such player online.` | O29 — wrong not-found wording (`NOPERSON`) |

**Already MATCHES (do NOT "fix"):** `say`, `tell`, `reply` (no target), `whisper`/`ask` happy path,
and self-`tell`. The simple co-located wording is already faithful.

## 2. Root cause

Directed speech is duplicated and drifted: session hand-rolls in `pkg/session/act_comm.go`
(whisper/ask, ~69-175) and `pkg/session/comm_cmds.go` (say/tell/reply, ~323-375, "simplified"
recipient handling, online-session-only), with parallel game copies (`pkg/game/act_comm.go`,
`comm_say.go`). **Consolidate to one game-owned eligibility/delivery core over `Act()`** (canonical
since F0a); session delegates. Boundary = per-recipient act() messages (this is a text domain, not a
structured-DTO one — same as recite/whisper in F0a).

## 3. Faithful C reference (act.comm.c) + the gates unit tests must lock

### 3a. `do_spec_comm` — whisper (SCMD_WHISPER) + ask (SCMD_ASK), act.comm.c:976, POS_RESTING, lvl 0
Shared handler. Verbs: whisper → sing `"whisper to"`, plur `"whispers to"`, others
`"$n whispers something to $N."`; ask → sing `"ask"`, plur `"asks"`, others `"$n asks $N a question."`
- `PLR_NOSHOUT` → `"Sorry, you cannot do that."`
- `half_chop` → target + message; either empty → `"Whom do you want to <sing>.. and what??"`
- target not visible in room → `NOPERSON` (`"No-one by that name here."`)
- **self (`vict == ch`) → `"You can't get your mouth close enough to your ear..."`** (fixes RED #2)
- victim (TO_VICT): **`"$n <plur> you, '<msg>'"`** with `delete_ansi_controls` applied to the message
  (fixes RED #1 — the "you" is mandatory; strip ANSI from user input, don't hard-code color)
- actor: NOREPEAT → `OK`; else `"You <sing> <victName>, '<msg>'"`
- room (TO_NOTVICT): the `others` string above

### 3b. `do_say` — act.comm.c:759, POS_RESTING
- `GET_WIS==0 || GET_INT==0` → refused (read the exact string in-source)
- drunk → `speak_drunk()` mangles the message
- punctuation switch on last char: `?` → `"$n asks, '%s'"`, `!` → exclaims, default → `"$n says, '%s'"`
  (actor `"You say, '%s'"`); NOREPEAT → `OK`

### 3c. `do_tell` — act.comm.c:901 / `do_reply` — :934
Gates (each its own message — read them): sender `PLR_NOSHOUT`; `ROOM_SOUNDPROOF` (sender side);
self-tell (`"You try to tell yourself something."`); target visibility; linkless / writing target;
recipient `PRF_NOTELL` or `ROOM_SOUNDPROOF`. Success: victim `"$n tells you, '<msg>'"`, actor
`"You tell $N, '<msg>'"`, and set the recipient's reply-target. `reply` sends to the last teller
(none → `"You have no-one to reply to!"` — verify exact string).

**Unit tests (the real gate for this PR — none of 3b/3c's gates are reachable by the co-located
scenario):** mute (INT/WIS 0), drunk mangling, NOREPEAT→OK, self-tell, NOTELL both directions,
SOUNDPROOF both directions, not-visible target, whisper/ask self + not-found + ansi-strip + victim
"you" form, empty-arg prompts. Assert the exact C strings.

## 4. Session adoption
Replace the hand-rolled bodies in `act_comm.go`/`comm_cmds.go` with delegations to the new game ops
(`DoSay`/`DoTell`/`DoReply`/`DoSpecComm(ch,arg,subcmd)`); keep `markDirty` where WS vars change.
Delete now-dead helpers if no other callers (check across session). Preserve delivery to **all**
in-room/online recipients via `Act()`/room broadcast, not just "online sessions."

## 5. Acceptance gate
1. **Oracle red→green:** `--scenario communication` → clean (Claude re-runs; run it yourself first).
2. Observer/victim broadcasts diffed — must match C.
3. **Unit tests** for every §3 gate with exact C strings (this is where most of O29/O31 lives).
4. Instance-safe; no WS schema break (`protocol_schema_test.go` golden green).
5. `make check-fmt vet` + `go test ./...` green.

## 6. Gotchas
- **Thin oracle surface by design.** Both newbies are healthy/unmuted/co-located, so only 3 lines
  diverge; the gates are proven by unit tests, not the oracle. Don't conclude "3 lines = small job."
- **`delete_ansi_controls`** strips ANSI from *user-supplied* message text (whisper/ask) — port it;
  do NOT hard-code color into the messages (that was part of O31).
- **ANSI blind spot:** the normalizer strips color, so any color parity is unit-test-only.
- Don't touch channels/page/socials — that's brief 6b.
