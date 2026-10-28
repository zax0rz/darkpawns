# Packaging dry run: every channel generated, nothing published

2026-10-28, local machine, GoReleaser v2.18.3 (the pinned CI version). The
release pipeline builds and signs each platform's bundle contents inside a
build post-hook (`scripts/sign-bundle-content.sh`), so GoReleaser's checksums
and every package channel pin the exact signed archive bytes; no artifact is
repacked after publication. `skip_upload`/`skip_push: auto` plus a
prerelease-looking local tag (`v0.1.7-rc.0`) kept every channel local; the
tokens used were dummies.

## Generated channels

- `darkpawns.rb` — Homebrew formula: per-OS/arch SHA-256 of the signed
  archives, libexec install of the whole bundle with a `bin` symlink,
  caveats, and a `darkpawns version` test.
- `darkpawns.json` — Scoop manifest: per-architecture URL + hash of the
  signed Windows zips, bins both executables.
- `winget-installer.yaml` — winget installer manifest:
  `InstallerType: zip` with `NestedInstallerType: portable` and
  `PortableCommandAlias` entries, per-arch `InstallerSha256` of the signed
  zips. The publisher emits the nested-portable structure natively.
- `checksums.txt` — GoReleaser's checksums over the six signed archives.

## Docker

`docker-run.txt` records the full local proof: the goreleaser buildx build
with `--platform=linux/amd64,linux/arm64` (arm64 under emulation) tagged
`ghcr.io/darkpawns/darkpawns:v0.1.7-rc.0` and `:latest` without pushing; the
container booted the game, answered `GET /health` with 200, ran PID 1 as uid
65532 (the entrypoint drops privileges before exec), kept the instance in the
`/data` volume across a restart, and the launcher's publisher gate verified
the snapshot-signed bundle at boot (an earlier image built without the key
linked into the launcher was refused with "publisher signature verification
failed" — the trust model held inside the container too).

## Trust note

The channels install the same signed archives as the bootstrap installer;
nothing bypasses the publisher gate, and the launcher re-verifies the bundle
signature on first run regardless of channel.
