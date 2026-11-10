#!/usr/bin/env python3
"""Apply/check only the reviewed modernizers, preserving R1/R3 exclusions.

Both modes first collect go fix -diff across the configured modules. Excluded
files are never written. Tool failures fail closed; --check never writes files.
The policy records the exact batches 1–5 exclusions and guards new RNG callers.
Like go fix ./..., analysis uses the current build platform and build tags.
"""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def exclusion(path, source, policy):
    if path in policy["excluded_files"]:
        return policy["excluded_files"][path]
    for prefix, reason in policy["excluded_prefixes"].items():
        if path.startswith(prefix):
            return reason
    for pattern, reason in policy["excluded_source_patterns"].items():
        if re.search(pattern, source):
            return reason
    return None


def filter_diff(diff, root, policy):
    accepted, excluded = [], {}
    # go fix emits unified file patches, with absolute paths and old/new labels.
    for patch in re.split(r"(?m)(?=^--- )", diff):
        if not patch:
            continue
        lines = patch.splitlines(keepends=True)
        if len(lines) < 3 or not lines[0].endswith(" (old)\n") or not lines[1].endswith(" (new)\n"):
            raise ValueError("unexpected go fix diff headers")
        old = Path(lines[0][4:-7]).resolve()
        new = Path(lines[1][4:-7]).resolve()
        if old != new or not old.is_file():
            raise ValueError("go fix patch does not describe an existing source file")
        path = old.relative_to(root).as_posix()
        reason = exclusion(path, old.read_text(), policy)
        if reason:
            excluded[path] = reason
            continue
        lines[0], lines[1] = f"--- a/{path}\n", f"+++ b/{path}\n"
        accepted.append("".join(lines))
    return "".join(accepted), excluded


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail on eligible changes; never write")
    args = parser.parse_args()
    policy = json.loads((ROOT / "scripts/modernize-policy.json").read_text())
    modules = subprocess.check_output(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "--", "go.mod", "**/go.mod"],
        cwd=ROOT, text=True,
    )
    if {str(Path(path).parent) for path in modules.splitlines()} != set(policy["modules"]):
        raise ValueError("module inventory changed; review modernize-policy.json")
    # A nested research module still targets Go 1.22. Use the root-selected
    # toolchain consistently instead of falling back to the host's older go.
    version = subprocess.check_output(["go", "env", "GOVERSION"], cwd=ROOT, text=True).strip()
    env = dict(os.environ, GOTOOLCHAIN=version)
    patches, exclusions = [], {}
    for module in policy["modules"]:
        result = subprocess.run(
            ["go", "fix", "-diff", *["-" + name for name in policy["analyzers"]], "./..."],
            cwd=ROOT / module, env=env, capture_output=True, text=True,
        )
        # -diff exits 1 when fixes are offered. Dependency download progress
        # is normal on fresh CI caches; every actual diagnostic fails closed,
        # even if another package also produced a usable partial patch.
        diagnostics = "\n".join(
            line for line in result.stderr.splitlines()
            if not re.fullmatch(r"go: downloading \S+ \S+", line)
        )
        if result.returncode not in (0, 1) or diagnostics or (result.returncode == 1 and not result.stdout):
            raise RuntimeError(f"go fix failed in {module} ({result.returncode}):\n{result.stderr}")
        patch, skipped = filter_diff(result.stdout, ROOT, policy)
        patches.append(patch)
        exclusions.update(skipped)
    patch = "".join(patches)
    for path, reason in sorted(exclusions.items()):
        print(f"modernize: excluded {path}: {reason}", file=sys.stderr)
    if not patch:
        print("modernize: clean")
        return 0
    if args.check:
        print(patch, end="")
        print("modernize: run make modernize and review the diff", file=sys.stderr)
        return 1
    # Git validates every hunk before any write; no shell and no stale-file
    # overwrites. It also refuses symlink/outside-worktree patch destinations.
    subprocess.run(["git", "apply", "--check", "-"], cwd=ROOT, input=patch, text=True, check=True)
    subprocess.run(["git", "apply", "-"], cwd=ROOT, input=patch, text=True, check=True)
    print("modernize: applied; review changes, run make fmt and repository gates")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"modernize: {exc}", file=sys.stderr)
        sys.exit(2)
