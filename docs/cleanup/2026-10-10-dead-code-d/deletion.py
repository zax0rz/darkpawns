#!/usr/bin/env python3
"""PR-1 deletion driver (dead-code-d).

Deletes the symbols listed in docs/cleanup/2026-10-10-polish-audit/
{uncalled,deadcode-delta}.tsv, minus the PR-2 deferrals, per the brief:

  - re-greps callers on comment-stripped source before each deletion (R5g):
    a TSV row is a lead, not proof; drift since the audit aborts the row.
  - deletes the whole decl block (doc comment + func ... closing brace).
  - excises test functions that reference deleted symbols (PR A restore
    rule: a test is deleted only if it calls a function that was deleted).
  - writes a per-symbol log for the commit.

Usage: deletion.py <worktree-root>
"""
import importlib.util
import re
import subprocess
import sys
from pathlib import Path

_spec = importlib.util.spec_from_file_location(
    'census', Path(__file__).parents[1] / '2026-10-10-polish-audit' / 'exported-symbol-census.py')
_census = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_census)
strip_comments = _census.strip_comments

EXCLUDE = {'ExecuteZoneReset', 'RegisterObjectInstance',  # PR-2 scope
           'DoScan'}  # only caller is command.CmdScan - class (b)/(c) family
                     # awaiting Zach's wire-or-delete ruling; defer with it
ROOTS = ['cmd', 'pkg', 'web', 'internal', 'admin-ui', 'archive', 'tools']
TSVS = ['docs/cleanup/2026-10-10-polish-audit/uncalled.tsv',
        'docs/cleanup/2026-10-10-polish-audit/deadcode-delta.tsv']


def load_rows(root: Path):
    rows = {}
    for tsv in TSVS:
        for ln in (root / tsv).read_text().splitlines()[1:]:
            f = ln.split('\t')
            if len(f) < 2:
                continue
            sym, loc = f[0], f[1]
            if sym in EXCLUDE:
                continue
            # uncalled.tsv: file:line   delta.tsv: file:line:col
            m = re.match(r'(.+?):(\d+)(?::\d+)?$', loc)
            rows.setdefault(sym, set()).add((m.group(1), int(m.group(2))))
    return rows


def find_decl_line(text: list, sym: str, near: int):
    """Locate the decl within +-25 lines of the TSV line.

    `Sym` pins a top-level func; `Receiver.Method` pins a method on that
    receiver type (deadcode-delta rows use the dotted form).
    """
    if '.' in sym:
        recv, meth = sym.rsplit('.', 1)
        pat = re.compile(r'^func\s+\([^)]*' + re.escape(recv) +
                         r'\)\s*' + re.escape(meth) + r'\s*\(')
    else:
        pat = re.compile(r'^func\s+(?:\([^)]*\)\s+)?' + re.escape(sym) + r'\s*\(')
    best = None
    for off in range(0, 26):
        for ln in (near - off, near + off):
            if 1 <= ln <= len(text) and pat.match(text[ln - 1]):
                best = ln
                if off == 0:
                    return ln
    return best


def block_range(text: list, decl_ln: int):
    """Doc comment above + func line + body down to the closing col-0 brace.

    The terminator is `}` at column 0 exactly — an indented `\t}` closing an
    inner if/for must not end the block early.
    """
    start = decl_ln
    while start - 2 >= 0 and text[start - 2].lstrip().startswith('//'):
        start -= 1
    end = decl_ln
    if not text[decl_ln - 1].rstrip().endswith('}'):
        while end < len(text) and text[end] != '}':
            end += 1
        # text[end] is 0-based index of the col-0 `}`; its 1-based line is
        # end+1, and the range is 1-based inclusive.
        return start, end + 1
    return start, end


def main() -> None:
    root = Path(sys.argv[1])
    tests_only = '--tests-only' in sys.argv
    rows = {}
    method_syms = {}
    if tests_only:
        # Recover the deleted-symbol set and method-ness from the log of a
        # previous full run (decl text is read from HEAD via git).
        log_path = root / 'docs/cleanup/2026-10-10-dead-code-d/deletion-log.tsv'
        for ln in log_path.read_text().splitlines()[1:]:
            f = ln.split('\t')
            if len(f) < 3 or f[1] != 'DELETE':
                continue
            sym = f[0]
            first = f[2].split(';')[0]
            path, rng = first.rsplit(':', 1)
            s, _e = rng.split('-')
            blob = subprocess.run(['git', '-C', str(root), 'show', f'HEAD:{path}'],
                                  capture_output=True, text=True).stdout.splitlines()
            method_syms[sym] = any(re.match(r'^func\s+\(', blob[i - 1])
                                   for i in range(int(s), min(int(_e) + 1, len(blob)))
                                   if 0 < i <= len(blob))
        deleted_syms = sorted(method_syms)
    else:
        rows = load_rows(root)
        deleted_syms = []
    files = set()
    for r in ROOTS:
        base = root / r
        if base.exists():
            files.update(p for p in base.rglob('*.go'))
    files = sorted(set(files))
    non_test = [p for p in files if not p.name.endswith('_test.go')]
    stripped = {p: strip_comments(p.read_text(errors='replace'))
                for p in non_test}
    stripped_lines = {p: s.splitlines() for p, s in stripped.items()}

    log = ['symbol\taction\tdecls_removed\ttests_removed\tnote']
    sym_spans = {}  # sym -> list of (path, start, end) on pristine files
    plan = {}  # path -> list of (start, end) 1-based inclusive, on pristine files

    # Phase A: resolve every site and drift-check on PRISTINE files, so
    # deletions elsewhere in the same file cannot shift line numbers.
    # (--tests-only mode seeds deleted_syms above and leaves rows empty,
    # so this loop and Phase B are no-ops.)
    for sym in sorted(rows):
        sites = []
        ok = True
        for path, near in sorted(rows[sym]):
            p = root / path
            if not p.exists():
                log.append(f'{sym}\tSKIP\t\t\tdecl file missing: {path}')
                ok = False
                break
            text = p.read_text(errors='replace').splitlines()
            decl = find_decl_line(text, sym, near)
            if decl is None:
                log.append(f'{sym}\tSKIP\t\t\tdecl not found near {path}:{near}')
                ok = False
                break
            method_syms[sym] = bool(re.match(r'^func\s+\(', text[decl - 1]))
            sites.append((path, decl))
        if not ok:
            continue
        # R5g: any live (non-test, non-decl-site) reference to the symbol?
        # Dotted (method) rows: grep cannot tie `x.Commit(` to a receiver
        # type, so the deadcode -test reachability result is the evidence.
        # Top-level funcs: a live ref is a call-shaped occurrence; bare
        # field keys (`Number:`) and selectors (`z.Number`) do not count,
        # and any value-pass reference the rule misses breaks the build
        # in the gate loop anyway.
        drift = []
        if '.' not in sym:
            decl_pkg = ''
            m = re.search(r'^package (\w+)', (root / sites[0][0])
                          .read_text(errors='replace'), re.M)
            if m:
                decl_pkg = m.group(1)
            # Live call shapes: bare `Sym(` or own-package `pkg.Sym(`.
            # `x.Sym(` is some receiver's method, not this function.
            call = re.compile(r'(?<![\w.])' + re.escape(sym) + r'\s*\(')
            qualified = re.compile(r'\b' + re.escape(decl_pkg) +
                                   r'\.' + re.escape(sym) + r'\s*\(')
            for p in non_test:
                body = stripped_lines[p]
                decl_sites = set(sites)
                for ln, line in enumerate(body, 1):
                    # Skip decl-shaped lines: `func ...` (incl. methods with
                    # the same name) and interface-member `Sym(...)` lines —
                    # neither can be a call to our symbol.
                    if re.match(r'^\s*(func\b|[A-Za-z_]\w*\()', line):
                        continue
                    if call.search(line) or qualified.search(line):
                        rel = str(p.relative_to(root))
                        if (rel, ln) in decl_sites:
                            continue
                        drift.append(f'{rel}:{ln}')
        if drift:
            log.append(f'{sym}\tSKIP\t\t\tlive refs since audit: {", ".join(drift[:3])}')
            continue
        for path, decl in sites:
            p = root / path
            text = p.read_text(errors='replace').splitlines()
            s, e = block_range(text, decl)
            plan.setdefault(path, []).append((s, e))
            sym_spans.setdefault(sym, []).append((path, s, e))
        deleted_syms.append(sym)
        removed = ';'.join(f'{pa}:{s}-{e}' for pa, s, e in sym_spans[sym])
        log.append(f'{sym}\tDELETE\t{removed}\t\t')

    # Phase B: apply per file, descending so earlier spans stay valid.
    for path, spans in plan.items():
        p = root / path
        lines = p.read_text(errors='replace').splitlines()
        for s, e in sorted(spans, reverse=True):
            del lines[s - 1:e]
        p.write_text('\n'.join(lines) + '\n')

    # Pass 2: excise tests (and cascading test helpers) that reference
    # deleted symbols. Top-level funcs match bare calls or own-package-
    # qualified calls (never `x.Number` accesses); METHODS (dot-notation
    # receivers) match any `\bName(` — tests call them as recv.Name(.
    tests_removed = {}
    topfunc = re.compile(r'^func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(')
    pats = {}
    for sym in deleted_syms:
        if '.' in sym:
            pats[sym] = re.compile(r'\b' + re.escape(sym.rsplit('.', 1)[1]) + r'\s*\(')
        elif method_syms.get(sym):
            pats[sym] = re.compile(r'\b' + re.escape(sym) + r'\s*\(')
        else:
            pats[sym] = re.compile(r'(?<![\w.])' + re.escape(sym) + r'\s*\(')
    for it in range(6):
        changed = False
        for p in files:
            if not p.name.endswith('_test.go'):
                continue
            lines = p.read_text(errors='replace').splitlines()
            body = strip_comments('\n'.join(lines))
            hits = [s for s in deleted_syms if pats[s].search(body)]
            if not hits:
                continue
            # excise every top-level func whose body mentions a hit
            funcs = []
            i = 0
            while i < len(lines):
                m = topfunc.match(lines[i])
                if m:
                    j = i
                    if not lines[j].rstrip().endswith('}'):
                        while j < len(lines) and lines[j] != '}':
                            j += 1
                    funcs.append((i + 1, j + 1, m.group(1)))
                    i = j + 1
                else:
                    i += 1
            drop = []
            for s, e, name in funcs:
                seg = '\n'.join(lines[s - 1:e])
                if any(pats[h].search(seg) for h in hits):
                    drop.append((s, e, name))
            if not drop:
                continue
            for s, e, name in sorted(drop, reverse=True):
                del lines[s - 1:e]
                tests_removed.setdefault(str(p.relative_to(root)), []).append(name)
            p.write_text('\n'.join(lines) + '\n')
            changed = True
        if not changed:
            break

    for path, names in sorted(tests_removed.items()):
        for sym in deleted_syms:
            pass
        log.append(f'-\tTESTS\t{path}\t{len(names)}\t{",".join(sorted(set(names)))}')

    out = root / 'docs/cleanup/2026-10-10-dead-code-d/deletion-log.tsv'
    out.write_text('\n'.join(log) + '\n')
    print(f'deleted={len(deleted_syms)} skipped={sum(1 for l in log if "\tSKIP\t" in l)} '
          f'testfiles={len(tests_removed)}')
    for l in log:
        if '\tSKIP\t' in l:
            print('SKIPPED: ' + l)


if __name__ == '__main__':
    main()
