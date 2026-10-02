#!/usr/bin/env python3
"""Prove actor-before-passive-peer capture with an assertion-only 0/1/0 triple."""
from pathlib import Path
import subprocess
import sys
repo=Path(sys.argv[1]).resolve(); logs=Path(sys.argv[2]).resolve();logs.mkdir(parents=True,exist_ok=True)
path=repo/'internal/oraclediff/scenario.go'; original=path.read_text()
mutant=original.replace('var output string\n\t\ttarget,', 'var output string\n\t\tvar earlyPeers []AudienceProbeBlock\n\t\ttarget,',1)
mutant=mutant.replace('output, err = target.ReadUntilQuiescent(quiescence)', 'earlyPeers, err = readAudiencePeers(audience, step, i, quiescence, mayClose)\n\t\t\tif err != nil { return blocks, err }\n\t\t\toutput, err = target.ReadUntilQuiescent(quiescence)',1)
mutant=mutant.replace('peerBlocks, peerErr := readAudiencePeers(audience, step, i, quiescence, mayClose)', 'peerBlocks, peerErr := earlyPeers, error(nil)',1)
assert mutant!=original
statuses=[]
try:
 for stage in ['baseline','reverted','restored']:
  path.write_text(mutant if stage=='reverted' else original)
  p=subprocess.run(['go','test','./internal/oraclediff','-run','^TestSlowIssuingConnectionCapturesPeerAfterActor$','-count=1'],cwd=repo,text=True,capture_output=True)
  text=p.stdout+p.stderr;(logs/(stage+'.log')).write_text(text);statuses.append(p.returncode)
  if stage=='reverted':assert '--- FAIL:' in text and 'capture lost output' in text and 'build failed' not in text,text
 assert statuses==[0,1,0],statuses
finally:path.write_text(original)
(logs/'revert-triples.tsv').write_text('mutation\ttest\tbefore\treverted\trestored\npeer-before-actor\tTestSlowIssuingConnectionCapturesPeerAfterActor\t0\t1\t0\n')
print('slow-command peer output: assertion-only 0/1/0 passed (primary and send:peer)')
