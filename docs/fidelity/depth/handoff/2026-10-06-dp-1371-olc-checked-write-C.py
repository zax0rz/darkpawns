import argparse,hashlib,json,os,pathlib,select,shutil,signal,socket,subprocess,tempfile,time
parser=argparse.ArgumentParser();parser.add_argument("--output",required=True);args=parser.parse_args()
root=pathlib.Path(args.output).resolve();root.mkdir(parents=True,exist_ok=True)
binary=pathlib.Path('/home/zach/darkpawns-c-oracle/bin/circle')
(root/'checked-write-reference.sha256').write_text(hashlib.sha256(binary.read_bytes()).hexdigest()+'\n')
lib=pathlib.Path(tempfile.mkdtemp(prefix='dp-olc-parent-C-'))/'lib'
shutil.copytree(binary.parent.parent/'lib',lib)
(lib/'etc/players').write_bytes(b'')
for directory in ['plrpoof','plralias','plrobjs','plrtext']:
 for bucket in ['A-E','F-J','K-O','P-T','U-Z','ZZZ']:(lib/directory/bucket).mkdir(parents=True,exist_ok=True)
sock=socket.socket();sock.bind(('127.0.0.1',0));port=sock.getsockname()[1];sock.close()
log=(root/'checked-write-process.log').open('wb')
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
 results=[]
 for commandName,extension in [('medit','mob'),('sedit','shp')]:
  target=lib/'world'/extension/('10.'+extension);backup=target.with_name(target.name+'-retained');target.rename(backup)
  try:
   target.symlink_to('/dev/full')
   data=command(commandName+' save 10')
   (root/('C-'+commandName+'-dev-full.txt')).write_bytes(data)
   results.append({'command':commandName,'device':'/dev/full','checked_write_diagnostic_seen':b'Cannot write' in data,'output':data.decode('latin1')})
  finally:
   target.unlink();backup.rename(target)
 (root/'checked-write-experiment.json').write_text(json.dumps(results,indent=2))
 print(json.dumps(results),flush=True)

finally:
 (root/'checked-write-transcript.json').write_text(json.dumps(transcript,indent=2))
 if connection:connection.close()
 process.send_signal(signal.SIGINT)
 try:process.wait(timeout=3)
 except subprocess.TimeoutExpired:process.kill();process.wait()
 log.close();shutil.rmtree(lib.parent)
