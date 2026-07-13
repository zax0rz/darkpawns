# Refactor Plan — Canonical game operations + result-DTO boundary (session vs game)

**Date:** 2026-07-13
**Author:** Claude, for Zach
**Motivates / closes:** most of the 46 oracle findings (DP-1083..DP-1130 / O1-O46). This is the
architectural spine the audits pointed at; individual fixes become domain migrations under it.
**Governing rule:** every migration is proven by an oracle red→green run (see
`[[darkpawns-oracle-proof-gate]]` — player-facing output is first-class fidelity).

---

## 1. The problem (why we can't just "fix bugs" or "just delegate")

The two audits (Tier A: DP-1083..1107; Tier B/C: DP-1108..1130) found three structural threads:

1. **Reimplemented-and-drifted (Bucket 1).** `pkg/session/*` command handlers reimplement logic that
   also lives in `pkg/game/*`, and the two copies have diverged from each other *and* from C
   (look, movement, channels, directed speech, doors, equipment, recite/zap).
2. **Delegates-but-game-buggy (Bucket 2).** Session correctly calls `pkg/game`, but the game
   operation itself diverges from C (get TAKE/container, give-to-mob, pour).
3. **Privilege / gate collapse.** The registry registers commands at `(level 0, pos 0)`, and the
   dispatcher only enforces a gate when the value is >0 — so admin commands leak to mortals and
   position gates vanish (O10/O24/O40/O42-O45).

Two facts make the naive fixes wrong:

- **You cannot blindly delegate session → today's game function.** GPT's audit showed the game
  copies are *also* incomplete vs C (game `look` misses room extra-descs & full directional look;
  both `doGenDoor` copies are door-only; game channels miss writing/soundproof gates). Delegation
  would trade bugs, not fix them.
- **The transport split blocks a single text-emitting core.** Session commands emit a structured
  `MsgState`/`ServerMessage` for the **WebSocket** client; the telnet listener's `formatState`
  renders that to text. `pkg/game` functions instead call `Player.SendMessage`, which reaches
  session only as an untyped `EventData{Type:"text"}`. So "just move the logic into game and send
  text" loses the WS structured payload the web client needs.

And a fourth, cross-cutting thread the fixes keep surfacing:

4. **Message-fidelity rot from a missing `act()`.** C's entire player-facing message system is
   `act(fmt, hide, ch, obj, vict, TO_*)`, which substitutes `$n/$N/$e/$s/$m/$p/$o/...` **per
   viewer** with visibility ("someone" when unseen) and emits to the right audience (TO_CHAR /
   TO_ROOM / TO_VICT / TO_NOTVICT). Go has no equivalent — `broadcastToRoom` is a dumb broadcaster
   and every message is hand-substituted inline. That is exactly how `zap` and `recite` leaked
   literal `$n`/`$m` to the room (DP-1110 fix / DP-1130), how `whisper` hard-codes ANSI (O31), and a
   large share of the comm/movement wording drift (O29/O30/O25). **A faithful `act()` is a
   foundation, not a detail.**

## 2. Target architecture

Four layers. The stable boundary is a **game-owned result** (structured facts + messages), **not
game-owned text**.

```
             ┌─────────────────────────────────────────────┐
   player →  │ dispatcher (one authoritative command table) │  ← gates: level/position/wait
             └───────────────┬─────────────────────────────┘
                             ▼
             ┌─────────────────────────────────────────────┐
             │ canonical game operation  (pkg/game)         │  validates C rules,
             │   DoLook / DoDrop / DoMove / DoTell / ...     │  mutates state,
             │   returns a Result (structured + Messages)    │  returns Result — NO text
             └───────────────┬─────────────────────────────┘
                 ┌───────────┴───────────┬───────────────────┐
                 ▼                       ▼                   ▼
        telnet renderer         WS/session renderer    mobprog/specproc
        act() → C-faithful      MsgState + typed        act() text adapter
        text                    events (+ act text)     (same Result, no 2nd rules impl)
```

### Layer 0 — the `act()` primitive (foundational; build first)
A faithful port of C `act()`: authored strings use C's `$` codes; a shared renderer resolves them
**per recipient** with visibility, position, and pronoun/sex rules, and delivers to the correct
audience set. Every player-facing message flows through it. Kills the whole message-fidelity class
(no more inline hand-substitution → no more `$n` leaks or wording drift). Pronoun tables key off
Go's actual `Sex` encoding (0=male/1=female/2=neutral, `pkg/game/player.go:71`) — verified during
the zap fix; note this is *not* C's raw constant order, so port the mapping, not the literals.

### Layer 1 — canonical operations returning a Result
One game-owned operation per command *contract*. It owns C's validation, mutation, and message
choice, and returns a `Result` — never rendered text. Shape:

```go
// Illustrative — exact types per domain.
type Result struct {
    Ok        bool
    Messages  []ActMessage   // {Format, Hide, Actor, Obj, Vict, Audience} — rendered by act()
    Structured any           // domain payload for WS (RoomView, ObjView, EquipChange, ...)
    Events    []SemanticEvent// typed events WS clients rely on (e.g. "drop", "position")
}
```

### Layer 2 — transport renderers (thin, no rules)
- **Telnet:** run `Result.Messages` through `act()` → text. Done.
- **WS/session:** emit `MsgState`/`ServerMessage` from `Result.Structured` + `Result.Events`; render
  any text portions via `act()` too. Must remain schema-compatible with today's web client.
- **Mobprog/specproc:** consume the same `Result` via an `act()` text adapter — never a second copy
  of the rules. (This is how we retire the duplicate implementations for good.)

### Layer 3 — one authoritative command table + dispatcher
A single source-of-truth command table (name, aliases, **min level**, **min position**, wait, flags)
generated/verified against C `src/interpreter.c`. The dispatcher enforces it uniformly (fixing the
"gate only when >0" hole). This mechanically closes the privilege/position cluster
(O10/O24/O40/O42-O45/DP-1094) and gives C-faithful `Huh?!?` for over-level commands (already proven
for summon/hcontrol in PR #295).

## 3. Migration strategy — strangler, one domain at a time, oracle-proven

Do **not** big-bang. `main` stays shippable throughout. For each domain:

1. Build/extend the canonical op(s) to full C fidelity (read C; the current game copy is a starting
   point, not the target).
2. Add the transport renderers (telnet via act(); WS via Result→MsgState).
3. Route the session command to the canonical op; **delete** the session reimplementation.
4. Point mobprog/specproc callers at the same op.
5. **Prove:** oracle red→green scenario(s) for the domain's findings; unit tests for state
   invariants (e.g. instance-not-prototype). Close the domain's Linear issues.

### Foundations first (unlock everything)
- **F0a — `act()` primitive.** Highest leverage; every domain depends on it. Ship with tests +
  retro-fix the known leaks (recite O46; audit whisper O31).
- **F0b — authoritative command table + dispatcher gates.** Mechanical, high-value, independent of
  act(). Closes the privilege/position cluster in one sweep. Pairs naturally with a generated
  "position/level matrix" test against C.

### First full domain — **Observation** (do first after foundations)
`look` / `look <target>` / `look <dir>` / `examine` / `exits` / `diagnose`. Why first:
- Read-only → lowest risk (no state mutation to get wrong).
- `pkg/game/look.go` already exists and is closest to canonical — least greenfield.
- Densest finding cluster: DP-1083, DP-1086, DP-1089, DP-1090, DP-1107, O23, O35 (+ resolves the
  O1 telnet-vnum-leak by design: the `RoomView` result carries the vnum, and only the *immortal*
  telnet renderer prints it — the WS payload can always include it).
- Forces us to solve the "Result feeds both WS structured + telnet text" problem on the safest
  surface, establishing the pattern for everything else.

### Then, in rough dependency order
| Domain | Canonical op(s) | Closes (representative) |
| --- | --- | --- |
| Consumables / instance values | DoUse/eat/drink/pour (instance GetValue/SetValue) | O21, O22, O41✓, O46; the prototype-mutation class |
| Object / inventory | DoGet/DoDrop/DoPut/inventory view | O7, O8, O12, O17, O18, DP-1091/1092/1102/1105, O2 |
| Equipment | one atomic explicit-slot op | O19 |
| Movement | one DoMove transaction (charm/mount/follower/hide/room-scripts) | O25, O26, O27, O28 |
| Communication | one eligibility/delivery core over act() | O29, O30, O31, O32, O44 |
| Character view | score/where/who over a viewer-aware view | O9/O14 (score), O11/O13 (where/who) |
| Skill / spell | **needs a product decision first** (see §5) | O36, O37, O38, O39 |

## 4. Sequencing (the roadmap)

1. **F0a act() primitive** + retro-fix recite (O46) / whisper (O31) as its first customers.
2. **F0b command table + dispatcher gates** → close privilege/position cluster (O10/O24/O40/O45;
   O42-44 already patched in PR #295 but should re-land on the table).
3. **Observation domain** — first full Result+dual-renderer migration; proves the pattern.
4. **Consumables** — small, high-value, kills the prototype-mutation class; act() already in place.
5. **Object/inventory**, then **Equipment**, then **Movement**, then **Communication**.
6. **Character view** (score/where/who).
7. **Skill/spell** — after the §5 decision.

Each step is its own PR(s) + oracle proof; each closes a named batch of DP issues.

## 5. Risks & decision gates

- **WS schema compatibility (hard constraint).** The Result→WS renderer must keep producing the
  `MsgState`/`ServerMessage` shapes the web client already consumes (add fields, don't break them).
  Capture the current schema as a golden before touching it.
- **Skill-system is a product decision, not a port (gate).** C's `practice`/guild model vs Go's
  redesigned skill system genuinely differ (O36/O37/O38). Decide *intended* behavior before
  "fixing," else we thrash. Everything else is faithful porting; this one needs Zach's product call.
- **The three-caller invariant.** Player command, mobprog, and spec-proc must all reach the *same*
  canonical op. Auditing for direct-to-mutation calls is part of each domain migration.
- **Don't regress the oracle harness's own gains.** Keep scenarios green (or expectedly-red until a
  fix lands); wire new domain scenarios as each migration proves out.
- **Scope honesty.** This is a multi-week/-month arc. The strangler keeps it incremental and always
  shippable; the value lands per-domain, not at the end.

## 6. How this plugs into the workflow

- Each domain becomes a worker brief (audit-derived, C-cited) under Claude review; foundations
  (act(), command table) may warrant frontier judgment.
- **Definition of done per domain = code + unit tests (state invariants) + oracle red→green
  (behavior *and* wording) + closed Linear issues.** A green build is never sign-off on its own.
- The 46 findings stop being a flat backlog and become ~9 domain migrations, each with a clean
  proof.

See also `[[darkpawns-c-oracle]]` (harness + findings), `[[darkpawns-oracle-proof-gate]]` (the
proof requirement), and the two audit docs under `docs/research/drafts/2026-07-13-session-vs-game-*`.
