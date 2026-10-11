#!/usr/bin/env python3
"""Re-derive the DP-1415/PR-4 Lua seam table from src/scripts.c (read-only).

For every site line in the lua-needs-proof survey, scan forward to the mudlog
call it guards and emit site, function, payload, type, level and file flag. The
test's expectation table is generated from this, so it stays citable to src/
(R5g) instead of being transcribed by hand.

    python3 docs/fidelity/depth/handoff/2026-10-11-dp-1371-lua-proofs-extract.py \
        > docs/fidelity/depth/handoff/2026-10-11-dp-1371-lua-proofs.tsv
"""
import pathlib
import re
import sys

SURVEY = pathlib.Path("docs/fidelity/depth/handoff/2026-10-09-dp-1371-lua-needs-proof-survey.tsv")
SCRIPTS = pathlib.Path("src/scripts.c")

lines = SCRIPTS.read_text().split("\n")


def literal(text, start):
    """Return (value, end) for the C string literal starting at index start."""
    out = []
    i = start + 1
    while text[i] != '"':
        if text[i] == "\\":
            nxt = text[i + 1]
            out.append({"n": "\n", "r": "\r", "t": "\t", '"': '"', "\\": "\\"}.get(nxt, nxt))
            i += 2
        else:
            out.append(text[i])
            i += 1
    return "".join(out), i + 1


def mudlog_at(site):
    for idx in range(site - 1, min(site + 24, len(lines))):
        pos = lines[idx].find("mudlog(")
        if pos < 0:
            continue
        chunk = lines[idx][pos:]
        depth, end = 0, 0
        for j, ch in enumerate(chunk):
            if ch == "(":
                depth += 1
            elif ch == ")":
                depth -= 1
                if depth == 0:
                    end = j
                    break
        chunk = chunk[: end + 1]
        payload, i = "", chunk.index("(") + 1
        while True:
            while i < len(chunk) and chunk[i] in " \t\n":
                i += 1
            if i < len(chunk) and chunk[i] == '"':
                piece, i = literal(chunk, i)
                payload += piece
            else:
                break
        typ = next((t for t in ("BRF", "NRM", "CMP") if t in chunk), "?")
        lvl = re.search(r"LVL_[A-Z]+", chunk)
        tofile = "TRUE" if re.search(r",\s*TRUE\s*\)", chunk) else "FALSE" if re.search(r",\s*FALSE\s*\)", chunk) else "?"
        return payload, typ, lvl.group(0) if lvl else "?", tofile
    return None, None, None, None


def main():
    print("site\tc_function\tpayload\ttype\tlevel\tfile")
    rows = [l.split("\t") for l in SURVEY.read_text().split("\n")[1:] if l.strip()]
    for row in rows:
        payload, typ, lvl, tofile = mudlog_at(int(row[0].split(":")[1]))
        if payload is None:
            print(f"{row[0]}\t{row[1]}\tNOT_FOUND\t\t\t", file=sys.stderr)
            continue
        print(f"{row[0]}\t{row[1]}\t{payload}\t{typ}\t{lvl}\t{tofile}")


if __name__ == "__main__":
    main()
