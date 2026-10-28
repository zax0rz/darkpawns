# Dark Pawns admin and webOLC

The React app in this directory is served by the Go server at `/admin/`. The
builder-facing guide is available at `/admin/workshop/help` after sign-in.

The header search (or Ctrl/⌘ K) searches loaded zones, rooms, mob prototypes,
object prototypes, and shops across VNUMs, names, keywords, descriptions, and
script names where present. A result opens its editor. List-page filters stay
useful for narrowing one entity type after browsing to that section. Search
uses committed live definitions, so an uncommitted draft appears only after
**Commit draft**. Results from outside a builder's assigned zone may be visible,
but the editor still enforces the server's OLC permissions.

## Give a zone-assigned builder access

Staff must give the character the appropriate in-game builder level and assign
its OLC zone. The web role shown under the character name is a navigation aid;
the OLC API checks the character's in-game level and assigned zone for each
resource. Builders below level 35 can edit only their assigned zone. Confirm
the zone assignment before sharing access. See `pkg/olc/authorization.go` and
`pkg/admin/olc.go` for the server rules.

Send the builder the `/admin/` URL and their character credentials through an
appropriate private channel. The app uses those credentials to obtain an admin
session. Do not include a password in issue reports or screenshots.

## First room edit

1. Sign in at `/admin/` and choose **Zones & rooms**.
2. Open the assigned zone. Its **Builder workshop** shows existing room VNUMs
   and a suggested free VNUM when room creation is permitted.
3. Open an existing room, then choose **Edit room**. To make a room, choose the
   suggested new-room action from the zone workshop.
4. Edit the draft. The editor claims the room while it is open; another web or
   in-game OLC editor may temporarily block the same resource.
5. Choose **Commit draft** to apply the changes to the live world. Choose
   **Discard** to leave it unchanged.
6. Choose **Save zone file** after committing. This writes the canonical world
   file. Confirm that the zone is absent from **Workshop & saves** → **Pending
   zone saves** before finishing.

Committing and saving are separate. A committed zone marked dirty has live
changes that still need a file save. The workshop lists held claims and pending
saves. If an action is refused, report the resource VNUM and exact error to
staff. The **Builder guide** page shows additional actions and access
requirements reported by the server schema for the signed-in character.

## Edit the welcome messages

Open **Staff tools → MOTD** or the dashboard's **MOTD** card. Builders can read;
GRGOD staff (level 38) can save. Choose **MOTD** or **Immortal MOTD**, edit the
draft, then check colors and wrapping at 40, 80 or 100 columns before saving.
Each tab keeps its draft. Saves take effect at once and are audited; a conflict
keeps your draft for comparison with the saved copy. No reboot is needed.

Discord insertion asks for your own HTTPS invite. Both snippets change only
the draft. Check Notifications before inserting the gossip notice: an automatic
notice may already be enabled, and the preview does not include it. Each message
has a 2,400-byte limit, including UTF-8 characters and two-byte line breaks.
See the [MOTD operator guide](../website-astro/src/content/docs/server/motd-editor.md)
for color codes, draft recovery and save conflicts.

## Development

```bash
cd admin-ui
npm ci
npm run dev
npm run lint
npm run build
```

Vite serves the UI at `http://127.0.0.1:5173/admin/` in development. API
requests need a Go server on port 4350; without one, the login page still
renders but authenticated editor work is unavailable. Set `ADMIN_API_TARGET` to
use another Go server address. The same `/admin/` asset base is used in development
and production; no `--base` build flag is needed.

The Go server loads the production build from `ADMIN_UI_DIR`, or from
`admin-ui-dist` in its working directory. See [DEPLOYMENT.md](../DEPLOYMENT.md#optional-admin-interface)
for the build and copy commands. The editor API is implemented in `pkg/admin/olc*.go`,
with shared OLC behavior in `pkg/olc/`.

## Edit a saved player

Choose **Staff tools → Players** (`/admin/staff/players`). This is a staff editor,
separate from the online-player inspector in Operations. It requires an admin
session and a current saved level of HIGOD or higher. The API rechecks the staff
record on every request. Staff below IMPL can edit only lower-level characters
and cannot grant a level above their own.

The list reports the total saved-character count (and matching count when
filtered). Beneath the selected name, **Last saved** shows the elapsed time with
an exact local date on hover. This is the existing character-save timestamp,
not a distinct login event; missing timestamps show **unknown**.

The character must be fully disconnected, including menus, character entry,
switched bodies and retained/linkdead characters. A session-manager name
reservation excludes new entry while a read or write runs. Deployments without
that provider refuse editing with 503.

- **Character stats:** level, experience, health/mana/movement and their maxima,
  and the six base attributes plus exceptional strength. Save or discard the
  draft before changing equipment or resetting a password.
- **Equipment & inventory:** grant a prototype by vnum, move an existing item
  to inventory or one of the 22 equipment positions, or remove it. Every action
  saves immediately. A container's contents and saved instance state travel
  with it; removing a container removes its contents. Occupied slots and
  equipping a stack are rejected. Slot selection is an explicit staff override,
  rather than a simulation of the in-game wear command. Normal entry-time
  equipment effects and checks still run on the next login.
- **Password:** set and confirm a password of 8 to 72 bytes. The server uses
  bcrypt and clears failed-login counts and the stored account lockout. Passwords
  and hashes are excluded from responses and audit details.

All writes require the revision returned by the read endpoint. A stale browser
receives 409 and keeps its draft for review; reload to obtain the current record.
Character, optional password, and equipment snapshot updates commit in one
SQLite transaction. Item edits preserve an existing rent/cryo header and create
an ordinary rented snapshot if none exists. Stats and password edits do not
create or rewrite object snapshots. Unrelated character data stays intact.
Legacy object JSON requiring migration is refused, never silently rewritten.

The Huma/OpenAPI operations are:

| Method | Route | Purpose |
| --- | --- | --- |
| GET | `/admin/characters?q=` | Search saved names |
| GET | `/admin/characters/{name}` | Read record, editable field catalog and revision |
| PATCH | `/admin/characters/{name}` | Save changed numeric `values` with `revision` |
| POST | `/admin/characters/{name}/items` | Grant, move or remove a saved item |
| POST | `/admin/characters/{name}/password` | Reset password and clear account lockout |

These are modern staff operations permitted by R4. They do not add telnet
commands or change ordinary gameplay output. Full/combined oracle verification
remains the batched pre-merge gate; targeted checks cover the shared offline
compare-and-save path, re-entry and rent reporting.
