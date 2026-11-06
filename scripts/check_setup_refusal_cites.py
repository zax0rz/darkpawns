#!/usr/bin/env python3
"""Check every src/ citation in internal/oraclediff/setuprefusal.go (R5g).

Each refusal table entry names src/<file>:<line> locations; this script greps
the cited lines out of src/ and verifies the refusal string actually appears
there, so a citation cannot drift from the source it claims (DP-1456 follow-up).
Exit 1 with a table of failures if any citation is wrong.
"""

from __future__ import annotations

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
TABLE = ROOT / "internal" / "oraclediff" / "setuprefusal.go"
CITE = re.compile(r"src/([\w.]+):(\d+)")

ENTRY = re.compile(r'\{"((?:[^"\\]|\\.)*)",\s*"((?:[^"\\]|\\.)*)"\}')


def main() -> int:
    src = TABLE.read_text()
    # Only the setupRefusalLines table (entries with a cite field).
    body = src.split("var setupRefusalLines", 1)[1]
    body = body.split("\n]", 1)[0]
    bad = 0
    checked = 0
    for m in ENTRY.finditer(body):
        line, cite = m.group(1), m.group(2)
        line = line.replace('\\"', '"').replace("\\'", "'")
        for c in CITE.finditer(cite):
            path, no = ROOT / "src" / c.group(1), int(c.group(2))
            checked += 1
            try:
                text = path.read_text().splitlines()[no - 1]
            except (OSError, IndexError):
                print(f"BAD   {cite}: {path.name}:{no} does not exist")
                bad += 1
                continue
            if line not in text:
                # A via-site may only name the macro: NOPERSON's text lives at
                # its config.c definition (cited by its own table entry), and
                # HCONTROL_FORMAT's first line is the cited string itself.
                macro_ok = ("NOPERSON" in text and line == "No-one by that name here.") or (
                    "HCONTROL_FORMAT" in text and line.startswith("Usage: hcontrol"))
                if not macro_ok:
                    print(f"BAD   cite src/{c.group(1)}:{no} for {line!r}: line is {text.strip()!r}")
                    bad += 1
    print(f"checked {checked} citations across the refusal table: {bad} bad")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
