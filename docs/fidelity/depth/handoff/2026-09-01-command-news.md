# Depth-fidelity handoff — `news`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R4, R5b, R5c, R5e

## Frontier

This session started on clean, pulled `main` after rereading
`docs/fidelity/DEPTH_TESTING.md` and the newest `neckbreak` handoff.  The
starting frontier was 2,634 cases: 2,564 proven/delegated, 22 blocked, and 48
excluded (99.1% actionable).  After `news` was merged, fresh `main` reports:

```text
Cases: 2638 total, 2568 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2568/2590 = 99.2%
```

The source-order sweep confirms the next actually unclaimed command row is
`newbie` at `src/interpreter.c:566` (`POS_SLEEPING`, `do_gen_comm`,
`SCMD_NEWBIE`).  The later `nibble` and `nod` rows are still queued after it;
previous family claims for `mail`, the social aliases `muhaha`/`mumble`, and
`murder` remain unchanged.

## C path and proof

The registered C row is `{ "news", POS_SLEEPING, do_gen_ps, 0, SCMD_NEWS }`
at `src/interpreter.c:565`.  `src/act.informative.c:2117-2158` selects the
`SCMD_NEWS` case and calls `page_string(ch->desc, news, 0)` without examining
the command argument.  `src/db.c:315-316` boot-loads the text, with
`NEWS_FILE` defined as `text/news` in `src/db.h:51`; the authoritative
`lib/text/news` file is a short single-page static text.

The existing Go path already matched that call path: the registered `news`
command uses `cmdNews`, which calls `sendCachedText(s, "news")`; that loader
uses the world-derived `LibTextDir`, caches the bytes, and routes them through
the shared `PageString`.  No player-visible Go behavior needed changing.  The
proof therefore records evidence only, preserving R1/R4 and avoiding an
invented implementation difference.

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/news-depth.txt` — bare `news` and trailing
  arguments, with `# depth-case` annotations for both intended C blocks.
- `pkg/session/news_depth_test.go` — C entry-gate and registered-command
  identity.
- `docs/fidelity/depth/news.tsv` — four rows covering entry, static page,
  ignored arguments, and delegated pager routing.

The `news-depth --show-oracle --seed 1` run reached both intended blocks and
showed the exact 12-line static page for bare and trailing-argument probes.
Seeds 1, 2, 3, 5, and 8 all returned no normalized divergence.  No `src/` or
C-oracle files were edited (R1-R5e).

## Gates and merge

Local gates passed on `glm/depth-news`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean

Feature PR #1040 (`glm/depth-news`) initially reported no checks.  The one
permitted retry, `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-news`,
started the workflow; hosted `lint`, `security`, and `test` then passed, with
build/deploy skipped by the PR workflow.  The PR was self-merged under the
2026-08-27 amendment as squash commit `f464dfb48`.

The post-merge `main` checkout/pull and frontier rerun passed with the counts
above.  The next session must start on `main`, pull, rerun
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this
handoff, then map and attempt only `newbie` at `src/interpreter.c:566`.
