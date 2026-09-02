# Depth handoff — 2026-09-02 — command `sha`

The `sha` depth slice is complete. The C path was audited from
`src/interpreter.c:686` through `src/new_cmds.c:1500-1541` and
`1552-1739`. The registered command is standing-position, level 0, and
dispatches to `do_kuji_kiri` with `SKILL_KK_SHA`. The call path establishes the
non-Ninja, fighting, active-kuji, mounted, shared-skill, and SHA-mastery gates;
the success heal floor/cap and affect transition; the failure message and
non-lockout behavior; and actor/room audience bytes. The shared generic Go
handler already matched the C path, so no implementation divergence was found
and no `src/` or `darkpawns-c-oracle/` file was edited.

Evidence is recorded in `docs/fidelity/depth/sha.tsv` and the vehicles
`sha-depth` and `sha-failure-depth`. The success vehicle is green on seeds 1,
2, 3, 5, and 8, with seed 1 run using `--show-oracle`; it proves the success
actor/room bytes and the immediate lockout. The failure vehicle is green on
seed 1 with `--show-oracle`; it proves the failed concentration bytes and the
absence of an aggregate kuji-kiri lockout before the subsequent successful
cast. Focused tests cover the C registration gate, low-level heal floor,
maximum-health cap, and failure no-lockout state transition. The skill-slot
and shared entry gates remain delegated to their owning manifests under R5b/R5c.

The repository gates passed on `main` before publication:
`make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, clean `gofumpt -l .`, clean `git diff --check`, and
clean oracle-tree diff. The feature/evidence PR #1159 (`glm/depth-sha`) passed
hosted lint, security, and test checks after the one authorized workflow retry
and was self-merged to `main` as `df08146d5`.

The frontier after this merge is:

- 3225 total cases
- 3143 proven/delegated
- 30 blocked
- 52 excluded

The one-time blocked-row vehicle remains satisfied and bounded:
`objmagic.sleep-entry-gates.cast` is green through the cast-sleep outlaw and
reagent arms in `sleep-spell-depth`; the separate object-magic entry row stays
blocked in `docs/fidelity/depth/object-magic.tsv` and must not be re-picked.

The next unclaimed source-order interpreter entry is `shadow` at
`src/interpreter.c:687`, registered to `do_follow` with the C table gate.
Begin it only after a fresh `main` checkout/pull, `make fidelity-depth`,
depth-guide read, newest-handoff read, and source/table audit. Fidelity basis
remains R1/R2/R3/R4/R5e, with shared ownership bounded by R5b/R5c.
