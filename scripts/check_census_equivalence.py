#!/usr/bin/env python3
"""Compare retained targeted census proofs without hiding retries or draw flakes."""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path


def inspect(run):
    if (run/'exit-code').read_text().strip() != '0' or not (run/'MANIFEST.md').exists():
        raise ValueError(f'{run}: not a completed clean run')
    summary = (run/'summary.txt').read_text()
    if 'verdict=CLEAN_AFTER_RECHECK' in summary:
        raise ValueError(f'{run}: infrastructure rechecks invalidate the no-new-flake proof')
    rows = {f[1]: f[0] for f in (l.split('\t') for l in (run/'results.tsv').read_text().splitlines())}
    retries = [l for l in (run/'census.log').read_text().splitlines() if l.startswith('RETRY ')]
    if retries:
        raise ValueError(f'{run}: internal retries retained; no-new-flake proof failed: {retries}')
    captures = {}
    for path in sorted(run.glob('initial-logs/attempt.*/*.capture*/blocks.json')):
        name, attempt = path.parent.name.rsplit('.capture',1)
        if attempt != '1' or name in captures:
            raise ValueError(f'{run}: duplicate or later capture for {name}')
        payload = json.loads(path.read_text())
        draws = {side:[l for l in (path.parent/f'{side}-draws.log').read_text().splitlines() if l.startswith('DRAW ')] for side in ('c','go')}
        if not draws['c'] or not draws['go']:
            raise ValueError(f'{run}: missing raw draw rows for {name}')
        log = path.parent.parent/(name+'.attempt1.log')
        reports = [l for l in log.read_text().splitlines() if l.startswith('draw-parity\t')]
        if len(reports)!=1 or not reports[0].startswith('draw-parity\tMATCH\t'):
            raise ValueError(f'{run}: draw mismatch for {name}: {reports}')
        # Foreign/bind failure must never become a clean attempted probe.
        for side in ('c','go'):
            engine_log=(path.parent/f'{side}-server.log').read_text()
            if 'listener failed' in engine_log or 'SYSERR: bind' in engine_log:
                raise ValueError(f'{run}: listener failure survived into capture for {name}')
        captures[name] = dict(blocks=payload,draws=draws,report=reports[0])
    if set(captures)!=set(rows):
        raise ValueError(f'{run}: capture coverage differs: {set(rows)^set(captures)}')
    return dict(rows=rows,captures=captures)


def compare(paths):
    baseline=inspect(paths[0])
    results={}
    for path in paths:
        actual=inspect(path)
        if actual!=baseline:
            changed=[name for name in baseline['rows'] if actual['rows'].get(name)!=baseline['rows'][name] or actual['captures'].get(name)!=baseline['captures'][name]]
            raise ValueError(f'{path}: verdict/transcript/draw differences: {changed}')
        content=json.dumps(actual,sort_keys=True).encode()
        results[str(path)] = dict(scenarios=len(actual['rows']),verdicts=dict(Counter(actual['rows'].values())),
                                 blocks=sum(len(c['blocks']) for c in actual['captures'].values()),
                                 sha256=hashlib.sha256(content).hexdigest())
    return results


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('runs',type=Path,nargs='+')
    parser.add_argument('--output',type=Path)
    args=parser.parse_args()
    try:
        result=compare(args.runs)
    except (ValueError,OSError) as exc:
        parser.exit(1,f'{exc}\n')
    text=json.dumps(result,indent=2)+'\n'
    if args.output:args.output.write_text(text)
    print(text,end='')


if __name__=='__main__':
    main()
