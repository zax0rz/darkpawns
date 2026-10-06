#!/bin/sh
# Reproducible enumeration of the C mudlog producer class (DP-1371 D7).
#
# The manifest's historical "about 120" is an estimate, not an audited
# denominator: a raw `mudlog(` grep also matches the consumer's own definition,
# comment prose and the whod.c LOG macro wrapper. This script prints every
# match so a reviewer can reconcile the inventory by hand; it claims no
# classification of its own.
#
# Usage: docs/fidelity/depth/handoff/2026-10-05-dp-1371-mudlog-inventory.sh > enumeration.txt
set -u
cd "$(dirname "$0")/../../../.." || exit 1

echo "# every mudlog( match in src/*.c, with line and text"
grep -rn 'mudlog(' src --include='*.c' | sed 's/[[:space:]]\+$//'

echo
echo "# totals per file"
grep -rn 'mudlog(' src --include='*.c' | cut -d: -f1 | sort | uniq -c | sort -rn

echo
echo "# non-comment, non-definition candidates (line does not open with a comment marker)"
grep -rn 'mudlog(' src --include='*.c' \
  | grep -vE ':[0-9]+:[[:space:]]*(\*|/\*)' \
  | grep -v 'void mudlog(char' \
  | sed 's/[[:space:]]\+$//'
