# Depth handoff — 2026-08-30 — `embrace`

## Frontier and queue position

- Started from clean `main` at `df554a475` after the merged `drool` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and read
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff,
  `2026-08-30-command-drool.md`.
- The frontier before this slice was 1,618 total, with 1,563
  proven/delegated, 14 blocked, and 41 excluded. This slice adds nine
  proven cases; the post-slice frontier is 1,627 total, 1,572
  proven/delegated, 14 blocked, 41 excluded, with actionable completion
  1,572/1,586 (99.1%).
- The source-order command gap was `embrace`, registered at
  `src/interpreter.c:431`. The next un-manifested command-table family is
  `enter` at `src/interpreter.c:432`; `eat`, `echo`, and `emote` are already
  manifested before this slice.

## C call path and branch inventory

`src/interpreter.c:431` registers `embrace` with `POS_STANDING` and
`do_action`. Its record is `lib/misc/socials:201-209`:

```text
embrace 0 0
You reach but come away empty.  :(
$n reaches out for an embrace, but no one is there.
You embrace $M warmly.
$n embraces $N warmly.  
$n embraces you warmly.  
Alas, your embracee is not here.  
You embrace yourself??  
$n wraps $s arms around $mself for a warm self-embrace.
```

The actual `src/act.social.c:102-151` order is the PLR_NOSHOUT early return,
record-driven `one_argument` parsing, no-argument actor/room pair, visible
target lookup, not-found actor line, self-target actor/room pair,
minimum-victim-position check, and target actor/room/victim trio. This record
has `hide=0` and minimum victim position zero, so a sleeping target is admitted
to the target branch while its plain `TO_VICT` private line remains suppressed
by `comm.c:2480-2555`; the command's `POS_STANDING` gate is handled before the
social handler.

## Coverage proof and result

The C-first `cmd/dp-oracle-diff/scenarios/embrace-depth.txt` vehicle covers the
no-argument, leading fill-word target, unknown target, self target, standing
target, sleeping target, sitting position rejection, and stand/wake recovery
branches. Seed 1 was run with `--show-oracle`, exposing the full target matrix
and the exact two-space record punctuation. The separate
`embrace-noshout.txt` vehicle sets PLR_NOSHOUT through the registered `mute`
path and proves the refusal precedes target lookup and all audience output.

Both vehicles are GREEN for seeds 1, 2, 3, 5, and 8; no Go divergence was
confirmed, so no production code was changed. The durable inventory is
`docs/fidelity/depth/embrace.tsv`, with nine cases covering the record-specific
branches and registered command gates.

## Gates

On `glm/depth-embrace`:

- `make fidelity-depth` — PASS
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS
- `golangci-lint run ./...` — PASS, 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean

This coverage slice follows R1/R2/R4 and R5e: C bytes, command registration,
the actual social dispatch path, and the source record remain authoritative;
no behavior was invented; and neither `src/` nor the C oracle tree was edited.
The next session should checkout `main`, pull, confirm the frontier, and take
`enter` in interpreter-table order.
