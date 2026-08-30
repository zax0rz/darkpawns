# Depth handoff — `dns` command

Date: 2026-08-30
Queue slice: `src/interpreter.c:418`, `dns` / `do_dns`
Starting main: `a5922b11c`
Merged main: `1c652f561` (PR #840)

## Queue decision

The special-procedure inventory remains exhausted and the one permitted
`objmagic.sleep-entry-gates` cast-sleep attempt remains blocked. The command
table sweep advanced from `display` to the next un-manifested family, `dns`.
The next un-manifested family after this slice is `doh` at
`src/interpreter.c:419`.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## C path and proof

The command table registers `dns` at `src/interpreter.c:418` with
`POS_DEAD` and `LVL_IMPL-1`, dispatching to `do_dns` in
`src/act.wizard.c:3106-3208`. The handler uses `half_chop` for the
subcommand, `is_abbrev` for `delete`/`add`/`list`, `sscanf` for the first
three or four IP octets, and `two_arguments` for the add address/name pair.
Defined branches are:

- empty subcommand warning;
- delete missing/malformed address;
- delete by matching the first three octets, marking every matching cache
  entry and emitting each hostname;
- add with missing arguments, three-octet `ip[3] = -1`, or four-octet address,
  prepending into `(ip0+ip1+ip2) % 257`, saving `etc/dns`, and acknowledging;
- unknown non-empty subcommand falling through silently.

The clean-main vehicle was RED for every defined command branch: C emitted
the DNS-specific bytes while the absent Go registration returned `Huh?!?`.
After the scoped port, the vehicle was GREEN at seed 1 with `--show-oracle`,
including `del` abbreviation, lowercased host names, prepend order, and
prefix-delete output. `TestExecDNSAddDeletePrefixAndPersistence` proves the
three-octet fourth-octet sentinel, bucket order, and deletion state.

The reachable `list` branch is explicitly excluded. C builds its output with
overlapping `sprintf(buf, "%s...", buf, ...)` source/destination arguments,
and this vehicle's empty `etc/dns` also reaches `boot_dns`'s uninitialized
EOF-buffer path. The shipped Linux oracle therefore emits unstable garbage
for that branch. This is the same explicit undefined-behavior treatment used
by `auto.no-arg-listing-undefined`, under R1/R5e, rather than inventing a
stable player-facing contract.

This follows R1/R2/R3/R4 and R5/R5e: exact defined bytes, authoritative
command gating and parsing, deterministic cache state, no invented list
contract, and verification of the actual C dispatch path.

## Evidence and gates

Added or changed:

- `cmd/dp-oracle-diff/scenarios/dns-depth.txt`
- `docs/fidelity/depth/dns.tsv`
- `pkg/game/dns.go`
- `pkg/game/dns_test.go`
- `pkg/game/world.go`
- `pkg/session/cmd_misc.go`
- `pkg/session/commands.go`

Focused proof:

```
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario dns-depth --seed 1 --show-oracle
```

The full local gates passed:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The post-manifest frontier is 1,585 total, 1,530 proven/delegated, 14
blocked, and 41 excluded; actionable completion is 1,530/1,544 (99.1%).

## Hosted review

PR #840 (`glm/depth-dns`) passed hosted `lint`, `security`, and `test`; the
build-and-push and deploy jobs were skipped by workflow policy. It was merged
only after all required hosted checks were green.

## Carry-forward

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then take `doh` in interpreter-table order.
