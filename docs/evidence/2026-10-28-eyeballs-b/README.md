# Eyeballs B: useful character activity

2026-10-28. This is B only; equipment and inventory are C. Network/ports (#10)
remain with GLM. Copy follows brand-voice §8.5.

## Browser proof

Chrome was unavailable in the connected browser inventory for this turn. These
captures use the in-app Chromium browser through CUA, against the opt-in,
disposable `adminpreview` fixture and the actual built admin bundle. No production
world, credentials or telemetry were used. All names, logs, history and deaths
are explicitly synthetic fixture data. Desktop is 1440 × 1000; mobile is
390 × 844. The sidebar breakpoint was also checked at 1024 × 768, with no
horizontal document overflow. Temporary viewport overrides are reset after QA.

| Surface | Desktop | Mobile |
| --- | --- | --- |
| Player-death tile | [desktop](dashboard-desktop.jpg) | [mobile](dashboard-mobile.jpg) |
| Deaths: names, rooms, causes | [desktop](deaths-desktop.jpg) | [mobile](deaths-mobile.jpg), [full records](deaths-mobile-details.jpg) |
| Accepted characters: names/classes | [desktop](characters-desktop.jpg) | [mobile](characters-mobile.jpg) |
| Returning logins: names/classes | [desktop](logins-desktop.jpg) | [mobile](logins-mobile.jpg) |

Name search was checked against Avery and an unmatched name; [empty search](filtered-empty-mobile.jpg).
The seven-day switch, release/uptime detail, online coverage disclosure, and the
error link carrying ERROR severity plus the UTC lower bound were exercised.
The browser reported no JavaScript errors before the intentional outage check.
After stopping only the disposable preview by its exact PID, the page reported
OFFLINE and showed both refresh failures with Try again controls
([capture](refresh-failed-desktop.jpg)); it did not present the failure as an
empty activity list. The preview was left stopped.

## Account-switch visibility

A final class audit found that the shared query cache survived sign-out. The
identity provider now clears queries before applying boot, sign-in or sign-out
state, so a different account (or the same character after demotion) cannot
inherit cached private query results. This covers activity, logs and editors.

The fixture's opt-in `ADMIN_PREVIEW_LOGIN=1` mode seeds two disposable panel
sign-ins and one event visible only at level 40. In the same browser session:
TestPlayer (40) loaded [the restricted row](visibility-implementor.jpg), signed
out through the account menu, and Returner (31) signed in through the real form.
On immediately reopening the same Logins window, [the builder view](visibility-builder-after-signout.jpg)
contained Mira and Avery, with zero Hiddenstaff cells. This exercised the UI
cache boundary in addition to the server's stored-level tests. The auth fixture
was stopped by exact PID afterward and the temporary viewport was reset.

## Data boundaries and tests

The old death counter remains intact for compatibility. The new player-death
counter observes accepted guarded deaths, raw kill, and death traps. Mob deaths,
positive-HP rejected deaths and repeat delivery do not inflate that counter.
Named login records require successful returning authentication and saved-password
reset. Accepted creations record only after a successful save, once.

Endpoint tests cover kind/range validation, name filtering, stable cursor paging,
cache prevention, missing history, actual stored immortal visibility, and immediate
loss of access after demotion. Game/session tests exercise real recording seams,
including mob versus player killers, traps, raw kill, rejected/duplicate deaths,
failed authentication, failed creation, and repeated acceptance. Telemetry tests
cover bounded buffers, lost details, seven-day expiry, clearing expired names from
memory, legacy-file migration, restart IDs and pending activity saved on shutdown.

The existing shutdown test now requires at least one saved minute bucket instead
of exactly one: a final shutdown sample may legitimately cross a minute boundary.
A separate test requires arrivals after the last regular sample to survive shutdown.

## Verification

- `make fmt`
- `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  each full Go gate run separately; lint reports zero issues.
- Focused `go test -race` for Activity and Shutdown tests in telemetry, admin,
  game and session.
- `make deadcode-ratchet`: 52 baselined, zero new.
- `make admin-api`; admin `npm run build` and `npm run lint`.
- `make build-site`: generated assets, voice/Markdown tests, content inventory,
  wordmark/palette checks, Astro build and redirect generation.
- Impeccable scoped polish and manual detector: no findings. Existing light
  paperback design and source-owned shadcn/Recharts components preserved.

## Fidelity evidence (R1/R3/R4/R5f)

Final run:
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-28/v017-eyeballs-b-security-final/`

[Summary](census-summary.txt): **9/9 CLEAN, zero draw divergences**, seed 1.
The wrapper retains both engines' blocks, draws, logs, frozen source inputs and
its own MANIFEST.md in that directory. All 13 changed production Go files were
compared byte-for-byte with that final run's frozen source after the run and
matched. The
branch was rebased over the reviewed packaging PR #2205; that changed no game
call paths. Generated docs inventory/search data were rebuilt to include both
changes, and the full local gates were rerun on the rebased tree.

Scenarios: character-creation; entry-menu-returning;
entry-password-retry-success; entry-usurp-saved-body-depth; combat-death;
death-menu-reentry; mudlog-death-trap-player; raw-kill-protection-backfire;
skillmsg-poison-tick-outcomes-depth. These cover the touched creation,
authentication/reconnect and guarded/raw/trap/periodic death boundaries. API and
persistence semantics are proved separately by endpoint and SQLite/session tests.

Earlier development runs are also retained at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-28/v017-eyeballs-b-activity/`.
That run and `v017-eyeballs-b-final/` were each 9/9 CLEAN with zero draw
divergences. The later `v017-eyeballs-b-snapshot-final/` run was also clean,
after retention buffer cleanup and use of the room snapshot getter. The final
security run includes the explicit unsigned count conversion. No full or combined release census was started; Claude owns that
gate for the reviewed batch.


## Finish review

The records-first desktop/mobile path, keyboard focus, search, empty state,
visibility tests, typed response, sample gaps and failed-refresh state were
reviewed together. No blocking findings remain. Screenshot files are native
browser JPEG captures, not product assets. Claude's review in Zach's session is
still the merge boundary.

Impeccable reported older product/schema and sidecar metadata, plus surface-path
resolution drift. The actual target files exist; this scoped pass preserved the
incumbent design. A separate `init`/`document` refresh can reconcile that metadata;
no unrelated metadata migration or visual-world replacement was performed.


## CI security correction

The first PR security run flagged the signed subtraction inside an otherwise
guarded `int`→`uint64` detail-removal count (G115). The length is now widened
before subtracting the capacity, which makes the nonnegative range explicit.
The pinned gosec v2.29.0 HIGH scan passes locally with no suppression. No game
or observation decision changed; the final retained census includes this form.
