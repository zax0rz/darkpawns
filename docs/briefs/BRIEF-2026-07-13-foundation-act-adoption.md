# BRIEF 2026-07-13 — Foundation 1: adopt the canonical `act()` messaging primitive

**Executor:** ChatGPT / frontier. **Read the refactor plan first for the big picture:**
`docs/research/drafts/2026-07-13-session-game-refactor-plan.md` — this brief is **Foundation F0a** in
that plan (the primitive everything else depends on). Claude reviews; **every behavior/wording change
is proven by an oracle red→green run** (`docs/research/drafts/2026-07-13-c-oracle-differential-testing.md`;
player-facing output is first-class fidelity, equal to the mechanical effect).
**Branch:** `refactor/act-adoption` off current `main`. **One PR** (or a tight series if large).

---

## The headline: `act()` ALREADY EXISTS and is faithful. Do NOT rebuild it.

`pkg/game/act.go` has a complete, correct port of C's `act()`:
- **`Act(world, hideInvisible, ch, vict, obj, victObj, format, arg2, actType)`** (act.go:456) — full
  audience handling (`ToChar`/`ToVict`/`ToRoom`/`ToNotVict`, `ToSleep`), room iteration, `hideInvisible`
  + `canSee` visibility ("someone"/"something" when unseen), capitalization.
- **`performAct`** substitutes the full C `$`-code set (n/N/m/M/s/S/e/E/o/O/p/P/a/A/T/F/$ — a superset of
  C comm.c:2418+), verified against `~/.openclaw/workspace/darkpawns-c-oracle/src/comm.c`.
- **Pronoun helpers** `hmhr`/`hshr`/`hssh` map `0→male, 1→female, 2→neutral` — **correct for Go's `Sex`
  encoding** (`pkg/game/player.go:71`). NOTE: the comment at act.go:47 wrongly says this "matches C
  SEX_* constants" — it does not (C is neutral=0/male=1/female=2; see DP-1104/O15). **Fix the comment,
  keep the code.**

**The problem is adoption, not implementation.** The session command layer and a pile of *competing*
helpers bypass `Act()` and hand-substitute names inline — which is exactly how `zap` (DP-1110) and
`recite` (DP-1130) leaked literal `$n`/`$m` to the room, how `whisper` hard-codes ANSI (O31), and a
chunk of comm/movement wording drifts. Your job is to make `Act()` **the one path** and retire the rest.

## Inventory to consolidate (the competing/duplicate messaging)

Canonical (keep, extend): `pkg/game/act.go` `Act()` + `SendToChar`/wrappers.
Retire or reduce to thin wrappers over `Act()`:
- `pkg/game/item_helpers.go:414 actToChar`, `:449 actToRoom`, `:517 actToRoomExclude`
- `pkg/game/other_helpers.go:76 actToRoom`
- `pkg/game/world.go:735 actToRoomMob`
Dumb broadcasters / inline substitution (the sprawl — do NOT convert wholesale here, see scope):
- `pkg/session/*` `broadcastToRoom` (movement_cmds.go:374) — no `$`-substitution; callers pre-substitute
  by hand (that's the bug surface).
- Inline `fmt.Sprintf("%s ...", s.player.Name, ...)` message sites across `pkg/session/*`.
- The ad-hoc `strings.ReplaceAll(msg, "$N", ...)` in `pkg/session/consider.go:275`.

## Scope — bounded (strangler; do NOT boil the ocean)

This is a **foundation**, not the whole migration. Deliver:

1. **Verify + document `Act()` as canonical.** Fix the act.go:47 comment. Add a short doc/comment block
   declaring `Act()` the single messaging primitive and how session/telnet consume it (below). Add unit
   tests that pin each `$`-code, each audience, visibility ("someone" when unseen), and pronouns for all
   three sexes (guards against regression + the O15 trap).
2. **Consolidate the duplicate `pkg/game` helpers** (`actToChar`/`actToRoom`/`actToRoomExclude`/
   `actToRoomMob`) into thin wrappers over `Act()` (or delete + migrate their callers). One code path.
3. **Migrate the two known-broken customers as the proof-of-pattern:** `recite` (DP-1130/O46) and
   `whisper` (O31). Route their messages through `Act()` with correct `$`-codes and C-faithful strings:
   - recite → actor `"You recite $p which dissolves."`, room `"$n recites $p."` (C spell_parser.c:684/688).
   - whisper → resolve any visible room character (not just players), reject self, honor norepeat,
     distinct `$n`/`$N` actor/victim/observer messages (C act.comm.c:976-1018); no hard-coded ANSI.
4. **Establish the convention + a guard.** A CONTRIBUTING note (or lint/vet check/test) that new
   player-facing messages MUST go through `Act()` — so the sprawl stops growing while later domain
   migrations (observation, movement, comm, …) convert the rest incrementally.

**Out of scope (later domain migrations own these):** converting every `broadcastToRoom`/Sprintf site;
the Result-DTO/structured-WS work; anything touching command gates (that's Foundation F0b, next brief).

## Transport coexistence (know this before wiring)

`Act()` produces **text** and delivers via `Actor.SendMessage`, which reaches the session as an
`EventData{Type:"text"}` event → telnet prints it; the WS client shows it as text. **That is correct
for the text half.** The *structured* `MsgState` payloads (room/objects/vitals for the web client) are a
SEPARATE concern handled by the Result-DTO in later domain migrations — do NOT try to fold structured
state into `Act()`. `Act()` is the message primitive; structured state rides alongside it. Keep the WS
`MsgState` schema untouched here.

## Oracle proof (required — this is the gate)

Add/extend a scenario (e.g. `cmd/dp-oracle-diff/scenarios/act-messages.txt`) that exercises recite and
whisper from a mortal, with a second character in the room so the TO_ROOM path is observed:
- **Red (pre-fix):** capture the current divergence — room sees literal `$n`, recite says "reads",
  whisper mis-targets/hard-codes color.
- **Green (post-fix):** the actor, victim, and observer lines match C (normalized). Paste both reports.
- Because room-observer text is the crux, ALSO add Go unit tests that drain a **room observer's** send
  channel (pattern from the zap fix `TestZap_BroadcastToRoom_...`) and assert the substituted output +
  the absence of any literal `$` token. The observer test + the oracle red→green together are sign-off;
  a green build alone is not.

## ⛔ Guardrails
- Do NOT rebuild `act()` — extend/adopt the existing one. Match C exactly; read `src/*.c`, don't invent.
- Do NOT modify the C oracle clone / `DP_SEED`.
- Do NOT change command gates/levels (Foundation F0b) or start structured-WS work (later domains).
- Keep the WS `MsgState` schema unchanged.
- Build gate green: `go build ./... && go vet ./... && go test ./...` + lint + gofumpt.

## Success criteria (PR shows ALL)
1. `Act()` documented as canonical; act.go:47 comment corrected; `$`-code/audience/visibility/pronoun
   unit tests present and passing.
2. Duplicate `pkg/game` act helpers consolidated into thin wrappers over `Act()` (or removed + callers
   migrated) — one messaging code path.
3. recite (DP-1130) and whisper (O31) routed through `Act()`; strings verified against C.
4. **Oracle red→green** for recite + whisper pasted inline; **room-observer unit tests** asserting
   substituted output and no literal `$` tokens.
5. Convention/guard in place so new messages must use `Act()`.
6. Build gate green; WS `MsgState` schema unchanged.

## Wrap-up
Commit; push; open PR with the red→green oracle reports + observer tests inline; STOP — Claude reviews
against `origin/main` + `src/*.c` (esp. pronoun correctness, C string parity, and that no literal `$`
leaks remain) and merges. **Next foundation brief (F0b):** one authoritative command table + dispatcher
gates (closes the privilege/position cluster O10/O24/O40/O42-45). **Then** the Observation domain is the
first full Result-DTO migration.
