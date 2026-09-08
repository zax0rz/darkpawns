# Repair validation — 2026-09-08

- `go build ./...`, `go vet ./...`, `go test ./...`: passed.
- `golangci-lint run ./...`: 0 issues (existing stale-cache warning).
- `make fmt`, `make hooks`: completed.
- Explicit local PostgreSQL `go test ./pkg/session -run '^TestEntry' -count=1 -v`: passed; see persistence-green.txt. Without DP_ENTRY_TEST_DATABASE_URL, integration tests skip.
- `node --test scripts/entry-client.test.mjs`: 4 passed, 0 failed. Handler contracts use terminal/WebSocket doubles, not full browser byte proof.
- `make build-site`: passed; after bundling pinned xterm, `npm run build` in website-astro passed.
- C oracle retry scenario: seeds 1,2,3,5,8, no normalized divergence; see oracle-green.txt.
- Built page at localhost:14350/play/ with disposable PostgreSQL: aiko -> N -> Aiko -> creation -> menu -> world succeeded. Reload, AIKO -> saved password -> menu -> world succeeded. Password was not visibly echoed. First local character was God, so mortal browser flow remains unproven.
- Full oracle census, complete entry branch matrix, and production deployment were not performed.

Commands and the remaining branch inventory are in the incident brief and depth/entry.tsv. No production records were modified.
