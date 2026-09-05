# Depth-fidelity handoff — `elements_guardian` — 2026-08-30

## Frontier and queue position

- Started from the freshly pulled `main`; the opening `make fidelity-depth`
  report was **1232 total, 1186 proven/delegated, 13 blocked, 33 excluded**,
  actionable **1186/1199 (98.9%)**.
- Consumed the next active special-procedure slice in C file/registration
  order: `SPECIAL(elements_guardian)` at `src/spec_procs3.c:1242-1295`,
  actively registered by `ASSIGNMOB(1314, elements_guardian)` at
  `src/spec_assign.c:196`.
- The local manifest now has **1237 total, 1191 proven/delegated, 13 blocked,
  33 excluded**, actionable **1191/1204 (98.9%)**.
- Next active special is `fly_exit_up`, defined at `src/spec_procs3.c:1289+`
  and registered by `ASSIGNROOM(1389, fly_exit_up)` at
  `src/spec_assign.c:635`.

## C call path and observable contract

The player-command path is `src/interpreter.c:1407-1456`: the current-room
special receives the player as `ch` and the registered mob as `me` before the
ordinary command handler. The autonomous path is `src/mobact.c:68-93`, which
passes the mob as both `ch` and `me` with `cmd=0`; C returns immediately on
`!cmd` before scanning the room.

For a command, C walks `world[ch->in_room].people` in its front-insert order,
skipping NPCs, players above `LVL_IMMORT`, and already-fighting players. For
the first remaining player it inspects the immediate next room occupant. If
that occupant is absent, an NPC, above the immortal threshold, or fighting,
C draws `number(10,50)`, calls `damage(ppl, ppl, dam, TYPE_UNDEFINED)`, then
sends the exact two Act messages for the observer and the affected player. If
the immediate next occupant is another eligible player, C sends three exact
audience-specific Act messages, calls synchronous `hit(ppl, next,
TYPE_UNDEFINED)`, and returns `FALSE`. The ordinary command therefore still
runs after the special.

## RED → GREEN

- Vehicle: `cmd/dp-oracle-diff/scenarios/spec-proc-elements-guardian.txt`.
- The initial main vehicle was RED: the old Go handler selected targets from
  an unordered player map, invented a `GuardianTargetself` string, used the
  wrong self-damage source, added charm effects absent from C, and used a
  direct one-point damage call instead of the player `hit()` path.
- The corrected vehicle uses registered mob 1314, three named mortal peers,
  and a one-record attack-type weapon (`lib/world/obj/162.obj` vnum 16212)
  acquired through the command path. The compared probe is only `say hello`;
  setup and positioning are discarded warmup. `--show-oracle` confirmed the
  C pair branch and its actor, victim, and third-party outputs.
- Final live oracle runs were GREEN with `result: no normalized divergence`
  for seeds **1, 2, 3, 5, and 8**.
- Focused proof in `pkg/game/spec_elements_guardian_test.go` covers commandless
  and nil entry, ordered pair selection, exact three-audience Act bytes,
  absence of invented charm state, solo self-damage, pronoun substitution, and
  the no-self-fighting invariant.

## Go changes

- Reconstructed deterministic C room-people order from connection/object
  insertion metadata rather than iterating Go maps.
- Matched the C commandless gate, immediate-next solo/pair branches, exact Act
  templates, `damage(ppl,ppl,...)` self-source, and ordinary player combat
  opener for `hit(ppl,next,TYPE_UNDEFINED)`.
- Fixed a confirmed vehicle-path divergence in `cmdForce`: C preserves the
  complete forced command tail, while Go previously executed only its first
  word. The guard now checks the command verb while executing the full tail.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Manifest, gates, and PR

Added six rows to `docs/fidelity/depth/spec-procs.tsv` for entry, autonomous
commandless dispatch, solo branch, pair branch, pair audience, and delegated
canonical hit behavior. The rows cite R1/R2/R3/R4/R5e; the shared hit
delegation also cites R5b/R5c.

Passed locally: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` clean, and
`git diff --check`.

The branch is `glm/spec-elements-guardian`; commit and PR status will be added
after the branch is pushed and GitHub checks complete. Do not merge unless all
checks are green.

## Next action

Start the next session from `main`, pull, rerun the frontier, reread the depth
testing guide and this handoff, then map and prove `fly_exit_up` in C file and
registration order. Continue the special-procedure inventory, then attempt the
one blocked `objmagic.sleep-entry-gates` row through the cast-sleep
outlaw/reagent vehicle before sweeping remaining un-manifested interpreter
families. Leave one dated handoff per session.
