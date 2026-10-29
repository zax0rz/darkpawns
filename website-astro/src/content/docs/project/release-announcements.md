---
title: Release announcements
description: Explain how the published GitHub release notes are posted to Discord by CI, and how contributors can preview or retry an announcement.
type: explainer
audience: developers
verified: v0.1.7
---

Dark Pawns release announcements are sent by the release workflow after a release is published. The game's Notifications page does not control them.

## What CI posts

The workflow reads the published release title, URL, and notes, then posts them to Discord. If the release notes exceed Discord's embed limit, CI truncates the post and links to the full notes.

The destination comes from the repository secret `DISCORD_ANNOUNCE_WEBHOOK`. If the secret is absent, the workflow skips the announcement.

## Preview or retry an announcement

1. Open the `Announce release on Discord` workflow in GitHub Actions.
2. Choose **Run workflow** and enter the published stable release tag, such as `v0.1.7`.
3. Leave `dry_run` enabled to print the payload without posting. Disable it to retry the announcement.

The workflow refuses tags that are not stable version tags and releases that are still drafts.

## How it works

The `announce-release.yml` workflow can run after publication through `workflow_call` or by hand through `workflow_dispatch`. It gets the notes from the GitHub release API and posts with mentions disabled.

See [Publishing releases](https://github.com/zax0rz/darkpawns/blob/main/docs/publishing-releases.md) for the release process.
