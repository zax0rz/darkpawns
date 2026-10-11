#!/usr/bin/env python3
"""R5h compiling overlays for the real death/extraction boundary."""
import json
from pathlib import Path
import subprocess
import tempfile

checks=(
 ('pkg/game/char_mgmt.go','w.DieFollower(p)','// removed PC cleanup','TestDeathReleasesFollowers'),
 ('pkg/game/char_mgmt.go','w.DieFollowerMob(m)','// removed NPC cleanup','TestNPCDeathReleasesFollowersAtExtraction'),
 ('pkg/game/follow.go','actorFollowingSequence(asActor(followers[i])) > actorFollowingSequence(asActor(followers[j]))','actorFollowingSequence(asActor(followers[i])) < actorFollowingSequence(asActor(followers[j]))','TestDeathReleasesFollowers'),
 ('pkg/game/follow.go','\tstop(body)\n','\t// removed outgoing master cleanup\n','TestDeathFollowerOwnMasterAndRemoteFollower'),
 ('pkg/game/death.go','\tplayer.StopFighting()\n','\tplayer.StopFighting()\n\tw.DieFollower(player)\n','TestDeathReleasesFollowers'),
)
command=['go','test','./pkg/game','-run','^(TestDeath.*Follower|TestNPCDeathReleasesFollowersAtExtraction)$','-count=1']
def run(args, failure=False, name=''):
 r=subprocess.run(args,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
 print(r.stdout,end='')
 if failure:
  assert r.returncode==1 and '[build failed]' not in r.stdout
  assert '--- FAIL: '+name in r.stdout
 else: assert r.returncode==0
run(command)
for path,old,new,test in checks:
 source=Path(path).resolve();original=source.read_text();assert old in original
 with tempfile.TemporaryDirectory() as d:
  root=Path(d);mutation=root/source.name
  mutation.write_text(original.replace(old,new,1))
  overlay=root/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(mutation)}}))
  run(['go','test','-overlay='+str(overlay),'./pkg/game','-run','^'+test+'$','-count=1'],True,test)
 run(command)
