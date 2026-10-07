import argparse,hashlib,json,os,pathlib,select,shutil,signal,socket,subprocess,tempfile,time
parser=argparse.ArgumentParser();parser.add_argument("--output",required=True);args=parser.parse_args()
assert os.getuid() != 0, 'permission proof requires non-root uid'
root=pathlib.Path(args.output).resolve();root.mkdir(parents=True,exist_ok=True)
binary=pathlib.Path('/home/zach/darkpawns-c-oracle/bin/circle')
(root/'reference.sha256').write_text(hashlib.sha256(binary.read_bytes()).hexdigest()+'\n')
lib=pathlib.Path(tempfile.mkdtemp(prefix='dp-olc-parent-C-'))/'lib'
shutil.copytree(binary.parent.parent/'lib',lib)
(lib/'etc/players').write_bytes(b'')
for directory in ['plrpoof','plralias','plrobjs','plrtext']:
 for bucket in ['A-E','F-J','K-O','P-T','U-Z','ZZZ']:(lib/directory/bucket).mkdir(parents=True,exist_ok=True)
sock=socket.socket();sock.bind(('127.0.0.1',0));port=sock.getsockname()[1];sock.close()
log=(root/'reference-process.log').open('wb')
process=subprocess.Popen([str(binary),'-d',str(lib),str(port)],cwd=binary.parent.parent,env={**os.environ,'DP_CLOCK':'1','DP_SEED':'1','DP_FIXED_TIME':'1760000000'},stdout=log,stderr=subprocess.STDOUT)
connection=None
transcript=[]
def capture():
 data=b'';limit=time.monotonic()+3
 while time.monotonic()<limit:
  ready,_,_=select.select([connection],[],[],0.3 if data else 2)
  if not ready:break
  b=connection.recv(65536)
  if not b:raise RuntimeError('C closed unexpectedly')
  data+=b
 return data

def command(line):
 connection.sendall((line+'\n').encode());data=capture();transcript.append((line,data.decode('latin1')));return data
try:
 deadline=time.monotonic()+30
 while True:
  if process.poll() is not None:raise RuntimeError('C failed to start')
  try:connection=socket.create_connection(('127.0.0.1',port),timeout=1);break
  except OSError:
   if time.monotonic()>deadline:raise
   time.sleep(0.1)
 transcript.append(('connect',capture().decode('latin1')))
 for line in ['Openactor','Y','oraclepass','oraclepass','N','M','H','W','K','Y','','1']:command(line)
 command('color off');command('syslog complete')
 cases=[('readable','remove'),('absent','absent'),('unreadable','unreadable'),('dangling','dangling'),('remove-error','remove-error'),('open-error','open-error')]
 for name,kind in cases:
  parent=lib/'scripts'/'room'/'fileproof'
  parent.mkdir(exist_ok=True)
  target=parent/'probe.lua'
  if kind=='dangling':target.symlink_to(parent/'missing.lua')
  elif kind!='absent':target.write_text('old line\n')
  command('luaedit room/fileproof probe')
  command('/c')
  if kind=='unreadable':target.chmod(0)
  if kind=='remove-error':parent.chmod(0o500)
  if kind=='open-error':
   command('replacement')
   target.unlink();parent.rmdir();parent.write_text('obstruction')
  try:
   data=command('/s')
   (root/('C-'+name+'.txt')).write_bytes(data)
   if kind in ['remove-error','open-error']:
    assert b'Deleted.' not in data and b'Saved.' not in data,data
   else:assert b'Deleted.' in data,data
   if kind in ['unreadable','dangling']:assert target.is_symlink() or target.exists()
   if kind=='readable':assert not target.exists()
  finally:
   if kind=='open-error':parent.unlink();parent.mkdir()
   parent.chmod(0o700)
   if target.is_symlink():target.unlink()
   elif target.exists():target.chmod(0o600);target.unlink()
 (lib/'scripts/mob/fileproof').mkdir(exist_ok=True)
 for name,cmd,logical in [('raw-dir-save','luaedit mob//fileproof proofraw','scripts/mob//fileproof/proofraw.lua'),('root-save','luaedit proofroot','scripts/proofroot.lua'),('subdir-save','luaedit mob proofsub','scripts/mob/proofsub.lua'),('help-save','tedit help','text/help/screen')]:
  command(cmd);command('/c');command('new line');data=command('/s')
  assert b'Saved.' in data,data
  assert (lib/logical).read_bytes()==b'new line\n',logical
  assert ("OLC: Openactor saves '"+logical+"'.").encode() in (root/'reference-process.log').read_bytes(),logical
  (root/('C-'+name+'.txt')).write_bytes(data)
 for payload in ["SYSERR: Can't delete file 'scripts/room/fileproof/probe.lua'.", "SYSERR: Can't write file 'scripts/room/fileproof/probe.lua'."]:
  assert payload.encode() in (root/'reference-process.log').read_bytes(),payload
 print('Reference C: readable/absent/unreadable/dangling/delete refusal/open-parent PASS',flush=True)
finally:
 (root/'reference-transcript.json').write_text(json.dumps(transcript,indent=2))
 if connection:connection.close()
 process.send_signal(signal.SIGINT)
 try:process.wait(timeout=3)
 except subprocess.TimeoutExpired:process.kill();process.wait()
 log.close();shutil.rmtree(lib.parent)
