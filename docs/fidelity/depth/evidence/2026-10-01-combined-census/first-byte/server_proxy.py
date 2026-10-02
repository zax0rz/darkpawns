#!/usr/bin/env python3
"""Disposable proof fixture: contend for the Go server's CPU only at zreset *."""
import os,signal,socket,subprocess,sys,threading,time
args=sys.argv[1:]; index=args.index('-telnet-port')+1; front=int(args[index])
reserve=socket.socket();reserve.bind(('127.0.0.1',0));back=reserve.getsockname()[1];reserve.close();args[index]=str(back)
core=min(os.sched_getaffinity(0)); burners=[]
server=subprocess.Popen(['taskset','-c',str(core),'nice','-n','19',os.environ['CPU_PROOF_SERVER_REAL'],*args])
listener=socket.socket();listener.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1)
def cleanup(*_):
 for p in burners:
  if p.poll() is None:p.terminate()
 if server.poll() is None:server.terminate()
 try:listener.close()
 except OSError:pass
 raise SystemExit(0)
signal.signal(signal.SIGTERM,cleanup);signal.signal(signal.SIGINT,cleanup)
while True:
 try:
  check=socket.create_connection(('127.0.0.1',back),timeout=.1);check.close();break
 except OSError:
  if server.poll() is not None:raise SystemExit(server.returncode)
  time.sleep(.01)
listener.bind(('127.0.0.1',front));listener.listen()
def load():
 for _ in range(2):
  p=subprocess.Popen(['taskset','-c',str(core),sys.executable,'-c',"import time; print('ready',flush=True); end=time.monotonic()+.95\nwhile time.monotonic()<end: pass"],stdout=subprocess.PIPE,text=True)
  burners.append(p);assert p.stdout.readline().strip()=='ready'
 print('CPU-PROOF: 2 busy processes, same CPU as Go, 950ms contention before forwarding zreset *',flush=True)
def bridge(client):
 backend=socket.create_connection(('127.0.0.1',back))
 def responses():
  try:
   while data:=backend.recv(65536):client.sendall(data)
  except OSError:pass
  finally:
   try:client.shutdown(socket.SHUT_WR)
   except OSError:pass
 threading.Thread(target=responses,daemon=True).start()
 pending=b''
 try:
  while data:=client.recv(65536):
   pending+=data
   while b'\n' in pending:
    line,pending=pending.split(b'\n',1)
    if line.strip()==b'zreset *':load()
    backend.sendall(line+b'\n')
 except OSError:pass
 finally:client.close();backend.close()
while True:
 client,_=listener.accept();threading.Thread(target=bridge,args=(client,),daemon=True).start()
