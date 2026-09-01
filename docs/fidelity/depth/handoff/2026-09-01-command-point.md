# Depth-fidelity handoff — `point`

Date: 2026-09-01

## Queue position

This session began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep reached `point` in source-table
order. The next unclaimed interpreter row is `poke` at `src/interpreter.c:610`.

Frontier before this slice: 2,798 total; 2,727 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,812 total; 2,741 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:609 */
{ "point", POS_RESTING, do_point, 0, 0 },
```

The handler is `src/new_cmds.c:2505-2560`. Its actual path is:

1. `one_argument` parses the first non-fill-word token.
2. `get_char_room_vis` takes precedence for a visible character target.
3. `get_obj_in_list_vis` searches visible room contents.
4. `ismove` accepts abbreviations of east, west, up, down, north, and south.
5. Remaining input falls through to the around-the-room actor/room pair.
6. Character targets split into named self or another character; another
   character splits again on whether `GET_EQ(ch, WEAR_WIELD)` is present.
7. `point_obj` emits the room act before the actor act, with `$p` for the
   wielded object and `$P` for the room object.

The direct C message and audience branches are therefore no argument, six
direction abbreviations, visible player/NPC target with and without a wielded
weapon, named self, `self` alias, visible room object with and without a wielded
weapon, unresolved-target fallback, and fill-word/trailing-input parsing. The
command's shared POS_RESTING gate and target/object visibility machinery are
separately claimed or delegated in the manifest.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/point-depth.txt`

Manifest: `docs/fidelity/depth/point.tsv` (14 rows)

Test: `pkg/session/point_depth_test.go`

The first live seed-1 vehicle was RED on main. It exposed these confirmed Go
divergences:

- direction abbreviations were missing;
- room objects were not searched;
- the command wrapper discarded the resolved victim, so the victim branch
  could not receive its C line;
- the self-room pronoun used C sex values against Go's sex encoding;
- target parsing did not apply `one_argument`; and
- the wielded-weapon branch was not reachable in the fixture until a separate
  sword was loaded, wielded, and removed around the object/character probes.

The Go fix is limited to the confirmed `do_point` class: the command wrapper
now parses and passes the resolved character target to the result fan-out;
`DoPoint` now follows C's character-before-object-before-direction order,
uses the canonical room-object resolver, emits exact object and weapon short
descriptions, and uses the correct Go sex encoding for the self-room line.
No unrelated command behavior was changed.

The corrected vehicle uses a scriptless generic mob, a named peer, a loaded
room gem, and the starting sword loaded and wielded during the compared probe.
`point-depth --show-oracle --seed 1` was GREEN; the matrix at seeds
`1,2,3,5,8` was GREEN with no normalized divergence. It proves actor, victim,
and non-victim room bytes for character targets, both weapon states, objects,
directions, self forms, fallback, and parsing.

## Verification and integration

Local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-point`

Feature commit: `d3b98ed53` (`fix: align point depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1069 — merged; main merge commit `7055cc79e`.
Hosted lint, security, and test checks were green; build-and-push and deploy
were skipped by repository workflow conditions. The PR was merged only after
the required hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
