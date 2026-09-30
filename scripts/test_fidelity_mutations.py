#!/usr/bin/env python3
"""Failure-sensitive tests for the mutation audit, separate from sampled game proofs."""
import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from fidelity_mutation_sample import compact_selection, sample
from fidelity_mutations.run import capture, classify, one_case
from fidelity_mutations.report import write_report


def result(action='pass', code=0, output='', child=None):
    events = [{'Test': 'TestClaim', 'Action': action}]
    if output:
        events.insert(0, {'Test': child or 'TestClaim', 'Action': 'output', 'Output': output})
    return {'events': events, 'exit': code}


class SamplingTests(unittest.TestCase):
    def test_hamilton_quotas_cover_small_strata_without_largest_stratum_takeover(self):
        frame = [{'package': package, 'category': 'state', 'symbol': f'{package}:Test{i}'}
                 for package, count in [('large', 80), ('medium', 18), ('tiny', 2)]
                 for i in range(count)]
        chosen, strata = sample(frame, 20, 20260930)
        self.assertEqual({r['package']: r['sample'] for r in strata},
                         {'large': 15, 'medium': 4, 'tiny': 1})
        self.assertEqual(len({c['symbol'] for c in chosen}), 20)
        self.assertEqual(sample(list(reversed(frame)), 20, 20260930), (chosen, strata))
        self.assertNotEqual(sample(frame, 20, 20260931)[0], chosen)

    def test_compact_selection_preserves_anchor_identity_without_full_frame(self):
        selection = json.loads((Path(__file__).parent / 'fidelity_mutations/selection.json').read_text())
        self.assertEqual(selection['seed'], 20260930)
        self.assertEqual(selection['frame_sha256'],
                         '8d4bcbeae99868a34ee1cf1ead47d67510a25fd1494c9ea5f1e9724095f77bf1')
        self.assertEqual(len(selection['selected']), 100)
        self.assertEqual(len({c['symbol'] for c in selection['selected']}), 100)
        self.assertEqual(len({c['package'] for c in selection['selected']}), 9)
        full = dict(selection, frame=[{'unselected': True}], selected=[dict(c, references=[],
                    anchor=dict(c['anchor'], notes='full frame metadata')) for c in selection['selected']])
        self.assertEqual(compact_selection(full, selection['frame_sha256']), selection)
        self.assertNotIn('frame', selection)
        with self.assertRaises(ValueError):
            sample([{'package': 'one', 'category': 'state', 'symbol': 'TestOne'}], 2, 1)


class ClassificationTests(unittest.TestCase):
    def classify(self, broken, clean=None, restored=None):
        return classify(clean or result(), broken, restored or result(), 'TestClaim', 'claim value')[0]

    def test_only_registered_claim_assertion_is_a_kill(self):
        self.assertEqual(self.classify(result('fail', 1, 'claim value = 0', 'TestClaim/anchor')), 'killed')
        self.assertEqual(self.classify(result('fail', 1, 'unrelated value = 0')), 'unmapped failure')
        self.assertEqual(self.classify(result('fail', 1, 'claim value = 0', 'TestOther')), 'unmapped failure')
        self.assertEqual(self.classify(result('pass', 1, 'claim value = 0')), 'unmapped failure')
        self.assertEqual(self.classify(result()), 'survived')
        self.assertEqual(self.classify(result('fail', 1, '--- FAIL: TestClaim/claim value (0.00s)')), 'unmapped failure')

    def test_skipped_descendants_and_failed_restore_are_not_proofs(self):
        skipped = result()
        skipped['events'].insert(0, {'Test': 'TestClaim/anchor', 'Action': 'skip'})
        self.assertEqual(self.classify(skipped), 'survived')
        skipped_failure = result('fail', 1, 'claim value')
        skipped_failure['events'].insert(0, {'Test': 'TestClaim/anchor', 'Action': 'skip'})
        self.assertEqual(self.classify(skipped_failure), 'survived')
        self.assertEqual(self.classify(result('fail', 1, 'claim value'), clean=skipped), 'baseline failure')
        self.assertEqual(self.classify(result('fail', 1, 'claim value'), restored=skipped), 'restoration failure')
        self.assertEqual(self.classify(result('fail', 1, 'claim value'), restored=result('fail', 1)), 'restoration failure')

    def test_panics_build_failures_and_timeouts_are_not_assertion_kills(self):
        for output in ['panic: claim value', 'fatal error: claim value', 'signal: killed; claim value', '[build failed] claim value']:
            with self.subTest(output=output):
                self.assertEqual(self.classify(result('fail', 1, output)), 'crash')
        for output, code in [('test timed out after 1s; claim value', 1), ('i/o timeout; claim value', 1), ('', 124)]:
            self.assertEqual(self.classify(result('fail', code, output)), 'timeout')
        raw_compile = result('missing', 1)
        raw_compile['output'] = '# fixture\n[build failed]'
        self.assertEqual(self.classify(raw_compile), 'crash')


class ExecutionTests(unittest.TestCase):
    def fixture(self, directory):
        root = Path(directory)
        (root / 'pkg/fixture').mkdir(parents=True)
        (root / 'go.mod').write_text('module proof-fixture\n\ngo 1.24\n')
        source = root / 'pkg/fixture/value.go'
        source.write_text('package fixture\nfunc Value() int { return 7 }\n')
        (root / 'pkg/fixture/value_test.go').write_text('''package fixture
import "testing"
func TestClaim(t *testing.T) { if Value() != 7 { t.Errorf("claim value = %d", Value()) } }
''')
        subprocess.run(['git', 'init', '-q', str(root)], check=True)
        subprocess.run(['git', 'add', '.'], cwd=root, check=True)
        subprocess.run(['git', '-c', 'user.name=Proof Fixture', '-c', 'user.email=fixture@example.invalid',
                        'commit', '-qm', 'fixture'], cwd=root, check=True)
        head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
        case = {'id': 'Mfixture', 'symbol': 'TestClaim', 'package': './pkg/fixture'}
        plan = {'path': 'pkg/fixture/value.go', 'before': 'return 7', 'after': 'return 6', 'assertion': 'claim value', 'behavior_changed': 'Return six instead of seven'}
        return root, source, head, case, plan

    def test_real_go_clean_broken_restore_and_evidence_refusal(self):
        with tempfile.TemporaryDirectory() as directory, tempfile.TemporaryDirectory() as archive:
            root, source, head, case, plan = self.fixture(directory)
            with contextlib.redirect_stdout(io.StringIO()):
                observed = one_case(case, plan, root, Path(archive), head)
            self.assertEqual(observed['outcome'], 'killed')
            self.assertEqual([observed[p]['exit'] for p in ['clean', 'broken', 'restored']], [0, 1, 0])
            self.assertEqual(observed['restored_git_status'], '')
            proof = Path(archive) / 'Mfixture'
            self.assertEqual((proof / 'source-clean.sha256').read_text(), (proof / 'source-restored.sha256').read_text())
            self.assertIn('-return', (proof / 'mutant.diff').read_text().replace('func Value() int { ', ''))
            self.assertEqual(json.loads((proof / 'PLAN.json').read_text())['head'], head)
            with self.assertRaises(ValueError):
                one_case(case, plan, root, Path(archive), head)
            self.assertIn('return 7', source.read_text())

    def test_exception_during_mutant_execution_restores_exact_bytes(self):
        with tempfile.TemporaryDirectory() as directory, tempfile.TemporaryDirectory() as archive:
            root, source, head, case, plan = self.fixture(directory)
            original = source.read_bytes()
            with patch('fidelity_mutations.run.capture', side_effect=[result(), RuntimeError('capture interrupted')]):
                with self.assertRaisesRegex(RuntimeError, 'capture interrupted'):
                    one_case(case, plan, root, Path(archive), head)
            self.assertEqual(source.read_bytes(), original)

    def test_report_rejects_changed_head_or_verdict_in_retained_evidence(self):
        with tempfile.TemporaryDirectory() as directory, tempfile.TemporaryDirectory() as archive:
            root, _, head, case, plan = self.fixture(directory)
            case.update({'file': 'pkg/fixture/value_test.go', 'line': 3, 'references': [{}],
                         'anchor': {'manifest': 'fixture.tsv', 'line': '2', 'case_id': 'fixture.value', 'c_site': 'fixture.c:1'}})
            evidence = Path(archive)
            (evidence / 'mutations').mkdir()
            with contextlib.redirect_stdout(io.StringIO()):
                one_case(case, plan, root, evidence / 'mutations', head)
            sample_file = evidence / 'sample.json'
            sample_file.write_text(json.dumps({'selected': [case], 'head': head, 'sample_size': 1,
                                               'frame_size': 1, 'created_utc': '2026-09-01T00:00:00+00:00'}))
            summary = write_report(sample_file, evidence, evidence / 'review')
            self.assertEqual(summary['outcomes'], {'killed': 1})
            self.assertIn('claim value = 6', (evidence / 'review/results.tsv').read_text())
            retained = evidence / 'mutations/Mfixture/RESULT.json'
            saved = retained.read_text()
            payload = json.loads(saved)
            payload['outcome'] = 'survived'
            retained.write_text(json.dumps(payload))
            with self.assertRaisesRegex(ValueError, 'classification differs'):
                write_report(sample_file, evidence, evidence / 'review')
            retained.write_text(saved)
            plan_file = evidence / 'mutations/Mfixture/PLAN.json'
            payload = json.loads(plan_file.read_text())
            payload['head'] = 'different'
            plan_file.write_text(json.dumps(payload))
            with self.assertRaisesRegex(ValueError, 'frame/head/restoration mismatch'):
                write_report(sample_file, evidence, evidence / 'review')

    def test_actual_go_panic_is_not_a_kill(self):
        with tempfile.TemporaryDirectory() as directory, tempfile.TemporaryDirectory() as archive:
            root, source, _, _, _ = self.fixture(directory)
            source.write_text('package fixture\nfunc Value() int { panic("claim value") }\n')
            broken = capture(['go', 'test', '-json', './pkg/fixture'], root, Path(archive), 'panic')
            self.assertEqual(classify(result(), broken, result(), 'TestClaim', 'claim value')[0], 'crash')


if __name__ == '__main__':
    unittest.main()
