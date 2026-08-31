# Depth-fidelity handoff: `imotd`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family
sweep after the special-procedure inventory and the one-time
`objmagic.sleep-entry-gates` attempt. The pre-slice frontier was 2,265 total
cases: 2,205 proven/delegated, 16 blocked, and 44 excluded (2,205 of 2,221
actionable cases, 99.3%). The four new `imotd` manifest rows bring the frontier
to 2,269 total: 2,209 proven/delegated, 16 blocked, and 44 excluded (2,209 of
2,225 actionable cases, 99.3%).

The special-procedure inventory remains exhausted and the explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
`imotd` feature slice was PR #964 on `glm/depth-imotd`, merged to `main` as
`787e11ffc` after all hosted checks were green. The prior howl handoff PR #953
has since also completed with green test, lint, and security checks. No
non-green PR was merged.

The next source-order unclaimed command is `immlist` at
`src/interpreter.c:514`.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:513: { "imotd", POS_DEAD, do_gen_ps, LVL_IMMORT, SCMD_IMOTD }
```

`src/act.informative.c:2117-2158` dispatches `SCMD_IMOTD` to
`page_string(ch->desc, imotd, 0)` without inspecting the argument. During
`boot_db`, `src/db.c:319` loads `IMOTD_FILE`, defined as `text/imotd` in
`src/db.h:53`. The authored `lib/text/imotd` is shorter than one pager page,
so its only ordinary success path sends the exact cached bytes without a
pager prompt. A player below `LVL_IMMORT` is rejected by the interpreter
with `Huh?!?` before `do_gen_ps` runs.

The audited player-visible cases are:

- the `LVL_IMMORT` and `POS_DEAD` entry gate;
- the mortal level gate and its early-return bytes;
- the immortal static page and its exact audience bytes; and
- ignored trailing command arguments.

The shared pager is not a distinct reachable `imotd` branch because the fixed
file fits one page. R1/R2/R3/R4 and the C-first call-path rule R5e apply. No
Go behavior change was confirmed or made; shared page behavior remains owned
by the existing pager/help evidence under R5b/R5c.

## RED/GREEN result

This was a pure-coverage round. The branch started from clean `main`, and the
Go implementation diff was empty before the proof run. The unchanged path was
therefore GREEN on main-equivalent code:

- `imotd-gate-depth --show-oracle --seed 1`: no normalized divergence; the
  oracle block is `Huh?!?`.
- `imotd-depth --show-oracle --seed 1`: no normalized divergence; bare and
  trailing-argument probes match the C static page byte-for-byte.
- `imotd-depth` is also green at seeds 2, 3, 5, and 8.
- `TestImotdRegistrationUsesCEntryGate` passes.

The durable proof is `cmd/dp-oracle-diff/scenarios/imotd-depth.txt`,
`cmd/dp-oracle-diff/scenarios/imotd-gate-depth.txt`,
`pkg/session/imotd_test.go`, and `docs/fidelity/depth/imotd.tsv`. No file under
`src/` or `darkpawns-c-oracle/` was edited.

## Verification

The complete local gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a clean
`gofumpt -l .` check. The hosted `test`, `lint`, and `security` checks for
PR #964 were green before merge; build and deploy were skipped by repository
workflow policy.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread the depth
testing guide and this newest handoff, then take only the unclaimed `immlist`
family at `src/interpreter.c:514`. Continue the command-table sweep in source
order with one slice and one PR.
