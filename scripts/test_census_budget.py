#!/usr/bin/env python3
"""Admission and lifecycle regression tests; no oracle required (R5h)."""
from pathlib import Path
import hashlib
import fcntl
import json
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

import census_budget as budget

SCRIPTS = Path(__file__).resolve().parent


class BudgetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name)
        self.memory = self.base/'memory'
        self.memory.write_text(str(32*1024*1024))
        self.env = patch.dict(os.environ, CENSUS_TESTING='1', CENSUS_TEST_RUNTIME=str(self.base/'runtime'), CENSUS_TEST_MEMORY=str(self.memory))
        self.env.start()
        self.oracle = self.base/'oracle'
        self.oracle.write_text('stub')
        self.runs = []

    def tearDown(self):
        self.env.stop()
        self.temp.cleanup()

    def register(self, name, **extra):
        run = self.base/name
        run.mkdir()
        record = dict(status='queued',pid=os.getpid(),birth=budget.identity(os.getpid()),jobs=36,release=False,exclusive=False,submitted=time.time(),oracle=str(self.oracle),oracle_sha=hashlib.sha256(self.oracle.read_bytes()).hexdigest(),started='test') | extra
        with budget.transaction() as state:
            state['runs'][str(run)] = record
        return str(run)

    def lease(self, run, kind='worker'):
        return dict(run=run,kind=kind,pid=os.getpid(),birth=budget.identity(os.getpid()),pgid=0,granted=time.time()-3)

    def test_host_and_per_run_caps_and_fairness(self):
        a,b,c = [self.register(x) for x in ('a','b','c')]
        with budget.transaction() as state:
            state['leases'] = {str(i):self.lease(a if i<36 else b) for i in range(72)}
            self.assertFalse(budget.eligible(state,dict(run=c,kind='worker')))
            state['leases'].pop('71')
            self.assertFalse(budget.eligible(state,dict(run=a,kind='worker')))
            self.assertTrue(budget.eligible(state,dict(run=c,kind='worker')))
            state['waiting'] = [dict(run=b,kind='worker',token='b'),dict(run=c,kind='worker',token='c')]
            state['cursor'] = b
            self.assertEqual(budget.selected(state),'c')

    def test_memory_hysteresis_and_build_reserve(self):
        a = self.register('a')
        with budget.transaction() as state:
            req = dict(run=a,kind='worker')
            self.assertEqual(budget.FLOOR_KIB,4*1024*1024)
            self.memory.write_text(str(3*1024*1024))
            self.assertFalse(budget.eligible(state,req))
            self.memory.write_text(str(budget.FLOOR_KIB+1))
            self.assertFalse(budget.eligible(state,req))
            self.memory.write_text(str(budget.RESUME_KIB))
            self.assertTrue(budget.eligible(state,req))
            self.assertFalse(budget.eligible(state,dict(run=a,kind='build')))
            self.memory.write_text(str(budget.FLOOR_KIB+budget.BUILD_RESERVE_KIB+36*16*1024))
            self.assertTrue(budget.eligible(state,dict(run=a,kind='build')))
            state['leases']['build'] = self.lease(a,'build')
            self.assertFalse(budget.eligible(state,dict(run=a,kind='build')))

    def test_release_drains_without_preempting_and_fifo(self):
        a = self.register('ordinary')
        r = self.register('release',release=True)
        later = self.register('later-release',release=True)
        with budget.transaction() as state:
            state['leases']['active'] = self.lease(a)
            self.assertFalse(budget.eligible(state,dict(run=a,kind='worker')))
            self.assertFalse(budget.eligible(state,dict(run=r,kind='build')))
            self.assertIn('active',state['leases'])
            state['leases'].clear()
            self.assertTrue(budget.eligible(state,dict(run=r,kind='build')))
            self.assertFalse(budget.eligible(state,dict(run=later,kind='worker')))

    def test_exclusive_drain_and_release_priority(self):
        a=self.register('ordinary');x=self.register('exclusive',exclusive=True)
        with budget.transaction() as state:
            state['leases']['active']=self.lease(a)
            self.assertFalse(budget.eligible(state,dict(run=a,kind='worker')))
            self.assertFalse(budget.eligible(state,dict(run=x,kind='build')))
            state['leases'].clear()
            self.assertTrue(budget.eligible(state,dict(run=x,kind='build')))
        r=self.register('release',release=True)
        with budget.transaction() as state:
            self.assertEqual(budget.release_owner(state),r)

    def test_crashed_client_keeps_permit_until_children_die(self):
        run = self.register('orphan')
        child = subprocess.Popen(['sleep','30'],start_new_session=True)
        self.addCleanup(lambda: budget.signal_group(child.pid,signal.SIGKILL))
        self.addCleanup(child.wait)
        with budget.transaction() as state:
            state['leases']['orphan'] = self.lease(run) | dict(pid=99999999,birth='dead',pgid=child.pid)
        with budget.transaction() as state:
            self.assertIn('orphan',state['leases'])
        child.wait(timeout=3)
        with budget.transaction() as state:
            self.assertNotIn('orphan',state['leases'])


class CLITests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name)
        self.repo = self.base/'repo'
        (self.repo/'scripts').mkdir(parents=True)
        for p in SCRIPTS.glob('*.py'):
            shutil.copy2(p,self.repo/'scripts'/p.name)
        for name in ('census.sh','census_runner.sh','census_build.sh','oracle_regression.sh','oracle_regression_worker.sh'):
            shutil.copy2(SCRIPTS/name,self.repo/'scripts'/name)
        cases = self.repo/'cmd/dp-oracle-diff/scenarios'
        cases.mkdir(parents=True)
        for name in ('alpha','beta'):
            (cases/f'{name}.txt').write_text('[probe]\nlook\n')
        oracle = self.base/'circle'
        oracle.write_text('#!/bin/sh\nexit 0\n')
        oracle.chmod(0o755)
        (cases.parent/'reference-oracle.sha256').write_text(hashlib.sha256(oracle.read_bytes()).hexdigest())
        (cases.parent/'expected_divergences.tsv').write_text('')
        (cases.parent/'expected_divergence_pins.tsv').write_text('')
        self.memory = self.base/'memory'
        self.memory.write_text(str(32*1024*1024))
        self.fakego = self.base/'go'
        # The real full/claims/combined drivers, build gate and classification
        # worker execute; only compilation/engines are replaced in this fixture.
        self.fakego.write_text('''#!/usr/bin/env python3
import sys,pathlib
if '-C' in sys.argv and sys.argv[1]!='-C':sys.exit('Go -C must be first')
p=pathlib.Path(sys.argv[sys.argv.index('-o')+1])
p.write_text("""#!/usr/bin/env python3
import os,time,pathlib,sys
hold=os.environ.get('STUB_HOLD')
while hold and pathlib.Path(hold).exists():time.sleep(.02)
mode=os.environ.get('STUB_MODE','pass')
if mode=='infra' and os.environ.get('ORACLE_REGRESSION_JOBS')!='1':
 print('listener ownership: synthetic foreign listener');sys.exit(1)
if mode=='fail':
 print('divergence-fingerprint\\\\tprobe\\\\t'+'a'*64)
 print('result: normalized divergence detected');sys.exit(3)
print('result: no normalized divergence')
if '--draw-parity' in sys.argv:print('draw-parity\\\\tMATCH\\\\tentries=1\\\\tstream=1')
""")
p.chmod(0o755)
''')
        self.fakego.chmod(0o755)
        self.env = dict(os.environ,CENSUS_TESTING='1',CENSUS_TEST_RUNTIME=str(self.base/'rt'),CENSUS_TEST_MEMORY=str(self.memory),
                        ORACLE_RUNS_ROOT=str(self.base/'runs'),DP_ORACLE_BIN=str(oracle),ORACLE_REGRESSION_GO=str(self.fakego),CENSUS_POLL_SECONDS='.02')
        self.env.pop('CENSUS_RUNNER',None)
        subprocess.run(['git','init','-q',str(self.repo)],check=True)
        subprocess.run(['git','-C',str(self.repo),'add','.'],check=True)
        subprocess.run(['git','-C',str(self.repo),'-c','user.email=test@example.org','-c','user.name=Test','commit','-qm','fixture'],check=True)
        self.runs=[]

    def tearDown(self):
        for run in self.runs:
            self.cli('cancel','--run',str(run))
        # Let cancellation reap detached fixture groups before removing artifacts.
        for run in self.runs:
            for _ in range(100):
                rec=json.loads((run/'admission.json').read_text())
                if not budget.alive(rec['pid'],rec['birth']):break
                time.sleep(.01)
        self.temp.cleanup()

    def cli(self,*args,env=None):
        return subprocess.run([str(self.repo/'scripts/census.sh'),*args],env=env or self.env,text=True,capture_output=True)

    def start(self,name,*args,env=None):
        result=self.cli('start','--name',name,*args,env=env)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        run=next((self.base/'runs').glob('*/'+name))
        self.runs.append(run)
        return run

    def wait(self,run,code=0):
        out=self.cli('wait','--run',str(run),'--max-seconds','15')
        self.assertEqual(out.returncode,code,out.stdout+out.stderr+(run/'census.log').read_text())
        self.assertEqual(len(out.stdout.splitlines()),1,out.stdout)
        if code in (0,1):self.assertTrue((run/'MANIFEST.md').exists())
        return out

    def test_real_dispatch_full_targeted_claims_and_combined(self):
        manifests=self.base/'manifests';manifests.mkdir()
        (manifests/'test.tsv').write_text('handler\tcommand\tcase_id\tdepth\tscope\tstatus\tproof\tc_site\tnotes\nh\tc\ta\tD1\ta\toracle-green-multiseed\talpha@1,2\tsrc/x:1\t\n')
        env=self.env|dict(CENSUS_MANIFEST_DIR=str(manifests))
        for name,flags in [('full',[]),('target',['--scenarios','alpha']),('claims',['--claims']),('combined',['--combined'])]:
            run=self.start(name,*flags,'--draw-parity',env=env)
            self.wait(run)
            events=[json.loads(l) for l in (run/'admission-events.jsonl').read_text().splitlines()]
            self.assertTrue(any(e['event']=='granted' and e.get('kind')=='worker' for e in events))
            self.assertTrue(any(e['event']=='granted' and e.get('kind')=='build' for e in events))
            self.assertIn('Reference oracle: yes',(run/'MANIFEST.md').read_text())

    def test_queued_snapshot_noqueue_and_cancel(self):
        self.memory.write_text('0')
        run=self.start('queued','--scenarios','alpha')
        self.assertIn('queued',self.cli('status','--run',str(run)).stdout)
        self.assertEqual(self.cli('start','--name','refused','--no-queue').returncode,5)
        original=(run/'source/cmd/dp-oracle-diff/scenarios/alpha.txt').read_text()
        (self.repo/'cmd/dp-oracle-diff/scenarios/alpha.txt').write_text('changed after registration\n')
        self.assertEqual((run/'source/cmd/dp-oracle-diff/scenarios/alpha.txt').read_text(),original)
        self.assertEqual(self.cli('cancel','--run',str(run)).returncode,0)
        self.wait(run,4)

    def test_parallel_identity_and_duplicate_names(self):
        hold=self.base/'hold';hold.touch()
        env=self.env|dict(STUB_HOLD=str(hold))
        a=self.start('a',env=env);b=self.start('b',env=env)
        self.assertEqual(self.cli('wait','--max-seconds','0').returncode,4)
        self.assertIn('--run',self.cli('status').stderr)
        self.assertEqual(self.cli('start','--name','a').returncode,2)
        self.assertEqual(self.cli('wait','--run',str(a),'--name','wrong').returncode,4)
        hold.unlink()
        self.wait(a);self.wait(b)
        self.assertTrue((a/'source-hashes.json').exists())

    def test_registration_queues_behind_legacy_exclusive_guard(self):
        rt=self.base/'rt';rt.mkdir(mode=0o700)
        with (rt/'legacy.lock').open('a+') as old:
            fcntl.flock(old,fcntl.LOCK_EX)
            run=self.start('legacy-queued')
            self.assertEqual(self.cli('start','--name','noqueue','--no-queue').returncode,5)
            self.assertNotIn('"event": "granted"',(run/'admission-events.jsonl').read_text())
        self.wait(run)

    def test_reference_and_job_cap(self):
        self.assertEqual(self.cli('start','--name','oversized','--jobs','37').returncode,2)
        self.assertEqual(self.cli('start','--name','release-invalid','--release').returncode,2)
        (self.repo/'cmd/dp-oracle-diff/reference-oracle.sha256').write_text('0'*64)
        self.assertEqual(self.cli('start','--name','bad-reference').returncode,6)

    def test_recheck_uses_budget_and_fail_not_rechecked(self):
        run=self.start('infra','--jobs','2',env=self.env|dict(STUB_MODE='infra'))
        self.assertIn('CLEAN_AFTER_RECHECK',self.wait(run).stdout)
        self.assertEqual(len((run/'recheck-results.tsv').read_text().splitlines()),2)
        events=[json.loads(l) for l in (run/'admission-events.jsonl').read_text().splitlines()]
        self.assertEqual(sum(e['event']=='granted' and e.get('kind')=='worker' for e in events),4)
        failed=self.start('fail',env=self.env|dict(STUB_MODE='fail'))
        self.wait(failed,1)
        self.assertFalse((failed/'recheck-results.tsv').exists())

    def test_release_drains_then_blocks_later_ordinary(self):
        hold=self.base/'hold';hold.touch()
        a=self.start('ordinary',env=self.env|dict(STUB_HOLD=str(hold)))
        for _ in range(300):
            data=json.loads((self.base/'rt/registry.json').read_text())
            if any(l['kind']=='worker' for l in data['leases'].values()):break
            time.sleep(.01)
        manifests=self.base/'claims';manifests.mkdir()
        (manifests/'x.tsv').write_text('handler\tcommand\tcase_id\tdepth\tscope\tstatus\tproof\tc_site\tnotes\nh\tc\tx\tD1\ta\toracle-green\talpha@1\tsrc/x:1\t\n')
        r=self.start('release','--release','--combined',env=self.env|dict(CENSUS_MANIFEST_DIR=str(manifests)))
        b=self.start('later')
        time.sleep(.1)
        self.assertNotIn('"event": "granted"',(b/'admission-events.jsonl').read_text())
        self.assertNotIn('"event": "granted"',(r/'admission-events.jsonl').read_text())
        hold.unlink();self.wait(a);self.wait(r);self.wait(b)
        events=lambda path:[json.loads(l) for l in (path/'admission-events.jsonl').read_text().splitlines()]
        finished=next(e['time'] for e in events(r) if e['event']=='finished')
        self.assertTrue(all(e['time']>=finished for e in events(b) if e['event']=='granted'))

    def test_runner_sigkill_reaps_leases(self):
        hold=self.base/'hold';hold.touch()
        run=self.start('kill',env=self.env|dict(STUB_HOLD=str(hold)))
        for _ in range(300):
            data=json.loads((self.base/'rt/registry.json').read_text())
            leases=[l for l in data['leases'].values() if l.get('pgid')]
            if leases:break
            time.sleep(.01)
        else:self.fail('no live lease')
        rec=json.loads((run/'admission.json').read_text())
        os.kill(rec['pid'],signal.SIGKILL)
        self.wait(run,4)
        for _ in range(300):
            self.cli('status','--run',str(run))
            if all(not budget.group_alive(l['pgid']) for l in leases):break
            time.sleep(.01)
        else:self.fail('engine group survived runner death')


if __name__ == '__main__':
    unittest.main()
