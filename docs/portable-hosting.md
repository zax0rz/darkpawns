# Hosting and adopting an existing world

Requires v0.1.1 or newer. Finish a signed public version-to-version update
rehearsal before migrating an existing service. The local setup wizard retains
its local-only defaults; public hosting is a deliberate operator configuration.
The original game command surface and output are unchanged (R1 through R5).

## Instance configuration

Keep `format`, `http_port` and `jwt_secret` from the existing `instance.json`.
Merge only the hosting fields you need, while the instance is stopped:

```json
{
  "telnet_port": 7777,
  "telnet_tls_port": 7778,
  "whod_port": 7779,
  "server_env": {
    "TELNET_TLS_CERT_FILE": "/etc/my-mud/tls/fullchain.pem",
    "TELNET_TLS_KEY_FILE": "/etc/my-mud/tls/privkey.pem",
    "DP_MUDLET_PACKAGE_URL": "https://mud.example.invalid/darkpawns.xml",
    "DP_MUDLET_MAP_URL": "https://mud.example.invalid/darkpawns-map.xml"
  },
  "mudlet_directory": "/srv/my-mud/mudlet"
}
```

This is a partial configuration, not a replacement for `instance.json`. Zero or
omitted listener ports disable them. Nonzero ports must be distinct from each
other and the HTTP port. TLS requires an explicit certificate and key; startup
checks the pair and listener availability before launching. Availability checks
are advisory because another process can bind between preflight and launch.

HTTP remains on loopback behind your reverse proxy. Only the four documented
`server_env` keys are accepted. Inherited TLS, Mudlet, database, admin and
notification settings are still stripped; this configuration explicitly opts
into hosting settings. Other environment variables, such as contact delivery and
trusted proxy configuration, follow the existing server environment contract.
Back up any external environment and certificate files separately.

The optional `mudlet_directory` must already exist and be writable by the game
account and readable by the web server. On each game start, the launcher atomically
publishes the bundled `darkpawns.xml` there before that server can advertise its
compiled Mudlet version. It replaces only that file and leaves other files alone.
Updates and failed-start recovery use the corresponding release's package. The
file is public (0644); private instance files remain private. Proxy the live
`/darkpawns-map.xml` route to the game, and serve `/darkpawns.xml` from this public
directory. Keep the Astro website deployment separate.

## An external service manager

`darkpawns supervise --instance /var/lib/my-mud` is a persistent foreground
manager for a hosted instance. It stays alive while the updater replaces the
game's authenticated background supervisor. It restarts a crashed game, respects
explicit `start`/`stop`/`restart` requests, and waits through pending update or
recovery transactions. An interrupted update requires explicit recovery; the
monitor does not guess that a failed transaction can be ignored.

A Linux systemd example, adapted to your service account and paths:

```ini
[Unit]
Description=Hosted Dark Pawns world
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=mud
WorkingDirectory=/var/lib/my-mud
EnvironmentFile=/etc/my-mud/server.env
ExecStart=/opt/my-mud/releases/v0.1.1/darkpawns supervise --instance /var/lib/my-mud
Restart=on-failure
RestartSec=5
KillMode=mixed
TimeoutStopSec=90

[Install]
WantedBy=multi-user.target
```

Keep the original launcher directory as the stable anchor. When enabling hosting
fields after upgrading from v0.1.0, also retain a v0.1.1-or-newer launcher as the
recovery anchor: v0.1.0 recovery cannot read the additional configuration fields. `start`, `stop`,
`restart`, `status`, `doctor` and `update` dispatch to the active release. A running
monitor is an ordinary long-lived process; restart the external unit when a
release specifically requires a new monitor/configuration interface. Updating
the game alone does not replace that process in place.

A fresh service-manager start requests a running world. `darkpawns stop` leaves
the manager alive with the world intentionally stopped; `darkpawns start` resumes
it. Stopping the external unit also stops the game. Sign-in startup registration
and system supervision are separate modes and cannot own the same instance.
Doctor reports monitor ownership and the requested run state.

## Adoption procedure

1. Inventory the live binary, complete flags/environment, working directory,
   database, complete mutable tree, boards, notification settings and encryption
   key, JWT secret, logs, service manager, proxy and public assets. Record which
   paths belong to state and which belong to replaceable code.
2. Prepare a new private instance directory outside every release. Download and
   verify the signed release before downtime. Never initialize a fresh starter
   world over existing state.
3. Back up the running SQLite database with its online backup API, verify
   integrity and record table counts. Preserve the old binary, unit, environment,
   public assets and complete world/state together. Check active players before
   the maintenance window.
4. Stop the old service and verify its process and listeners have exited. Make a
   final stopped-state backup. Copy the complete old `lib/` into the new instance,
   including edited world files, boards, database/sidecars, notification settings
   and the original encryption key. Refuse links or special files rather than
   following them into unrelated data. Keep the original installation intact.
5. Create a private format-1 `instance.json` with the same 64-hex JWT secret, HTTP
   port and explicit hosting fields. Do not rotate credentials during adoption.
   Restrict the instance directory to 0700 and configuration to 0600. Preserve
   the service account and all external configuration deliberately.
6. Run doctor offline, then configure the external unit to run `supervise` from
   the verified release. Keep the proxy routes and public ports consistent with
   the inventoried service. Reconcile backup paths before the next scheduled
   backup; a backup still reading the old tree is a failed migration.
7. Start once. Verify origin/public health, startup persistence logs, all public
   listeners and TLS identity, browser/admin assets, Mudlet package/map, player
   counts, boards, credentials and stable JWT. Verify a service-manager restart
   and a stopped/running lifecycle round trip during the maintenance window.
8. Save a receipt with release/tag/commit, archive digest, backup location,
   pre/post state checks, unit changes and verification. Keep the old deployment
   and final stopped backup through the rollback window.

For rollback, stop the new manager and verify the new game has exited. Restore
the matching old unit/configuration and pre-migration state together, then start
and verify once. If players have written new state, preserve it before restoring
the snapshot and reconcile those writes explicitly; an older executable cannot
undo a schema change. Never discard the migrated tree to make rollback simpler.

The migration changes runtime ownership and packaging. It does not overwrite
world edits with a newer seed, promote another character, reset a password,
regenerate notification keys or change the game's persistence format.
