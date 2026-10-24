# usage: stage2.sh <wayne-room> <mickey-room> <mallory-room> <witness-room|none> out.json
W="$4"
set -- "$1" "$2" "$3" "$4" "$5"
extra=""
if [ "$W" != none ]; then
  extra="\"goto scagnetti:0.5\" \"cast 'invisibility' scagnetti:1\" \"goto $W:0.5\" \"transfer scagnetti:0.5\""
fi
eval python3 login.py Mcclusky imppass1 $5 '"goto wayne:1"' '"restore wayne:0.5"' '"load obj 18222:0.3"' '"load obj 8022:0.3"' '"load obj 179:0.3"' '"load obj 176:0.3"' '"load obj 178:0.3"' '"load obj 173:0.3"' '"load obj 180:0.3"' '"load obj 171:0.3"' '"load obj 175:0.3"' '"give all wayne:1"' '"restore scagnetti:12"' "\"goto $1:0.5\"" '"transfer wayne:0.5"' $extra '"goto mickey:0.5"' "\"goto $2:0.5\"" '"transfer mickey:0.5"' '"goto mallory:0.5"' "\"goto $3:0.5\"" '"transfer mallory:0.5"' '"goto 1204:0.5"' '"quit:1"'
