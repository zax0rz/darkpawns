#!/usr/bin/env python3
"""Check numeric position literals against C's POS_* defines (DP-1467).

A Go numeric literal carrying a `// POS_...` or `// Pos...` comment is a
transcription of a C `#define` in src/structs.h. A wrong literal reads green in
the oracle when the position only surfaces in a later command: Hellfire's
`SetPosition(5) // POS_SITTING` (DP-1467) knocked a victim to resting (5), not
sitting (6), and no Hellfire scenario ever looked at the victim.

This script fails when a commented literal disagrees with its named C define,
and separately checks the generated pkg/game/social_positions.go table against
C's command table (src/interpreter.c cmd_info[]), its real source. The socials
data file (lib/misc/socials, read by src/act.social.c) stores a different,
per-message *victim* position, so it is not the source of this table.

`make fidelity-depth` runs it. --self-test is the R5h control.
"""
from __future__ import annotations

import argparse
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]

# The number closest to the comment: `SetPosition(int(PosSleeping)) // ...` has
# no digit before its comment and is skipped; `f(5, 6, 7) // POS_X` reports 7.
LITERAL = re.compile(r"(\d+)[^\d]*//\s*(POS_[A-Z]+|Pos[A-Za-z]+)")
DEFINE = re.compile(r"^#define\s+(POS_[A-Z]+)\s+(\d+)\b", re.M)
CMD_ROW = re.compile(r'^\s*\{\s*"([^"]+)"\s*,\s*(POS_[A-Z]+)\s*,', re.M)
SOCIAL_ROW = re.compile(r'^\s*"([^"]+)":\s*(\d+)\s*,', re.M)


def short_name(token: str) -> str:
    if token.startswith("POS_"):
        return token[4:]
    if token.startswith("Pos"):
        return re.sub(r"(?<=[a-z])(?=[A-Z])", "_", token[3:]).upper()
    return token


def read(path: pathlib.Path) -> str:
    return path.read_text(encoding="latin-1")


def go_sources() -> list[tuple[str, str]]:
    files = []
    for path in sorted((ROOT / "pkg").rglob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        files.append((str(path.relative_to(ROOT)), read(path)))
    return files


def literal_errors(files, defines) -> list[str]:
    errors = []
    for name, text in files:
        for lineno, line in enumerate(text.splitlines(), 1):
            match = LITERAL.search(line)
            if not match:
                continue
            value, token = int(match.group(1)), match.group(2)
            want = defines.get("POS_" + short_name(token))
            if want is None:
                errors.append(f"{name}:{lineno}: unknown position {token!r}")
            elif value != want:
                errors.append(f"{name}:{lineno}: literal {value} but {token} = {want} (src/structs.h)")
    return errors


def social_errors(defines) -> list[str]:
    cmd = {m.group(1): m.group(2) for m in CMD_ROW.finditer(read(ROOT / "src" / "interpreter.c"))}
    table = {m.group(1): int(m.group(2)) for m in SOCIAL_ROW.finditer(read(ROOT / "pkg" / "game" / "social_positions.go"))}
    errors = []
    for name, value in sorted(table.items()):
        token = cmd.get(name)
        if token is None:
            errors.append(f"pkg/game/social_positions.go: {name!r} has no src/interpreter.c cmd_info[] row")
        elif defines.get(token) != value:
            errors.append(f"pkg/game/social_positions.go: {name} = {value} but cmd_info[] {token} = {defines[token]}")
    return errors


def self_test() -> int:
    defines = {"POS_RESTING": 5, "POS_SITTING": 6, "POS_STANDING": 8}
    wrong = [("x.go", "\t\tcn.SetPosition(5) // POS_SITTING\n")]
    right = [("x.go", "\t\tme.SetPosition(8) // PosStanding\nNoLiteral // PosSleeping\n")]
    if not literal_errors(wrong, defines):
        print("self-test: the wrong literal was not flagged", file=sys.stderr)
        return 1
    if literal_errors(right, defines):
        print("self-test: a correct literal or a comment-only line was flagged", file=sys.stderr)
        return 1
    print("check-position-literals self-test: wrong literal flagged; correct literal and comment-only line clean")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        return self_test()

    defines = {m.group(1): int(m.group(2)) for m in DEFINE.finditer(read(ROOT / "src" / "structs.h"))}
    if not defines:
        print("check-position-literals: no POS_ defines parsed from src/structs.h", file=sys.stderr)
        return 1
    errors = literal_errors(go_sources(), defines) + social_errors(defines)
    for error in errors:
        print(f"check-position-literals: {error}", file=sys.stderr)
    if errors:
        print(f"check-position-literals: {len(errors)} mismatch(es)", file=sys.stderr)
        return 1
    print("check-position-literals: OK (commented position literals and the social table match C)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
