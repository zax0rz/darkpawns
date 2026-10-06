#!/usr/bin/env python3
"""Read-only source/site reconciliation; run from the repository root."""
import collections
import csv
from pathlib import Path
import re

TABLE = Path(__file__).with_name("2026-10-06-dp-1371-mudlog-sites.tsv")
TOKEN = re.compile(r'/\*.*?\*/|//[^\n]*|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'', re.S)


def uncomment(text):
    return TOKEN.sub(lambda m: re.sub(r"[^\n]", " ", m[0]) if m[0].startswith("/") else m[0], text)


def call_at(text, offset):
    opening = text.index("(", offset)
    depth, quoted, escaped = 1, False, False
    for i in range(opening + 1, len(text)):
        c = text[i]
        if quoted:
            if escaped:
                escaped = False
            elif c == "\\":
                escaped = True
            elif c == '"':
                quoted = False
        elif c == '"':
            quoted = True
        elif c == "(":
            depth += 1
        elif c == ")":
            depth -= 1
            if depth == 0:
                return " ".join(text[offset:i + 1].split())
    raise ValueError("unclosed call")


def main():
    sources = {}
    omitted = {"src/utils.c:212": "consumer", "src/whod.c:39": "LOG macro definition"}
    for path in sorted(Path("src").glob("*.c")):
        text = uncomment(path.read_text())
        pattern = r"\b(?:mudlog|LOG)\s*\(" if path.name == "whod.c" else r"\bmudlog\s*\("
        for match in re.finditer(pattern, text):
            site = f"{path}:{text.count(chr(10), 0, match.start()) + 1}"
            if site in omitted:
                continue
            sources[site] = call_at(text, match.start())
    with TABLE.open() as stream:
        rows = list(csv.DictReader(stream, delimiter="\t"))
    assert len({r["site"] for r in rows}) == len(rows), "duplicate inventory site"
    assert set(sources) == {r["site"] for r in rows}, "C sites and inventory differ"
    required = ("function", "payload", "type", "level", "file", "go_owner", "state", "proof", "group", "notes")
    for row in rows:
        assert sources[row["site"]] == row["expression"], f"changed C call: {row['site']}"
        assert all(row[field] for field in required), f"incomplete row: {row['site']}"
    counts = collections.Counter(r["group"] for r in rows)
    assert all(n <= 8 for group, n in counts.items() if group != "landed"), "train exceeds eight sites"
    disabled = [r for r in rows if r["state"] == "disabled-C"]
    assert len(disabled) == 1 and disabled[0]["site"] == "src/comm.c:1581"
    comm = Path("src/comm.c").read_text().splitlines()
    assert "#if 0" in comm[1577] and "#endif" in comm[1581], "disabled connection branch changed"
    print(f"mudlog inventory: {len(rows)} retained sites; {len(rows) - len(disabled)} active; eight LOG invocations; all calls reconciled")
    for group, n in counts.items():
        print(f"{group}\t{n}")


if __name__ == "__main__":
    main()
