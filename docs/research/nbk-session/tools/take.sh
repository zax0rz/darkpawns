#!/bin/sh
# take.sh NAME : fresh boot, run multi.py with remaining args, stop server.
S=/tmp/claude-0/-home-user-darkpawns/950883f4-8a1e-543f-8181-0ecc0f69af13/scratchpad
cd $S/nbk
bin/circle 4300 > boot-$1.log 2>&1 &
echo $! > $S/circle.pid
sleep 4
shift
cd $S/tools
timeout 330 python3 multi.py 4300 "$@"
kill $(cat $S/circle.pid)
