# Depth-fidelity handoff — `motd`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R4, R5b, R5e

## Frontier

Started from fresh `main` after the `mlist` handoff.  `make fidelity-depth`
reported 2,568 cases: 2,500 proven/delegated, 22 blocked, and 46 excluded
(99.1% actionable).  After the `motd` evidence was merged, the frontier is
2,572 total: 2,504 proven/delegated, 22 blocked, and 46 excluded (99.1%
actionable).

The next un-manifested command family is `mail`, the registered C row at
`src/interpreter.c:556` (`POS_STANDING`, level 1, `do_not_here`).  No existing
depth manifest row claims `mail`.

## C path and proof

The queue item was `{ "motd", POS_DEAD, do_gen_ps, 0, SCMD_MOTD }` at
`src/interpreter.c:555`.  C dispatch reaches `do_gen_ps` in
`src/act.informative.c:2117-2158`; the `SCMD_MOTD` branch passes the boot-cached
`motd` string to `page_string` without inspecting the command argument.
`src/db.c:318` loads `MOTD_FILE`, defined as `text/motd` at `src/db.h:52`.

The Go command was already on the matching path: `cmdMotd` delegates to the
cached `lib/text/motd` loader and `PageString`, and the authoritative command
gate table has the unrestricted `POS_DEAD` entry.  No behavioral fix was
warranted after the C-first comparison; R4 forbids inventing one.

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/motd-depth.txt` — exact static page and
  trailing-argument-ignored branches.
- `pkg/session/motd_depth_test.go` — C gate and registered-entry identity.
- `docs/fidelity/depth/motd.tsv` — four explicit rows, including shared
  pager delegation.

The live vehicle was green at seeds 1, 2, 3, 5, and 8, and was run with
`--show-oracle` at seed 1.  The static text matched after the normalizer’s
volatile release-line masking; the fixed authored MOTD and argument-ignored
behavior were both present in the C block.  No `src/` or C-oracle files were
edited.

## Gates and merge

Local gates passed on `glm/depth-motd`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature/evidence PR #1030 (`glm/depth-motd`) had green hosted `lint`,
`security`, and `test` checks; build/deploy were skipped by the PR workflow.
It was self-merged as squash commit `3c9d4fe1c`.

The post-merge `main` frontier was rerun and passed with the counts above.

The next session must start on `main`, pull, rerun `make fidelity-depth`,
re-read `docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only
the unclaimed `mail` family at `src/interpreter.c:556`.
