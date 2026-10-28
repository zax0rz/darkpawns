# CLI polish verification

Base: `8dda7d6b5` (including the merged dashboard and settings work).
Before launcher: `45002c6a9`; `cmd/darkpawns` is unchanged between those bases.

## Gates

The final source passed each gate individually, in order:

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`

Additional checks: `go test -race ./cmd/darkpawns`, `make deadcode-ratchet`,
`git diff --check`, and the opt-in `clipreview` executable build.

`scripts/cli_plain_compare.py` compares old/new launchers at the same executable
and instance paths. Seven cases match exact stdout, stderr and exit code: status,
status JSON, doctor, doctor JSON, update check, local-archive check and setup plain.
The test suite additionally proves non-TTY `status --watch` is the old single
snapshot, including its JSON form. It protects anchor dispatch when `--plain`
appears before setup/init and checks that JSON never enters rich presentation.

Unit tests exercise 40/80/120-column layouts, navigation/follow/focus, sanitized
bounded tails, actual in-memory cookie-to-bearer reads, configured-loopback and
redirect guards, measured/confirmed stage transitions, and cancellation waiting
for work to return before a plan can be closed.

## Visual evidence

VHS tapes: `scripts/vhs/cli-before.tape`, `cli-after.tape`, `cli-picker.tape`,
`cli-compact.tape`. Dark wide and light compact captures cover status, players,
log viewport, keyboard help, progress, filtering, confirmation and rendered notes.
The doctor segment uses real diagnostics on a disposable stopped instance; its
missing signed release deliberately produces repair checklist failures.

The host tool shell sets `NO_COLOR=1` and `TERM=dumb`, so rich tapes explicitly
unset NO_COLOR and set a colour terminal. These settings affect the rehearsal
shell only. The normal plain fallback is separately compared against the base.

UI fixtures are opt-in, build-tagged and labelled synthetic. No live world,
credentials, update transaction or service registration is mutated for captures.
The test executable is excluded from release builds. Tapes and images are retained
at `/home/zach/Archives/darkpawns/cli/2026-10-27/cli-polish/` with hashes and logs.

## Scope and limits

`internal/portable` and every installer script have no diff. Presentation observes
existing staging files/journals and invokes the same PrepareUpdate/Apply/Close,
Doctor, Status, Start, Stop and Restart APIs. It never writes their state or
reimplements signatures, backup, rollback or health decisions.

Remote archive totals are not guessed. Verify/stage remain indeterminate until
PrepareUpdate returns; health is not marked passed for a stopped instance.
Matching remote release notes are informational. Local archives remain offline.

No player output, draw, command, save format or game behavior changes (R1/R3/R4).
No oracle census is required for this launcher-only presentation change. Claude
owns the final v0.1.7 combined release gate.

Decisions for Zach: none blocking. Finer verify/stage percentages would require
future progress telemetry from GLM's updater. This PR keeps that boundary intact.
