#!/usr/bin/env python3
"""Report C line citations that may use the oracle's line numbers (R5g).

src/ is the citation target. darkpawns-c-oracle/ adds harness
instrumentation to a few files, so their line numbers drift from src/. For
each citation of a drifted file whose numbering differs between the copies,
this prints the cited src/ line and the oracle line, and flags the citation
when the src/ line is blank or a bare brace but the oracle line holds code
(a likely oracle-numbered citation). It is a triage report, not a gate.

  DP_ORACLE_ROOT  oracle checkout (default /home/zach/darkpawns-c-oracle)
"""
import difflib
import os
import re
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ORACLE = os.environ.get("DP_ORACLE_ROOT", "/home/zach/darkpawns-c-oracle")
SKIP_DIRS = {"src", ".git", "node_modules", "website", "website-astro"}


def read(path):
    with open(path, encoding="latin-1") as f:
        return f.read().split("\n")


def drifted_files():
    out = {}
    for name in sorted(os.listdir(os.path.join(REPO, "src"))):
        if not name.endswith((".c", ".h")):
            continue
        oracle_path = os.path.join(ORACLE, "src", name)
        if not os.path.exists(oracle_path):
            continue
        a, b = read(os.path.join(REPO, "src", name)), read(oracle_path)
        if a != b:
            out[name] = (a, b)
    return out


def trivial(line):
    return line.strip() in ("", "{", "}", "};")


def main():
    files = drifted_files()
    if not files:
        print("cite-drift: src/ and the oracle agree; nothing to check")
        return 0
    names = "|".join(re.escape(n[:-2]) for n in files)
    pat = re.compile(r"(?<![A-Za-z_.])(?:src/)?(" + names + r")\.([ch]):(\d+)(?:-(\d+))?")
    flagged = total = 0
    for root, dirs, fnames in os.walk(REPO):
        dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
        for fn in fnames:
            if not fn.endswith((".go", ".tsv", ".md", ".py", ".txt")):
                continue
            path = os.path.join(root, fn)
            try:
                text = open(path, encoding="utf-8").read()
            except (UnicodeDecodeError, OSError):
                continue
            for m in pat.finditer(text):
                name = m.group(1) + "." + m.group(2)
                if name not in files:
                    continue
                src_lines, oracle_lines = files[name]
                lo = int(m.group(3))
                if lo > len(src_lines) or lo > len(oracle_lines) or src_lines[lo - 1] == oracle_lines[lo - 1]:
                    continue
                total += 1
                if trivial(src_lines[lo - 1]) and not trivial(oracle_lines[lo - 1]):
                    flagged += 1
                    rel = os.path.relpath(path, REPO)
                    print(f"{rel}: {m.group(0)}\n    src:    {src_lines[lo - 1].strip()[:80]!r}\n    oracle: {oracle_lines[lo - 1].strip()[:80]!r}")
    print(f"cite-drift: {total} citations in drifted lines, {flagged} flagged as likely oracle-numbered")
    return 0


if __name__ == "__main__":
    sys.exit(main())
