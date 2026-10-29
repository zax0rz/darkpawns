---
title: "Install Dark Pawns"
description: "Install the signed portable release with Homebrew, Scoop, winget, Docker, or the bootstrap script, then create your first server instance."
type: guide
audience: operators
verified: v0.1.7
---

Homebrew, Scoop, winget, Docker, and the bootstrap installer deliver the
portable release. The launcher verifies the publisher-signed bundle record on
first run. A package manager's download checks are separate from that publisher
signature check.

Choose the package channel for the machine that will run the world. Every channel
installs the same signed release.

## Before you start

- A machine for the server and a place to keep its instance data.
- Network access to the chosen package source.

## Homebrew (macOS, Linux)

```sh
brew install darkpawns/tap/darkpawns
darkpawns setup            # create an instance and open the setup wizard
```

The formula links the `darkpawns` launcher with its release assets into the
Homebrew prefix. Upgrade with `brew upgrade`.

## Scoop (Windows)

```powershell
scoop bucket add darkpawns https://github.com/darkpawns/scoop-bucket
scoop install darkpawns
darkpawns setup
```

## winget (Windows)

```powershell
winget install DarkPawns.DarkPawns
```

winget installs the portable zip and puts `darkpawns` on PATH. New package
submissions to the winget community repository are reviewed by humans, so the
first version may lag the release by a day or two.

## Docker

```sh
docker run -d --name darkpawns \
  --network host \
  -v darkpawns-data:/data \
  ghcr.io/darkpawns/darkpawns:v0.1.7
```

The image runs the portable launcher as an unprivileged user with the
instance in the `/data` volume; the first boot creates the instance and every
boot verifies the signed bundle. Use `--network host` for now: the launcher
binds its web client to loopback inside the container, which bridge
publishing cannot reach. `latest` always points at the newest release; pin a
version tag for anything players depend on.

## Bootstrap installer (Linux, macOS, or a POSIX shell)

```sh
curl -fsSL https://darkpawns.org/install.sh -o install.sh
less install.sh
bash install.sh
```

The downloaded site stub verifies the installer's publisher signature before
running it. That installer verifies the signed release manifest and archive
before extraction. Reading the stub before you run it shows the URL and code
that will execute. See the [portable release notes](https://github.com/zax0rz/darkpawns/blob/main/docs/portable-release.md)
for the files it installs and how to remove them.

After any install, `darkpawns setup` walks through the first instance, or
`darkpawns init` creates one unattended; `darkpawns update` (or the admin
console's Updates page) keeps it current.

## Check it worked

Run `darkpawns status` from a new terminal. The launcher should report the
instance state and release. Then open the local web client address printed by
`darkpawns setup`.

## If it went wrong

- **The `darkpawns` command is not found.** Open a new terminal so it reads the
  updated PATH, or add the installer's printed command directory to PATH.
- **Setup says a port is unavailable.** Choose another unused local port in the
  setup review.
- **The server starts but remote players cannot connect.** The default setup
  binds to loopback. Follow [Portable hosting and adoption](https://github.com/zax0rz/darkpawns/blob/main/docs/portable-hosting.md)
  before opening access beyond this machine.

## Related

- [Your first login](/docs/host/first-login/)
- [Players can't connect](/docs/host/players-cannot-connect/)
