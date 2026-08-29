# Dated Handoff: 2026-08-29 (special-procedure `eq_thief`)

## Frontier

`make fidelity-depth` on `main` after PR #768 reports:

- Cases: 1094 total
- Proven/delegated: 1050
- Blocked: 13
- Excluded: 31
- Actionable completion: 1050/1063 (98.8%)

The next source-order special-procedure item is `portal_room` at
`src/spec_procs2.c:1648`; this handoff does not claim it. Its declaration is
present at `src/spec_assign.c:586`, but no `ASSIGNMOB(..., portal_room)` row was
found in the registration tables. Rebuild the C inventory and claim that
reachability result in the next session before advancing.

## Queue slice and registrations

This slice consumed the next registered procedure after `medusa`:

- `ASSIGNMOB(7979, eq_thief)` — `src/spec_assign.c:266`
- `ASSIGNMOB(12118, eq_thief)` — `src/spec_assign.c:350`
- `ASSIGNMOB(14225, eq_thief)` — `src/spec_assign.c:366`

Mob 7979 is the first registration, but `src/db.c:2113-2135` hard-codes 79xx
reset mobs into a random non-city room, so it cannot be pinned to a fixture
room with `spawn-mob`. Mob 12118 is the fixed-room vehicle used by
`cmd/dp-oracle-diff/scenarios/spec-proc-eq-thief.txt`; it is scriptless in
`lib/world/mob/121.mob:278-291`. Mob 14225 is the third registration and has
the random-zone behavior that makes it unsuitable as the fixed vehicle.

## C path and branches

The verified autonomous path is `src/mobact.c:68-93`, which skips fighting or
asleep mobs and invokes the special as `(ch, ch, 0, "")` at lines 82-91.
`SPECIAL(eq_thief)` then runs at `src/spec_procs2.c:1613-1646`; its helper
`npc_kender_steal()` is at lines 1583-1611.

The complete player-visible and state branch map is:

1. Non-empty command returns `FALSE`; commandless mobile dispatch is the only
   entry surface.
2. A non-standing thief returns `FALSE`.
3. Each mortal player in the room is considered in list order; immortal players
   are skipped, and each candidate consumes the outer `number(0,4)` draw.
4. For visible carried objects, the helper consumes the per-item
   `number(0,60)` gate, then the percent draw before applying sleeping,
   peaceful-room, immortal-victim, and immortal-thief overrides. A successful
   mortal transfer moves the first selected object to the thief.
5. If the first carried object is a container, `all.black <container>` performs
   the visible black-item retrieval; the room sees the retrieval, while the
   mob's `TO_CHAR` text is unobservable.
6. The first visible black carried object is junked, with the exact actor and
   room messages from lines 1636-1642, then extracted. The special returns
   `TRUE` for the selected victim and `FALSE` when no candidate triggers.
7. The outer draw, per-item gate, pre-override percent draw, final outcome draw,
   and continuation stream are preserved; no player command consumes these
   autonomous draws.

## Proof

Main RED was established before the port: with the old command-based Go
handler, the fixed 12118 vehicle produced a clean C inventory but retained the
loaded bread in Go on seed 1. The corrected implementation is GREEN on the
same 2000-pulse vehicle for seeds 2, 3, 5, and 8, covering no-trigger and
successful-transfer arms. Seed 1 remained a shared boot-stream arm where C
steals and Go does not after the long warmup, so it is deliberately not claimed
by the manifest; do not silently broaden the claimed seed set.

Focused tests in `pkg/game/spec_eq_thief_test.go` cover entry gates, mortal
target filtering, first-visible-item transfer, exact four-draw continuation,
container retrieval, junk extraction, and audience behavior. The scenario
manifest rows are:

- `mob.eq-thief-commandless-entry`
- `mob.eq-thief-mortal-target-gate`
- `mob.eq-thief-steal-transfer`
- `mob.eq-thief-pulse-dispatch`
- `mob.eq-thief-container-black-junk`
- `mob.eq-thief-junk-audience`
- `mob.eq-thief-rng-order`

All seven rows are accepted by the frontier validator. The local full gates
passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test
./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.

## Delivery and rules

The implementation and proof landed through PR #768, squash commit
`82409c3d4` on `main`. Its first CI run exposed the reachability ratchet's
static-registration heuristic for `offer`; the existing generic `do_not_here`
registration was made literal in commit `d5a314dcd`, after which the updated
PR checks were all green (test, security, and lint). No ungreen PR was merged.

This slice applies R1 (exact player-facing bytes), R2 (the commandless
autonomous surface and generic command fallthrough), R3 (draw order and
multiseed determinism), R4 (no invented command hook or output), and R5e
(verified C call path and registration order). The Go port changed no files in
`src/` or `darkpawns-c-oracle/`.
