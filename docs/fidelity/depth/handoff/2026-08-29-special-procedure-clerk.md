# Depth-fidelity handoff — clerk

Date: 2026-08-29  
Slice: special procedure `clerk`  
Registration: mob vnum `18228` at `src/spec_assign.c:407`  
C definition: `src/spec_procs3.c:75-142`  
Code commit: `65e694bf1`  
PR: #779, merged as `7a64840a4`; hosted run `33289478546`

## Queue position

The special-procedure inventory was refreshed through the end of
`src/spec_procs2.c` and into `src/spec_procs3.c`. `clerk` is claimed here and
must not be repicked. The next active unclaimed special is `butler`, defined
next in `src/spec_procs3.c` (refresh its exact extent before work) and
registered as mob vnum `8092` at `src/spec_assign.c:300`.

## C call path and branch map

The audit followed the registered-mobile dispatch in `special()` at
`src/interpreter.c:1407-1456`, the `SPECIAL(clerk)` body at
`src/spec_procs3.c:75-142`, and `do_tell()` at `src/act.comm.c:901-930`.
The script-bearing authored mob `18228` was spawned and its script stripped so
the native procedure was isolated.

The procedure rejects commandless calls, a fighting player, and a sleeping
player before transaction work. It maps zone numbers 80, 182, and 212 to the
Homet1, Homet2, and Homet3 citizenship names. An unknown zone sends the
verified default warning to the actor before the command falls through.
`list` tells the price directly. `buy` requires the exact `citizenship`
argument, then branches on visibility, insufficient gold, already matching
hometown, and successful hometown/gold mutation. The C `CAN_SEE(mobile,ch)`
visibility path was checked in `src/utils.h:513-530`; hide is not part of that
predicate. Transaction tells use C's direct `do_tell` audience, not a room
broadcast.

## RED → GREEN

On main, the first vehicle showed C's direct clerk tells being broadcast to a
same-room peer by Go, with the actor name also embedded inside the quoted
transaction text. The vehicle also exposed the existing wizard gold-set
acknowledgement mismatch (`ClerkActor's gold set to 3000.` in C versus Go's
generic `Gold set to 3000.`), which was corrected because the vehicle depends
on that enabler. The confirmed fixes restored the direct audience, exact
transaction bytes, zone/default ordering, C visibility semantics, and
citizenship state transitions. No `src/` or `darkpawns-c-oracle/` files were
edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-clerk.txt`  
Focused tests: `pkg/game/spec_clerk_test.go`

The vehicle is oracle-green with `--show-oracle` at seeds 1, 2, 3, 5, and 8.
Focused tests cover entry gates, direct tell audience, argument and gold
gates, success/already-citizen state, visibility, unknown-zone warning order,
and all three mapped hometown names.

Required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

Hosted `test`, `security`, and `lint` checks all passed in run
`33289478546`. The optional `build-and-push` and `deploy` jobs were skipped by
workflow policy; no CI retry was needed because the checks fired normally.

The manifest now reports:

```text
1163 total; 1118 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1118/1131 = 98.9%
```

## Fidelity rulings

This slice follows R1 (exact direct tell bytes and audiences), R2 (the
registered `list`/`buy` command surface), R3 (deterministic multi-seed
transaction behavior), R4 (no invented room or transaction output), and R5e
(verified `special()`, `SPECIAL(clerk)`, `do_tell()`, `CAN_SEE()`, and wizard
enabler call paths). R5b/R5c apply to the shared save boundary; this slice
claims the clerk's dispatch, visibility, output, zone mapping, and state
branches.

## Next action

Return to `main`, pull, run the frontier, and reread the depth-testing
instructions and this newest handoff. Then map `SPECIAL(butler)` in
`src/spec_procs3.c` and its registration before taking the next slice in
file-and-registration order.
