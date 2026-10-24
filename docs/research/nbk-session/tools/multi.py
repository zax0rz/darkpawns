"""Run several mud.py sessions concurrently: multi.py port delay:steps.json:out.log ..."""
import sys, threading, time, json
import importlib.util
spec=importlib.util.spec_from_file_location('mud', sys.path[0]+'/mud.py'); mud=importlib.util.module_from_spec(spec); spec.loader.exec_module(mud)
port=int(sys.argv[1]); ths=[]
for a in sys.argv[2:]:
    d,st,out=a.split(':')
    def go(d=float(d),st=st,out=out):
        time.sleep(d); mud.run(port, json.load(open(st)), out)
    t=threading.Thread(target=go); t.start(); ths.append(t)
for t in ths: t.join()
