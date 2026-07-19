#!/usr/bin/env python3
"""Parse git conventional commits → website/data/changelog.json for the changelog page."""

import json
import re
import subprocess
import sys
from pathlib import Path

# Paths
REPO_ROOT = Path(__file__).parent.parent.parent
OUTPUT_FILE = Path(__file__).parent.parent / "data/changelog.json"

# Constants
COMMIT_TYPES = {"feat", "fix", "refactor", "perf", "docs", "chore", "test"}
MAX_ENTRIES = 200

# Matches "type(scope): subject", "type!: subject", and "type: subject"
COMMIT_RE = re.compile(r"^(?P<type>[a-z]+)(?:\((?P<scope>[^)]*)\))?!?:\s+(?P<subject>.+)$")


def git_log():
    """Return raw 'hash|date|subject' lines from the enclosing repo, newest first."""
    result = subprocess.run(
        ["git", "-C", str(REPO_ROOT), "log", "--no-merges",
         "--pretty=format:%h%x1f%ad%x1f%s", "--date=short"],
        capture_output=True, text=True, check=True
    )
    return result.stdout.splitlines()


def parse_commits(lines):
    """Filter to conventional commits of tracked types, capped at MAX_ENTRIES."""
    entries = []
    for line in lines:
        parts = line.split("\x1f", 2)
        if len(parts) != 3:
            continue
        commit_hash, date, subject = parts
        match = COMMIT_RE.match(subject)
        if not match or match.group("type") not in COMMIT_TYPES:
            continue
        entries.append({
            "date": date,
            "type": match.group("type"),
            "scope": match.group("scope"),
            "subject": match.group("subject"),
            "hash": commit_hash
        })
        if len(entries) >= MAX_ENTRIES:
            break
    return entries


def main():
    entries = parse_commits(git_log())

    OUTPUT_FILE.parent.mkdir(parents=True, exist_ok=True)
    with open(OUTPUT_FILE, "w") as f:
        json.dump(entries, f, indent=2)
        f.write("\n")

    print(f"     Compiled changelog.json: {len(entries)} entries", file=sys.stderr)


if __name__ == "__main__":
    main()
