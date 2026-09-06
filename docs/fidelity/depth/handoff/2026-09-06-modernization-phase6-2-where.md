# Modernization Phase 6.2 — `cmdWhere` builder

Date: 2026-09-06  
Scope: one proven loop-concatenation target from roadmap item 6.2

## Change

Replaced `cmdWhere`'s repeated string concatenation in the no-argument player
listing with `strings.Builder` and `fmt.Fprintf`. The header, fixed-width
player rows, empty-list fallback, ordering, and final send are unchanged.

The targeted loop is a pure output-buffering change. The argument-search branch
was left untouched, as were the command gate and session ordering logic (R1,
R2, R3, R5e).

## Fidelity basis

- C source path: `src/act.informative.c:2253-2280`.
- Existing depth rows: `docs/fidelity/depth/info.tsv`, covering the no-argument
  listing, descriptor-order parity, mortal gate, and argument path.
- Go path: `pkg/session/cmd_info.go`, `cmdWhere`.

No player-facing bytes or state semantics are changed. This is the first small
proven slice of the broader Phase 6.2 string-cleanup wave; unrelated loops and
unproven output paths remain out of scope (R4).

## Verification

Focused unit test:

```text
/usr/local/go/bin/go test ./pkg/session       PASS
```

Focused oracle matrix:

```text
info-basic              PASS
info-where-immort       PASS
where-immort-zone-arg   PASS
```

Oracle summary: 3 scenarios, 3 passed, 0 failed, 0 infrastructure failures,
0 unpinnable, 0 timed out.

Full repository gates remain required before merge:

```text
/usr/local/go/bin/go build ./...
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
PATH=/usr/local/go/bin:$PATH golangci-lint run ./...
make fidelity-depth
python3 scripts/gen_expected_divergences.py --check-pins
git diff --check
```
