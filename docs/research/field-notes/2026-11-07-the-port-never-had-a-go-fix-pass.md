# The port never had a `go fix` pass

2026-11-07, Claude. A field note, not a finding. The cause is a hypothesis.

## What was observed

A Muse agent dry-ran `go fix` over `origin/main` at `4a2d74b77` with Go 1.27.3. Its report is archived at `~/Archives/darkpawns/notes/go127-dryrun-2026-11-07/dp-go127.md`.

- With `go.mod` left at the declared `go 1.26.10`, `go fix` proposes **703 hunks in 252 files**. None of them needs Go 1.27.
- Raising `go.mod` to 1.27.3 adds only 42 hunks in 11 files, almost all `embedlit`.
- The bulk is old modernizers:
  - `interface{}` → `any`: about 399 hunks. `git grep` counts 539 `interface{}` occurrences in `*.go` on the same commit.
  - three-clause loops → `for i := range N`: about 297
  - `sync.WaitGroup` boilerplate → `wg.Go`: about 31
  - `strings.Index`/`HasPrefix` → `Cut`/`CutPrefix`: about 33
- The report says it found no bug-class issues, only modernization.

## Why it might matter

The port's code is written in idioms several releases older than the toolchain it declares and builds with. `any` arrived in Go 1.18 and range-over-int in 1.22. The repository has never run `go fix`, so the gap is accumulated, not introduced recently.

Hypothesis, untested: the code-writing models reproduce the idiom vintage of their training data, not the declared toolchain's. And nothing in the harness (CI, lint, or the oracle) pushes back, because the old idioms are correct. The oracle checks bytes, not style.

## How to test it

- Date the old idioms with `git log -S`: did code written in 2026 by recent models still introduce `interface{}`?
- Compare across agents. The commit trailers name the lab.
- After the planned post-release pass, compare census results before and after. That's the idiom-change equivalent of the round-two idiom experiment (PR #2276).
