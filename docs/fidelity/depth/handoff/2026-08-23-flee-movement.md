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
  represent the pair once. Focused script recording now proves the final shared
  movement ordering boundary: destination look, then mob greet, then room-enter
  script. The movement depth manifest has no remaining actionable gaps.
