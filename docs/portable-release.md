# Portable releases

Portable releases include a guided setup wizard, background lifecycle commands,
doctor, optional per-user automatic startup, manual updates and snapshot recovery.
Official releases carry signed metadata; locally built snapshots are unsigned
development artifacts. Publication is manually approved; see
[publishing releases](publishing-releases.md).

The archive contains `darkpawns`, `darkpawns-server` (both `.exe` on Windows),
`release-assets/`, and the original license notices. Go, Node, Python and a source
checkout are not needed to **run** an extracted bundle.

## Bootstrap installation

Linux/macOS can download and run the installer in one shell command once a
versioned release is available:

```sh
curl -fsSL https://darkpawns.org/install.sh | bash
```

Windows PowerShell 5.1 or newer:

```powershell
$installer = Join-Path ([IO.Path]::GetTempPath()) ('darkpawns-' + [Guid]::NewGuid().ToString('N') + '.ps1'); try { Invoke-WebRequest -UseBasicParsing 'https://github.com/zax0rz/darkpawns/releases/latest/download/install.ps1' -OutFile $installer; & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer; if ($LASTEXITCODE -ne 0) { throw 'Installer did not complete.' } } finally { Remove-Item -LiteralPath $installer -ErrorAction SilentlyContinue }
```

The installer checks the signed release manifest and native archive before
extracting it, installs under your user account, and opens `darkpawns setup`.
Linux/macOS require the standard `curl`, OpenSSL and tar utilities; there is no
compiler or language runtime to install. Windows uses built-in PowerShell/.NET.
Nothing needs administrator privileges. This is a shell/PowerShell bootstrap,
not an MSI, macOS application or OS code-signing certificate. The entry script
is delivered over HTTPS from darkpawns.org and downloads the release installer
over GitHub HTTPS. The release installer pins the publisher key that authenticates
the manifest and bundle. Independently inspecting that script/key is the trust
boundary for first installation.

To inspect before running, download the entry script first:

```sh
curl -fsSL https://darkpawns.org/install.sh -o install.sh
less install.sh
bash install.sh
```

The entry script downloads a second script from the latest official release;
inspect that release's `install.sh` and pinned key too if auditing the bootstrap.

Default release folders are `$XDG_DATA_HOME/darkpawns/releases` (falling back to
`~/.local/share/darkpawns/releases`) on POSIX and
`%LOCALAPPDATA%\darkpawns\releases` on Windows. The world remains in the wizard's
separate instance folder. POSIX installs a new `~/.local/bin/darkpawns` symlink;
it prints the command folder so you can add it to PATH when needed. Windows
installs a UTF-8 `darkpawns.cmd` wrapper and adds its bin folder to your user PATH;
open a new terminal to pick that up. The Windows wrapper sets console code page
65001 so Unicode installation paths work. Existing commands and release folders
are preserved; use `darkpawns update` for an existing installation.

Downloaded scripts also accept `--install-dir`, `--bin-dir`, `--no-setup` and
`--source-dir` on POSIX; Windows uses `-InstallDir`, `-BinDir`, `-NoSetup`,
`-SourceDir` and optional `-NoPath`. Source-dir mode reads an offline release
folder and still requires the pinned signature; it is not an unsigned bypass.
In a headless POSIX session without a controlling terminal, installation prints
the setup command instead of reading prompts from a download pipe.

## Windows double-click setup (v0.1.3)

After extracting the complete Windows archive, double-click `darkpawns.exe` to
open the setup wizard in its console. On completion, cancellation or error, the
console waits for Enter so you can read the result. Setup still requires your
review before creating a world. It starts in background mode by default, so
closing the completed setup window leaves the world running.

A second double-click opens setup again. Existing world folders are refused;
choose a new folder (and available ports) for another world, or cancel. Manage
your existing world with the lifecycle commands below. Setup does not reset,
overwrite, or automatically restart an existing world.

Commands launched from an existing terminal keep their usual behavior, without
the extra pause. Shortcuts with explicit arguments keep those arguments and
wait before closing when launched through Explorer. This does not change Windows
SmartScreen or publisher-signing behavior.

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
(default `4350`) and a local telnet port (default `7777`, or `0` to disable), then
reviews the settings before creating anything. Both listeners bind to IPv4
loopback; other machines cannot connect directly. Use Tab or
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

`setup --instance <new-folder> --port <number> --telnet-port <number>` preselects
different values. Enabled ports must be available and distinct.
Missing parent folders are created only after confirmation and bundle verification.
Existing folders are refused, even when empty; resume a world with `run` rather
than rerunning setup. Cancellation, EOF and an occupied port create no instance.
Initialization or startup errors retain any created instance for inspection and
print recovery instructions. Port checks are advisory; another process can take
the port between the check and the server binding it.

The interactive wizard uses the original login artwork and oxblood focus accents,
with the terminal's own background and text colors. Step headings show progress
through the three screens. Small terminals use plain prompts automatically.

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
The admin console is at `/admin/`. New v0.1.2 instances also listen for telnet at
`127.0.0.1:7777` (or your chosen `--telnet-port`); use `telnet 127.0.0.1 7777`
or your MUD client. `--telnet-port 0` disables it. TLS telnet and WHOD remain off.
Updating an existing instance preserves its ports and bind configuration. The
splash page shows only active listeners and their observed address and port.
For deliberate public hosting, see [portable-hosting.md](portable-hosting.md).

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
must not copy starter files again. Managed updates keep replaceable releases alongside private update history under
`<instance>/.updates/`. They preserve the instance's edited starter files rather
than copying a newer seed over them. Keep the original launcher directory: PATH
shortcuts and native sign-in jobs use it as a stable entry point and dispatch to
the active release. Never extract a release over your instance.
A partially failed initialization is retained for inspection rather than erased.

For public listeners, external system supervision and existing-world migration,
see [hosting and adoption](portable-hosting.md). These require v0.1.1 or newer.

## Manual updates

From an **updater-capable portable release**, run:

```sh
darkpawns update --check                 # stage and verify; keep the world running
darkpawns update                         # review, then approve the stop/backup/update
darkpawns update --yes                   # explicit unattended approval
# Add --instance /absolute/path for a nondefault world.
```

The default source is the latest published stable release of this repository,
with an archive for the native OS/architecture. The updater obtains the expected
SHA-256 from the GitHub Releases API over HTTPS and requires it to match the
publisher's signed manifest and archive size. The signature is RSA PKCS#1 v1.5
with SHA-256, verified against the embedded public key in
`internal/releaseauth/trusted-key.pem`. Changed metadata, foreign keys, mismatched
release identities and unsigned official downloads are refused before downtime.

No portable release is published simply by merging or pushing a tag. Until the
first stable release is deliberately approved, the default command reports that
none is available. See [publishing releases](publishing-releases.md). A publisher
signature proves possession of that key; native compatibility tests, protected
publication review, signing-key security and TLS/script delivery remain separate
parts of the release process. It is not Apple notarization or Windows Authenticode.

For development or an offline host, supply a **trusted** archive and its digest:

```sh
darkpawns update --archive /path/to/native-release.tar.gz --sha256 <64-hex-digits>
# Windows archives are .zip. Obtain the expected digest independently of a download.
```

A local digest detects a mismatch; it does not make an untrusted archive safe to
execute. Both modes check the payload manifest, native Go binary targets and
explicit updater/data format declarations. An identical payload is a no-op.
`--check`, rejection, EOF and declined approval never stop the game. Cancellation
removes unapplied staging. No tools, compiler, Git or Node are needed on the host.

After approval, a lifecycle lock serializes operations. The launcher stops the
current game and verifies its shutdown receipt, obtains the run lock, checks
SQLite read-only, and copies the stopped instance to a private backup. It includes
world edits, boards, database files/sidecars, admin and notification settings,
notification encryption keys, stable signing configuration and logs. Updater
history and transient process/lock records are excluded. Links and special files
in the backed-up tree are refused. Backup file hashes are recorded and verified;
the updater does not claim a filesystem-wide snapshot of unrelated writers.
Keep enough free disk for releases, the whole backup, and a retained failed tree;
there is no automatic history deletion or disk-space reservation yet.

New releases live in unique directories; a durable active-release record selects
the payload. Running Windows executables are never overwritten. If the world was
running, the candidate supervisor must become locally healthy before commit.
A stopped world stays stopped, so that path verifies the payload and backup but
**does not certify runtime startup**. The game continues to use its existing
instance world, not the release's fresh starter world.

If candidate startup fails, the updater stops it, verifies the backup, preserves
its attempted `lib/` as `failed-lib`, restores the saved `lib/` and byte-identical
`instance.json`, selects the previous release and restarts it only if it was
running before. This is **snapshot recovery**, not a claim that old binaries can
undo a schema migration. Any writes accepted during the candidate's short startup
window remain in `failed-lib`; they are not merged into the restored snapshot.
Keep players out during a manual update. Automatic scheduling/idle-player gates
are not implemented in this slice.

An interrupted transaction blocks normal startup. Inspect `doctor`, retain the
`.updates/` history, then use the original launcher if the active payload is
unavailable:

```sh
/absolute/path/to/original/darkpawns update recover --instance /path/to/world
```

Recovery is retryable after interruption, refuses a held game lock or damaged
backup, and retains attempted state. A partial backup made before activation is
never restored. If restart fails after restoration, fix the reported environment
or service issue and run `start`; recovery never deletes the player's world to
start fresh. Journals and per-update receipts record the source, selected release,
archive digest, previous run state and outcome. `doctor` reports pending recovery.
There is no general-purpose restore command for a successfully committed update.

Releases declare `update_format: 1` and `data_format: 1`. Maintainers must change
the data epoch when a release cannot safely run against the current instance
format or recover using the previous snapshot. Unknown target epochs are refused;
there is no force-compatibility flag. The epoch is a producer declaration, not an
automatic schema-diff proof. The pre-updater bundles can supply format-1 instance
data, but sign-in jobs anchored to their old launchers must be removed/reinstalled
with an updater-capable release before updating.

These commands manage **portable instances**, not an arbitrary checkout or custom
system daemon. Migrating an existing deployment requires preserving its flags,
environment, networking, service account and state paths first. They do not deploy
the separate Astro website, change the reverse proxy, or alter production services.

## Build and verification

Maintainers need Go from `go.mod`, Node 24, Python 3, Git and GoReleaser **2.18.3**:

```sh
make portable-snapshot
```

This builds Linux/macOS/Windows amd64 and arm64 archives, with checksums. Starter
assets come from the current **committed** Git tree; the admin UI is freshly
built. Checksums detect damaged payloads, not publisher authenticity. No artifacts
are published automatically on tags. The manually approved publication workflow
signs the archive manifest and publishes the bootstrap scripts; macOS
notarization and Windows Authenticode/MSI remain follow-ups.

The portable CI workflow builds native unsigned snapshots, then tests initialization,
loopback HTTP/browser/admin assets, first-immortal/second-mortal creation over the
browser's WebSocket protocol, duplicate-run rejection, graceful stop, release
relocation and saved login after restart. It checks that edited instance world
files survive changing the release seed. A cross-build is not a runtime proof;
Windows console and macOS runtime results must come from those native jobs.
Native CI also exercises local-archive updates: checksum/epoch refusal before
downtime, check/cancel, complete backup, active-launcher dispatch through the
original entry point, saved login, native registration across updates, failed-start
snapshot recovery and preservation of a stopped state. Unit tests cover hostile
archives, damaged backups and interruption during restore. This does not certify
a successful live published feed activation, OS code signing, power-loss durability or arbitrary
schema migrations.
The same native journey now goes through the line-mode setup wizard, checking
cancellation/EOF, port conflicts and retained state after a failed server start.
Unit tests cover prompt correction, review and readiness timeout/cancellation.
The graphical browser and terminal renderer still require desktop verification;
native headless CI exercises the shared setup plan and server protocol, rather
than certifying desktop browser integration (R5f).

## Follow-up milestones

- Windows ACLs, desktop installation checks, real sign-out/reboot verification and crash supervision.
- macOS notarization, Windows Authenticode and a native Windows installer.
- Opt-in scheduled updates using that same engine: notify-only or a maintenance
  window, no connected players, independently delivered success/failure/recovery
  notifications from an updater outside the game process. Never promise binary
  rollback can undo an incompatible database or world migration.

## Release greeting and starter MOTD (v0.1.2)

New instances set `greeting_overlay` to `true`. This replaces only the historical
pwipe notice with the running server's build version and `https://darkpawns.org`.
The original logo, attribution and blank lines stay intact. Set it to `false`
while stopped, then restart to restore C's greeting. Omitted means `false`, so
an update never silently changes an existing world's greeting. The standalone
server equivalent is `-greeting-overlay`; it is off by default.

The release starter world receives a short neutral `lib/text/motd`. After
initialization that file belongs to the operator. Updates preserve it, including
production's custom MOTD; canonical repository text is not changed. Older
launchers cannot directly read new configuration fields: use a matching
v0.1.2-or-newer recovery launcher when adding these fields to an existing world.
