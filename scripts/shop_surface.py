#!/usr/bin/env python3
"""Validate the cross-file shop behavior map, independently of output-site maps.

Rows are branches (including silent state/math gates), not act/send call sites.
Evidence tokens are only scenarios with explicit seeds or Go test symbols.
"""
from __future__ import annotations

import csv
import pathlib
import re
import sys
import tempfile
from collections import Counter

ROOT = pathlib.Path(__file__).resolve().parent.parent
HEADER = ['system', 'c_site', 'function', 'branch', 'status', 'proof', 'owner', 'notes']
STATUSES = {'proven', 'delegated', 'fixed', 'unreachable', 'blocked'}
SITE = re.compile(r'(src/[\w.-]+\.c):([1-9][0-9]*)(?:-([1-9][0-9]*))?$')
TOKEN = re.compile(r'(?:Test\w+|[a-z0-9][a-z0-9-]*@[1-9][0-9]*(?:,[1-9][0-9]*)*)$')
FAMILIES = {'core': 'core shop transaction surface', 'special': 'special-procedure shop surface'}


def check(root=ROOT, verify_inventory=True):
    path = root / 'docs/fidelity/depth/surface/shops-behavior.tsv'
    errors = []
    with path.open() as stream:
        reader = csv.DictReader(stream, delimiter='\t')
        if reader.fieldnames != HEADER:
            return ['shop-surface: invalid header']
        rows = list(reader)
    tests = set()
    for file in root.rglob('*_test.go'):
        tests.update(re.findall(r'^func (Test\w+)\(', file.read_text(), re.M))
    seen = set()
    previous = {}
    for line, row in enumerate(rows, 2):
        where = f'{path.name}:{line}'
        if None in row or any(v is None for v in row.values()):
            errors.append(f'{where}: wrong field count')
            continue
        if row['system'] not in FAMILIES or row['status'] not in STATUSES:
            errors.append(f'{where}: unknown system/status')
        key = tuple(row[k] for k in ['system', 'function', 'branch'])
        if key in seen:
            errors.append(f'{where}: duplicate branch {key}')
        seen.add(key)
        site = SITE.fullmatch(row['c_site'])
        if not site or not (root / site[1]).is_file():
            errors.append(f'{where}: invalid C citation')
        else:
            start, end = int(site[2]), int(site[3] or site[2])
            if not 1 <= start <= end <= len((root / site[1]).read_text(encoding='latin1').splitlines()):
                errors.append(f'{where}: invalid C line range')
            if previous.get(site[1], 0) > start:
                errors.append(f'{where}: not in C source order')
            previous[site[1]] = start
        tokens = row['proof'].split(';') if row['proof'] else []
        if row['status'] in {'proven', 'fixed', 'delegated'} and not tokens:
            errors.append(f'{where}: evidence required')
        if not row['owner'] or not row['notes']:
            errors.append(f'{where}: owner and reasoning required')
        for token in tokens:
            if not TOKEN.fullmatch(token):
                errors.append(f'{where}: invalid proof token {token!r}')
            elif token.startswith('Test'):
                if token not in tests:
                    errors.append(f'{where}: missing test {token}')
            elif not (root / 'cmd/dp-oracle-diff/scenarios' / (token.split('@')[0] + '.txt')).is_file():
                errors.append(f'{where}: missing scenario {token}')
    if verify_inventory:
        inventory = root / 'docs/fidelity/depth/surface-inventory.tsv'
        with inventory.open() as stream:
            families = {r['family']: r for r in csv.DictReader(stream, delimiter='\t')}
        for system, family in FAMILIES.items():
            residual = sum(r['system'] == system and r['status'] == 'blocked' for r in rows)
            row = families[family]
            if row['units'] != str(residual):
                errors.append(f'{family}: units {row["units"]} != {residual} blocked branches')
            if (row['status'] == 'proven-already') != (residual == 0):
                errors.append(f'{family}: status disagrees with branch residual')
            if 'surface/shops-behavior.tsv' not in row['proof_or_attempt'].split(';'):
                errors.append(f'{family}: inventory must reference behavior map')
    return errors


def self_test():
    with tempfile.TemporaryDirectory() as tmp:
        root = pathlib.Path(tmp)
        path = root / 'docs/fidelity/depth/surface/shops-behavior.tsv'
        path.parent.mkdir(parents=True)
        (root / 'src').mkdir()
        (root / 'src/shop.c').write_text('void f() {}\n')
        (root / 'proof_test.go').write_text('func TestShopProof(t *testing.T) {}\n')
        row = ['core', 'src/shop.c:1', 'f', 'branch', 'delegated', 'TestShopProof', 'shared owner', 'boundary proof']
        def run(changed):
            with path.open('w') as stream:
                writer = csv.writer(stream, delimiter='\t')
                writer.writerow(HEADER)
                writer.writerows(changed)
            return check(root, False)
        assert not run([row])
        for index, value in [(0, 'wrong'), (1, 'src/shop.c:99'), (4, 'guess'), (5, 'run-name'), (5, 'TestMissing'), (5, 'missing@1'), (6, '')]:
            changed = row.copy()
            changed[index] = value
            assert run([changed]), (index, value)
        assert run([row, row]), 'duplicate branch control'
    print('shop-surface: self-test passed')


if __name__ == '__main__':
    if '--self-test' in sys.argv:
        self_test()
    else:
        errors = check()
        for error in errors:
            print(error)
        with (ROOT / 'docs/fidelity/depth/surface/shops-behavior.tsv').open() as stream:
            rows = list(csv.DictReader(stream, delimiter='\t'))
        for system in FAMILIES:
            print(f'{system}: {dict(Counter(r["status"] for r in rows if r["system"] == system))}')
        print(f'shop-surface: {len(rows)} branches, {len(errors)} error(s)')
        sys.exit(bool(errors))
