import argparse,hashlib,json,os,pathlib,select,shutil,signal,socket,subprocess,tempfile,time
parser=argparse.ArgumentParser();parser.add_argument("--output",required=True);args=parser.parse_args()
root=pathlib.Path(args.output).resolve();root.mkdir(parents=True,exist_ok=True)
binary=pathlib.Path('/home/zach/darkpawns-c-oracle/bin/circle')
(root/'reference.sha256').write_text(hashlib.sha256(binary.read_bytes()).hexdigest()+'\n')
lib=pathlib.Path(tempfile.mkdtemp(prefix='dp-olc-parent-C-'))/'lib'
shutil.copytree(binary.parent.parent/'lib',lib)
(lib/'etc/players').write_bytes(b'')
for directory in ['plrpoof','plralias','plrobjs','plrtext']:
 for bucket in ['A-E','F-J','K-O','P-T','U-Z','ZZZ']:(lib/directory/bucket).mkdir(parents=True,exist_ok=True)
zone=lib/'world/zon/10.zon'
lines=zone.read_text().splitlines()
zone.write_text('\n'.join(lines[:3]+['G 0 1 0','X 1 0 0','Z 1 0 0','S','$'])+'\n')
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
 command('color off');command('syslog brief')
 data=command('zedit save 10')
 for op in ['X','Z']:
  payload=("SYSERR: OLC: z_save_to_disk(): Unknown cmd '%s' - NOT saving" % op).encode()
  assert data.count(payload)==1,(op,data)
 (root/'C-unknown-success.txt').write_bytes(data)
 saved=zone.read_text();(root/'C-saved-zone.txt').write_text(saved)
 assert not any(line.startswith(('X ','Z ')) for line in saved.splitlines()),saved
 parent=lib/'world/zon';backup=parent.with_name('zon-retained');parent.rename(backup)
 try:
  parent.write_text('parent obstruction\n')
  data=command('zedit save 10')
  assert b"Can't write zone 10." in data and b'Unknown cmd' not in data,data
  (root/'C-unknown-open-failure.txt').write_bytes(data)
 finally:
  parent.unlink();backup.rename(parent)
 print('Reference C: retained unknown opcodes, successful skip and open-first silence PASS',flush=True)

finally:
 (root/'reference-transcript.json').write_text(json.dumps(transcript,indent=2))
 if connection:connection.close()
 process.send_signal(signal.SIGINT)
 try:process.wait(timeout=3)
 except subprocess.TimeoutExpired:process.kill();process.wait()
 log.close();shutil.rmtree(lib.parent)
