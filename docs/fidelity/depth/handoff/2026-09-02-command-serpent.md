# Depth handoff — 2026-09-02 — command `serpent`

The `serpent` depth slice is complete. The C path was audited from
`src/interpreter.c:685` through `src/new_cmds2.c:693-743`, the shared damage
path in `src/fight.c:1023-1092` and `1314-1718`, `improve_skill()` in
`src/act.other.c:1704-1727`, and `create_mobile()` in
`src/new_cmds2.c:588-618`. This established the skill gate, one-argument
target parsing and fighting fallback, self/mounted ordering, sleeping-target
auto-hit, set-156 actor/victim/room messages, zero-damage miss enrollment,
post-damage training draw, quiet level-derived hunting-mobile creation, and
deferred improvement ordering.

The initial RED probes found three confirmed divergences: Go used invented
serpent-kick literals instead of C fight-message set 156, joined trailing
target words instead of C `one_argument()`, and only fell back to fighting on
an empty command. The skillset vehicle also exposed that the multiword C skill
name needed the Go `serpent_kick` gameplay key. The fixes preserve the C
call-path order and keep the shared damage behavior delegated under R5b/R5c;
neither `src/` nor `darkpawns-c-oracle/` was edited.

Evidence is recorded in `docs/fidelity/depth/serpent.tsv` and the vehicles
`serpent-gate-depth` and `serpent-depth`. The gate vehicle is green on seed 1
with oracle output shown. The depth vehicle is green on seeds 1, 2, 3, 5, and
8. Focused tests cover registration, skillset key mapping, C gate ordering,
set-156 result contracts, sleeping auto-hit, create-mobile stats and XP, the
quiet room-18201 hunting spawn, and the post-create improvement draw.

Feature/evidence PR #1157 (`glm/depth-serpent`) passed hosted lint, security,
and test checks and was self-merged to `main` as
`763c82600cdd7bbc69af7a1e7e0dd106837b4925`. The repository gates passed:
`make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, clean `gofumpt -l .`, and clean oracle-tree diff.

The frontier after this merge is:

- 3210 total cases
- 3128 proven/delegated
- 30 blocked
- 52 excluded

The required one-time blocked-row vehicle remains satisfied and bounded:
`objmagic.sleep-entry-gates.cast` is green through the cast-sleep outlaw and
reagent arms in `sleep-spell-depth`; the separate object-magic entry row stays
blocked in `docs/fidelity/depth/object-magic.tsv` and must not be re-picked.

The next unclaimed source-order interpreter entry is `sha` at
`src/interpreter.c:686`, registered to `do_sha` with the C table gate. Begin it
only after a fresh `main` checkout/pull, `make fidelity-depth`, depth-guide
read, newest-handoff read, and source/table audit. Fidelity basis remains
R1/R2/R3/R4/R5e, with shared ownership bounded by R5b/R5c.
