#!/bin/sh
# First boot creates the instance in /data (the launcher's run command
# expects an existing one), then the process drops to the unprivileged user.
set -eu

http_port="${DP_HTTP_PORT:-4350}"
telnet_port="${DP_TELNET_PORT:-7777}"

if [ ! -f /data/instance.json ]; then
  # init refuses an existing directory, so create the instance beside the
  # volume and move it in.
  echo "entrypoint: initializing instance in /data (http ${http_port}, telnet ${telnet_port})" >&2
  seed="/tmp/dp-instance-seed"
  rm -rf "$seed"
  /app/darkpawns --instance "$seed" init --port "$http_port" --telnet-port "$telnet_port"
  cp -a "$seed/." /data/
  rm -rf "$seed"
  chown -R 65532:65532 /data
fi

exec setpriv --reuid 65532 --regid 65532 --init-groups \
  /app/darkpawns --instance /data run
