#!/usr/bin/env python3
"""Exported-symbol caller census for the polish audit.

Indexes every exported top-level func and method declared under the given
roots, strips comments, and counts references to each identifier in non-test
files and in test files separately. Zero-non-test-reference symbols are the
'registered but never called' candidates; hand-verify each with grep.

Usage: polish-sweep.py <repo-root> [roots...]
"""
import re
import sys
from pathlib import Path

DECL = re.compile(r'^func\s+(?:\([^)]*\)\s+)?([A-Z]\w*)\s*\(')
METHOD = re.compile(r'^func\s+\((\w+)\s+\*?(\w+)\)\s+([A-Z]\w*)\s*\(')
IDENT = re.compile(r'\b([A-Z]\w*)\b')


def strip_comments(text: str) -> str:
    out = []
    i, n = 0, len(text)
    in_str = in_chr = in_raw = in_line = in_block = False
    while i < n:
        c = text[i]
        nxt = text[i + 1] if i + 1 < n else ''
        if in_raw:
            out.append(c)
            if c == '`':
                in_raw = False
            i += 1
        elif in_line:
            if c == '\n':
                in_line = False
                out.append(c)
            i += 1
        elif in_block:
            if c == '*' and nxt == '/':
                in_block = False
                i += 2
            else:
                if c == '\n':
                    out.append(c)
                i += 1
        elif in_str:
            out.append(c)
            if c == '\\':
                if i + 1 < n:
                    out.append(nxt)
                i += 2
            elif c == '"':
                in_str = False
                i += 1
            else:
                i += 1
        elif in_chr:
            out.append(c)
            if c == '\\':
                if i + 1 < n:
                    out.append(nxt)
                i += 2
            elif c == "'":
                in_chr = False
                i += 1
            else:
                i += 1
        else:
            if c == '/' and nxt == '/':
                in_line = True
                i += 2
            elif c == '/' and nxt == '*':
                in_block = True
                i += 2
            elif c == '"':
                in_str = True
                out.append(c)
                i += 1
            elif c == "'":
                in_chr = True
                out.append(c)
                i += 1
            elif c == '`':
                in_raw = True
                out.append(c)
                i += 1
            else:
                out.append(c)
                i += 1
    return ''.join(out)


def main() -> None:
    root = Path(sys.argv[1])
    roots = sys.argv[2:] or ['cmd', 'pkg', 'web', 'internal', 'admin-ui', 'archive', 'tools']
    decls = {}  # name -> list of (kind, recv, file, line)
    live_refs = {}  # name -> count in non-test files
    test_refs = {}  # name -> count in test files

    files = []
    for r in roots:
        base = root / r
        if not base.exists():
            continue
        files.extend(p for p in base.rglob('*.go'))
    files = sorted(set(files))

    # Pass 1: declarations. Skip _test.go entirely: Benchmark/Test helpers
    # there are go-test surface, not the game's wiring.
    for p in files:
        if p.name.endswith('_test.go'):
            continue
        for ln, line in enumerate(p.read_text(errors='replace').splitlines(), 1):
            m = DECL.match(line)
            if not m:
                continue
            mm = METHOD.match(line)
            if mm:
                decls.setdefault(mm.group(3), []).append(
                    ('method', mm.group(2), p, ln))
            else:
                decls.setdefault(m.group(1), []).append(
                    ('func', '-', p, ln))

    names = set(decls)
    # Pass 2: references (comments stripped).
    for p in files:
        body = strip_comments(p.read_text(errors='replace'))
        is_test = p.name.endswith('_test.go')
        lines = body.splitlines()
        for ln, line in enumerate(lines, 1):
            for m in IDENT.finditer(line):
                name = m.group(1)
                if name not in names:
                    continue
                # Skip self-declaration lines.
                skip = False
                for kind, recv, dp, dln in decls[name]:
                    if dp == p and dln == ln:
                        skip = True
                if skip:
                    continue
                tgt = test_refs if is_test else live_refs
                tgt[name] = tgt.get(name, 0) + 1

    rows = []
    for name, ds in sorted(decls.items()):
        live = live_refs.get(name, 0)
        test = test_refs.get(name, 0)
        if live == 0:
            for kind, recv, p, ln in ds:
                rel = p.relative_to(root)
                rows.append(f'{name}\t{kind}\t{recv}\t{rel}:{ln}\ttest_refs={test}')
    print('\n'.join(rows) if rows else 'NO CANDIDATES')
    print(f'## symbols={len(decls)} files={len(files)} zero-live={len(rows)}',
          file=sys.stderr)


if __name__ == '__main__':
    main()
