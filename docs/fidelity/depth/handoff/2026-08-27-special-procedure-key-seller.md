# Dated Handoff: 2026-08-27 (special-procedure key-seller wave)

The deterministic special-procedure slice is complete on `main` at
`1681a1073` (self-merged PR #690). This round followed the assigned
`no_get` procedure (PR #689, handed off earlier) with `key_seller`. The
manifest gate is green and the existing blocked frontier remains untouched.

## Round — `key_seller` (PR #690)

The C assignment and call path were verified before changing Go:

- `src/spec_assign.c:452` assigns mob VNum **19610** to `key_seller`.
- `src/spec_procs2.c:1870-1932` first accepts only `buy`/`list`, resolves the
  seller's room through `find_house`, checks the exact owner ID, then routes
  each response through `do_tell`.
- `src/house.c:238-312` and `House_boot` validate the room, atrium, exit, and
  owner before the record becomes active.
- The source and `/home/zach/darkpawns-c-oracle` were not edited.

The disposable differential harness now writes equivalent native C and JSON
house-control records. The owner vehicle uses seeded C player Testor (ID 1)
and an explicit Go-side owner ID 0 because the no-DB creation harness assigns
new Go characters ID 0. It covers C's immortal room 1204 and Go's post-birth
room 8162 with valid down/north exits. The non-owner vehicle uses C owner ID 2
(seeded Oraklet) and Go owner ID 999 while probing as Testor.

RED on clean `main` was established before the fix: Go invented a 50-gold
shop response and failed to match C's house-gated behavior. The confirmed
Go-only corrections are:

- invoke the existing `World.PostInit` house-boot seam during server startup;
- remove `HouseBoot`'s boot-time recursive lock around `houseLoad`;
- replace the invented key-seller shop with the C owner gate, exact list/buy
  tells, 10,000-gold affordability check, configured house-key vnum, and C
  object/audience Act path;
- capitalize mob tell senders to match C's `do_tell` output.

GREEN vehicles:

- `spec-proc-key-seller`: owner `list`, bare `buy`, `buy junk`, and
  unaffordable `buy key`;
- `spec-proc-key-seller-non-owner`: non-owner `list` and `buy key`.

The owning manifest `docs/fidelity/depth/spec-procs.tsv` gained five D3
`oracle-green` rows. `make fidelity-depth` exits 0:

```text
Cases: 529 total, 518 proven/delegated, 1 blocked, 10 excluded
Actionable completion: 518/519 = 99.8%
```

Local gates all passed: `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, `gofumpt -l .`, and local `gosec -severity=high
-quiet ./...`. Hosted lint, security, and test checks passed after one
security retry caught and then cleared the fixture encoder's G115 checks.

## Frontier

No fight-, percent-, heartbeat-, or deep-engine backlog item was attempted in
this wave. Leave those rows with their existing owners and blocked/deferred
status until the next dedicated vehicle has the required combat state, random
draw, or pulse fixture. The next command-family pivot may begin with the
deterministic `consider` manifest in `src/act.informative.c` (`do_consider`),
then proceed to other special procedures only with source-verified fixtures.

