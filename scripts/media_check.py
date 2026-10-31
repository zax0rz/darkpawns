#!/usr/bin/env python3
"""Warn when a generated media source is newer than its output."""

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "docs/images/generated/manifest.json"
REQUIRED_OUTPUTS = {
    "admin-configuration.png", "admin-object-editor.png", "admin-mob-editor.png",
    "admin-zone-editor.png", "admin-lua-scripts.png",
    "admin-dashboard.png", "admin-player-editor.png", "admin-player-editor.gif",
    "admin-room-editor.png", "admin-notifications.png", "admin-operations.png",
    "admin-zones.png", "admin-updates.png", "admin-motd.png", "web-client-first-steps.png",
    "mudlet-dock.png", "install-and-setup.gif", "lifecycle-commands.gif", "update-check.gif",
}


def warn(path: str, reason: str) -> None:
    print(f"::warning file={path}::{reason}")


def main() -> int:
    if not MANIFEST.exists():
        print("::warning file=docs/images/generated/manifest.json::media manifest is missing; run make media")
        return 0
    try:
        manifest = json.loads(MANIFEST.read_text())
    except (OSError, json.JSONDecodeError) as error:
        warn("docs/images/generated/manifest.json", f"cannot read media manifest ({error}); run make media")
        return 0
    if not isinstance(manifest, dict):
        warn("docs/images/generated/manifest.json", "media manifest must be a JSON object; run make media")
        return 0
    if manifest.get("schema") != 1:
        warn("docs/images/generated/manifest.json", "unsupported or missing manifest schema; run make media")
    for field in ("schema", "release", "git_head", "working_tree_dirty", "media"):
        if field not in manifest:
            warn("docs/images/generated/manifest.json", f"manifest is missing {field}; run make media")
    if not isinstance(manifest.get("media"), list):
        warn("docs/images/generated/manifest.json", "manifest media must be an array; run make media")
        return 0
    listed = set()
    stale = []
    for item in manifest["media"]:
        if not isinstance(item, dict):
            warn("docs/images/generated/manifest.json", "media entry must be an object; run make media")
            continue
        path_value = item.get("path")
        if not isinstance(path_value, str) or not path_value:
            warn("docs/images/generated/manifest.json", "media entry has no output path; run make media")
            continue
        listed.add(Path(path_value).name)
        for field in ("alt", "sources", "release"):
            if field not in item:
                warn(path_value, f"manifest entry is missing {field}; run make media")
        alt = item.get("alt")
        if not isinstance(alt, str) or not alt.strip():
            warn(path_value, "manifest alt text must be a non-empty string; run make media")
        elif len(alt) > 150:
            warn(path_value, f"alt text is {len(alt)} characters; keep it under 150")
        if not isinstance(item.get("sources"), list) or not item.get("sources"):
            warn(path_value, "manifest sources must be a non-empty array; run make media")
            continue
        output = ROOT / path_value
        if not output.exists():
            stale.append((item["path"], "missing output"))
            continue
        for source in item["sources"]:
            if not isinstance(source, str):
                warn(path_value, "manifest source paths must be strings; run make media")
                continue
            path = ROOT / source
            if not path.exists():
                stale.append((item["path"], f"missing source {source}"))
            elif path.stat().st_mtime_ns > output.stat().st_mtime_ns:
                stale.append((item["path"], f"newer source {source}"))
    for missing in sorted(REQUIRED_OUTPUTS - listed):
        warn("docs/images/generated/manifest.json", f"manifest is missing {missing}; run make media")
    if stale:
        for output, reason in stale:
            warn(output, f"{reason}; run make media")
    else:
        print("media sources are current")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
