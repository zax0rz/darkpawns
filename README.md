```
        (_____)           (_)    (_____)
  _     /  __ \           | |    |  __ \                            _
 ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;
  =    /| |  | |/ _` | '__| |/ / |  ___/ _` \ \ /\ / / '_ \/ __|    =
.***.  /| |__| | (_| | |  |   <  | |  | (_| |\ V  V /| | | \__ \  .***.
~~~~~  /|_____/ \__,_|_|  |_|\_\ |||   \__,_| \_/\_/ |_| |_|___/  ~~~~~
                                 |||
                                 |||
                                 `.'
```

# Dark Pawns

Dark Pawns is a multiplayer text RPG in the DikuMUD/CircleMUD lineage, with its
original C server ported to Go. The server loads the preserved world files and
accepts players through telnet and a browser client.

[Play in your browser](https://darkpawns.org/play) ·
[Website](https://darkpawns.org) ·
[Report a bug](https://github.com/zax0rz/darkpawns/issues)

```sh
telnet darkpawns.org 7777
```

## Project status

The Go server is running publicly at `darkpawns.org`. This repository contains
the game, original world data, browser client, Astro website, and development
tools.

The port aims to preserve the original game's player-facing behavior. Fidelity
work is ongoing: having a Go implementation or one passing comparison does not
prove every branch matches the C server. The
[rulebook](docs/fidelity/RULEBOOK.md) defines the contract, and
[depth-testing guide](docs/fidelity/DEPTH_TESTING.md) explains the evidence and
remaining verification work. The original C in `src/` is the read-only reference.

Versioned distribution, an npm entry point, and an installation TUI are planned.
The supported installation today is a native Go binary with PostgreSQL.

## Run your own instance

Install Go at the version required by [go.mod](go.mod) and provision a PostgreSQL
database. The server creates and migrates its schema at startup.

```sh
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

export DATABASE_URL='postgres://USER:PASSWORD@localhost:5432/darkpawns?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"

go build -o server ./cmd/server
./server -world ./lib/world -web ./web -port 4350 -telnet-port 7777
```

Replace the database credentials with your own. Keep the signing secret stable
across restarts. The binary reads environment variables; it does not load
`.env` automatically.

Connect with `telnet localhost 7777` or open
[the local browser client](http://localhost:4350).

On an empty database, the first character becomes the game administrator. Create
that character before opening the instance to other players.

With this layout, the server runs from `lib/` and writes runtime state under
`lib/data/`. Preserve the database and the instance's `lib/` tree. See
[Running Dark Pawns](DEPLOYMENT.md) for database setup, configuration, backups,
and the optional admin frontend. The
[native installation check](docs/maintenance/native-install-check.md) covers
character creation and saved login after a server restart.

Docker, Compose, and Kubernetes deployment recipes are retired.

## What is here

- **The game:** combat, spells, skills, equipment, shops, clans, houses, bulletin
  boards, and Lua scripts, with the original C behavior as the reference.
- **Player connections:** telnet and a browser client backed by WebSocket sessions.
- **Operations:** PostgreSQL persistence, a React admin interface, audit logging,
  and a Prometheus metrics endpoint.
- **Agent tooling:** a command-line client and server-side memory integration.
  See the [agent CLI guide](docs/agents/dp-agent.md) and
  [research notebook](docs/research/README.md) for their scope and status.
- **Fidelity tooling:** a C-versus-Go differential harness, scenario fixtures,
  and per-command evidence manifests.

These systems share the same server. Their presence is not a claim that all
behavior has been verified; use the fidelity records when assessing coverage.

## Repository layout

| Path | Contents |
| --- | --- |
| `cmd/server/` | Game server entry point |
| `pkg/` | Game logic, sessions, transports, persistence, and supporting services |
| `cmd/dp-oracle-diff/` | Differential test harness and scenarios |
| `lib/world/` | World files and scripts loaded by the server |
| `lib/text/` | Preserved game text and credits |
| `src/` | Original C source, kept as a read-only reference |
| `web/` | Browser game client and HTTP handlers |
| `admin-ui/` | React admin frontend |
| `website-astro/` | Authored Astro website |
| `website/` | Shared static assets, generators, history data, and Caddy references |
| `docs/` | Maintained guides, fidelity evidence, research, and historical notes |

Official host access, secrets, and deploy/rollback procedures live in the private
`zax0rz/darkpawns-ops` repository. The public running guide describes how to
operate your own instance.

## Development

Read [AGENTS.md](AGENTS.md) for repository conventions and required checks.
Gameplay changes must follow the [fidelity rulebook](docs/fidelity/RULEBOOK.md);
do not re-port or edit the C reference files.

```sh
make hooks
make fmt
go build ./...
go vet ./...
go test ./...
go test ./pkg/game/...
golangci-lint run ./...
```

The Go unit tests do not require the separate C oracle. See
[development setup](docs/DEV-SETUP.md) for configuring the oracle harness and
[the installation check](docs/maintenance/native-install-check.md) for the
database-backed login test. CI uses disposable database service containers for
testing; they are not a server deployment requirement.

Bug reports are useful when they include the commands entered, what happened,
and what you expected. For fidelity changes, check the actual C call path and
include evidence rather than relying on an older brief.

## Documentation

Start with the [documentation index](docs/README.md). The main entry points are:

- [Running Dark Pawns](DEPLOYMENT.md)
- [Fidelity rulebook](docs/fidelity/RULEBOOK.md)
- [Depth testing and handoffs](docs/fidelity/DEPTH_TESTING.md)
- [Development setup](docs/DEV-SETUP.md)
- [Agent CLI](docs/agents/dp-agent.md)
- [Research notebook](docs/research/README.md)

Historical briefs and reports are preserved, but are not a current task queue.

## Credits and license

Dark Pawns grew from the work of its original developers, world builders, and
players. The preserved [game credits](lib/text/credits) identify CircleMUD 3.0,
Jeremy Elson, and its DikuMUD foundations. The
[original C repository](https://github.com/rparet/darkpawns) was published by
R.E. Paret (Frontline). The Go port is maintained by
[zax0rz](https://github.com/zax0rz).

See [LICENSE](LICENSE) and the original source notices.
