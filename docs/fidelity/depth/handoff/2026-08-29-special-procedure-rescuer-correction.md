# 2026-08-29 — `rescuer` publication correction

PR #757 (`feat: port rescuer special procedure`) was squash-merged into
`main` as `6a814d179`. GitHub `lint`, `security`, and `test` checks were all
green; the workflow's build-and-push and deploy jobs were skipped by policy.
The local `make fidelity-depth`, build, vet, test, lint, and gofumpt gates were
green before review. The original rescuer handoff's pending-publication note
is therefore closed.

The post-merge depth frontier is 1010 total cases: 974 proven/delegated, 12
blocked, and 24 excluded; actionable completion is 974/986 (98.8%). The next
unclaimed active source-order special is `pissedalchemist` at
`src/spec_procs2.c:546`, registered for mob vnum 15814 at
`src/spec_assign.c:422`. Continue there; do not repick `rescuer` or its
already-recorded vnum 15808 exclusion.
