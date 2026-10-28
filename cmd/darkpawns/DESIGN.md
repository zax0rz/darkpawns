# Launcher presentation

Mode: Operate. Keep setup's established terminal identity: the operator's own
font and background, normal terminal foreground, and one adaptive oxblood accent
(`#A8201A` on light terminals, `#CF4B42` on dark terminals). This extends
`setup_theme.go`; it does not paint a paper background or introduce a second shell.

Use Charm components for familiar terminal tasks: Bubble Tea owns live screens,
Bubbles supplies player tables, log viewport, filterable instance list, progress,
spinners and help. Huh owns reviewed confirmations and masked in-memory sign-in.
Glamour renders release notes with the same restrained accent and terminal text.

The status screen reads process state first, then uptime/release, players, recent
ERROR records and a bounded local log tail. Keyboard controls remain visible;
small terminals use a compact footer and `?` opens full help. `Tab` switches
player/log focus. Scroll pauses log following; `f` resumes it. Refreshes do not
multiply the polling timer. Unknown player information never appears as zero.

Progress reflects filesystem observations and portable API completion only.
Download bytes can be measured; verification and staging are an opaque operation,
so they stay indeterminate until PrepareUpdate returns. Apply success confirms
health only when the instance was running. A stopped instance reports that its
health gate was not run. Ctrl+C waits for the existing operation and recovery.

Plain output is a compatibility boundary. Non-TTY stdout, `--plain`, `TERM=dumb`
and `NO_COLOR` take the original status/doctor/update paths. `--json` remains
structured and `status --watch` returns one existing status response outside an
interactive terminal. The setup wizard retains its existing plain prompts.
No alternate screen, cursor animation, sign-in or picker enters the legacy path.

Untrusted log text, release prose, player names and paths are stripped of terminal
control sequences before rich rendering. Credentials are masked; the admin cookie
is held only in process memory and sent as a bearer token to the configured
loopback server. Redirects and proxy routing are refused. No JWT is fabricated.

The instance picker scans valid direct children of the default Dark Pawns data
folder. It does not crawl the home directory. `--instance` always selects an
explicit path without prompting. There is no global instance registry to mutate.

Verification uses VHS at wide and compact sizes, existing and added command tests,
exact before/after plain-output comparisons, cancellation tests and Go gates.
Rehearsal fixtures are opt-in build-tagged tests and visibly synthetic; no running
world or trusted updater is replaced for a capture.
