#!/usr/bin/env python3
"""Summarize retained mutation evidence without treating arbitrary failures as kills."""
import argparse
from collections import Counter
import csv
from datetime import datetime
import hashlib
import json
from pathlib import Path
import re
import statistics

if __package__:
    from .run import classify
else:
    from run import classify


def write_report(sample_path, evidence, out):
    sample = json.loads(sample_path.read_text())
    rows, costs, starts, ends, patches = [], [], [], [], []
    counts = Counter()
    for case in sample['selected']:
        directory = evidence / 'mutations' / case['id']
        plan = json.loads((directory / 'PLAN.json').read_text())
        result = json.loads((directory / 'RESULT.json').read_text())
        if plan['head'] != sample['head'] or plan['sample'] != case or result['restored_git_status']:
            raise ValueError(f'{case["id"]}: frame/head/restoration mismatch')
        outcome, detail = classify(result['clean'], result['broken'], result['restored'], case['symbol'], plan['assertion'])
        if outcome != result['outcome']:
            raise ValueError(f'{case["id"]}: retained classification differs: {outcome}')
        matched = [e for e in result['broken']['events'] if
                   (e.get('Test') == case['symbol'] or e.get('Test', '').startswith(case['symbol'] + '/'))
                   and e.get('Action') == 'output' and re.search(plan['assertion'], e.get('Output', ''))
                   and not e.get('Output', '').startswith(('===', '---'))]
        if outcome == 'killed' and not matched:
            raise ValueError(f'{case["id"]}: no assertion output mapped')
        counts[outcome] += 1
        seconds = sum(result[p]['seconds'] for p in ['clean', 'broken', 'restored'])
        costs.append(seconds)
        starts.append(datetime.fromisoformat(plan['started_utc']))
        ends.extend(datetime.fromisoformat(e['Time']) for e in result['restored']['events'] if 'Time' in e)
        anchor = case['anchor']
        rows.append({'id': case['id'], 'package': case['package'], 'symbol': case['symbol'],
                     'test': f'{case["file"]}:{case["line"]}', 'manifest': f'{anchor["manifest"]}:{anchor["line"]}',
                     'claim': anchor['case_id'], 'c_site': anchor['c_site'], 'outcome': outcome,
                     'mutation_path': plan['path'], 'mutation': plan['behavior_changed'],
                     'assertion_pattern': plan['assertion'],
                     'failing_test': ';'.join(dict.fromkeys(e['Test'] for e in matched)) if outcome == 'killed' else '',
                     'assertion_output': '\\n'.join(e['Output'].strip() for e in matched) if outcome == 'killed' else '',
                     'clean_exit': result['clean']['exit'], 'broken_exit': result['broken']['exit'],
                     'restored_exit': result['restored']['exit'], 'seconds': f'{seconds:.6f}',
                     'references_not_certified': len(case['references']) - 1,
                     'evidence': f'mutations/{case["id"]}/'})
        patches.append({'id': case['id'], 'claim': anchor['case_id'], 'outcome': outcome,
                        'diff': (directory / 'mutant.diff').read_text()})
    if len(rows) != sample['sample_size']:
        raise ValueError('incomplete sample')
    out.mkdir(parents=True, exist_ok=True)
    with (out / 'results.tsv').open('w') as stream:
        writer = csv.DictWriter(stream, fieldnames=list(rows[0]), delimiter='\t', lineterminator='\n')
        writer.writeheader()
        writer.writerows(rows)
    (out / 'mutations.json').write_text(json.dumps(patches, indent=2) + '\n')
    payload = {'sample_sha256': hashlib.sha256(sample_path.read_bytes()).hexdigest(), 'head': sample['head'],
               'population': sample['frame_size'], 'selected': len(rows), 'outcomes': dict(counts),
               'mapped_anchors': counts['killed'], 'other_references_not_certified': sum(r['references_not_certified'] for r in rows),
               'execution_seconds_sum': sum(costs), 'execution_seconds_mean': statistics.mean(costs),
               'execution_seconds_median': statistics.median(costs), 'execution_seconds_p95_nearest_rank': sorted(costs)[(len(costs) * 95 + 99) // 100 - 1],
               'first_execution_utc': min(starts).isoformat(), 'last_restored_event_utc': max(ends).isoformat(),
               'campaign_wall_seconds_from_lock': (max(ends) - datetime.fromisoformat(sample['created_utc'])).total_seconds(),
               'serial_seconds_projected_population': statistics.mean(costs) * sample['frame_size'],
               'serial_seconds_projected_remaining': statistics.mean(costs) * (sample['frame_size'] - len(rows)),
               'evidence_root': str(evidence)}
    (out / 'summary.json').write_text(json.dumps(payload, indent=2, sort_keys=True) + '\n')
    checksums = []
    for file in sorted(evidence.rglob('*')):
        if file.is_file() and not any(p.endswith('-cache') or p == '__pycache__' for p in file.parts) and file.suffix in {'.json', '.jsonl', '.diff', '.txt', '.tsv', '.md', '.log', '.py'}:
            checksums.append(f'{hashlib.sha256(file.read_bytes()).hexdigest()}\t{file.relative_to(evidence)}')
    (out / 'evidence-sha256.tsv').write_text('sha256\tpath\n' + '\n'.join(checksums) + '\n')
    return payload


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--sample', type=Path, required=True)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(write_report(args.sample, args.evidence, args.out), indent=2))


if __name__ == '__main__':
    main()
