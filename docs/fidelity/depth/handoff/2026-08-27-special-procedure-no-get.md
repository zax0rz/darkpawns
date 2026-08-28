# Special-procedure depth handoff — 2026-08-27 no_get slice

## Checkpoint

The assigned `no_get` slice is complete on `main` at `91c5f0beb` (PR #689,
self-merged after hosted lint, security, and test checks):

- `make fidelity-depth`: **524 total, 513 proven/delegated, 1 blocked, 10
  excluded**; exit 0.
- Actionable completion: **513/514 = 99.8%**.
- The only blocked row remains the intentional object-magic sleep entry gap.

## Vehicle and call path

C assigns both Grey Keep mobs 14416 and 14430 to `no_get` at
`src/spec_assign.c:385,388`. The handler is `src/new_cmds2.c:552-574`: it
requires an awake, living mobile, intercepts `get`, `palm`, and `take`, runs
`two_arguments()`, and stays active for zero or one token while returning
`FALSE` when a second token is present. The direct outcome is a
`TO_NOTVICT`/`TO_VICT` hand-strike pair followed by
`hit(me, ch, TYPE_UNDEFINED)`.

The live vehicle uses assigned mob 14416 in room 8162 with a primary actor and
peer. The five scenarios cover one-argument `get`, bare `get`, `palm`, `take`,
and two-argument `get` fall-through:

- `cmd/dp-oracle-diff/scenarios/spec-proc-no-get.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-no-get-bare.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-no-get-palm.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-no-get-take.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-no-get-two-args.txt`

On clean `main`, RED showed Go's invented ward message or ordinary command
fall-through where C emitted `A taipan man strikes at your hand!` to the actor
and `A taipan man strikes at Specactor's hand!` to the peer. The Go-only fix
now follows the C gates and token count, uses canonical `Act` audience routing,
recognizes all three command arms, and enters the shared combat state without
inventing a swing message for C's silent `TYPE_UNDEFINED` path.

The owning `docs/fidelity/depth/spec-procs.tsv` manifest has five `oracle-green`
rows. No `src/` or `darkpawns-c-oracle/` files were edited. This follows R1,
R4, R5c, and R5e: exact player bytes, no invented behavior, class-aware
shared-behavior review, and a verified reachable C assignment.

## Verification

- five no_get oracle vehicles — GREEN with `--show-oracle --seed 1`.
- `make fidelity-depth` — pass, counts above.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- PR #689 hosted test, lint, and security checks — pass.

## Next frontier

Begin the next deterministic assigned vehicle, `key_seller`, from current
`main`. Verify the house/ownership and object-creation call path before making
a fixture. Keep fight-, percent-, heartbeat-, and deep-engine backlog rows
blocked until their own controlled vehicles exist.
