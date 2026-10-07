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
import re
obj=lib/'world/obj/10.obj'
original=obj.read_text()
match=re.search(r'^#(\d+)',original,re.M);assert match
vnum=int(match[1])
next_match=re.search(r'^#\d+|^\$~?',original[match.end():],re.M);assert next_match
end=match.end()+next_match.start()
obj.write_text(original[:end]+'E\n~\nBroken description~\n'+original[end:])
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
 data=command('oedit save 10')
 payload=b'SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!'
 assert data.count(payload)==1,data
 (root/'C-loaded-null.txt').write_bytes(data)
 for line in ['oedit '+str(vnum),'f','1','allocated','2','/c','single line','/d1','/s','0','q','y']:
  command(line)
 data=command('oedit save 10')
 assert payload not in data,data
 (root/'C-allocated-empty.txt').write_bytes(data)
 saved=obj.read_text();(root/'C-saved-object.txt').write_text(saved)
 assert 'E\nallocated~\n~\n' in saved,saved
 print('Reference C: loaded NULL logs; real-editor allocated empty remains valid and saved PASS',flush=True)

finally:
 (root/'reference-transcript.json').write_text(json.dumps(transcript,indent=2))
 if connection:connection.close()
 process.send_signal(signal.SIGINT)
 try:process.wait(timeout=3)
 except subprocess.TimeoutExpired:process.kill();process.wait()
 log.close();shutil.rmtree(lib.parent)
