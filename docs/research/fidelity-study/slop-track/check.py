#!/usr/bin/env python3
"""Checks for slop.tsv (v2).

1. The brief's checker: every fix_commit and birth_commit resolves, and every
   coded row's c_truth names an existing src/ file and an in-range line.
2. Every c_truth_quote appears verbatim in its cited file, and (where the
   quote's line count matches the citation range) equals the cited lines.

Prints nothing when both pass.
"""
import csv, os, subprocess, sys

REPO = '/home/zach/darkpawns'
TSV = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'slop.tsv')


def git(*a):
    return subprocess.run(['git', '-C', REPO] + list(a), capture_output=True,
                          text=True, errors='replace').stdout


rows = list(csv.DictReader(open(TSV), delimiter='\t'))
cache = {}
PROBLEMS = []


def fail(*a):
    print(*a)
    PROBLEMS.append(a)


def filetext(f):
    if f not in cache:
        cache[f] = git('show', 'origin/main:' + f)
    return cache[f]


# ---- check 1 (the brief's) ----
seen = set()
for r in rows:
    for col in ('fix_commit', 'birth_commit'):
        c = r[col]
        if c and c not in seen:
            seen.add(c)
            if subprocess.run(['git', '-C', REPO, 'cat-file', '-e', c + '^{commit}'],
                              capture_output=True).returncode != 0:
                fail('MISSING COMMIT:', c)
for r in rows:
    if r['status'] != 'coded':
        continue
    ct = r['c_truth']
    if not ct:
        fail('EMPTY C_TRUTH:', r['id'])
        continue
    f = ct.split(':')[0]
    n = ct.split(':')[1].split('-')[0]
    if subprocess.run(['git', '-C', REPO, 'cat-file', '-e', 'origin/main:' + f],
                      capture_output=True).returncode != 0:
        fail('MISSING FILE:', ct)
        continue
    if len(filetext(f).splitlines()) < int(n):
        fail('LINE OUT OF RANGE:', ct)

# ---- check 2 (quotes) ----
for r in rows:
    if r['status'] != 'coded':
        continue
    ct, q = r['c_truth'], r['c_truth_quote']
    if not q:
        fail('EMPTY C_TRUTH_QUOTE:', r['id'])
        continue
    f, rng = ct.split(':')
    a = int(rng.split('-')[0])
    b = int(rng.split('-')[1]) if '-' in rng else a
    cited = filetext(f).splitlines()[a - 1:b]
    parts = q.split(' ~~~ ')
    if len(parts) != len(cited):
        fail('QUOTE LINE COUNT MISMATCH:', r['id'], ct, len(parts), len(cited))
        continue
    filelines = set(filetext(f).splitlines())
    for part in parts:
        if part not in filelines:
            fail('QUOTE NOT VERBATIM IN FILE:', r['id'], ct, repr(part[:60]))
            break
    if parts != cited:
        fail('QUOTE DIFFERS FROM CITED LINES:', r['id'], ct)

print('checks complete', file=sys.stderr)
sys.exit(1 if PROBLEMS else 0)
