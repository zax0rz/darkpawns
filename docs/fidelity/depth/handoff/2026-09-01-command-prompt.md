# Depth-fidelity handoff — `prompt`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `pout` to
`prompt`.

The next unmanifested interpreter family is `puke` at
`src/interpreter.c:621`; `pray` at line 620 is already covered.

Frontier before this slice: 2,861 total; 2,790 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,869 total; 2,798 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:619 */
{ "prompt"   , POS_DEAD    , do_display  , 0, 0 },
```

`src/act.other.c:1024-1082` implements the handler. It rejects NPCs, skips
leading argument whitespace, emits the exact usage line for an empty
argument, treats `on` and `all` as the five-display-flag enable branch, clears
all five flags before the general letter loop, returns silently for `off`,
and otherwise enables recognized h/m/v/t/f letters before sending `Okay.`.
The command is available at every position with no authority gate; shared
entry-gate details are pinned by the focused test.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/prompt-depth.txt`

Manifest: `docs/fidelity/depth/prompt.tsv` (8 rows)

Test: `pkg/session/prompt_depth_test.go`

The untouched-main scenario was RED: C returned `Okay.` for `all`, `on`,
`hmv`, and `none`, and was silent for `off`; Go instead emitted custom
prompt-string messages. The focused test then proved the five C display-bit
state transitions, including selective `hmv` and silent `off` clearing.

The confirmed divergence was fixed by routing the Go `cmdPrompt` command
through the existing C-faithful `ExecDisplay`/`doDisplay` implementation and
removing the custom prompt-string responses from this command surface. The
seed-1 `--show-oracle` vehicle and matrix at seeds `1,2,3,5,8` were GREEN with
no normalized divergence. No C source or oracle checkout was edited.

## Verification and integration

All local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-prompt`

Feature commit: `b846d0e12` (`fix: align prompt depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1086 — merged; main commit `b7e650ddb`. Hosted lint, security,
and test checks were green; build-and-push and deploy were skipped by
repository workflow conditions. The PR was merged only after the required
hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
