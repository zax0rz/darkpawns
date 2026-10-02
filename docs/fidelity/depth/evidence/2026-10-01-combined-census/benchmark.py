#!/usr/bin/env python3
"""Run the approved serial comparison through census.sh on one fixed HEAD.

Usage: benchmark.py CHECKOUT ARCHIVE-DIR
All benchmark invocations must start from a clean checkout. No parallel census.
"""
from pathlib import Path
import json
import os
import re
import subprocess
import sys
import time

repo = Path(sys.argv[1]).resolve()
evidence = Path(sys.argv[2]).resolve()
evidence.mkdir(parents=True, exist_ok=True)
def command(args, **kwargs):
    return subprocess.run(args, cwd=repo, check=True, text=True, capture_output=True, **kwargs).stdout.strip()
head = command(['git', 'rev-parse', 'HEAD'])
assert not command(['git', 'status', '--porcelain']), 'benchmark needs a clean tree'
records = {}

def run(label, flags, jobs, timings=None):
    assert command(['git', 'rev-parse', 'HEAD']) == head
    assert not command(['git', 'status', '--porcelain'])
    env = dict(os.environ)
    env.pop('CENSUS_TIMINGS_FILE', None)
    if timings:
        env['CENSUS_TIMINGS_FILE'] = str(timings)
    before = time.monotonic()
    args = [str(repo/'scripts/census.sh'), 'start', '--name', f'dp-1371-combined-fixed-{label}', '--jobs', str(jobs)] + flags
    out = command(args, env=env)
    print(out, flush=True)
    directory = Path(out.removeprefix('census started: '))
    while True:
        # Wrapper blocks; never poll status or read a running census.log.
        p = subprocess.run([str(repo/'scripts/census.sh'), 'wait', '--run', str(directory)], cwd=repo,
                           env=env, text=True, capture_output=True)
        print(p.stdout.strip(), flush=True)
        if p.returncode == 3:
            continue
        if p.returncode:
            raise RuntimeError(f'{label} failed: {p.stdout} {p.stderr}')
        break
    summary = (directory/'summary.txt').read_text().strip()
    seconds = float(re.search(r'elapsed=([0-9.]+)s', summary)[1])
    records[label] = dict(path=str(directory), elapsed=seconds, wrapper_wall=time.monotonic()-before,
                          summary=summary, head=(directory/'go-head.txt').read_text().strip(), command=args)
    assert records[label]['head'] == head
    (evidence/'benchmark-progress.json').write_text(json.dumps(records, indent=2)+'\n')
    return directory


def results(directory, claims=False):
    rows = {}
    prefix = 'claims-' if claims else ''
    for filename in (f'{prefix}results.tsv', f'{prefix}recheck-results.tsv'):
        path = directory/filename
        if not path.exists():
            continue
        seen = set()
        for line in path.read_text().splitlines():
            fields = line.split('\t')
            kind, name = fields[:2]
            pair = (int(fields[2]), name) if claims else (1, name)
            assert pair not in seen, f'duplicate {pair} in {path}'
            seen.add(pair)
            rows[pair] = kind
    return rows

full = run('baseline-full', [], 36)
claims = run('baseline-claims', ['--claims'], 36)
frozen = evidence/'frozen-timings.tsv'
timings = {}
for directory, claimed in ((full, False), (claims, True)):
    for line in (directory/('claims-results.tsv' if claimed else 'results.tsv')).read_text().splitlines():
        fields = line.split('\t')
        seed, seconds = (int(fields[2]), fields[3]) if claimed else (1, fields[2])
        timings[seed, fields[1]] = seconds
frozen.write_text(''.join(f'{seed}\t{name}\t{seconds}\n' for (seed, name), seconds in sorted(timings.items())))
comparison = []
for workers in (24, 36):
    directory = run(f'workers-{workers}', ['--combined'], workers, frozen)
    for projection, baseline, claimed in (('full', full, False), ('claims', claims, True)):
        old, new = results(baseline, claimed), results(directory, claimed)
        differences = [dict(seed=pair[0], scenario=pair[1], baseline=old.get(pair), combined=new.get(pair))
                       for pair in sorted(set(old)|set(new)) if old.get(pair) != new.get(pair)]
        row = dict(workers=workers, projection=projection, pairs=len(old), differences=differences,
                   baseline_verdict=re.search(r'verdict=([A-Z_]+)', (baseline/'summary.txt').read_text())[1],
                   combined_verdict=(directory/f'{projection}-verdict').read_text().strip())
        comparison.append(row)
    # No skipped union pair: the combined initial rows must be exactly the union.
    union = results(full) | results(claims, True)
    observed = {tuple((int(f[2]), f[1])) for f in
                (line.split('\t') for line in (directory/'combined-results.tsv').read_text().splitlines())}
    assert set(union) == observed, 'combined union coverage differs'
    (evidence/'comparisons.json').write_text(json.dumps(comparison, indent=2)+'\n')
    if any(row['differences'] for row in comparison):
        raise RuntimeError('per-pair classifications differ; inspect comparisons.json, do not hide them')
records['baseline-total'] = records['baseline-full']['elapsed'] + records['baseline-claims']['elapsed']
(evidence/'benchmark-results.json').write_text(json.dumps(dict(head=head, runs=records, comparisons=comparison), indent=2)+'\n')
(evidence/'benchmark-done').write_text('all final classifications match; all verdicts clean\n')
print(json.dumps(dict(head=head, runs=records, comparisons=comparison), indent=2), flush=True)
