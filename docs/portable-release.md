# Portable development bundle

This is the first installation milestone, not the public installer. Archives are
unsigned development snapshots. No TUI, automatic browser launch, background
service, doctor, updater, or public release/signing workflow is included yet.

The archive contains `darkpawns`, `darkpawns-server` (both `.exe` on Windows),
`release-assets/`, and the original license notices. Go, Node, Python and a source
checkout are not needed to **run** an extracted bundle.

## Create and run

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
The TUI, graphical browser journey and actual Windows/Mac desktop experience
remain untested until those features exist.

## Follow-up milestones

- Huh setup wizard and Cobra lifecycle/doctor commands, browser handoff and
  headless instructions; platform-native instance locations and Windows ACLs.
- Native user services and explicit ownership of start/stop/restart.
- Signed releases, macOS notarization and Windows installation experience.
- Manual updates with consistent backups, compatibility gates and recovery.
- Opt-in scheduled updates using that same engine: notify-only or a maintenance
  window, no connected players, independently delivered success/failure/recovery
  notifications from an updater outside the game process. Never promise binary
  rollback can undo an incompatible database or world migration.
