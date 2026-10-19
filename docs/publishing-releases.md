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
   race tests and lint. GoReleaser builds Linux/macOS/Windows amd64 and arm64 archives
   from the tagged tree, preserving the runtime licenses. A cross-build is not
   runtime proof for all six architectures.
4. Review the source commit, tag, CI and generated artifacts, then approve the
   protected publication job. It renders installers, signs the complete manifest,
   creates a draft with all artifacts and publishes only after upload succeeds.
   It refuses existing releases/assets; no `--clobber` or silent draft reuse.
5. Check the public assets, signature and manifest against the retained build
   receipt. Test a fresh bootstrap and the installed wizard. For the second
   release, exercise a real published version-to-version update with saved login
   before approving production adoption.

A failed upload may leave a draft that is invisible to latest-release consumers.
Inspect it and preserve evidence. Delete or repair that draft only through a
separate deliberate operator action, then rerun; the workflow does not guess that
an existing draft belongs to its attempt. The release environment serializes
publication runs. This workflow does not create tags or migrate/deploy production.

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
