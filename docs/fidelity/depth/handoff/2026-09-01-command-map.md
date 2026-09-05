# Depth handoff — 2026-09-01 — `map`

## Queue position and frontier

This session continued the command-family queue after the merged `luaedit`
scope slice. The special-procedure inventory remains exhausted. The one
blocked `objmagic.sleep-entry-gates` row was already attempted once through
the cast-sleep outlaw/reagent vehicle and remains blocked; this session did
not repick or relabel it. The next un-manifested interpreter-table family was
`map` at `src/interpreter.c:548`.

After this slice, `make fidelity-depth` reports **2,510 total cases, 2,447
proven/delegated, 18 blocked, and 45 excluded**; actionable completion is
**2,447/2,465 = 99.3%**.

## C-first path and findings

The registered row is `{ "map", POS_SLEEPING, do_map, 0, 0 }`. The verified
path is `command_interpreter` → `any_one_arg` → `do_map` in
`src/mapcode.c:141-218`. The interpreter passes `do_map` the pointer returned
by `any_one_arg`, which still points at the delimiter space. Consequently the
compiled C handler's `argument[1]` selects `a`, `b`, or `c` only for commands
typed as `map a`, `map b`, or `map c`; `map xa`, `map xb`, and `map xc` all take
the default mode. This call-path detail was the cause of the initial RED
probes and is recorded under R2/R5e rather than inferred from the command
comment.

The handler also has the direct mortal help gate, a 76×25 display, recursive
N/E/S/W traversal with collision modes 0/1/2/3, up/down markers, the legacy
post-loop `dir == 4` overlap access, and the exact C key legend. C's
`renum_world()` in `src/db.c:951-961` resolves exit targets to RNUMs before
`map()` runs; ordinary Go gameplay retains VNUMs, so the map now builds a
private RNUM-indexed view and resolves missing positive targets to `-1` at the
same boundary.

## Port and proof result

The map implementation now:

- emits the exact C legend bytes;
- treats exit presence as the C pointer-presence test, including exits whose
  target is `-2` or lower;
- uses the C boot-time RNUM identity for traversal and sector rendering;
- preserves the compiled oracle's observed `offx[4]/offy[4]` overlap marker
  behavior (`x-1`, `y-2` on this build); and
- maps explicit options using the normalized equivalent of C's
  delimiter-preserving argument pointer.

Added six rows to `docs/fidelity/depth/map.tsv` and six disposable oracle
vehicles: `map-gate`, `map-depth`, `map-all-depth`, `map-overlap-depth`,
`map-underlap-depth`, and `map-unknown-depth`. All six are GREEN. The complete
`map-depth` matrix matched on seeds **1, 2, 3, 5, and 8**; the `map a`
vehicle was also run with `--show-oracle` to verify that the intended C block
executed. No `src/` or `darkpawns-c-oracle/` file was edited.

Local gates passed:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (`0 issues`)
- `gofumpt -l .` clean
- `git diff --check`

Feature PR [#1016](https://github.com/zax0rz/darkpawns/pull/1016) passed hosted
lint, security, and test checks; build/deploy were correctly skipped. It was
self-merged only after all required checks were green as
`e7002a2e2` (`feat: deepen map command fidelity`).

## Next item

Return to a fresh `main` checkout and frontier check, reread
`DEPTH_TESTING.md` and this handoff, then take the next un-manifested
interpreter-table family: **`mindlink`**, registered immediately after `map`
at `src/interpreter.c:549` as `do_mindlink`, `POS_STANDING`, no minimum level.

