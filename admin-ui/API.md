# Generated admin API

`make admin-api` exports Huma registrations with `cmd/admin-openapi`, then runs
openapi-typescript. `src/api/schema.d.ts` is generated; do not edit it. The
openapi-fetch client checks paths, methods, parameters and request bodies at call
sites. `http.ts` preserves same-origin cookies, the CSRF header, unauthorized
redirects and structured API errors. OLC normalizers remain view models for the
server's existing field naming; they are not an independent wire contract.

`ADMIN_OPENAPI_URL` can select a running server's `/api/openapi.json`. The offline
export includes conditional mudlog operations so generation is complete without
production data or credentials. The public health endpoint returns text, not JSON.

## Audit of the five HandleFunc routes in the release brief

All five already have method-qualified Huma operations in
`pkg/admin/huma_completion.go`. The unqualified registrations in `router.go`
handle unsupported methods with legacy parsing and error responses. They must
remain for compatibility; they are not untyped implementations of the supported
operations.

| Path | Typed operation | Response |
| --- | --- | --- |
| `/admin/players/{name}` | GET `get-player` | `playerDetailResponse` |
| `/admin/players/{name}/save` | POST `save-player` | `statusOutput` |
| `/admin/players/{name}/kick` | POST `kick-player` | Registered operation; errors remain server-defined |
| `/admin/shops/{keeper}` | GET `get-shop` | `shopResponse` |
| `/admin/findings/{id}` | PUT `update-finding` | `Finding` |

The actual non-Huma allowlist in `pkg/admin/drift_gate_test.go` consists of the
SPA redirect/fallback, static assets, Prometheus text exposition and the mudlog
SSE stream. No JSON admin operation needs conversion. The drift gate checks that
every Huma path is mounted and every ordinary route is documented or allowlisted.

Finding update fields are independently optional: the handler accepts status,
Linear issue ID, or both. Their input schema now reflects that existing contract.
The generator currently declares a TypeScript 5 peer. The npm override uses the
console's TypeScript 6 installation; generation, compilation and lint are verified
together. Keep the lockfile when changing either tool.

Nullable arrays are handled explicitly by views instead of hidden by handwritten
TypeScript interfaces. Online player items expose `name`, `count`, `slot`, `vnum`;
the old frontend's `short_desc` and `wear_location` properties were not API fields.

## Preview and evidence

Run the opt-in real-handler fixture server from the repository root:

```sh
ADMIN_PREVIEW=1 go test -tags adminpreview ./pkg/admin -run '^TestConsolePreview$' -timeout 0
```

All mutable fixture data uses temporary directories. It binds only to
`127.0.0.1:4350`, or to `ADMIN_PREVIEW_PORT` when that is set, so a second agent
can run its own fixture beside a first. Run Vite separately, then capture and
smoke-test every route:

```sh
npm run dev --prefix admin-ui -- --host 127.0.0.1 --port 5187
cd admin-ui && node scripts/smoke.mjs
```

The dev server proxies `/admin`, `/health`, `/ws`, `/preview-session` and
`/preview-reset` to `ADMIN_API_TARGET` (default `http://localhost:4350`), so
pointing it at a fixture on another port keeps cookies, session and reset on one
origin. `ADMIN_PREVIEW_URL` selects the fixture address the smoke script calls
(default `http://127.0.0.1:4350`).

The development-only `/admin/component-preview.html` displays the chart foundation with
labelled synthetic data. It is excluded from the production build. Set
`ADMIN_COMPONENT_PREVIEW=1` when running the smoke script to retain that proof.

Install Playwright's Chromium if needed, or set `CHROMIUM_PATH` to an existing
Chromium executable. `ADMIN_UI_URL` selects another local baseline preview;
`ADMIN_EVIDENCE_DIR` selects the output directory. Captures use synthetic world
records and real API handlers. They do not test a live game connection or send
outbound notifications. Never point this fixture workflow at production.

### Check asset serving

Run `node scripts/asset-smoke.mjs` from `admin-ui` against the disposable preview
server above. Set `ADMIN_ASSET_ORIGIN` to its address. For a built console, set
`ADMIN_UI_DIST` to the same directory as the preview server's `ADMIN_UI_DIR`.
The check compares every asset's bytes and loads the dashboard and file editor
in a browser. Root `/assets/` requests fail the check.

Run it again with `ADMIN_ASSET_MODE=dev` and `ADMIN_ASSET_ORIGIN` pointing at
Vite. This also checks API forwarding, hot reload's connection, the component
preview and refreshing the sign-in page.
