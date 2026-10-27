# Configuration and updates in the admin console

Status: implemented for v0.1.7 (game-owned settings, typed pages, backup and
restore, audit history, and the launcher update handoff). Listener editing
stays with the network settings design. Nothing here changes telnet
dialogue, game rules, or draw parity (R1–R4). The design stops at the transport, lifecycle and persistence
decisions Zach asked for; implementation briefs follow approval, and assume
stage A landed: every admin endpoint typed in Huma, the UI client generated
from `/openapi.json`. It extends
[`admin-network-settings.md`](admin-network-settings.md), which stays the
authority for listener exposure; where this doc adds browser control of
updates it supersedes that doc's "update policy" clause — Zach has since
decided both a Check and an Update button exist.

## Authority model: who owns a setting

A setting has exactly one owning process, and the owner is the process that
reads it at boot. The console never becomes a second uncoordinated writer; it
edits each setting through the owner's typed API.

- **Launcher-owned** (`instance.json`, `internal/portable/instance.go:32-45`):
  listeners and ports, `jwt_secret`, `greeting_overlay`, the allowlisted
  `server_env` extras, `mudlet_directory`. Edited through the private control
  channel per the network design; `instance.json` while stopped remains the
  advanced operator path.
- **Game-owned** (new `lib/data/settings.json`): every server-process setting
  that today rides an environment variable — connection limits and idle
  timeouts, proxy trust, CORS origins, Grapevine, staff push (ntfy), contact
  form. The file is revisioned, atomically written, and strict-read like the
  launcher's own JSON (`DisallowUnknownFields`), because game and file ship in
  the same release. The launcher passes its path the same way it passes
  `DP_NOTIFICATION_CONFIG_PATH` (`internal/portable/run.go:181-202`) and
  scrubs the corresponding environment (`run.go:188-193`), so a portable
  instance has exactly one authority. Checkout deployments keep env vars as
  an override, and the console shows an "overridden by environment" badge
  rather than silently losing the operator's flag file.
- **Subsystem-owned**: `data/notifications.json` is already console-managed
  with revision conflicts (`pkg/notify/service.go:153-186`) and is linked, not
  duplicated. Ban lists, text files and help stay owned by their in-game
  commands, which already apply live.

**Migration**: on first boot with no settings file, current env values are
imported once, logged as a migration in the audit trail, and from then on the
file is authoritative. **Secrets never live in `settings.json`**: they go to
a shared encrypted vault that generalizes the notifications vault
(AES-256-GCM under a mode-0600 key file, `pkg/notify/credentials.go`), and
are write-only through the API.

The managed-instance capability from the network design gates everything
launcher-owned: the launcher hands the game a **scoped control token** at
start (env, never exposed through any admin endpoint, never the raw launcher
token) permitting exactly `config-update`, `restart`, `update-check`,
`update-trigger` and `update-status`. Without it the launcher-owned groups
are read-only with pointers to server flags — today's control protocol is
only `GET /status` and `POST /stop` (`internal/portable/control.go:52-74`),
so all five messages are new, on the same bearer-token, loopback-only,
Origin-rejecting transport.

## Inventory of every setting

"Apply" is the state after this design: **live** (takes effect without
restart), **restart** (saved now, applied by the restart transaction),
**command-live** (already live via an in-game command), **display**
(read-only in the console), or **locked** (a fidelity constant, deliberately
not a setting; R4 — engine tunables such as pulse rates, combat cadence and
autosave interval stay compiled in, so the port cannot drift from the oracle
by a settings screen).

### Launcher-owned — `instance.json`

| Field | Type | Default | Governs | Apply | Risk |
|---|---|---|---|---|---|
| `http_port` | int | 4350 | Web/admin/ws port | restart | exposure |
| `telnet_bind` / `telnet_tls_bind` / `whod_bind` | string | `127.0.0.1` / empty | listener interfaces | restart | exposure |
| `telnet_port` / `telnet_tls_port` / `whod_port` | int | 7777 / 0 / 0 | listener ports (0 = off; WHOD derives telnet+1 when unset) | restart | exposure |
| `greeting_overlay` | bool | true | build version in login greeting | restart | cosmetic |
| `jwt_secret` | generated 32 B | — | admin/web auth signing | display only | auth |
| `server_env` (allowlist: `DP_MUDLET_PACKAGE_URL`, `DP_MUDLET_MAP_URL`, `TELNET_TLS_CERT_FILE`, `TELNET_TLS_KEY_FILE`) | map | empty | validated extras (`hosting.go:38-55`) | restart | exposure/secrets |
| `mudlet_directory` | path | empty | where the Mudlet package is published | restart | ops |

Port validation covers range, cross-listener collisions (telnet/TLS/WHOD/HTTP)
and the bind-surface explanation from the network design. TLS cert/key paths
enter only as named profile selections, never free-text paths.

### Game-owned — today environment variables, moved to `settings.json`

| Setting | Type | Default | Read today | Apply | Risk |
|---|---|---|---|---|---|
| `LOGIN_IDLE_TIMEOUT` | seconds | 120 | telnet + ws pre-auth | live | availability |
| `TELNET_MAX_CONNS` / `TELNET_MAX_CONNS_PER_IP` / `WEBSOCKET_MAX_CONNS_PER_IP` | int | 200 / 8 / 8 | listener/manager init | live | availability |
| `DP_ACCOUNT_FAILURE_BUDGET` / `_WINDOW` | count / duration | off / 1h | `pkg/auth/ratelimit.go:418-422` | live | auth |
| `TRUSTED_PROXIES` | CIDR list | `127.0.0.0/8,::1` | boot, `main.go:470` | restart | auth (drives IP bans and rate limits) |
| `CORS_ALLOWED_ORIGINS` / `ADMIN_CORS_ORIGIN` | origin list | prod: `https://darkpawns.org` / unset | per request | live | exposure |
| `GRAPEVINE_MODE` / `GRAPEVINE_URL` / client id+secret | enum / URL / secret | off / `wss://grapevine.haus` | `pkg/grapevine/client.go` | restart (reconnect) | exposure/secrets |
| `DP_MUDLOG_NTFY_URL` / `_TOKEN` / `_LEVEL` / `_TYPE` / `_RATE` | URL / secret / 31 / 1 / 30min⁻¹ | off | `pkg/mudlog/ntfy.go` | restart | ops/secrets |
| `CONTACT_TO` + `CONTACT_SMTP_HOST/PORT/USER/PASSWORD` + `CONTACT_TURNSTILE_SECRET/HOSTNAMES` | strings | all-or-handler-off | `pkg/contact/contact.go:55-61` | restart | ops/secrets |
| `DP_MUDLET_PACKAGE_URL` / `DP_MUDLET_MAP_URL` | HTTPS URL | off | boot | restart | exposure |
| `TLS_CERT_FILE` + `TLS_KEY_FILE` (+ `USE_TLS`) | named profile | off | boot | restart | secrets |

### Already console- or command-managed

`data/notifications.json` (revisioned, live, HIGOD-gated); `banned` site
list (live via in-game `ban`/`unban`, `pkg/game/bans.go`); MOTD/news/etc.
and help (live via `tedit` and `reload`, wiz_system); `misc/*` report queues.
These get links and read-only status, not duplicate editors. `text/xnames`
has no reload path: the console edits it as a typed string list and marks
**restart required** — adding an in-game `reload xnames` would change the
command surface (R2) and is not proposed.

### Runtime toggles (in-memory, no file today)

`wizlock` (login gating, `pkg/session/manager.go`), `nameserver_is_slow`
(`slowns`), `ident` (`pkg/game/other_settings.go`). All three become live
toggles on a Maintenance card driving the same manager state the in-game
commands drive, so no second semantics exist.

### Deliberately not settings

Engine constants (pulse periods, `PULSE_*`, `SECS_PER_MUD_HOUR`,
`autosaveTime`, combat and save-throw formulas), board capacities, JWT
lifetimes, lockout thresholds, SQLite pragmas, HTTP server timeouts, the
production WebSocket origin allowlist: locked, per R4. Harness-only vars
(`DP_SEED`, `DP_CLOCK`, `DP_FIXED_TIME`, `DP_DRAW_LOG*`, `DP_FRESH_MUD`,
`DP_ALLOW_NO_DB`, `DP_DUMP_API_ROUTES`) and launcher plumbing
(`ADMIN_UI_DIR`, `ADMIN_STORE_PATH`, `DP_NOTIFICATION_*_PATH`, `DATABASE_URL`,
`DP_SQLITE_PATH`, world/scripts/db paths) are display-only or absent from
the console. `ENVIRONMENT` is shown read-only as the running mode.

## Typed schemas, pages, and applying changes

One **Configuration** page under Staff tools, implementor-gated (see
Decisions), with a card per group: Network (the network design's scope),
Sessions & limits, Security & access, Integrations, Maintenance, Deployment
(read-only: version, commit, mode, db/world paths, managed vs checkout).
Each group is a Huma-typed struct; per-field metadata — owner, apply mode,
risk class, secret, env-override — rides schema extensions so the generated
UI client renders badges without a parallel hand-maintained list.
Validation lives in the schema and handler: port ranges and collisions, CIDR
parsing, HTTPS-only URLs, enum modes, secret minimum lengths; a rejected
save names the field, never echoes a secret (the notifications precedent,
`admin-notifications.md`).

Saving follows the notifications pattern exactly: one file-level revision,
`PUT /admin/settings/{group}` with the loaded revision, stale revision →
409 "another administrator changed these settings; reload to compare", sticky
save bar, dirty diff summary, `useBlocker` unsaved-changes guard. Each save
keeps one previous generation (`settings.json.previous`) for trivial
recovery, audits every changed field old→new with secrets redacted, and
recomputes the restart banner: the fields that need a restart are listed by
name with an **Apply and restart** action that runs the network design's
restart transaction (who-is-online check, confirm, receipt, owned-process
readiness plus requested-listener bind, restore-previous on failure). Live
fields take effect through a settings manager that swaps values into the
atomics the code already reads; v1 live scope is exactly the fields the
inventory marks live — everything else stays restart-marked rather than
growing new concurrency surface.

## Backup, restore, audit

**Export** (`GET /admin/settings/export`) downloads a format-stamped JSON
bundle of every group with the revision, server version and date. Secrets
are excluded and appear as `"not-exported"` placeholders; the instance-level
full backup (secrets included) already exists inside every update snapshot,
and the launcher CLI remains the path for offline copies.

**Restore** is two-phase so the preview is server-validated, not
client-guessed. Upload (`POST /admin/settings/restore/validate`) parses and
validates the bundle against the schemas; the response is a field-by-field
diff (the UI reuses `admin-ui/src/lib/lineDiff.ts`; the conflict view never
merges), a list of unknown keys that will be ignored, which fields are
secret and therefore untouched, and whether the result needs a restart.
`POST /admin/settings/restore/apply` with explicit confirmation then goes
through the same save path — revision-checked, audited as `settings.restore`
with the bundle's version and field count, restart banner recomputed.

**Audit**: every mutation above writes the existing JSONL audit log via the
`auditAdmin` helper (`pkg/audit/logger.go`; actions `settings.save.<group>`,
`settings.restore.apply`, `settings.restart`), with field-level old→new
details, secrets redacted, and the initiating admin and pseudonymized IP.
A History card on the page reads a server-filtered
`GET /admin/settings/audit`. The launcher's on-disk receipts remain the
transaction truth for restarts and updates; the audit trail records the
human initiator beside them.

## Updates

**Current release display**: release builds already stamp `main.version`
(`cmd/server/main.go:276`, same ldflags as the launcher), so
`GET /admin/server` grows `version`, `commit`, `managed` (launcher present)
and, when managed, the bundle path from control status. The dashboard's
"current release / last update" tile (stage B) reads the same endpoint; its
drill-down lands on the Updates page.

**Check for updates** (`POST /admin/updates/check`, read-only, rate-limited,
audited): the console cannot reach GitHub and should not gain updater code —
the game forwards a `update-check` control message and the launcher, which
owns release fetching, queries the published latest-release metadata only:
no download, no staging, nothing stopping. The response carries current and
latest versions, up-to-date flag, asset size and release URL. Publisher
verification (signed release manifest, pinned key) stays where it is — at
staging time — untouched by this read path.

**Update now** runs the exact `darkpawns update` transaction
(`internal/portable/update.go`), which already has the hard parts: stage and
verify while the world keeps running (`PrepareUpdate`, update.go:29-144),
then the only-downtime phase — graceful stop with forced save, full
instance snapshot with per-file hashes, active-bundle pointer flip (no
running executable is replaced), candidate started by its own supervisor
(`_serve-update --transaction`), health-gated on owned-process readiness
plus bundle identity within 40 s, commit receipt, and **automatic rollback
to the verified snapshot on any observed failure**; an interrupted
transaction is journalled on disk and blocks start until
`darkpawns update recover` — the browser never invents a parallel updater.

The console flow:

1. **Pre-flight**: the dialog shows who is online (`GET /admin/players`
   names, levels, rooms), warns if other admin sessions are open, and
   refuses while another update transaction is journalled.
2. **Confirm**: previous → new version, SHA-256, expected downtime ("about
   a minute; players are disconnected and every character is saved" — the
   existing stop semantics, main.go's graceful drain), and, per the network
   design's honesty rule, that local success says nothing about LAN/WAN
   reachability.
3. **Apply** (`POST /admin/updates/apply` → control `update-trigger`): the
   launcher answers `202` with a transaction id immediately and runs
   stage → apply. The game process that accepted the browser request is
   about to be stopped, so it never blocks on the outcome — the network
   design's rule that no restart lives in a dying process's goroutine
   applies verbatim to updates.
4. **Status that survives the restart**: the UI keeps the transaction id in
   session storage and polls `GET /admin/updates/status`, treating refused
   connections during the stop/start window as "applying", with elapsed
   time. When the new server answers, that endpoint reports the on-disk
   journal and receipt (`.updates/transaction.json`,
   `.updates/<id>/receipt.json` — read-only, same account, private
   instance directory; reading state is not control): staged, applying,
   testing, **committed** (receipt id matches the stored one; new version
   shown), **rolled back** (previous restored, backup retained), or
   **pending recovery** with the exact `darkpawns update recover` command
   rendered from the journal. A browser refresh mid-update lands on the
   same status view — that is the point of journal-backed status.

Only one update may run: the launcher's `.lifecycle.lock` and pending-update
gate serialize it, the external `supervise` monitor already defers to an
update transaction, and sign-in service anchors are already updater-capable
or refused — all inherited because the browser path calls the same code.
On non-portable (checkout/systemd) deployments the Updates page is
read-only: version shown, check and update disabled with a pointer to
`DEPLOYMENT.md`, because there is no launcher to hand off to.

## What stays outside the browser

Extending the network design's list: JWT secret creation/rotation and TLS
profile provisioning (launcher/CLI), service registration and supervision
mode, update channels and policy (latest stable only; scheduling and
maintenance windows are future work, portable-release.md), filesystem
permissions and service accounts, database/world/scripts locations
(display), harness variables, and every fidelity-locked engine constant.
`instance.json` while stopped remains the operator escape hatch.

## Validation boundaries

Implementation tests must cover: schema rejection of out-of-range ports,
listener collisions, malformed CIDRs, non-HTTPS URLs; revision conflicts on
save and restore; write-only secrets (never echoed in any response or audit
detail, including schema-error paths); export containing no secret material;
restore diff correctness and apply atomicity with the previous-generation
file retained on failure; env-override badge precedence; the settings
manager swapping live values without a dropped request; update status
rendered from journal fixtures for every phase including committed,
rolled-back and pending-recovery; update-check leaving nothing staged; and
the scoped control token being unable to invoke raw `/stop`. Each test must
be able to fail against a plausible wrong implementation (R5h).

## Backlog (captured ideas, not promises)

Update channels and scheduling; a maintenance-window warning broadcast
before Update now; a browser `update recover` button; start/stop lifecycle
buttons for stopped instances; JWT rotation flow; passphrase-encrypted
full-settings export; per-group authority tiering; manual "back up now"
through the control channel; editing `mudlet_directory` and the DNS cache
toggle.

## Decisions for Zach

- **Persistence split**: game settings move to a revisioned
  `lib/data/settings.json` plus a shared encrypted vault; `instance.json`
  stays launcher-owned via the control channel. Alternative rejected:
  folding everything into `instance.json`'s `server_env`, which is an
  unvalidated string map and would put secrets in env.
- **Environment precedence after migration**: the file wins on portable
  instances; env remains an override for checkout deployments, surfaced as
  a badge. Alternative: env always wins, which would silently break the
  "nobody hand-edits config" promise the page makes.
- **Live scope v1**: only the fields the inventory marks live (per-use
  reads and existing atomics) hot-apply; everything else is restart-marked.
- **Authority**: the Configuration and Updates pages are implementor-only
  (`admin` role, level 40, a `staffAdminGate`-style middleware), matching
  the network design; the Maintenance toggles additionally respect their
  in-game command levels. Alternative: HIGOD (36) for game-owned groups,
  implementor for launcher-owned.
- **Update-now semantics**: immediate stop with forced save, exactly the
  CLI's behavior, no new player-facing bytes (R4). The maintenance-window
  warning broadcast is backlog. Alternative: warn-and-wait in v1.
- **Exports never contain secrets**; restore touches only non-secret
  fields. Alternative: a passphrase-encrypted full bundle (backlog).
- **No browser recovery in v1**: a pending or interrupted transaction
  renders the exact `darkpawns update recover` command instead of a button,
  so the recovery path stays single-sourced.
- **Engine tunables stay locked** (R4): pulses, autosave, combat cadence
  and the rest are constants, not settings; confirm this remains the line.
- **One previous generation** of `settings.json` is retained per save;
  deeper history is the audit log's job, not the file's.
