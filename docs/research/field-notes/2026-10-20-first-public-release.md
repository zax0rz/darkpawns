# The first public release, and a deploy that is one command

*Field note, 2026-10-20. Ledger: PF-081.*

Dark Pawns v0.1.0 was published at 12:10 UTC on 2026-10-20 from main `3c2f3e318`: portable archives for Linux, macOS and Windows on amd64 and arm64, `install.sh` and `install.ps1` bootstrap installers, and a signed `release-manifest.txt` (`release-manifest.sig`). An hour later, v0.1.1 (`88e44552e`, 13:22 UTC) added hosted instances and system supervision, and production moved onto it. Production's deploy procedure is now `ssh root@darkpawns.org darkpawns update`.

## What shipped

Seven PRs over about 18 hours, by Zach working with Sol (Codex):

| PR | What |
|---|---|
| #2110 | portable bundles and an isolated instance launcher |
| #2111 | guided setup from a new world to first login, with a browser handoff |
| #2112 | background lifecycle, `doctor`, native user startup (launchd, systemd, Windows) |
| #2113 | verified updates and snapshot recovery |
| #2114 | signed release publishing and the bootstrap installers |
| #2115 | installer polish and the v0.1.0 release notes |
| #2116 | hosted instances and update-aware supervision (v0.1.1) |

The publish job waits on a GitHub environment (`portable-release`) whose only reviewer is Zach. The v0.1.0 run (Actions run 39416454550) built and validated on all three platforms before that gate, and published only after his approval.

In September, this chain was the project's stated product thesis (install, then a setup wizard, then a page with a client, then a first login as an immortal) with its first two links missing: there was no installer and the wizard was a draft spec. Every link now exists and is published.

## What it rests on

The release sits on census-clean game code. The last two full combined draw-parity censuses before it, `~/Archives/darkpawns/oracle-runs/2026-10-18/dp-493-pr2106` (Go HEAD `2921e019c`) and `2026-10-19/sol-deploy-player-editor-310761c2a` (Go HEAD `310761c2a`), were each CLEAN_AFTER_RECHECK over 5,246 C-versus-Go pairs. Between `310761c2a` and v0.1.1, the game path (`pkg/game`, `pkg/combat`, `pkg/spells`, `pkg/session`, `lib/world`) changed only for notification hooks. These publish a death or creation event and emit no player output, draw no random numbers and change no game state (reviewed by Claude, 2026-10-20). v0.1.0 to v0.1.1 changed no game-path file. So the bytes a player sees in v0.1.1 are the bytes those censuses compared against the C original.

## The cutover

As Sol reported it (agent prose, not yet independently verified beyond the checks below): two version-to-version update rehearsals passed. The first production cutover rolled back automatically on a systemd quoting mistake, and the retry succeeded. All 19 player records, the JWT secret, boards and notification credentials were preserved. The old hand-built deployment and its rollback bundles were left intact.

Claude checked production read-only afterwards: `dark-pawns.service` is active and runs `/opt/darkpawns-portable/releases/darkpawns_0.1.1_linux_amd64/darkpawns supervise --instance /var/lib/darkpawns`, `darkpawns --version` reports `0.1.1 (88e44552e320…)`, and `darkpawns status` reports the instance running. The procedure is documented in `DEPLOYMENT.md` and `docs/portable-release.md`.

## Why it matters

Two things are worth recording.

The first is that the release did not need a release-specific fidelity pass. The batch census rhythm of the previous week (an agent's PRs, merged into one branch, one combined census, a fast-forward to main) meant main was already census-clean when the release was cut. The fidelity process and the release process met at main.

The second is the rollback. The first real cutover failed, on a quoting mistake of exactly the kind that hand-run deploys make, and the updater put production back by itself. The hand-built deploys of the previous weeks (a database backup with an integrity receipt, a binary swap, a restart, a checklist) carried the same safety by human discipline. The updater now carries it as code, and on its first production run it was the code, not a person, that caught the mistake.

## Open

- An independent check of a later `darkpawns update` against the old checklist: no players online at restart (or a warning), database integrity, listeners on 4350, 7777, 7778 and 7779, the banner, and new-process journal errors.
- Prod now runs tagged releases, not arbitrary main commits. Census-clean work reaches players only when a release is cut, so a batch now ends in a tag rather than a deploy.
