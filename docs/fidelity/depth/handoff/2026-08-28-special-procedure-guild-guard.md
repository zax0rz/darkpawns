# Depth-fidelity handoff — guild_guard special procedure

Date: 2026-08-28
Branch: `glm/spec-guild-guard`
Starting main: `2053de355` (paladin handoff)

## Queue position and inventory

The session refreshed `main`, ran `make fidelity-depth`, read
`docs/fidelity/DEPTH_TESTING.md`, and read the newest prior handoff. The
source-and-registration ordered next active procedure was `guild_guard`,
defined at `src/spec_procs.c:569-608`. Its active registrations are mob vnums
8017, 8016, 8018, 8019, 8025, and 8083 from `src/spec_assign.c:271-276,306`;
the earlier 8014 registration is overridden later by the ordinary `guild`
assignment and is not an active guild_guard dispatch.

Before this slice the frontier was 558 total: 545 proven/delegated, 2
blocked, and 11 excluded. This slice adds five rows. The resulting frontier
is 563 total: 550 proven/delegated, 2 blocked, and 11 excluded; actionable
completion is 550/552 (99.6%).

## C path and branch claims

The player command path resolves aliases to a canonical command index in
`src/interpreter.c:889-940`, then invokes mob specials from
`src/interpreter.c:1380-1456`. `SPECIAL(guild_guard)` first intercepts the
exact canonical `flee`, `escape`, and `retreat` commands. It then accepts only
movement commands, bypasses movement interception for a blind guard, exempts
immortals, remort-only classes, and C's player-path `HUNTING` result, and
compares the player's class, room vnum, and canonical direction against the
`guild_info` table in `src/class.c:275-290`. Non-movement or blind fighting
fallback calls `fighter(guard, guard, 0, NULL)`.

The Go procedure now uses the shared `Act` audience/pronoun path, maps the
registered movement aliases to C direction indexes, uses the C guild table,
implements the C immortal/remort gates, and delegates autonomous fallback to
the guard as fighter actor. It does not invent a player hunting state: the C
`HUNTING(ch)` macro returns NULL for a non-NPC player on this command path.

## Evidence

Focused RED/GREEN coverage is in
`pkg/game/spec_guild_guard_test.go`:

- `TestSpecGuildGuard_GuildInfoMatchesC` pins every C table row and sentinel.
- `TestSpecGuildGuard_EntryGatesAndAudiences` covers unauthorized and
  authorized movement, aliases, immortal/remort exemptions, and exact actor
  versus room audiences.
- `TestSpecGuildGuard_FleeAliasesAndExactRoomText` covers `flee`, `escape`,
  and `retreat` with the exact C room text.
- `TestSpecGuildGuard_BlindAndNonMoveDelegateToFighter` covers both fighter
  fallback gates at the special boundary.

The C-first live vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-guild-guard.txt`. On the pre-fix
main-equivalent code it was RED: movement did not intercept, the peer received
the actor's flee room line, and the room wording differed. After the focused
fix, the vehicle is GREEN with `--show-oracle --seed 1`; seeds 2, 3, 5, and 8
also report no normalized divergence. The live scenario uses active mob 8017,
room 8015, a mortal thief, a warrior peer, and the C level gate is restored by
setting the primary player to level 1 after warmup.

## Verification and next queue item

The focused guild_guard tests and all five live differential runs pass. Full
repository gates are run before the PR is opened: `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, clean
`gofumpt -l .`, and `make fidelity-depth`.

After this slice merges, refresh `main` and take the next active
source-and-registration ordered special procedure, `puff` at
`src/spec_procs.c:611` / mob vnum 1. The blocked
`objmagic.sleep-entry-gates` row remains queued after the special-procedure
inventory is exhausted.

Rules applied: R1, R2, R4, R5b, R5c, and R5e.
