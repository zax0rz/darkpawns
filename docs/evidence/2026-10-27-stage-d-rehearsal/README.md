# Stage D rehearsal: console-driven updates on a local portable instance

2026-10-27, local machine only. This is the end-to-end proof Zach's review
asked for before merge: install a real release, serve locally built
snapshots as the "new" release, click Check and Update now in the console,
and show the page surviving the restart — then a release that fails its
health gate and the automatic rollback.

## Topology

- **Bootstrap**: the real published v0.1.6 release archive created the
  instance (`darkpawns setup --yes`, first character created over the
  browser WebSocket protocol, auto-crowned implementor).
- **Key boundary**: v0.1.6's updater verifies against the production
  publisher key, so it correctly refuses locally signed snapshots (the
  publisher gate doing its job). The running anchor is therefore the stage D
  build itself stamped `v0.1.7-local.1` — exactly the production shape of
  "v0.1.7 shipped, next update comes from the console". All binaries were
  built with a disposable snapshot publisher key
  (`DP_SNAPSHOT_PUBLISHER_KEY`), the same mechanism portable CI uses.
- **Feed**: `DP_RELEASE_ENDPOINT=http://127.0.0.1:8399` points the launcher
  at a local fixture serving GitHub's layout (latest metadata, archives,
  signed release manifests). The signed-manifest publisher gate stayed in
  force for every byte.
- **Candidates**: `v0.1.8` (healthy) and `v0.1.9` (its `darkpawns-server`
  is a native Go executable that writes a marker and exits 9 — it passes
  preflight and fails the real readiness boundary).

## Success case: v0.1.7-local.1 → v0.1.8

`updates-check-desktop.png` (check result), `updates-confirm-desktop.png`
(confirm dialog "Update to v0.1.8?": who is online by name, verification
before the world stops, automatic return of the current release),
`updates-inprogress-desktop.png`
(staging, captured after a mid-transaction page refresh — the refresh landed
back on the status view), `updates-committed-desktop.png` (same browser tab
across the restart, reporting v0.1.8 and "Committed").

The durable journal recorded:

```json
{"id":"5783c24984e738defdd5b550259fb83e","phase":"committed",
 "previous":".../darkpawns_v0.1.7-local.1_linux_amd64",
 "candidate":".../.updates/5783c24984e738defdd5b550259fb83e/release/darkpawns_v0.1.8_linux_amd64",
 "sha256":"8fe665646b380e78b804333a20587d0c1259e6e955f6120fe41cef5d91a96c96",
 "was_running":true}
```

Receipt: `journals/receipt-committed.json`.

## Rollback case: v0.1.8 → v0.1.9 (fails health gate)

The broken candidate exited before readiness; the launcher rolled the
instance back to the verified v0.1.8 snapshot and restarted it.
`updates-rolledback-desktop.png` shows the page's report — still serving
v0.1.8, "Rolled back (rolled-back)", backup retained.

The journal recorded the exact failure and outcome
(`journals/transaction-rollback.json`, `journals/receipt-rollback.json`):

```json
{"phase":"rolled-back",
 "error":"new release exited before readiness: server exited before becoming ready\nexit status 9",
 "was_running":true}
```

The transaction directory retained `backup/`, `backup-manifest.json` and
`failed-lib/` for inspection.

## Downgrade refusal (review polish)

With v0.1.8 running and the feed offering the older v0.1.7: the console's
check names the direction and offers no plain update
(`updates-downgrade-refused-desktop.png`); the CLI refuses with
`refusing to downgrade from v0.1.8 to v0.1.7; your world was not stopped;
rerun with --allow-downgrade to force`, and `--allow-downgrade` stages it
(`cli-downgrade-refusal.txt`). This closes the hostile-mirror case for
`DP_RELEASE_ENDPOINT` feeds: a mirror cannot walk an instance backwards
without the operator's explicit flag, which the console never sends. An
equal version carrying a different commit is refused the same way
(unit-tested in `internal/portable/version_guard_test.go`).

## Retry after rollback (review polish)

After the broken v0.1.9 was rolled back a second time, the page names the
failed release, hides the plain Update now, offers **Retry v0.1.9 anyway**
with the same confirm dialog (titled "Retry update to v0.1.9?"), and links
the updates documentation (`updates-rolledback-retry-desktop.png`,
`updates-retry-confirm-desktop.png`). Retrying re-ran the transaction, failed
the same health gate, and restored v0.1.8 again.

## Configuration page

`settings-default-desktop.png` (all groups; the access card carries the
`env overrides: trusted_proxies` badge because the launcher was started
with `TRUSTED_PROXIES` exported — an inherited variable the file cannot
silently beat), `settings-restart-banner-desktop.png` (a Grapevine save —
mode presence, client id, and a write-only secret — raising the restart
banner; the secret field reports "set" after saving, never a value),
`settings-restore-preview-desktop.png` (an uploaded bundle validated
server-side with a field-level diff, unknown-key handling, and the
secrets-untouched note). Mobile: `settings-mobile.png`,
`updates-mobile.png` (390×844, bottom tab bar layout).

## Bugs the rehearsal caught (all fixed in this PR)

(Polish round notes: the first update click after the rebuild surfaced the
one-slot journal showing the previous transaction's terminal state while the
new one staged — pre-existing single-journal behavior; the page reports the
new transaction as soon as its journal is written.)

1. **Snapshot-linked builds could not verify a release feed** — the feed
   path pinned the production key directly instead of `TrustedKeyBytes()`
   (identical in published builds; wrong for snapshot builds).
2. **A dead staging wedged Update now** — the button disabled on the
   staging marker alone; it now disables only on journal-backed phases,
   leaving serialization to the launcher's locks.
3. **Console saves were rejected with 422** — the page's save body carried
   view-only fields (`client_secret_set` …) that strict body validation
   refuses.
4. **First boot manufactured phantom env overrides** — the boot installer
   wrote compiled-in defaults like `GRAPEVINE_MODE=off` into the
   environment, which then beat every later file save ("still overridden
   by environment: grapevine.mode" in the audit trail was the tell). The
   installer now installs only real file changes and never touches an
   inherited variable. This one would have shipped to production without
   the rehearsal.
