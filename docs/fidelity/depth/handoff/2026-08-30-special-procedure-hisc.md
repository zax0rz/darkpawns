# Depth-fidelity handoff — hisc

Date: 2026-08-30  
Slice: special procedure `hisc`  
C declaration: `SPECIAL(hisc)` in `src/spec_assign.c:166`  
C definition: `src/spec_procs3.c:894-903`  
Only assignment: commented `ASSIGNMOB(14412, hisc)` at `src/spec_assign.c:382`  
Manifest commit: `9aaa4f7f1`  
PR: #783, merged as `212f5b27b`; hosted run `33292905576`

## Queue position

The inventory pass after `conjured` found `hisc` as the next source-order C
definition. Its only apparent mob assignment is commented out, and searches of
the active mob, object, and room assignment tables found no registration. The
unassigned body is claimed here and must not be repicked. The next active
unclaimed procedure is the room special `elements_master_column`, defined at
`src/spec_procs3.c:936-1002` and registered at room vnum `1315` in
`src/spec_assign.c:622`.

## C call path and reachability ruling

`SPECIAL(hisc)` declares the existing `no_move_south` and `cleric` procedures.
For a south command it delegates to `no_move_south(ch, me, cmd, NULL)`; for
all other calls it delegates to `cleric(ch, me, 0, NULL)`. A commandless call
would therefore be a cleric path, while a player command could expose the
south gate, but only if the containing special pointer were installed.

The actual C dispatchers are the player-command special path in
`src/interpreter.c:1407-1456` and autonomous mobile activity in
`src/mobact.c:68-93`. `assign_mobiles` has only the local declaration and the
commented assignment at `src/spec_assign.c:382`; `assign_objects` and
`assign_rooms` have no `hisc` assignment. Therefore no live C mob, object, or
room can enter this procedure through a registered special pointer. Promoting
the Go-only `RegisterSpec("hisc", ...)` entry or synthesizing mob vnum 14412
would violate the C registration surface, so no oracle vehicle is valid.

## RED → GREEN / scope

This is a documentation-only inventory slice. No Go behavior was changed and
no `src/` or `darkpawns-c-oracle/` file was edited. The manifest now contains
`mob.hisc-unassigned` as D5 `excluded`, with no synthetic scenario. This keeps
the latent C body visible in the durable inventory while preserving R2's
registered command surface and R4's no-invention rule.

## Verification

The inventory evidence was checked against all active `ASSIGNMOB`, `ASSIGNOBJ`,
and `ASSIGNROOM` tables in `src/spec_assign.c`. The exclusion PR was opened
after local gates passed and merged only after the one prescribed workflow
retry produced green hosted checks:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

Hosted `test`, `security`, and `lint` all passed in run `33292905576`; the
optional `build-and-push` and `deploy` jobs were skipped. The initial PR had no
checks, so `gh workflow run "Dark Pawns CI/CD" --ref glm/spec-hisc` was used
once as required; no further retry was needed.

The manifest frontier changed from:

```text
1193 total; 1148 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1148/1161 = 98.9%
```

to:

```text
1194 total; 1148 proven/delegated, 13 blocked, 33 excluded
Actionable completion: 1148/1161 = 98.9%
```

## Fidelity rulings

This slice follows R2 (the registered C dispatch surface), R4 (no synthetic
vnum or invented proof), and R5e (the active assignment tables and both real
special dispatchers were checked). R5b/R5c require keeping this unreachable
classification distinct from the reachable `no_move_south` and `cleric`
callee behavior already owned by their active registrations.

## Next action

Start from `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove
`elements_master_column` at room `1315` in C source and registration order.
Continue the special-procedure inventory afterward, then attempt the blocked
`objmagic.sleep-entry-gates` row once via cast-sleep before sweeping remaining
un-manifested command families from `src/interpreter.c`. Leave one dated
handoff for the next session.
