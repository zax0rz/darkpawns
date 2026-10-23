#!/bin/sh
# Stable entry point at darkpawns.org/install.sh. The publisher key below is
# pinned here and changes only with a deliberate stub update on key rotation;
# every release publishes install.sh.sig beside install.sh, and the installer
# runs only after that signature verifies — never before.
set -eu
umask 077
base=https://github.com/zax0rz/darkpawns/releases/latest/download
work=$(mktemp -d "${TMPDIR:-/tmp}/darkpawns-bootstrap.XXXXXXXX")
trap 'rm -rf "$work"' EXIT
trap 'exit 1' HUP INT TERM
for tool in curl openssl mktemp wc; do command -v "$tool" >/dev/null 2>&1 || { echo "Required system utility is missing: $tool" >&2; exit 1; }; done
cat > "$work/publisher.pem" <<'PUBLIC_KEY'
-----BEGIN PUBLIC KEY-----
MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAyFZWeEzNp0iaUB3CZtI0
N3uEelzM9QYWmsqfKf9Uo/Q5LfCtM/kwE4dDKaFYZ7AnY8gp3CquBUsi4K2+Om+Z
RHeTAgwbjRfBDP+Z7dFHrvTh9l9bH/gxX6Qv7W0Rq2PcogtCmZTJdOmFSVMQ8VnA
zEVo5R5ZeQoyfuwR9a2W8MZ5r91lsfebSTz+oxSmmcwVDNhq3M4nV5mIaEwvnxS9
lPvqj0zdf0L8Q5AQxBd4G7t4Gt7ipp+7h/JrQ6wSOZvcNC+ly6fIMuswp7sAwwrq
wawhnX+s/Ucw148NepkiMlZ6kCF3qgNfDfPrtEkZvDSjHN36M9JQErghxCW1rgpk
bi38qr3M8zm74RSM4imW4l0cnISOKusk6OWhAuTJOXPgckg7tQ6l/LgbbBI7eEBx
ORwBDCCZHSt9yNc58OzIt2wvzvDlQ0mK/RSh1MFpPYc707dk5JZILMRYL0jJbr2s
dLZWgNMrcaDmvHTp2ckSOIJI5Cq02NHCnrm5LiO2IRQFAgMBAAE=
-----END PUBLIC KEY-----
PUBLIC_KEY
curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 --max-time 180 --max-filesize 1048576 "$base/install.sh" -o "$work/install.sh"
curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 --max-time 60 --max-filesize 4096 "$base/install.sh.sig" -o "$work/install.sh.sig"
[ "$(wc -c < "$work/install.sh")" -le 1048576 ] && [ "$(wc -c < "$work/install.sh.sig")" -le 1024 ] || { echo 'Installer download exceeds size limit.' >&2; exit 1; }
openssl dgst -sha256 -verify "$work/publisher.pem" -signature "$work/install.sh.sig" "$work/install.sh" >/dev/null || { echo 'Publisher verification failed. Nothing ran.' >&2; exit 1; }
sh "$work/install.sh" "$@"
