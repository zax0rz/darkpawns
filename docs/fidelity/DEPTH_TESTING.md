# Command Depth Fidelity

This is the handoff guide for finishing the Dark Pawns port by behavior rather
than by file count. The Go implementation may exist and a command may have one
green oracle scenario while substantial player-visible branches remain wrong.

## Strategy: Breadth, Then Depth

Breadth answers: **does every registered command have at least one live C-vs-Go
probe?** It catches missing handlers, obvious messages, and broad command-surface
gaps. PR #598 completed the registered-command breadth backlog.

Depth answers: **does one command match across its reachable behavior tree?** A
depth pass maps the actual C call path, enumerates gates and outcomes, proves
each audience and state transition, checks RNG draw parity, and identifies
shared behavior that belongs to another manifest.

Do not infer completion from a breadth scenario. Do not duplicate every shared
callee branch under every caller: prove the caller/callee boundary, then delegate
the callee's full matrix to its own depth manifest.

## Proof Levels

Use D1-D5 as a practical review order, not as a claim that complexity is linear:

- **D1 — entry gates:** position, argument, authorization, and early-return bytes.
- **D2 — direct outcomes:** success/failure branches and terminal messages.
- **D3 — audiences and topology:** actor, victim, origin room, destination room,
  object/room state, and failures returned by direct callees.
- **D4 — state transitions:** combat teardown, XP/resources, affects, cooldowns,
  ordering, and exact arithmetic. Use focused unit tests when state is not visible.
- **D5 — hidden/re-entrant behavior:** RNG draw parity, automatic combat calls,
  NPC/specproc entry, wait-state interactions, and other non-command call sites.

The C source is the case inventory. Follow the real call path (R5e); summaries,
Go tests, and the existing Go implementation are not behavioral authority.

## Per-Command Workflow

1. Locate the registered Go handler and the C handler/call sites.
2. Enumerate reachable branches and player audiences before changing Go.
3. Create `docs/fidelity/depth/<command>.tsv`, one row per case.
4. Prefer an oracle scenario for observable bytes. Add named peers when room or
   victim output matters. Use unit tests for exact hidden state/arithmetic.
5. Add `# depth-case: <case-id>` to each proving scenario.
6. For RNG, run several `--seed` values and expose the selected draw in output;
   equal final outcomes alone do not establish R3.
7. Run with `--show-oracle` at least once to verify the intended C block executed.
8. Fix only confirmed divergences, checking the entire behavior class under R5c.
9. Run `make fidelity-depth`, the relevant oracle matrix, and all repository gates.

Manifest statuses are deliberately explicit:

- `oracle-green` / `oracle-green-multiseed`: live transcript proof.
- `unit-green`: focused state-level proof tied to a real test symbol.
- `delegated`: shared callee behavior owned by another named manifest.
- `excluded`: unreachable from this surface; state the owning surface.
- `blocked`: a real proof gap. Do not relabel inconvenience as exclusion.

`scripts/gen_fidelity_depth.py` validates scenario annotations and unit-test
symbols, then prints the actionable completion report. The manifest is the
durable evidence; prose percentages are only snapshots.

## Harness Lessons From the `flee` Pilot

- World topology is test input. Disposable `replace-room-exits` and
  `set-room-flag` fixtures make movement branches deterministic without touching
  `src/`, the C checkout, or checked-in world data.
- Recipient bytes matter. A primary actor plus origin/destination peers exposed
  canonical leave/arrival text that an actor-only probe missed.
- Seed matching is necessary but insufficient. All six exits in the success
  fixture led to one destination, while observer arrival text revealed which
  direction the RNG selected.
- Successful movement must use the canonical movement path. Low-level relocation
  can match destination state while inventing or omitting player-facing bytes.
- Some descriptor-issued timing states are unreachable because command input is
  drained only after wait reaches zero. A synchronous `force` or focused unit
  test can exercise the same handler call path without pretending queued input did.
- `--show-oracle` is a development requirement for timing-sensitive probes; a
  no-diff report is meaningless if the intended block never ran.
- State such as exact XP deltas may be silent in C. Prove the arithmetic in a
  unit test and separately prove that no invented player message appears (R4).

## Dated Handoff: 2026-08-23

- Registered-command breadth coverage was completed and merged in PR #598.
- `flee` is the first depth pilot, implemented in PR #601.
- Its manifest is `docs/fidelity/depth/flee.tsv`: 14 mapped cases, 13 actionable
  cases proven/delegated, zero blocked, and one NPC-only case excluded to the
  future mob/specproc surface.
- Transitive boat, charm, tunnel, mount, special, and greet failures belong in a
  movement depth manifest; `flee` proves the callee false-return edge.
- The movement depth pass is now captured in `docs/fidelity/depth/movement.tsv`.
  It exposed and fixed destination-look ordering during follower recursion and
  added disposable exit-keyword and room-sector fixtures. Ordinary movement,
  closed exits, boats, tunnels, vertical audiences, follower state, resource
  costs, and death traps are proven.
- Mounted movement is now implemented and proven as a vertical slice: spawned
  mobs have C's 50-point movement pool, rider/mount pairs transfer together,
  only mounts pay movement cost, failure gates match, and room observations
  represent the pair once. Greet plus room-enter trigger ordering are the two
  remaining shared-movement proof gaps.

Update this dated section when the frontier materially changes. Keep the rest
of this document stable unless the methodology itself changes.
