#!/usr/bin/env bash
set -u

state_dir=${FAKE_STATE_DIR:?FAKE_STATE_DIR is required}
sequence=${FAKE_SEQUENCE:?FAKE_SEQUENCE is required}
mkdir -p -- "$state_dir"
count_file="$state_dir/attempt-count"
attempt=0
if [[ -f "$count_file" ]]; then
	attempt=$(<"$count_file")
fi
attempt=$((attempt + 1))
printf '%d\n' "$attempt" >"$count_file"

IFS='|' read -r -a steps <<<"$sequence"
step=${steps[$((attempt - 1))]:-infra}
case "$step" in
P)
	printf 'fake harness successful content attempt %d\n' "$attempt"
	exit 0
	;;
I)
	printf 'fake harness: connection reset before readiness\n' >&2
	exit 1
	;;
T)
	printf 'fake harness timeout attempt %d\n' "$attempt" >&2
	exit 124
	;;
M)
	printf 'fake harness divergence without fingerprints\n'
	exit 3
	;;
D:*)
	records=${step#D:}
	IFS=',' read -r -a fingerprints <<<"$records"
	for record in "${fingerprints[@]}"; do
		label=${record%%:*}
		digest=${record#*:}
		printf 'divergence-fingerprint\t%s\t%s\n' "$label" "$digest"
	done
	printf 'fake harness content divergence attempt %d\n' "$attempt"
	exit 3
	;;
X:*)
	record=${step#X:}
	label=${record%%:*}
	digest=${record#*:}
	printf 'divergence-fingerprint\t%s\t%s\n' "$label" "$digest"
	printf 'fake harness malformed fingerprint attempt %d\n' "$attempt"
	exit 3
	;;
*)
	printf 'fake harness: unknown deterministic step %s\n' "$step" >&2
	exit 125
	;;
esac
