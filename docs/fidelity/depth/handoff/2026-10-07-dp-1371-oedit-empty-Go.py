#!/usr/bin/env python3
"""Retain a compiling assertion-red reproduction; never installs a failing test."""
import argparse,json,os,pathlib,subprocess
p=argparse.ArgumentParser();p.add_argument('--output',required=True);a=p.parse_args()
root=pathlib.Path(a.output).expanduser().resolve();root.mkdir(parents=True,exist_ok=True)
(root/'HEAD.txt').write_bytes(subprocess.check_output(['git','rev-parse','HEAD']))
source=root/'oedit-empty-repro_test.go'
source.write_text('''package session
import("os";"path/filepath";"strings";"testing")
func TestOeditAllocatedEmptyReferenceReproduction(t *testing.T) {
 w:=makeOeditTestWorld(t);m:=newTestManager(t,w,nil)
 s:=makeOeditTestSession(t,m,"Emptygod",40);openOedit(t,s,"3001")
 for _,line:=range []string{"f","1","allocated","2","single line","/d1","/s","0","q","y"} {
  s.handleOeditInput(line);drainSessionText(t,s)
 }
 obj,ok:=w.SnapshotObj(3001)
 if !ok || len(obj.ExtraDescs)!=1 || obj.ExtraDescs[0].Keywords!="allocated" || obj.ExtraDescs[0].Description!="" {t.Fatalf("not the intended allocated-empty working copy: %#v",obj)}
 zone,ok:=w.SnapshotZone(30);if !ok {t.Fatal("missing zone")}
 if err:=saveOeditZone(w,&zone);err!=nil {t.Fatal(err)}
 data,err:=os.ReadFile(filepath.Join(w.GetParsedWorld().SourceDir,"obj","30.obj"));if err!=nil {t.Fatal(err)}
 if !strings.Contains(string(data),"E\\nallocated~\\n~\\n") {t.Fatal("Go drops the valid allocated-empty C description during object save")}
}
''')
overlay=root/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(pathlib.Path('pkg/session/oedit_empty_repro_test.go').resolve()):str(source)}}))
r=subprocess.run(['go','test','-p','2','-overlay='+str(overlay),'./pkg/session','-run','^TestOeditAllocatedEmptyReferenceReproduction$','-count=1'],env=dict(os.environ,GOMAXPROCS='2'),text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT)
(root/'reproduction.txt').write_text(r.stdout+'\nEXIT='+str(r.returncode)+'\n')
assert r.returncode!=0 and '[build failed]' not in r.stdout and '--- FAIL: TestOeditAllocatedEmptyReferenceReproduction' in r.stdout and 'Go drops the valid allocated-empty' in r.stdout,r.stdout
print('Expected compiling Go assertion red: allocated-empty description lost on save',flush=True)
