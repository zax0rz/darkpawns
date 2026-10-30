# Docs media acceptance evidence — 2026-10-29

The capture implementation and usage are described in [MEDIA.md](../../MEDIA.md). All 14 published assets are in [the canonical folder](../../images/generated/), with per-image alt text, source lists and release in its manifest. Source provenance truthfully records a development worktree rather than claiming a clean tagged build.

## Verification

Two complete, sequential `make media` runs exited 0. `repeatability.json` records their PNG SHA-256 hashes, decoded-pixel comparisons, and GIF frame counts/durations. Original outputs and logs are retained at `/home/zach/Archives/darkpawns/media-release-proof/run-a/` and `run-b/` on the capture host. Earlier failed comparisons are retained separately under `media-ship/`, rather than overwritten.

Reproducibility fixes came from actual differences: Operations and Dashboard crops included pixels from neighboring live metrics; their padding now excludes those panels. The room screenshot focuses the field editor instead of its ticking claim lease. Captures wait for fonts before measuring panel outlines and use software Chromium rendering with fixed color handling. Tiny antialiasing variations at rounded panel corners persisted; the dashboard crop now sits inside those decorative edges, and the zones capture focuses the table rather than the create-zone form. The annotation ring is inset to remain visible. Mudlet's Qt hash iteration changed overlapping room draw order; its isolated process now sets `QT_HASH_SEED=0`. The screenshot remains the real native Mudlet renderer. Browser captures settle the web client’s smooth scroll before measuring its crop, and comparison checks dimensions as well as pixels. Archive metadata and VCS build stamping are fixed so the update-check digest does not change merely because generated files dirty the checkout.

The editor GIF uses an explicit 18-frame sequence. VHS records live terminal timing; its measured frame counts and durations are recorded alongside the hashes. The three recordings run the actual setup, status/doctor and offline update-check commands. They never apply an update.

The staleness negative check advanced `admin-ui/src/pages/DashboardPage.tsx`'s mtime past its image, ran `make media-check`, asserted both the expected `newer source` warning and exit 0, then restored the original mtime. `staleness-touch.log` contains the output. This proves the advisory check can flag its intended condition (R5h).

## Gates and visual review

Each of `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...` and `golangci-lint run ./...` passed separately. The root agent ran the Go gates itself; lint used an isolated cache. `make deadcode-ratchet`, `make site-check` and the Astro build also passed. Logs are retained under `/home/zach/Archives/darkpawns/media-ship/` and `/home/zach/Archives/darkpawns/docs-media-*.log`.

Captures use synthetic preview identities or new disposable Guest sessions. No live player data or production connection is used. The terminal examples use a generic temporary path, an explicitly public capture-only signing key and a local release candidate. The image manifest's alt text is under 150 characters for every entry. All GIFs loop, have no audio and last under 15 seconds.

No census or deployment was run: this changes documentation, capture tooling and an admin-preview clock seam, with the production clock retaining its existing real-time behavior (R1).

## Measured GIF timing

| Recording | Frames A / B | Duration A / B (ms) |
| --- | --- | --- |
| admin-player-editor.gif | 18 / 18 | 3000 / 3000 |
| install-and-setup.gif | 198 / 196 | 7920 / 7840 |
| lifecycle-commands.gif | 166 / 166 | 6640 / 6640 |
| update-check.gif | 171 / 171 | 6840 / 6840 |
