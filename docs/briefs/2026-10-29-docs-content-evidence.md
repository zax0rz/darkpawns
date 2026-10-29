# Docs prose migration evidence, 2026-10-29

This note records the prose-only content migration and reader-driven corrections. Page facts were checked against the listed repository sources. The site renderer, generated commands/API/releases, routes, sidebar, styles, search, redirects, and media capture pipeline belong to the integration agent.

## Existing route map

| Old route | New route | Purpose / audience |
| --- | --- | --- |
| `/docs/getting-started/local-server/` | `/docs/build/local-server/` | Run a source checkout locally; developers. |
| `/docs/research/port-fidelity/` | `/docs/project/port-fidelity/` | Explain oracle-based fidelity work; developers. |
| `/docs/server/install/` | `/docs/host/install/` | Install the portable release; operators. |
| `/docs/server/admin-console/` | `/docs/host/admin-console/` | Find console pages, tasks, and access gates; operators. |
| `/docs/server/admin-configuration/` | `/docs/host/admin-configuration/` | Explain managed settings, restart behavior, backups, and checkout overrides; operators. Update transaction details moved to the dedicated update guide. |
| `/docs/server/dashboard-and-logs/` | `/docs/host/dashboard-and-logs/` | Look up dashboard counters, activity, log filtering, and exports; operators. |
| `/docs/server/player-management/` | `/docs/host/player-management/` | Manage saved characters and understand save boundaries; operators. |
| `/docs/server/notifications/` | `/docs/host/notifications/` | Configure outbound notifications and inspect delivery; operators. CI release announcements moved to Project. |
| `/docs/server/motd-editor/` | `/docs/host/motd-editor/` | Edit and save welcome messages; operators. |
| `/docs/server/architecture/` | `/docs/project/architecture/` | Trace server startup and package roles; developers. |
| `/docs/server/contributing/` | `/docs/project/contributing/` | Follow fidelity, formatting, and verification rules; developers. |

## New pages and fact sources

| Page | Purpose / audience | Sources checked | Fresh-reader check needed |
| --- | --- | --- | --- |
| `play/first-hour` | Get a new player from starting gear to a safe training area; players. | `lib/text/help/commands.hlp` topics `WEAR` and `NEWBIE`; `lib/text/help/info.hlp` topics `FAQ`, `MAP`, `RENT`, `GUILD`, `HOMETOWN`; `lib/world/wld/80.wld` rooms 8004, 8008, 8021, 8074, 8101, 8102; `lib/world/wld/163.wld` rooms 16300 through 16304; trainee guard 16303 in `lib/world/mob/163.mob`; `consider`, `kill`, and `flee` help; `lib/world/zon/163.zon`; `pkg/session/comm_cmds.go` newbie channel behavior. | Confirm the Kir Drax'in-only route is clear and the hometown alternative is sufficient. |
| `play/connect-client` | Connect using the web client, Mudlet, or telnet; players. | `website-astro/src/pages/play.astro`; `web/public/index.html`; `web/public/mud-client.js`; `docs/portable-release.md` guided setup/listener defaults. | Confirm first-connection instructions make auto-connect and character creation clear. |
| `host/install` | Install a portable release; operators. | `docs/portable-release.md` channel and signature checks; `cmd/darkpawns/setup.go`; `scripts` and site install stub. | Confirm channel distinction, inspect-before-run sequence, and first successful command are obvious. |
| `host/first-login` | Complete setup, make the first character, and sign into the console; operators. | `docs/portable-release.md` guided setup; `cmd/darkpawns/setup.go`; `pkg/db/player.go`; `cmd/server/main.go`; `src/db.c:3016`; `admin-ui/src/pages/LoginPage.tsx`. | Confirm Implementor (level 40) consequence and local-only listener behavior are unambiguous. |
| `host/admin-console` | Locate tools and understand access gates; operators. | Migrated `website-astro/src/content/docs/server/admin-console.md`; `admin-ui/src/Layout.tsx`; linked dashboard, MOTD, notifications, player editor, and configuration pages. | Confirm the page works as a map rather than an exhaustive reference. |
| `host/update-server` | Run and monitor a managed update; operators. | `admin-ui/src/pages/UpdatesPage.tsx`; `cmd/darkpawns/update_presentation.go`; `docs/portable-release.md` manual update section. | Confirm the transaction continues independently of the browser and rollback diagram matches the operator's mental model. |
| `host/players-cannot-connect` | Diagnose connection failures; operators. | `internal/portable/network_config.go`; `cmd/darkpawns/setup.go`; migrated admin configuration page; `docs/portable-hosting.md`; server listener and web-client connection code. | Confirm local listener check is distinct from external reachability. |
| `reference/glossary` | Look up the player-facing terms appearing in docs; players. | `docs/docs-style.md` §5 and §12; `lib/text/help/info.hlp` topics `AC`, `NEWBIE`, `MOB`, `MANA`, `REMORT`, `RENT`, `TANK`, `GUILD`, `HIGOD`, `GRGOD`, `WHOD`; `lib/text/help/commands.hlp` prompt and score topics; `src/whod.c`; world files for zone/vnum usage. | Confirm terms are brief and definitions answer likely search queries without prior game knowledge. |
| `project/release-announcements` | Explain CI release-note announcements; developers. | `.github/workflows/announce-release.yml`; `docs/publishing-releases.md`; the former section in Notifications. | Confirm contributors can distinguish this workflow from server notifications. |

## Existing pages retained in new sections

- `build/local-server`: checkout requirements, build/run examples, local browser and telnet endpoints, and developer checks from the original page. `DP_ALLOW_NO_DB=1` is now explicitly marked temporary and non-playable; the page explains lost persistence and mail identity. Sources: `go.mod`, `cmd/server`, `lib/world/`, `cmd/server/main.go`, `pkg/db/`, and `web/public/index.html`.
- `project/port-fidelity`: the five-rule overview and C-oracle boundary from the original page. The scenario command is explicitly one example, not a completeness proof. Sources: `docs/fidelity/RULEBOOK.md`, `docs/DEV-SETUP.md`, and `cmd/dp-oracle-diff`.
- `host/admin-configuration`: managed settings behavior and checkout overrides from the original page, with implementation and storage detail moved to “How it works”. Sources: original article, `docs/design/admin-configuration.md`, `internal/portable/`, admin UI settings pages.
- `host/dashboard-and-logs`: dashboard measurements and export scope from the original page. The event time versus next-sample bucket distinction is stated once, with the midnight/hour boundary consequence. Sources: original article, admin UI operations/history pages, logging and history implementation.
- `host/player-management`: edit, rename, delete, and flag behavior from the original page. Deep identity and storage implementation details are reduced to one “How it works” section. Sources: original article, `admin-ui/src/pages/PlayersPage.tsx`, corresponding admin APIs, and the cited C call paths in `src/`.
- `host/notifications`: channel behavior and troubleshooting from the original page; CI release announcements moved to Project. The restore key instruction identifies the matching backup. Sources: original article, `admin-ui/src/pages/NotificationsPage.tsx`, notification configuration and delivery code.
- `host/motd-editor`: draft, save, access, and preview behavior from the original page. It distinguishes builder read visibility from GRGOD save access. Sources: original article, `admin-ui/src/pages/MotdPage.tsx`, `pkg/session`, and the cited C command gate.
- `project/architecture`: startup/package map from the original page. Sources: `cmd/server/main.go`, the listed `pkg/` packages, and `docs/fidelity/RULEBOOK.md`.
- `project/contributing`: formatter, gate, and read-only oracle requirements from the original page and repository instructions. Adds the enforced `make deadcode-ratchet` gate and links setup/process material. Sources: `AGENTS.md`, `Makefile`, `.golangci.yml`, and fidelity docs.

## Media still needed

The Play client, Mudlet, setup, and saved-character editor pages contain explicit `TODO(media)` notes. The media agent is producing captures for the web client and setup, but there is no committed generated asset for the exact Mudlet profile or character flags/identity view in this content branch yet. Existing player-editor screenshots were removed from the page because they are not tied to the required capture pipeline.

## Integrated review

The independent [reader review](2026-10-29-docs-starlight/reader-review.md) records all three reader answers for every authored page, including the first-fight route and recovery commands. [UI review](2026-10-29-docs-starlight/ui-review.md) and desktop/mobile captures accompany the integration.
