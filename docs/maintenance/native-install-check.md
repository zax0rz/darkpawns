# Native installation check — 2026-09-09

Verified source: `0505c87b7` (`origin/main`, including PR #1428). The check ran
in a detached worktree with PostgreSQL 16 databases created just for this test.
No production database, player, service, or deployment was used.

## Results

| Check | Result |
| --- | --- |
| Build from committed source with `CGO_ENABLED=0 go build ./cmd/server` | Passed |
| Boot with `-world ./lib` | Failed: parser cannot find `lib/wld`; old deployment guide was wrong |
| Boot with `-world ./lib/world -web ./web` | Passed |
| Fresh PostgreSQL schema initialized by the server | Passed |
| HTTP browser HTML, JavaScript, and CSS | Served successfully |
| First character created through telnet on an empty database | Entered The Board Room Of The Immortals |
| Second character created through telnet | Entered A Burning Hut as a mortal |
| Stop the process, start it again, log in with saved character | Passed; entered Temple Infirmary |
| Lowercase spelling of the saved name | Loaded the same character |
| Wrong password followed by the correct password | Rejected the wrong password; successful retry |
| `score` after reconnecting | Returned the saved character's name |
| Existing real-database telnet persistence test | Passed, with `-count=1` |
| Admin frontend `npm ci` and `npm run build` from clean source | Passed; bundle-size warning only |

The live native check used a production-mode signing secret, a real database,
and no `DP_ALLOW_NO_DB`, frozen clock, fixed seed, or seeded player rows. The
first and second characters went through the normal creation prompts. HTTP asset
checks do not certify interactive browser login or administrative operations (R5f).

## Reproduce

Follow [Running Dark Pawns](../../DEPLOYMENT.md), using a fresh database. From
the checkout root, the verified command shape is:

```bash
CGO_ENABLED=0 go build -o darkpawns-server ./cmd/server
./darkpawns-server -world ./lib/world -web ./web -port 4350 -telnet-port 7777
```

`DATABASE_URL` and a stable `JWT_SECRET` must already be exported. Create the
first administrator, then a second character. Save, stop the server normally,
restart with the same configuration, and log back in as the second character.

For the existing automated persistence check, use a disposable database:

```bash
export DP_TEST_DB_URL='postgres://USER:PASSWORD@localhost:5432/darkpawns_test?sslmode=disable'
go test ./tests/e2e -run '^TestTelnetSmoke_PersistenceRoundTrip$' -count=1 -v
```

That test creates and deletes its own test rows and starts server processes.
Never point it at a live player database. Unlike the manual first-character check,
it seeds a sentinel player so it can specifically test mortal persistence.

## Corrections made

- Restored README `-world ./lib/world` after the preceding housekeeping pass
  incorrectly changed it to `./lib`; corrected the older deployment guide too.
- Fixed the same incorrect world-directory default in `make run` and provided
  an explicit web asset directory there.
- Documented that process-relative state lives in `lib/data/` for this layout.
- Made `.env.example` describe native configuration: local database host, explicit
  environment export, optional agent settings, and CLI flags for paths and ports.
- Commented out placeholder TLS certificate paths: supplying both enables HTTPS
  even when `USE_TLS=false`, so loading the old example would fail to serve HTTP.
- Documented the admin frontend build and `lib/admin-ui-dist/` location.

## Container workflow remains unverified

**Disposition:** the subsequent [container retirement](container-retirement.md)
removes this workflow from the supported installation paths. The findings below
record why it was unverified; they are not a pending launch checklist.

Docker and Podman were unavailable on the verification host. No container build
or Compose startup is claimed. The native result does not certify Docker (R5f).
The following are confirmed source-level gaps to resolve in a container pass:

- Compose passes `/app/lib` instead of `/app/lib/world`.
- Compose does not provide the signing configuration or publish telnet.
- The image requires prebuilt admin output instead of building it.
- Admin output is copied to `/app/admin-ui-dist`, but the process changes to
  `/app/lib` when using `/app/lib/world`, so the default asset path differs.
- The legacy read-only world bind mount does not support `lib/data/` writes.
- The Python sidecar, its credentials, the bootstrap SQL, and persistent volumes
  need validation as part of the Compose workflow.

No container configuration was changed in this native verification pass.
