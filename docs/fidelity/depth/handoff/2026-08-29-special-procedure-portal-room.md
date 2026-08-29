# Dated Handoff: 2026-08-29 (special-procedure `portal_room` inventory slice)

## Frontier and inventory

At the start of this session, `main` was checked out, fast-forwarded, and
validated with `make fidelity-depth`; `docs/fidelity/DEPTH_TESTING.md` and the
newest prior handoff were reread. The refreshed C census found 113 `SPECIAL`
definitions: 41 in `src/spec_procs.c`, 43 in `src/spec_procs2.c`, and 29 in
`src/spec_procs3.c`. The dispatch-table census found 233 `ASSIGNMOB` calls,
plus the object and room registrations, with 228 unique mob VNUMs and 66
unique final mob-procedure names as recorded by the existing scout.

After this slice, `make fidelity-depth` reports:

- Cases: 1095 total
- Proven/delegated: 1050
- Blocked: 13
- Excluded: 32
- Actionable completion: 1050/1063 (98.8%)

## C-first result

The next source-order definition after `eq_thief` is
`SPECIAL(portal_room)` at `src/spec_procs2.c:1648-1677`. Its body has these
latent branches:

- accept only `say` or apostrophe commands, unless `mini_mud` is active;
- trim the argument and require the exact case-insensitive word
  `kallinistra`;
- call `do_say`, send the actor's teleport line, broadcast disappearance in
  the origin room, move to real room 21264, broadcast arrival, and call
  `do_look`.

The real command path would be `special()` at `src/interpreter.c:1407-1416`,
where a room function pointer is called before ordinary command dispatch. The
actual registration tables were checked in `src/spec_assign.c`: mobile rows
`182-508`, object rows `528-561`, and room rows `605-635`. They contain no
`ASSIGNMOB`, `ASSIGNOBJ`, or `ASSIGNROOM` entry for `portal_room`; the only
occurrence there is its declaration at line 586. Therefore no registered C
room/mobile/object pointer can reach this body, and an oracle fixture would
invent a player-visible surface. Under R2, R4, and R5e, the correct durable
status is excluded rather than a synthetic proof.

## Delivery

Added the manifest row `room.portal-room-unassigned` to
`docs/fidelity/depth/spec-procs.tsv` with `D5 / command / excluded` and the
complete C branch and registration evidence above. No scenario or Go change
was made, and neither `src/` nor `darkpawns-c-oracle/` was edited.

The documentation-only change landed through PR #769, squash commit
`a39163556` on `main`. Local gates passed: `make fidelity-depth`, `go build
./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
`gofumpt -l .`. CI test, security, and lint checks were all green; no ungreen
PR was merged.

The next source-order unclaimed registered procedure is `carrion`, defined at
`src/spec_procs2.c:1726` and registered for room 14305 by
`src/spec_assign.c:612`. First map its room-command call path and corpse
branches; do not repick `portal_room`.

This slice applies R2 (registered command/special surface), R4 (no synthetic
vehicle or invented output), and R5e (actual C dispatch and registration
verification).
