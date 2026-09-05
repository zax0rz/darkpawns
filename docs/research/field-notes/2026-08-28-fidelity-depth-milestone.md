# Fidelity depth milestone — 2026-08-28

## Scope and question

- Track: legacy-port fidelity and agent-assisted software engineering.
- Question: what did the long-running depth loop actually complete, does the
  merged batch remain coherent on `main`, and what remains after deployment?
- Window: PRs #687 through #731, excluding unused numbers #696 and #697.
- Verified revision: `c74789ef9` on `main`.

## Primary-artifact observations

- GitHub reports 43 merged PRs in the range. The batch spans 227 files and
  changes 14,255 inserted lines and 875 deleted lines relative to
  `b5d238292`, the parent immediately before PR #687.
- The work has two ordered strands: active C special procedures in source and
  registration order, followed by distinct interpreter command families in
  table order through `carve`.
- `scripts/gen_fidelity_depth.py` reports 877 modeled cases: 855
  proven/delegated, six blocked, and 16 excluded. Actionable completion within
  the modeled ledger is 855/861, or 99.3%. This is not a percentage of the
  entire game or command surface.
- The six blocked rows are paladin combat dispatch, janitor pulse dispatch,
  cityguard breed-killer selection, cityguard pulse dispatch, dragon-breath
  combat transcript, and object-magic sleep entry gates.
- The deterministic reachability report covers all 508 C command-table entries:
  291 registered commands, 183 socials, eight special procedures, two
  abbreviation stubs, ten implemented-but-unwired commands, and 14 missing
  commands. Registered-command breadth was completed earlier; remaining
  command depth continues after `carve` in interpreter-table order.
- Release verification at `c74789ef9` passed `gofumpt`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  `make fidelity-depth`, and `make reachability`.
- The Linux/amd64 server built from this revision was deployed to CT 120 on
  2026-08-28. The service restarted with a fresh PID and zero restarts, the
  database connected, `/health` returned `OK`, and the required public HTML,
  Markdown, 404, and OpenAPI probes passed.

## Coherence and limitations

- Manifest references and named unit-test symbols pass the depth generator's
  validation, and the full repository gates remain green after the complete
  merge sequence. No tracked formatting changes were produced by `gofumpt`.
- The batch preserves explicit blockers instead of converting failed or
  unreachable vehicles into false greens. That is consistent with R4 and R5.
- Production logged an existing Lua `snake.lua` nil-opponent fight-trigger
  error after restart. It is operational follow-up, not evidence that the C
  `SPECIAL(snake)` row is reachable; the latter remains excluded based on the
  verified C assignment call path.
- The server shutdown path also missed its heartbeat deadline before restart.
  No players were connected, the replacement started successfully, and the
  database reconnected, but shutdown hygiene remains a separate operational
  concern.

## Proposed claims

- PF-006: 43 merged PRs advanced the depth ledger to 855/861 actionable modeled
  cases (99.3%) while retaining six explicit blockers. State: verified.
- PF-007: the current C command surface still has 14 missing and ten
  implemented-but-unwired entries, so depth-ledger completion must not be
  presented as whole-port completion. State: verified.
- MA-005: one long-running agent loop produced an ordered, handoff-backed batch
  that remained green after integration and production deployment. State:
  partially verified; repository and deployment facts are verified, but model
  duration, token use, and autonomous-session provenance were not independently
  reconstructed.
