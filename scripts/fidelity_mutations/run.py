#!/usr/bin/env python3
"""Execute registered proof mutations in detached, package-isolated worktrees."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time


def capture(command, cwd, directory, label, timeout=180):
    started = time.monotonic()
    try:
        run = subprocess.run(command, cwd=cwd, capture_output=True, text=True, timeout=timeout)
        status, output = run.returncode, run.stdout + run.stderr
    except subprocess.TimeoutExpired as exc:
        status = 124
        output = (exc.stdout or b'').decode() if isinstance(exc.stdout, bytes) else exc.stdout or ''
        output += (exc.stderr or b'').decode() if isinstance(exc.stderr, bytes) else exc.stderr or ''
    (directory / f'{label}.jsonl').write_text(output)
    events = []
    for line in output.splitlines():
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            pass
    result = {'command': command, 'cwd': str(cwd), 'exit': status, 'seconds': time.monotonic() - started, 'events': events, 'output': output}
    (directory / f'{label}-execution.json').write_text(json.dumps(result, indent=2) + '\n')
    return result


def terminal(run, symbol):
    return next((e['Action'] for e in reversed(run['events']) if e.get('Test') == symbol and e.get('Action') in {'pass', 'fail', 'skip'}), 'missing')


def skipped(run, symbol):
    return any(e.get('Action') == 'skip' and (e.get('Test') == symbol or e.get('Test', '').startswith(symbol + '/')) for e in run['events'])


def classify(clean, broken, restored, symbol, assertion):
    if terminal(clean, symbol) != 'pass' or clean['exit'] or skipped(clean, symbol):
        return 'baseline failure', 'clean selected test did not pass'
    if terminal(restored, symbol) != 'pass' or restored['exit'] or skipped(restored, symbol):
        return 'restoration failure', 'restored selected test did not pass'
    text = broken.get('output', ''.join(e.get('Output', '') for e in broken['events']))
    if broken['exit'] == 124 or 'test timed out after' in text or 'i/o timeout' in text:
        return 'timeout', 'mutant exceeded the retained timeout'
    if re.search(r'panic:|fatal error:|signal:|\[build failed\]', text):
        return 'crash', 'panic, signal, fatal runtime error or build failure; not an assertion kill'
    if skipped(broken, symbol):
        return 'survived', 'mutant made selected test skip; not proven'
    if terminal(broken, symbol) == 'pass' and broken['exit'] == 0:
        return 'survived', 'selected test passed despite the claimed behavior change'
    own_output = ''.join(e.get('Output', '') for e in broken['events'] if e.get('Test', '').split('/')[0] == symbol and e.get('Action') == 'output' and not e.get('Output', '').startswith(('===', '---')))
    if terminal(broken, symbol) == 'fail' and re.search(assertion, own_output):
        return 'killed', 'selected assertion for the anchor claim failed'
    return 'unmapped failure', 'mutant failed outside the registered claim assertion; not credited'


def one_case(case, plan, tree, out, head):
    directory = out / case['id']
    if directory.exists():
        raise ValueError(f'refusing to overwrite evidence {directory}')
    directory.mkdir()
    spec = dict(plan)
    spec['sample'] = case
    spec['head'] = head
    spec['started_utc'] = datetime.now(timezone.utc).isoformat()
    (directory / 'PLAN.json').write_text(json.dumps(spec, indent=2) + '\n')
    symbol = case['symbol'].split(':')[-1]
    command = ['go', 'test', '-json', '-count=1', '-timeout', '120s', '-run', '^' + re.escape(symbol) + '$', case['package']]
    if 'reason_no_mutation' in plan:
        clean = capture(command, tree, directory, 'clean')
        result = {'id': case['id'], 'outcome': 'no meaningful mutation possible', 'detail': plan['reason_no_mutation'], 'clean': clean}
    else:
        path = Path(plan['path'])
        if not str(path).startswith('pkg/') or path.suffix != '.go' or path.name.endswith('_test.go') or '..' in path.parts:
            raise ValueError('mutation must affect a production Go file under pkg/')
        target = tree / path
        original = target.read_bytes()
        before, after = plan['before'], plan['after']
        text = original.decode()
        if text.count(before) != 1 or before == after:
            raise ValueError(f'{case["id"]}: mutation must have one exact source match')
        clean = capture(command, tree, directory, 'clean')
        try:
            target.write_text(text.replace(before, after, 1))
            diff = subprocess.check_output(['git', 'diff', '--', str(path)], cwd=tree, text=True)
            (directory / 'mutant.diff').write_text(diff)
            (directory / 'source-clean.sha256').write_text(hashlib.sha256(original).hexdigest() + '\n')
            broken = capture(command, tree, directory, 'broken')
        finally:
            target.write_bytes(original)
        (directory / 'source-restored.sha256').write_text(hashlib.sha256(target.read_bytes()).hexdigest() + '\n')
        restored = capture(command, tree, directory, 'restored')
        verdict, detail = classify(clean, broken, restored, symbol, plan['assertion'])
        result = {'id': case['id'], 'outcome': verdict, 'detail': detail, 'clean': clean, 'broken': broken, 'restored': restored}
    state = subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=no'], cwd=tree, text=True).strip()
    result['restored_git_status'] = state
    if state:
        raise ValueError(f'{case["id"]}: mutant tree not restored: {state}')
    (directory / 'RESULT.json').write_text(json.dumps(result, indent=2) + '\n')
    print(f'{case["id"]} {result["outcome"]}', flush=True)
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--sample', type=Path, required=True)
    p.add_argument('--plans', type=Path, required=True)
    p.add_argument('--root', type=Path, required=True)
    p.add_argument('--trees', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('--ids', help='comma-separated IDs; omitted means every planned ID')
    p.add_argument('--jobs', type=int, default=3)
    a = p.parse_args()
    sample = json.loads(a.sample.read_text())
    plans = json.loads(a.plans.read_text())
    wanted = set(a.ids.split(',')) if a.ids else set(plans)
    cases = [c for c in sample['selected'] if c['id'] in wanted]
    if len(cases) != len(wanted) or not wanted <= set(plans):
        p.error('requested IDs must be in both the locked sample and plans')
    head = sample['head']
    a.out.mkdir(parents=True, exist_ok=True)
    groups = {}
    for case in cases:
        groups.setdefault(case['package'], []).append(case)
    def group(package, members):
        tree = a.trees / package.removeprefix('./').replace('/', '-')
        if not tree.exists():
            tree.parent.mkdir(parents=True, exist_ok=True)
            subprocess.run(['git', 'worktree', 'add', '--detach', str(tree), head], cwd=a.root, check=True, capture_output=True)
        if subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=tree, text=True).strip() != head:
            raise ValueError('mutant tree HEAD differs from registered frame')
        if subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=no'], cwd=tree, text=True).strip():
            raise ValueError('mutant tree dirty before execution')
        return [one_case(c, plans[c['id']], tree, a.out, head) for c in members]
    started = time.monotonic()
    with ThreadPoolExecutor(max_workers=a.jobs) as pool:
        futures = [pool.submit(group, package, members) for package, members in groups.items()]
        for future in futures:
            future.result()
    print(f'completed {len(cases)} mutations in {time.monotonic() - started:.3f}s', flush=True)

if __name__ == '__main__':
    main()
