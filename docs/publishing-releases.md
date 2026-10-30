# Publishing a portable release

Publication is deliberate. Merging code and pushing tags do not publish binaries.
The `Publish portable release` workflow must be dispatched from **main** with an
existing reviewed `vMAJOR.MINOR.PATCH` tag. The first planned release is `v0.1.0`;
create it only after its source and release notes are approved. This document is
an operator procedure, not authorization to publish.

## Trust and ownership

`internal/releaseauth/trusted-key.pem` is the public RSA publisher key pinned by
the updater and rendered into both bootstrap scripts. The protected
`portable-release` GitHub environment holds `RELEASE_SIGNING_KEY` as a private
PKCS#8 PEM secret. Require a human reviewer and restrict the environment to main.
Self-approval can remain enabled for a one-maintainer project; approving the
publication job is still a separate explicit action after validation finishes.
No private key, real secret or signing-key backup belongs in this repository,
its build artifacts, or logs. Keep an offline backup in the maintainer's secret
store. A missing secret or a key that differs from the committed public key stops
publication.

The manifest uses RSA PKCS#1 v1.5 with SHA-256, supported by Go's standard library,
OpenSSL and Windows PowerShell's .NET RSA verifier. It includes the stable tag,
full source commit, each platform/architecture, filename, byte length and SHA-256
for all six archives and the two installers. Its detached signature is binary.
The exact format is `internal/releaseauth/manifest.go`; do not hand-edit a signed
manifest or change its line endings. The updater cross-checks signed asset data
against the official GitHub release API before downloading a bundle.

This authenticates release bytes; it does not prove a private signing key has
never been compromised. It does not add Apple notarization, Windows Authenticode
or a timestamping service. Initial installer-script delivery trusts GitHub HTTPS;
the script carries the pin used before executing a bundle. Signature verification
cannot protect someone who substitutes both the bootstrap script and its key.

For a new signing identity, generate a key **outside** the checkout, preserve the
private file, review the public-key change and bootstrap pins, then configure the
protected environment. `go run ./cmd/dp-release --generate-key /private/path/key.pem
--public /private/path/public.pem` refuses existing files. Replace the committed
public key only as an explicit trust-root change. Old installed launchers pin the
old key: rotation requires a transition release trusted by the old key before
switching publishers. There is no silent key download or trust-on-first-use path.

## Release sequence

1. Finish source review, native CI and the applicable fidelity batch gate. Choose
   the exact main commit and prepare release notes describing compatibility and
   known boundaries. Release publishing does not replace the fidelity process.
2. Create the stable tag on that reviewed main commit and push it. Confirm it
   resolves to the intended commit. The workflow refuses nonstable tag syntax or
   commits outside main's history.
3. Dispatch `Publish portable release` from main with that tag. It independently
   runs native instance/lifecycle/update/bootstrap journeys and full build, vet,
   race tests and lint. A cross-build is not runtime proof for all six architectures.
4. Review the source commit, tag, CI and generated artifacts, then approve the
   protected publication job. It builds the six archives from the tagged tree
   inside the protected environment, and a GoReleaser build post-hook signs each
   platform's bundle contents (both executables plus the data manifest) with
   `RELEASE_SIGNING_KEY` as the archives are assembled — so GoReleaser's
   `checksums.txt`, the release assets, and every package channel pin the exact
   signed bytes. The job then renders installers, signs the release manifest
   (`dp-release --manifest-only`; the archives are already signed), creates a
   draft with all artifacts and publishes only after upload succeeds. It refuses
   existing releases/assets; no `--clobber` or silent draft reuse.
5. Check the public assets, signature and manifest against the retained build
   receipt. Test a fresh bootstrap and the installed wizard. For the second
   release, exercise a real published version-to-version update with saved login
   before approving production adoption.

A failed upload may leave a draft that is invisible to latest-release consumers.
Inspect it and preserve evidence. Delete or repair that draft only through a
separate deliberate operator action, then rerun; the workflow does not guess that
an existing draft belongs to its attempt. The release environment serializes
publication runs. This workflow does not create tags or migrate/deploy production.

## Package-manager channels

The same protected publication job feeds four package channels, using
GoReleaser's built-in publishers against the signed archives:

- **Homebrew tap** `darkpawns/homebrew-tap` (formula, macOS and Linux),
- **Scoop bucket** `darkpawns/scoop-bucket` (Windows manifest),
- **winget**: manifests committed to a versioned branch of
  `darkpawns/winget-pkgs` (a fork of `microsoft/winget-pkgs`); the job then
  opens the pull request (`scripts/winget-pull-request.sh`). The first
  submission of a new package identifier is reviewed by the winget team, so
  expect a human delay there.
- **Docker** `ghcr.io/darkpawns/darkpawns`, a multi-arch image
  (linux/amd64, linux/arm64) built from the signed archives with this run's
  checksum verification, tagged `vX.Y.Z` and `latest`, running as a
  non-root user with the instance in `/data`. The image is built by an
  explicit buildx step **after** the signing step, so no key file exists
  while third-party base images execute during the build.

All four pin per-release checksums of the signed archives; none bypasses the
signed-bundle trust model, and the launcher re-verifies the publisher
signature at first run regardless of channel.

### Code-execution boundary in the publish job

No third-party code may run where the signing key or the package token is
present. The admin UI (whose `npm ci` install scripts are third-party
code) is built by the secret-less verify job and passed as an artifact;
`DP_PREBUILT_UI` collapses the goreleaser npm before-hooks to no-ops. The
signing and publishing step runs only the pinned GoReleaser binary, `go
build` of this repository, our `cmd/dp-release`, and our staging script —
never `go generate`, third-party `go run`, or npm. The key file is created
immediately before that step and deleted by its exit trap; the GHCR login
exists only inside the container-image step, which runs after the key is
gone. `npm ci --ignore-scripts` everywhere npm does run is defence in
depth, and `internal/releaseguard` fails the build if a publish-job step
ever references npm, pnpm or yarn again.

The channels are generated into `dist/` by GoReleaser (`skip_upload: true`)
and pushed by `scripts/publish-package-channels.sh` only **after** the
GitHub release and its download URLs are live, so no formula, bucket entry
or winget manifest ever points at a URL that 404s. Every push is
idempotent (blob shas, winget branch reuse, open pull request reuse), so a
failed channel step is retriable against a complete release. A test in
`internal/releaseguard` fails the build if the channel step ever moves
ahead of the release publication.

These channels need `PACKAGES_GITHUB_TOKEN` in the protected
`portable-release` environment (created by the maintainer, never committed).
It must be a **classic** personal access token with exactly two scopes,
**`public_repo`** and **`write:packages`**. A fine-grained token can't do
this job: the GitHub container registry accepts only classic tokens, and a
fine-grained token can't open the pull request on `microsoft/winget-pkgs`,
which the organization doesn't own. `public_repo` reaches every public
repository the token's owner can push to, so the token lives only in the
protected environment and expires after 90 days; a dedicated account that
belongs only to the `darkpawns` organization would narrow it further. The
organization must allow classic tokens (Settings → Personal access tokens).

### Dry run (no publishing)

Every channel generates its artifacts locally under a prerelease-looking tag,
because `skip_upload`/`skip_push` are set to `auto`:

```sh
export DP_SNAPSHOT_PUBLISHER_KEY=$(base64 -w0 snapshot-public.pem)
export SNAPSHOT_PRIVATE_PEM=…snapshot-private.pem SNAPSHOT_PUBLIC_PEM=…snapshot-public.pem
export PACKAGES_GITHUB_TOKEN=dry-run-token GITHUB_TOKEN=dry-run-token
git tag -f v0.1.7-rc.0
goreleaser release --clean --skip=validate     # add --skip=docker to skip the image build
```

`dist/homebrew/Formula/darkpawns.rb`, `dist/scoop/darkpawns.json`,
`dist/winget/manifests/**` and `dist/checksums.txt` can then be inspected;
nothing is pushed (prerelease tag, and no real token). The retained example
output lives in `docs/evidence/2026-10-28-packaging-dryrun/`.

## Verification boundaries

PR CI uses disposable private/public keys and locally signed native archives.
It tests rejection of changed signatures, manifests and archives, native selection,
existing-install preservation, command paths with apostrophes/Unicode/percent,
and startup from the installed payload. No external notification is sent.
The same native suite also verifies saved-player persistence and manual updates.
These tests do not claim that a real release was published or that its public
URLs, approval permissions, first-install trust delivery or OS security dialogs
have been validated. Perform that check after deliberate first publication.

The `.goreleaser.yaml` default still disables publication; the approved workflow
owns signing and upload. Snapshot archives from PR jobs remain unsigned developer
artifacts. Explicit `update --archive --sha256` remains an operator-trusted local
mode; it does not turn unsigned files into authenticated official releases.


## Release announcements

The `announce-release.yml` workflow posts a published release's title, URL and
notes to Discord after publication, through `workflow_call` or a manual
`workflow_dispatch`. It truncates notes that exceed Discord's embed limit and
links to the full release. Mentions are disabled. The game's Notifications page
doesn't control these announcements.

The destination is the repository secret `DISCORD_ANNOUNCE_WEBHOOK`. An absent
secret skips the announcement. To preview or retry it:

1. Open **Announce release on Discord** in GitHub Actions.
2. Choose **Run workflow** and enter the published stable tag, such as `v0.1.7`.
3. Leave `dry_run` enabled to print the payload. Disable it to retry the post.

The workflow refuses unstable version tags and draft releases.
