from pathlib import Path
import subprocess,sys
repo=Path(sys.argv[1]).resolve(); logs=Path(sys.argv[2]).resolve();logs.mkdir(parents=True,exist_ok=True)
cases=[('tcp-cpu','internal/oraclediff/conn.go','firstWait = max(d, c.firstByteWait)','firstWait = d','TestTCPFirstByteWaitUnderCPULoad'),('ws-cpu-echo','internal/oraclediff/wsconn.go','firstWait = max(d, c.firstByteWait)','firstWait = d','TestWSFirstByteWaitUnderCPULoadIgnoresLocalEcho'),('trailing-silence','internal/oraclediff/conn.go','deadline = time.Now().Add(d)','deadline = time.Now().Add(max(d, c.firstByteWait))','TestTCPFirstByteWaitKeepsTrailingSilence')]
rows=[]
for name,file,before,after,test in cases:
 p=repo/file; original=p.read_text(); assert before in original
 statuses=[]
 try:
  for stage in ['baseline','reverted','restored']:
   p.write_text(original.replace(before,after) if stage=='reverted' else original)
   result=subprocess.run(['go','test','./internal/oraclediff','-run','^'+test+'$','-count=1'],cwd=repo,text=True,capture_output=True)
   text=result.stdout+result.stderr;(logs/(name+'-'+stage+'.log')).write_text(text);statuses.append(result.returncode)
   if stage=='reverted':assert '--- FAIL:' in text and 'lost' in text if name!='trailing-silence' else 'extended trailing silence' in text
  assert statuses==[0,1,0],(name,statuses)
 finally:p.write_text(original)
 rows.append(name+'\t'+test+'\t0\t1\t0\n')
(logs/'revert-triples.tsv').write_text('mutation\ttest\tbefore\treverted\trestored\n'+''.join(rows));print('3 first-byte assertion-only triples passed')
