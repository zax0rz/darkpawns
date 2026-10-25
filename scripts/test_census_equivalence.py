from pathlib import Path
import json
import shutil
import tempfile
import unittest

from check_census_equivalence import compare


class EquivalenceTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.a=Path(self.temp.name)/'a';self.a.mkdir()
        for name,text in [('exit-code','0'),('MANIFEST.md','retained'),('summary.txt','verdict=CLEAN'),('results.tsv','PASS\talpha\t1\n'),('census.log','PASS alpha\n')]:
            (self.a/name).write_text(text)
        self.cap=self.a/'initial-logs/attempt.x/alpha.capture1';self.cap.mkdir(parents=True)
        (self.cap/'blocks.json').write_text(json.dumps([dict(Command='look',Oracle='Room\n',Go='Room\n',Diff='')]))
        for side in ('c','go'):
            (self.cap/f'{side}-draws.log').write_text('DRAW 0 1 Number(1,6) = 3\n')
            (self.cap/f'{side}-server.log').write_text('ready\n')
        (self.cap.parent/'alpha.attempt1.log').write_text('draw-parity\tMATCH\tentries=1\tstream=1\n')
        self.b=Path(self.temp.name)/'b';shutil.copytree(self.a,self.b)

    def test_equal_evidence_and_changed_transcript(self):
        self.assertEqual(len(compare([self.a,self.b])),2)
        p=self.b/'initial-logs/attempt.x/alpha.capture1/blocks.json'
        rows=json.loads(p.read_text());rows[0]['Go']='missing response\n';p.write_text(json.dumps(rows))
        with self.assertRaisesRegex(ValueError,'differences'):compare([self.a,self.b])

    def test_draw_change_retry_and_foreign_listener_are_red(self):
        p=self.b/'initial-logs/attempt.x/alpha.capture1/go-draws.log'
        p.write_text('DRAW 0 1 Number(1,6) = 4\n')
        with self.assertRaisesRegex(ValueError,'differences'):compare([self.a,self.b])
        shutil.copy2(self.cap/'go-draws.log',p)
        (self.b/'census.log').write_text('RETRY alpha\nPASS alpha\n')
        with self.assertRaisesRegex(ValueError,'retries'):compare([self.a,self.b])
        (self.b/'census.log').write_text('PASS alpha\n')
        (p.parent/'go-server.log').write_text('Telnet listener failed: bind address already in use\n')
        with self.assertRaisesRegex(ValueError,'listener failure'):compare([self.a,self.b])
