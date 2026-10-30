# Regenerating documentation media

Run `make media` from the repository root. It creates the ten PNGs and four looping GIFs in `docs/images/generated/`; that directory and its `manifest.json` are the shared source for the README and documentation site. Commit the generated files with capture changes.

The capture host is Linux x86-64 with Go, Node/npm, Python 3, curl, tar, apt/dpkg, VHS, ffmpeg and ffprobe available. The bootstrap installs the admin UI's locked npm dependencies and Playwright Chromium when absent. It downloads the SHA-256-pinned official Mudlet 5.0.2 AppImage and extracts Xvfb/xdotool packages under `.local/`. It does not install system packages. Mudlet and Xvfb still need their normal host shared libraries; bootstrap reports missing libraries.

Keep localhost ports 4350, 5187, 4360, 7780, 8088 and 4371 and X display :88 free. Captures refuse occupied ports. The scripts start disposable preview/game processes and stop only their own process groups. CLI recordings stage in `/tmp/darkpawns-docs-demo`; an ownership marker prevents replacing unrelated content, and a failed shutdown preserves the instance for diagnosis.

## What the captures show

The admin preview uses a fixed clock, synthetic characters, login/death history and logs. Its routes use the current console components. Browser captures share `crop`, `highlight`, `badge` and `gif` helpers in `admin-ui/scripts/media-capture.mjs`. The editor animation holds, zooms and pulls back; PNGs focus the relevant panel with the site's oxblood outline.

The browser client and official Mudlet connect to a disposable local game. The Mudlet image shows its native dock and map. `QT_HASH_SEED=0` fixes Qt's overlapping map-room draw order. The game output is not rewritten for screenshots (R1).

Three VHS tapes record real portable setup, status/doctor and an offline `update --check`. The capture bundle has fixed archive metadata and explicitly public test-only signing keys from `scripts/fixtures/`. Those keys are used through a build overlay for this disposable CLI; they must never sign a real release. The update recording verifies a local candidate without applying it. Builds disable VCS stamping so Git's dirty bit cannot change the displayed demo archive hash.

## Checking freshness and repeatability

Run `make media-check`. It warns about missing outputs, invalid manifest entries, missing sources or source mtimes newer than a capture. This check is deliberately advisory, including in CI. A fresh checkout can give source files newer mtimes; regenerate before treating that warning as proof of visual drift.

Each manifest entry declares its alt text (under 150 characters), source files and release. The manifest also retains the source Git HEAD and whether the working tree was dirty. It records provenance rather than claiming that a development capture came from a clean release.

To check reproducibility, save `docs/images/generated/` after one run, run `make media` again, and compare PNG SHA-256 hashes. The editor GIF uses explicit frames. VHS records a terminal process in real time, so terminal GIF frame counts and durations can differ slightly; inspect decoded content as well as hashes. The dated evidence report records the measured comparison for this implementation.
