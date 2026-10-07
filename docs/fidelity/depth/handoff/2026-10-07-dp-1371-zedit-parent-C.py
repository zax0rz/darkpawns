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
 cases=[('zedit','zon','zone','zone information')]
 for commandName,extension,noun,announcement in cases:
  parent=lib/'world'/extension;backup=parent.with_name(extension+'-retained');parent.rename(backup)
  try:
   parent.write_text('parent obstruction\n')
   data=command(commandName+' save 10')
   payload=b"SYSERR: OLC: zedit_save_to_disk:  Can't write zone 10."
   assert data.count(payload)==1,(commandName,data)
   assert data.index(b'Saving all zone information.')<data.index(payload),(commandName,data)
   (root/('C-'+commandName+'-parent.txt')).write_bytes(data)
  finally:
   if parent.exists():parent.unlink()
   backup.rename(parent)
 print('Reference C: ZEDIT non-directory parent PASS',flush=True)
finally:
 (root/'reference-transcript.json').write_text(json.dumps(transcript,indent=2))
 if connection:connection.close()
 process.send_signal(signal.SIGINT)
 try:process.wait(timeout=3)
 except subprocess.TimeoutExpired:process.kill();process.wait()
 log.close();shutil.rmtree(lib.parent)
