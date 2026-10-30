#!/usr/bin/env python3
"""Ignored-Go-files gate — fail when a source file is silently left out of the build.

``go list`` reports every file it refuses to compile in ``IgnoredGoFiles``. A
file lands there for one of two reasons:

* an explicit ``//go:build`` constraint the current host does not satisfy
  (``//go:build windows``, ``//go:build clipreview``), which is deliberate; or
* an *implicit* GOOS/GOARCH suffix in the file name — ``x_arm_test.go`` is read
  as "GOARCH=arm" and dropped on amd64, ``x_windows.go`` as "GOOS=windows".

The second kind is the dangerous one, because it is invisible: the file sits in
the tree, ``go build`` and ``go test`` both succeed, and the tests inside it
never run. It cost a real debugging session on 2026-10-30 — a new test file
named ``skill_pain_arm_test.go`` was silently excluded, so its tests "passed" by
not running and nothing failed. The same trap hides a whole file of new tests
behind any of Go's GOOS or GOARCH tokens.

This gate therefore requires every ignored file to be named in ``ALLOWED``
below with the tag that justifies it. A new ignored file fails the build until
someone either fixes the name or records why it is ignored — the review the
suffix rule never gets.

Cite: R5h (a green must be able to fail) in ``docs/fidelity/RULEBOOK.md`` — a
test file that never compiles into the binary is a green that cannot fail.

Exit codes:
    0  — every ignored file is allowlisted (healthy)
    1  — at least one ignored file is not allowlisted (printed with its package)
    2  — ``go list`` failed, or its output could not be parsed (infra)

Usage:
    python3 scripts/check_ignored_go_files.py             # CI invocation
    python3 scripts/check_ignored_go_files.py --list      # show the allowlist
    python3 scripts/check_ignored_go_files.py --root DIR  # scan another tree
"""

import argparse
import json
import os
import subprocess
import sys

# Repo-relative paths that may be ignored, each with the constraint that
# justifies it. Keys are exact paths; a rename or a new file fails the gate.
ALLOWED = {
    # Opt-in preview harnesses, built only with -tags clipreview / adminpreview.
    "cmd/darkpawns/cli_preview_test.go": "//go:build clipreview",
    "pkg/admin/console_preview_test.go": "//go:build adminpreview",
    # Non-Linux listener ownership seam.
    "cmd/dp-oracle-diff/listener_ownership_other.go": "//go:build !linux",
    # Windows-only sources; ignored on the Linux host by their _windows suffix.
    "internal/portable/detach_windows.go": "GOOS=windows",
    "internal/portable/process_windows.go": "GOOS=windows",
    "internal/portable/update_forward_windows.go": "GOOS=windows",
    "internal/portable/update_replace_windows.go": "GOOS=windows",
    "pkg/session/file_editor_access_windows.go": "GOOS=windows",
    "pkg/session/file_editor_access_windows_test.go": "GOOS=windows",
}


def parse_go_list(stdout):
    """Return [(package, ignored_file)] from ``go list -json`` output.

    ``go list -json`` emits a stream of concatenated JSON objects, one per
    package, so they are decoded with a raw JSON decoder rather than one call.
    """
    decoder = json.JSONDecoder()
    text = stdout.strip()
    index = 0
    pairs = []
    while index < len(text):
        while index < len(text) and text[index] in " \t\r\n":
            index += 1
        if index >= len(text):
            break
        obj, end = decoder.raw_decode(text, index)
        index = end
        package = obj.get("ImportPath", "")
        for name in obj.get("IgnoredGoFiles") or []:
            pairs.append((package, name))
    return pairs


def collect_ignored(root):
    """Run ``go list -json ./...`` in root and return its ignored files."""
    result = subprocess.run(
        ["go", "list", "-json", "./..."],
        cwd=root,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            "go list -json ./... failed (%d):\n%s" % (result.returncode, result.stderr.strip())
        )
    return parse_go_list(result.stdout)


def repo_relative(package, name, module):
    """Map a package import path plus file name to a repo-relative path.

    ``go list`` reports ``ImportPath`` and a bare file name; the module prefix
    is stripped so the allowlist can hold readable repo paths.
    """
    prefix = module + "/"
    if package == module:
        directory = ""
    elif package.startswith(prefix):
        directory = package[len(prefix):]
    else:
        directory = package
    return os.path.join(directory, name) if directory else name


def unexpected(pairs, module, allowed=ALLOWED):
    """Return the repo-relative paths of ignored files that are not allowlisted."""
    found = []
    for package, name in pairs:
        relative = repo_relative(package, name, module)
        if relative not in allowed:
            found.append(relative)
    return sorted(found)


def module_path(root):
    with open(os.path.join(root, "go.mod"), encoding="utf-8") as handle:
        for line in handle:
            if line.startswith("module "):
                return line.split(None, 1)[1].strip()
    raise RuntimeError("no module line in %s/go.mod" % root)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", default=".", help="tree to scan (default: the current directory)")
    parser.add_argument("--list", action="store_true", help="print the allowlist and exit")
    args = parser.parse_args(argv)

    if args.list:
        for path in sorted(ALLOWED):
            print("%s\t%s" % (path, ALLOWED[path]))
        return 0

    root = os.path.abspath(args.root)
    try:
        module = module_path(root)
        pairs = collect_ignored(root)
    except (RuntimeError, OSError) as error:
        print("ignored-go-files: %s" % error, file=sys.stderr)
        return 2

    offenders = unexpected(pairs, module)
    if offenders:
        print(
            "ignored-go-files: %d source file(s) are excluded from the build and not "
            "allowlisted:" % len(offenders),
            file=sys.stderr,
        )
        for path in offenders:
            print("  %s" % path, file=sys.stderr)
        print(
            "\nA file is ignored either by an explicit //go:build constraint or by a\n"
            "GOOS/GOARCH token in its name. If the name is the cause (a test file ending\n"
            "in _arm, _amd64, _wasm, _windows, ...), rename it: the tests inside it never\n"
            "run and nothing fails. Otherwise add the path to ALLOWED in\n"
            "scripts/check_ignored_go_files.py with the tag that justifies it, so the\n"
            "exclusion is a reviewed decision.",
            file=sys.stderr,
        )
        return 1

    print("ignored-go-files: ok (%d ignored file(s), all allowlisted)" % len(pairs))
    return 0


if __name__ == "__main__":
    sys.exit(main())
