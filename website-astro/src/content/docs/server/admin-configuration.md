---
title: "Configuration and updates"
description: "Edit every runtime setting from the console, back up and restore the settings bundle, and run launcher updates with automatic rollback."
section: "server"
audience: "operator"
order: 16
sourcePath: "website-astro/src/content/docs/server/admin-configuration.md"
updated: 2026-10-27
draft: false
---

The Configuration and Updates pages sit under Staff tools and require
implementor access (level 40). Configuration edits the game-owned settings
file (`lib/data/settings.json` on a portable instance); Updates drives the
portable launcher's update transaction. Listener ports and bind addresses
remain launcher-owned: change them in `instance.json` while the world is
stopped, as the network settings design describes.

## What is a setting

Engine rules (pulse rates, combat cadence, autosave interval, board
capacities) are compiled in and never configurable, so a running server
cannot drift from the audited game. The console edits operator surface only:
connection limits, login idle timeouts, proxy trust, browser origins, the
Grapevine intermud relay, the staff-log ntfy push, and the contact form
delivery.

Each field shows whether it applies **live** (the next connection sees it) or
needs a **restart**. A save that touches restart-marked fields raises a banner
listing exactly those fields, the number of players online, and an
**Apply and restart** action. The launcher owns that restart: the world stops
gracefully with a forced save, starts again, and must pass readiness before
the console reports success.

Secrets (relay credentials, push tokens, SMTP and Turnstile secrets) are
stored encrypted with AES-256-GCM beside the settings file. They are write-only:
an input replaces the sealed value, a clear button removes it, and no
response, export, or audit entry ever contains one.

## Revisions, overrides, and recovery

Saves are revision-checked. If another administrator saved first, the console
refuses the save and asks you to reload; last-writer-wins is never used. On a
checkout deployment (running the server binary directly), exported
environment variables still win over the file, and the affected fields carry
an env-override badge rather than pretending the file won. A first run
imports the current environment into the settings file once.

Every save keeps the previous file as `settings.json.previous`. A damaged
settings file does not stop the server: it runs on defaults, refuses saves,
and the boot log names the repair path. Each change is written to the audit
log (who, what changed, which fields need a restart), and the History card
on the page reads it back.

## Backup and restore

**Download bundle** exports every group with its revision and server version.
Secrets are never part of an export. **Upload bundle** validates the file
server-side first and shows a field-by-field diff with the current values,
any unknown keys it will ignore, and which secrets stay untouched; applying
is a separate confirmed step through the same revision-checked save path.

## Updates

The Updates page shows the running release and build commit, and whether the
server is a **managed instance** (started by the portable launcher). On an
unmanaged server the page is read-only: deploy with `darkpawns` or your
existing procedure.

**Check for updates** is read-only: the launcher looks up the latest stable
release metadata and compares versions. Nothing is downloaded, staged, or
stopped, and publisher verification still happens at staging time.

**Update now** runs exactly the transaction `darkpawns update` runs:

1. The dialog shows who is online and what will happen: stage and verify
   while the world runs, then stop with a forced save, snapshot the whole
   instance, swap the active release pointer, start the candidate, and
   health-check it. A failed health check rolls back to the previous
   release automatically.
2. The request returns immediately; the game process never waits on its own
   update. The page polls the launcher's journal-backed status, which
   survives both the restart and a browser refresh.
3. Status reads staging → applying → testing → committed (or rolled back).
   After a rollback the page names the release that failed its health
   check and offers an explicit **Retry** of that same release rather than
   a plain update button. If a transaction was interrupted (the machine died mid-update), the page
   says recovery is needed and prints the exact `darkpawns update recover`
   command. Recovery stays single-sourced in the CLI; the browser never
   invents its own updater.

Only one update may run at a time; the launcher serializes lifecycle
operations and an external `supervise` monitor defers to an update
transaction. A successful local update says nothing about LAN or WAN
reachability: firewall, proxy, NAT, and DNS remain operator
responsibilities.
