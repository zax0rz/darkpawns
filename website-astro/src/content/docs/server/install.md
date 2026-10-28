---
title: "Installing the server"
description: "Four package-manager channels and the bootstrap installer, all shipping the same signed release archives."
section: "server"
audience: "operator"
order: 5
sourcePath: "website-astro/src/content/docs/server/install.md"
updated: 2026-10-28
draft: false
---

Every channel installs the same publisher-signed portable archives from the
GitHub release: a package manager only adds its own download and checksum
handling, and the launcher verifies the bundle signature again on first run.
A tampered mirror cannot pass either check.

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

## Bootstrap installer (any platform with curl)

```sh
curl -fsSL https://darkpawns.org/install.sh | sh
```

The installer verifies its own signature before running and then verifies the
release archive it downloads. See the repository's portable release notes for
what it writes and how to remove it.

After any install, `darkpawns setup` walks through the first instance, or
`darkpawns init` creates one unattended; `darkpawns update` (or the admin
console's Updates page) keeps it current.
