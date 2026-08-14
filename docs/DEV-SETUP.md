# Dev Environment Setup

How to stand up a full Dark Pawns development environment from scratch — the Go
port **and** the C oracle differential harness — on a Linux workstation.

This is the reproducible replacement for tribal knowledge that used to live only
on the mac-mini. If you are an agent or a future maintainer picking this up on a
fresh machine, start here.

## What you're setting up

Three moving parts:

1. **The Go port** — this repo. The thing under development.
2. **The C oracle** — [`zax0rz/darkpawns-c-oracle`](https://github.com/zax0rz/darkpawns-c-oracle)
   (private), a deterministic fork of the original C server. The *answer key*.
   The differential harness diffs the Go port's output against it.
3. **The deploy path** — production runs on a Linux LXC (CT 120). See
   [`DEPLOYMENT.md`](../DEPLOYMENT.md).

## Prerequisites

```bash
# Go (see the `go` directive in go.mod for the exact minimum; currently 1.26.5)
go version

# Build toolchain for the C oracle (autotools + a C compiler)
sudo apt-get install -y build-essential autoconf automake gcc make git

# GitHub CLI (for cloning the private oracle repo and for gh-based workflows)
gh auth status
```

## 1. Clone the Go port

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build ./...        # sanity check the toolchain
go test ./...         # unit tests should pass with no oracle present
```

The unit tests do **not** need the C oracle. The differential harness does.

## 2. Build the C oracle

The oracle is a separate repo so its DikuMUD-derived history and license stay
cleanly bounded. Its default branch, `dp-oracle-seam`, carries two determinism
commits (`DP_SEED` / `DP_CLOCK` / `DP_FIXED_TIME`) on top of upstream — that seam
is the entire reason a run is reproducible. See that repo's README for detail.

```bash
# Clone next to the Go repo (any location works; you'll point DP_ORACLE_BIN at it)
cd ..
gh repo clone zax0rz/darkpawns-c-oracle
cd darkpawns-c-oracle

# Native build on Linux. Two CFLAGS are load-bearing (notes below):
#   -fcommon : the vintage C has duplicate tentative globals (e.g. buf2) that
#              GNU ld rejects by default; macOS ld coalesces them, so this is
#              only needed on Linux.
#   -O0      : deliberate — build the fixture oracle unoptimized. See below.
./configure CFLAGS="-g -O0 -fcommon"
make
file bin/circle       # expect: ELF 64-bit LSB executable, x86-64
```

`bin/circle` needs its sibling `lib/` tree (world files, text, player dir) to
boot. Keep them together as cloned. The repo ships a baseline `lib/etc/players`;
without it the first character created is auto-crowned Implementor (stock
CircleMUD), and every mortal scenario silently diverges (oracle sees a god, the
Go port sees a mortal).

### Why -O0 — read before you "optimize" the fixture

The DikuMUD-derived C carries a lot of latent undefined behavior: self-aliasing
`sprintf(dst, "…%s…", dst, …)` appends (~180 sites), `1 << 31` signed shifts,
etc. At `-O0` — and on the reference arm64/clang build — these are tolerated; at
`-O2`/amd64 the optimizer exploits them into arch-specific Heisenbugs that
surface as spurious differential divergences (the first one found: a fresh
char's prompt dropping its H/M segments to render move-only `NNV >`). This is a
*test* oracle: runtime speed is irrelevant, and determinism is seed-driven, so
opt level cannot affect parity. Building at `-O0` reproduces arm64's UB-tolerance
across all sites at once. Hot paths still get real source fixes as they surface
(belt and suspenders); `-O0` is the blanket mitigation, not a cure.

### If `make` fails wanting `aclocal-1.14`

git checkout can set the autotools file mtimes so `make` tries to regenerate
`aclocal.m4` with an ancient automake. The committed `configure` / `Makefile.in`
are valid — just refresh their timestamps (there is no maintainer-mode switch):

```bash
find . \( -name configure.ac -o -name Makefile.am \) -exec touch {} +
sleep 1; touch aclocal.m4
sleep 1; touch configure; find . -name Makefile.in -exec touch {} +
```

## 3. Point the harness at the oracle

The harness finds the oracle via `DP_ORACLE_BIN`. Export it (add to your shell
profile so it persists):

```bash
export DP_ORACLE_BIN="$HOME/darkpawns-c-oracle/bin/circle"   # adjust to your path
```

If `DP_ORACLE_BIN` is unset, the harness prints `SKIP: DP_ORACLE_BIN is unset`
and does nothing — that's the tell you forgot this step.

## 4. Verify parity end to end

From the Go repo, run a known-green scenario:

```bash
cd ../darkpawns
DP_ORACLE_BIN=$DP_ORACLE_BIN DP_SEED=1 DP_FRESH_MUD=1 \
  go run ./cmd/dp-oracle-diff --scenario mortal-batch14
```

- Exit code is **0 even on divergence** — do not trust it. Read the `result:` line.
- `result: no normalized divergence` → **green**, environment is correct.
- If instead you get a divergence, first suspect the oracle build (wrong branch,
  missing determinism commits) before suspecting the Go port.

### Harness invocation reference

| Env var | Meaning |
|---------|---------|
| `DP_ORACLE_BIN` | Path to the built `bin/circle`. Required. |
| `DP_SEED` | RNG seed; both servers use the same one. Conventionally `1`. |
| `DP_FRESH_MUD` | Boots an empty player file; with the god-harness fixture, crowns the first character `LVL_IMPL` to probe immortal commands. |
| `DP_CLOCK` / `DP_FIXED_TIME` | Set internally by the harness to freeze the clock/calendar. You don't set these by hand. |

Note: the harness **normalizes line terminators** (`\r\n` vs `\n\r`) before
diffing, so terminator-only infidelities pass green here. Byte-exact terminator
behavior is guarded by Go unit tests (e.g. `TestCanUseSkill_Audited_UnknownSkill_ExactMessages`),
not by the oracle. Trust the unit test, not the diff, for terminators.

## 5. Coverage reporting

```bash
make scenario-coverage        # writes docs/reports/scenario-coverage-<date>.tsv
```

**Always measure coverage against a clean checkout of `origin/main`**, never in a
working directory that has in-flight branch work — a stale HEAD undercounts and
manufactures phantom regressions. Use a throwaway worktree:

```bash
git worktree add /tmp/cov origin/main && cd /tmp/cov && make scenario-coverage
```

## 6. Deploy access

Production deploys (see [`DEPLOYMENT.md`](../DEPLOYMENT.md)) go over SSH to CT 120.
The deploy key must be present on this workstation:

- Copy `~/.ssh/id_ed25519_darkpawns` (+ `.pub`) from the old machine, **or**
- Generate a fresh key here and append its public half to
  `root@192.168.1.121:~/.ssh/authorized_keys`.

Confirm access: `ssh root@192.168.1.121 'systemctl is-active dark-pawns.service'`
should print `active`.

On a Linux workstation the deploy build is **native** (`go build`), not a
cross-compile — DEPLOYMENT.md documents both paths.

## 7. Continuity notes (optional, for the maintaining agent)

Persistent working memory from earlier sessions lives outside the repo, under the
Claude Code project directory keyed by the repo's absolute path. When the repo
path changes (new machine, new home dir), that memory won't auto-follow. To carry
it over, copy the old `memory/` contents into the new project's memory directory
after the first launch here. This is convenience, not correctness — everything
load-bearing is captured in this repo and the oracle repo.
