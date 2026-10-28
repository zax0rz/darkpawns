# Operator launcher

Interactive presentation extends the existing portable launcher. It never changes
`internal/portable` update, signature verification, rollback or health logic.
The shell installer is unchanged and gains no gum dependency.

## Live status

`darkpawns --instance /path/to/world status --watch`

The Bubble Tea view refreshes every two seconds. It shows the owned process
state, running bundle version, server-reported uptime, online player table,
recent ERROR records and a scrollable tail of `lib/logs/launcher.log`.
The tail reads at most 64 KiB, rather than loading the entire file.

A running world offers an immortal sign-in to read its existing admin API. The
password is masked, the session stays in memory, and no credential file or JWT
is created. The session cookie is used as a bearer token for non-browser reads.
Requests use the configured loopback address and refuse redirects/proxies.
Cancelling sign-in keeps local process/log information available. Unknown or
unauthorized player information is labelled unavailable; it never becomes zero.

Keys: `q`/Ctrl+C leave, `r` refresh, arrows/Page Up/Page Down scroll, `f` toggle
log following, `Tab` switch player/log focus, and `?` open/close full help.
Scrolling pauses follow mode. A stopped instance has no players in its world;
restart the watch after starting it if you want to sign in for player details.

## Diagnostics and updates

`doctor` uses a ✓/✗ checklist in a colour terminal. Checks and failure exit codes
come from the existing portable doctor. `update --check` prepares and verifies
through the existing updater, displays a checklist and renders matching release
notes with Glamour. A missing/mismatched notes response is reported separately;
release prose never supplies trusted update inputs. Local archives remain offline
and report unavailable notes rather than contacting GitHub for them. Private
release feeds also skip the optional public notes lookup; portable alone owns
their configured endpoint.

`update` shows download, verify, stage and health-gate tracks. Local-archive byte
counts have an exact total; remote downloads show observed bytes without an
invented percentage. Verify and stage are one opaque portable operation and
remain indeterminate until it returns. The UI reads new staging files and the
existing transaction journal; it does not write either or duplicate validation.
A health gate completes only after Apply confirms success for a running instance.
For a previously stopped instance it is marked not run.

Ctrl+C cancels the operation context and waits for portable's result and recovery
before closing a plan. Terminal failures also wait for that result. Backup,
rollback, candidate activation and health decisions remain portable's responsibility.

Interactive update/restart confirmations show the current online names, or an
explicit unknown state when live details were skipped/unavailable. `--yes` skips
these prompts for intentional unattended operation. Plain-mode update retains its
original line confirmation. Plain-mode restart retains its original behavior.

## Choosing instances

`darkpawns instances` opens a filterable list. `/` filters, Enter chooses, and
`q`/Esc cancels. With several known instances, interactive lifecycle/update commands
offer this same picker unless `--instance` is explicit. Discovery reads valid
instance directories directly under the default Dark Pawns data folder. It does
not create a registry, scan the home tree or search arbitrary filesystem paths.
Use `--instance /path` for worlds stored elsewhere.

## Plain output is a compatibility boundary

Non-TTY stdout, `--plain`, `TERM=dumb`, or `NO_COLOR` preserve the existing
status/doctor/update output. No prompts, colour or live-screen control sequences
are added to scripts. `--json` never launches an instance picker or a live view.
`status --watch` prints one legacy status snapshot when presentation is disabled.
Existing setup plain prompts remain unchanged.

The help text necessarily lists the new flags and `instances` command. The
compatibility comparison covers existing command output, not that expanded help.

## Captures and validation

VHS tapes live in `scripts/vhs/cli-{before,after,picker,compact}.tape`. Build a
launcher into `.local/cli-evidence/after/darkpawns` and the opt-in rehearsal with:

```sh
go test -c -tags clipreview -o .local/cli-evidence/preview.test ./cmd/darkpawns
vhs scripts/vhs/cli-after.tape
vhs scripts/vhs/cli-picker.tape
vhs scripts/vhs/cli-compact.tape
```

`TestCLIPreview` exists only in that build-tagged test executable. Its live data,
progress, paths and release notes are labelled synthetic. It runs shipping UI
components without updating or restarting any world. The doctor capture uses
actual diagnostics on a disposable stopped instance, including expected failures
for its deliberately absent signed release.

`scripts/cli_plain_compare.py BEFORE AFTER` retains exact output comparisons.
Before/after plain stdout, stderr and exit status were compared across status,
status JSON, doctor, doctor JSON, update check, local-archive check and setup plain.
Added tests cover narrow layouts, navigation, cancellation, bounded/sanitized
logs, in-memory sessions, redirect refusal and confirmed progress semantics.
No game code or player bytes change (R1/R3/R4); no oracle census is required.

Decisions for Zach: no blocking decisions. Fine-grained verify/stage percentages
would require telemetry from GLM's portable updater; this PR deliberately uses
indeterminate waits rather than changing that boundary. Ready for Claude review
in Zach's session.
