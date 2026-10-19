# Portable development bundle

These are unsigned development snapshots with a guided setup wizard. Background lifecycle commands, doctor and optional per-user automatic startup are
included. Updater and the public release/signing workflow are later milestones.

The archive contains `darkpawns`, `darkpawns-server` (both `.exe` on Windows),
`release-assets/`, and the original license notices. Go, Node, Python and a source
checkout are not needed to **run** an extracted bundle.

## Guided setup

Extract the whole archive, then run:

```sh
/path/to/release/darkpawns setup
```

PowerShell:

```powershell
& 'C:\DarkPawns\release\darkpawns.exe' setup
```

The three-step terminal wizard asks for a new instance folder, a local web port
(default `4350`), then reviews the settings before creating anything. Use Tab or
Shift+Tab to move between fields, Enter to continue, and Ctrl+C to cancel. The
review includes the browser choice and explains that the first character becomes
the immortal. Background mode is the default: setup returns once the world is ready, and you
can close the terminal. Setup waits for `/health`, then opens the browser when a
desktop session is available. Choose foreground mode (or `--foreground`) to keep
it attached to the terminal; Ctrl+C then saves and stops it. Automatic startup at
sign-in is a separate, optional choice, disabled by default. It requires background mode.
A missing or failed browser opener leaves the server running and prints the URL.

Default instance folders (separate from the extracted release):

| Platform | Default |
|---|---|
| Linux | `$XDG_DATA_HOME/darkpawns/world`, otherwise `~/.local/share/darkpawns/world` |
| macOS | `~/Library/Application Support/darkpawns/world` |
| Windows | `%LOCALAPPDATA%\darkpawns\world` |

`setup --instance <new-folder> --port <number>` preselects different values.
Missing parent folders are created only after confirmation and bundle verification.
Existing folders are refused, even when empty; resume a world with `run` rather
than rerunning setup. Cancellation, EOF and an occupied port create no instance.
Initialization or startup errors retain any created instance for inspection and
print recovery instructions. Port checks are advisory; another process can take
the port between the check and the server binding it.

For screen readers or a basic terminal, use `setup --plain`: line prompts with an
explicit yes at review (the default is no). Non-terminal input requires `--plain`
or `--yes`. Small, `TERM=dumb` or `NO_COLOR` terminals use line prompts automatically.
`setup --yes --no-browser --instance <new-folder>` explicitly accepts
the selected settings and starts the background server without prompts. Add `--foreground` for terminal-attached startup.
`--autostart` explicitly opts into native automatic startup during unattended setup.

For a headless host, use `setup --no-browser` (also the default over SSH and on
Linux without a desktop display). The URL is local to the server machine. To use
your laptop browser, forward the chosen port, for example:

```sh
ssh -L 4350:127.0.0.1:4350 your-host
```

Run setup in that session, then open `http://127.0.0.1:4350/` on your laptop.

After stopping, restart using the printed command, or:

```sh
/path/to/release/darkpawns run --instance '/path/to/your/world'
```

The first successfully created character becomes the immortal. Create it before
exposing the server to other people. Setup does not create an admin account or
change the existing character creation dialogue (R4).

## Start, stop, restart and status

Every lifecycle command accepts `--instance <folder>`. If omitted, it uses the
same platform default folder as setup. These commands do not create a world;
run setup first.

```sh
/path/to/release/darkpawns start --instance '/path/to/your/world'
/path/to/release/darkpawns status --instance '/path/to/your/world'
/path/to/release/darkpawns restart --instance '/path/to/your/world'
/path/to/release/darkpawns stop --instance '/path/to/your/world'
/path/to/release/darkpawns doctor --instance '/path/to/your/world'
```

`start` returns only after readiness; starting an already-running world is
idempotent. `stop` asks the instance's own launcher to shut down and waits for its
save/shutdown path. `restart` waits for that stop before starting again. These
operations serialize per instance. A private `.last-exit.json` record ties the
shutdown result to that run; stop reports a failed shutdown rather than treating
lock release alone as success. The launcher allows 30 seconds for the game to
exit, then terminates a stuck child and reports failure. Inspect the log and data
before restarting after that failure. `status --json` and `doctor --json` provide
structured output; doctor exits nonzero when it finds a problem.

Background output goes to `lib/logs/launcher.log`. Keep the extracted release
in place while it is running. Foreground `run` instances can also be inspected
and stopped by these commands. Stopped worlds keep all their data and secrets.

The launcher exposes a separate authenticated loopback control endpoint, with a
fresh per-run capability in `.control.json`. Never share that private runtime
file. Game/admin authentication cannot control it. Lifecycle commands do not
signal PIDs read from disk or treat an arbitrary healthy web port as ownership.
`.run.lock` remains held through the child's shutdown; `.lifecycle.lock` serializes
operator changes. Do not delete a held lock. If the supervisor dies abruptly,
use doctor and the OS tools to identify any orphaned game before restarting;
a busy configured port is refused before starting another server.

## Automatic startup at sign-in

Choose it during setup, or configure an existing instance:

```sh
/path/to/release/darkpawns service install --instance '/path/to/your/world'
/path/to/release/darkpawns start --instance '/path/to/your/world'
/path/to/release/darkpawns service remove --instance '/path/to/your/world'
```

Removal gracefully stops the instance and removes its owned startup registration,
without deleting the world. Registration names derive from the canonical instance
path, so multiple worlds stay separate. Existing definitions are refused. Changes to owned Linux/macOS files or the
Windows launch action are refused. A failed registration retains owned files for diagnosis and cleanup;
Windows rolls back its local record if task creation fails.

| Platform | Current-user startup mechanism | Requirements |
|---|---|---|
| Linux | systemd user unit under `$XDG_CONFIG_HOME/systemd/user` or `~/.config/systemd/user` | Active user service manager; a user session bus |
| macOS | LaunchAgent under `~/Library/LaunchAgents` | A logged-in GUI session for the account |
| Windows | Task Scheduler logon trigger with `InteractiveToken` and least privileges | Task Scheduler permissions for your own account; an interactive sign-in |

These are user jobs, not system-wide boot services. No root/admin elevation is
requested. Local Windows policy may restrict task registration; errors are
reported. On headless macOS without a GUI login, use ordinary background `start`
or foreground `run`. Linux administrators may independently configure lingering
if the user service should remain after logout. Registration pins the launcher's
absolute release path; remove it before moving the release and reinstall from the
new path. This is not an updater or migration mechanism. Native definitions do not
automatically restart a crashed world in this slice.

Doctor checks release checksums, the server executable, instance configuration,
POSIX private permissions, launcher state, local web availability and owned startup
files/tasks. While stopped, it opens the existing SQLite database read-only and
runs `quick_check`; it reports a missing, not-yet-created database separately.
It reports a failed last shutdown when stopped and skips offline integrity
checking while a launcher is active. It performs no
repairs, password resets or credential changes. Windows ACL inspection and real
logout/reboot behavior remain desktop verification work.

## Manual create and run

Extract the whole archive into a release directory. Choose a **new directory
outside that release** for your instance; its parent must already exist. Existing
directories are refused, never overwritten. Examples (replace the paths):

```sh
/path/to/release/darkpawns init --instance /path/to/my-world --port 4350
/path/to/release/darkpawns run --instance /path/to/my-world
```

PowerShell:

```powershell
& 'C:\DarkPawns\release\darkpawns.exe' init --instance 'C:\DarkPawns\my-world'
& 'C:\DarkPawns\release\darkpawns.exe' run --instance 'C:\DarkPawns\my-world'
```

Run these from any working directory. The foreground server listens on
`http://127.0.0.1:4350/` (or your chosen port). Open it manually in your browser.
The admin console is at `/admin/`. Telnet and WHOD are disabled for this first,
local-only launcher. The existing standalone server flags remain unchanged.

Create your first character through the existing client to become the immortal;
the second character is mortal. Ctrl+C stops the launcher and asks the server to
save and stop normally. Closing the terminal abruptly is not graceful shutdown.
A second launcher cannot run the same instance while its process lock is held.

## What belongs to you

`instance.json` stores the instance format, web port and randomly generated,
stable signing secret. Keep it private. The instance's `lib/` holds its own copy
of the starter world, scripts and text. SQLite and notification credentials live
in `lib/data/`; boards live in `lib/world/etc/`; audit logs live in `lib/logs/`.
No existing repository player database, player index, rent file, or board is
seeded. Initialization copies the starter files exactly once.

Back up the **entire instance** using the running guide's consistent SQLite
backup procedure. Preserve notification settings and their original encryption
key together. POSIX instance directories/files are created as `0700`/`0600`;
Windows uses inherited filesystem ACLs, which need review before public launch.
The launcher rejects inherited `DP_*`, `ADMIN_*`, `DARKPAWNS_*`, database, signing
and TLS configuration so another installation's environment cannot redirect this
instance. The raw server is still available for advanced deployments.

Replaceable binaries and browser/admin assets stay in the release directory.
Running a different compatible development bundle against the same instance
must not copy starter files again. This separation is **not** a safe general
upgrade/rollback system: schema compatibility, backups, signing and controlled
activation still need implementation. Never extract a release over your instance.
A partially failed initialization is retained for inspection rather than erased.

## Build and verification

Maintainers need Go from `go.mod`, Node 24, Python 3, Git and GoReleaser **2.18.3**:

```sh
make portable-snapshot
```

This builds Linux/macOS/Windows amd64 and arm64 archives, with checksums. Starter
assets come from the current **committed** Git tree; the admin UI is freshly
built. Checksums detect damaged payloads, not publisher authenticity. No artifacts
are published automatically, including on tags. Signing/notarization and the
one-command bootstrap remain later milestones.

The portable CI workflow builds native unsigned snapshots, then tests initialization,
loopback HTTP/browser/admin assets, first-immortal/second-mortal creation over the
browser's WebSocket protocol, duplicate-run rejection, graceful stop, release
relocation and saved login after restart. It checks that edited instance world
files survive changing the release seed. A cross-build is not a runtime proof;
Windows console and macOS runtime results must come from those native jobs.
The same native journey now goes through the line-mode setup wizard, checking
cancellation/EOF, port conflicts and retained state after a failed server start.
Unit tests cover prompt correction, review and readiness timeout/cancellation.
The graphical browser and terminal renderer still require desktop verification;
native headless CI exercises the shared setup plan and server protocol, rather
than certifying desktop browser integration (R5f).

## Follow-up milestones

- Windows ACLs, desktop installation checks, real sign-out/reboot verification and crash supervision.
- Signed releases, macOS notarization and Windows installation experience.
- Manual updates with consistent backups, compatibility gates and recovery.
- Opt-in scheduled updates using that same engine: notify-only or a maintenance
  window, no connected players, independently delivered success/failure/recovery
  notifications from an updater outside the game process. Never promise binary
  rollback can undo an incompatible database or world migration.
