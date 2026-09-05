# Depth handoff — 2026-08-30 — `drool`

## Frontier and queue position

- Started from clean `main` at `85caf99bb` after the merged `dream` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and read
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff,
  `2026-08-30-command-dream.md`.
- The frontier before this slice was 1,610 total, with 1,555
  proven/delegated, 14 blocked, and 41 excluded. This slice adds eight
  proven cases; the post-slice frontier is 1,618 total, 1,563
  proven/delegated, 14 blocked, 41 excluded, with actionable completion
  1,563/1,577 (99.1%).
- The source-order command gap was `drool`, registered at
  `src/interpreter.c:425`. The preceding `drink` and `drop` families are
  already manifested. The next un-manifested command-table family is
  `embrace` at `src/interpreter.c:431`; `eat`, `echo`, and `emote` between
  them are already manifested.

## C call path and branch inventory

`src/interpreter.c:425` registers `drool` with `POS_RESTING` and
`do_action`. Its record is `lib/misc/socials:191-199`:

```text
drool 1 0
You start to drool.
$n starts to drool.
You drool all over $N.
$n drools all over $N.
$n drools all over you.
Pardon??
Sure, go ahead and drool...yuk!
$n drools on $mself.  What a sight.
```

The registered command's position gate is handled before `do_action` by the
interpreter command path. Inside `src/act.social.c:102-151`, the C order is
the PLR_NOSHOUT early return, optional `one_argument` parsing, no-argument
actor/room pair, visible-target lookup, not-found actor line, self-target
actor/room pair, minimum-victim-position check, and target actor/room/victim
trio. The record's `hide=1` and `min_victim_position=0` mean a sleeping target
is admitted to the target branch, but plain `TO_VICT` does not deliver its
private line while asleep. `comm.c:2480-2555` supplies the recipient gates.

## RED, coverage proof, and result

The C-first `cmd/dp-oracle-diff/scenarios/drool-depth.txt` vehicle exercises
no argument, leading fill-word plus target, unknown target, self target, a
sleeping target, and the sleeping actor's command-position rejection. The
clean-main vehicle was byte-equal on seed 1; the `--show-oracle` run exposed
the intended complete target matrix, including the absent sleeping-victim
line.

The separate `drool-noshout.txt` vehicle uses the registered `mute` path to
set PLR_NOSHOUT on a mortal, then proves the refusal occurs before argument
lookup or audience output. It is also byte-equal on seed 1. The two vehicles
are GREEN for the main matrix seeds 1, 2, 3, 5, and 8; no Go divergence was
confirmed, so no production code was changed. The durable inventory is
`docs/fidelity/depth/drool.tsv`, with eight cases covering the complete
record-specific branch and gate surface.

## Gates

On `glm/depth-drool`:

- `make fidelity-depth` — PASS
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS
- `golangci-lint run ./...` — PASS, 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean

This coverage slice follows R1/R2/R4 and R5e: C bytes, the command surface,
the actual registered dispatch path, and the social record remain
authoritative; no divergence was invented; and neither `src/` nor the C
oracle tree was edited. The next session should checkout `main`, pull,
confirm the frontier, and take `embrace` in interpreter-table order.
