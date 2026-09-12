#!/usr/bin/env bash
set -u

output=
previous=
for argument in "$@"; do
	if [[ "$previous" == -o ]]; then
		output=$argument
	fi
	previous=$argument
done
if [[ -z "$output" || -z "${FAKE_HARNESS_SOURCE:-}" ]]; then
	exit 2
fi
cp -- "$FAKE_HARNESS_SOURCE" "$output"
chmod +x "$output"
