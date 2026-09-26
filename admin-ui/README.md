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
renders but authenticated editor work is unavailable. The Go admin router serves the production
build from `admin-ui/dist`; see `pkg/admin/router.go` and the root Makefile for
the build integration. The editor API is implemented in `pkg/admin/olc*.go`,
with shared OLC behavior in `pkg/olc/`.
