[![CI](https://github.com/zax0rz/darkpawns/actions/workflows/ci.yml/badge.svg)](https://github.com/zax0rz/darkpawns/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/zax0rz/darkpawns)](go.mod)
[![License](https://img.shields.io/github/license/zax0rz/darkpawns)](LICENSE)

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

**CircleMUD 3.0, rewritten in Go.** Dark Pawns is a multiplayer text RPG in the
DikuMUD lineage — the classic C server faithfully ported to a single modern Go
binary, running live today and built so anyone can host their own instance.

[Play in your browser](https://darkpawns.org/play) ·
[Website](https://darkpawns.org) ·
[Player guide](docs/player-guide/player-guide.md) ·
[Report a bug](https://github.com/zax0rz/darkpawns/issues)

## Play

The live game is up now. No account signup, no download.

```sh
telnet darkpawns.org 7777
```

Or play in the browser at [darkpawns.org/play](https://darkpawns.org/play) —
same game, same world, WebSocket under the hood.

Returning from the 2004 era? The world files are the preserved originals, and
the port's prime directive is that the game plays byte-for-byte like the C
server did: same commands, same combat, same quirks. See
[the player guide](docs/player-guide/player-guide.md) for classes, character
creation, and how to connect.

## Run your own

This is the point of the project: live MUDs out there running Dark Pawns. One
Go binary, no external services required — persistence is an embedded SQLite
database by default (PostgreSQL is optional).

```sh
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

export JWT_SECRET="$(openssl rand -hex 32)"

go build -o server ./cmd/server
./server
```

That's a running MUD. Connect with `telnet localhost 7777` or open
[http://localhost:4350](http://localhost:4350). On an empty database, the first
character you create becomes the game administrator — create it before opening
the instance to other players.

One binary, three surfaces: telnet (`-telnet-port`, default 7777), HTTP +
WebSocket (`-port`, default 4350, serving the browser client and a
[Huma](https://huma.rocks)-powered JSON API with OpenAPI at `/openapi.json`),
and an embedded database. `./server -h` lists every flag.

For everything beyond the quickstart — PostgreSQL, reverse proxies, backups,
the admin frontend — see [Running Dark Pawns](DEPLOYMENT.md). Official host
access and deploy procedures for `darkpawns.org` itself live in a private ops
repo; the public guide covers operating *your* instance.

## What it is

- **The game:** combat, spells, skills, equipment, shops, clans, houses,
  bulletin boards, and Lua scripting — ported from the original C with the
  original behavior as the reference implementation.
- **The fidelity contract:** the Go server must emit the same player-facing
  bytes as the C original. The [rulebook](docs/fidelity/RULEBOOK.md) is the
  law; a C-versus-Go differential harness (`cmd/dp-oracle-diff`) plus scenario
  fixtures and per-command evidence manifests back it up. Presence of a system
  here is not a claim that every branch is verified — check the fidelity
  records for coverage.
- **The API:** REST endpoints via Huma with generated OpenAPI, WebSocket
  sessions, Prometheus metrics, audit logging, and a React admin frontend
  (`admin-ui/`).
- **webOLC (in development):** a web-based online creator — rooms, mobs,
  objects, shops, and zones editable from the browser through
  `/admin/olc/*`, sharing one editor core with the classic telnet OLC so both
  stay byte-identical. Under active development; see `pkg/olc/` and
  `admin-ui/src/components/olc/`.
- **Agent tooling:** a command-line client and server-side hooks for AI agents
  playing as full players under the same rules. Scope and status:
  [agent CLI guide](docs/agents/dp-agent.md),
  [research notebook](docs/research/README.md).

## Repository layout

| Path | Contents |
| --- | --- |
| `cmd/server/` | Game server entry point (the one binary) |
| `pkg/` | Game logic, sessions, transports, persistence, OLC, and services |
| `cmd/dp-oracle-diff/` | C-versus-Go differential harness and scenarios |
| `lib/world/` | Preserved world files and scripts loaded by the server |
| `lib/text/` | Preserved game text and credits |
| `src/` | Original C source — read-only reference, never edited |
| `web/` | Browser game client and HTTP handlers |
| `admin-ui/` | React admin frontend (incl. webOLC components) |
| `website-astro/` | Authored Astro website ([darkpawns.org](https://darkpawns.org)) |
| `docs/` | Guides, fidelity evidence, research, and history |

## Develop

Read [AGENTS.md](AGENTS.md) first — repository conventions and required checks.
[CONTRIBUTING.md](docs/CONTRIBUTING.md) covers the contribution workflow, the
fidelity contract, and where help is wanted.
Gameplay changes must follow the [fidelity rulebook](docs/fidelity/RULEBOOK.md):
player-facing bytes are law, the C source wins disputes, nothing is invented.

```sh
make hooks        # one-time: install the pre-push hook
make fmt          # gofumpt formatting (CI enforces it)
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

No commit without all four passing. The Go unit tests don't need the C oracle;
[development setup](docs/DEV-SETUP.md) covers the optional oracle harness.

Bug reports are most useful with the commands entered, what happened, and what
you expected. For fidelity work, verify the actual C call path and bring
evidence.

## Documentation

Start at the [documentation index](docs/README.md). Entry points:

- [Running Dark Pawns](DEPLOYMENT.md) — operate your own instance
- [Fidelity rulebook](docs/fidelity/RULEBOOK.md) — the port contract
- [Depth testing](docs/fidelity/DEPTH_TESTING.md) — evidence and remaining work
- [Development setup](docs/DEV-SETUP.md) — toolchain and oracle
- [Agent CLI](docs/agents/dp-agent.md) — agents as players
- [Research notebook](docs/research/README.md) — the open research artifact

Historical briefs and reports are preserved for context, not as a task queue.

## Credits and license

Dark Pawns grew from the work of its original developers, world builders, and
players. The preserved [game credits](lib/text/credits) identify CircleMUD
3.0, Jeremy Elson, and its DikuMUD foundations. The
[original C repository](https://github.com/rparet/darkpawns) was published by
R.E. Paret (Frontline). The Go port is maintained by
[zax0rz](https://github.com/zax0rz).

[MIT License](LICENSE) — run it, host it, fork it.
