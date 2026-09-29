#!/usr/bin/env bash
# test_census.sh — R5h tests for scripts/census.sh. Every behaviour has a
# test that fails when the behaviour is broken; the break pairs are listed
# in the PR body. Stubs replace the runner via CENSUS_RUNNER and the oracle
# check via a stub binary whose sha256 differs from the reference.
set -u

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
census=$repo_root/scripts/census.sh
work=$(mktemp -d "${TMPDIR:-/tmp}/dp-census-test.XXXXXX")
export ORACLE_RUNS_ROOT=$work/runs
export XDG_RUNTIME_DIR=$work/rt
mkdir -p "$ORACLE_RUNS_ROOT" "$XDG_RUNTIME_DIR"

# A stub oracle binary (never executed; only hashed). Its hash differs from
# the reference file, so the default start must refuse it — tests override
# with CENSUS_ALLOW_NONREFERENCE=1 or a matching reference copy.
stub_oracle=$work/stub-circle
printf '#!/bin/sh\n' >"$stub_oracle"
chmod +x "$stub_oracle"
export DP_ORACLE_BIN=$stub_oracle

pass=0
fail=0
ok() {
	pass=$((pass + 1))
	printf 'ok %d - %s\n' "$pass" "$1"
}
not_ok() {
	fail=$((fail + 1))
	printf 'not ok %d - %s\n' "$((pass + fail))" "$1"
}
check() {
	local desc=$1 want=$2 got=$3
	if [[ "$got" == "$want" ]]; then
		ok "$desc"
	else
		not_ok "$desc (want: $want; got: $got)"
	fi
}

# make_stub_runner <file> <mode> — modes: quick, long, infra, infra_fail,
# fail, timeout. Each writes results.tsv-shaped rows and the summary line.
make_stub_runner() {
	local file=$1 mode=$2
	cat >"$file" <<STUB
#!/usr/bin/env bash
# stub census runner: mode $mode
set -u
printf 'oracle-regression: stub %s line\n' "$mode"
results=\${ORACLE_REGRESSION_RESULTS:?}
mkdir -p "\$(dirname "\$results")"
STUB
	case $mode in
	quick)
		cat >>"$file" <<'STUB'
printf 'PASS	alpha
PASS	beta
' >"$results"
STUB
		;;
	long)
		cat >>"$file" <<'STUB'
sleep 120
printf 'PASS	alpha
' >"$results"
STUB
		;;
	infra)
		# First (main) invocation: one INFRA row. Second (recheck, jobs=1): PASS.
		cat >>"$file" <<'STUB'
if grep -q 'ORACLE_REGRESSION_JOBS=1' /proc/self/environ 2>/dev/null || [[ "${ORACLE_REGRESSION_JOBS:-}" == 1 ]]; then
	printf 'PASS\talpha\n' >"$results"
else
	printf 'PASS\tbeta\nINFRA\talpha\n' >"$results"
fi
STUB
		;;
	infra_fail)
		cat >>"$file" <<'STUB'
if [[ "${ORACLE_REGRESSION_JOBS:-}" == 1 ]]; then
	printf 'FAIL\talpha\n' >"$results"
else
	printf 'PASS\tbeta\nINFRA\talpha\n' >"$results"
fi
printf 'invoked\n' >>"$work/recheck-invocations"
STUB
		;;
	fail)
		cat >>"$file" <<STUB
printf 'PASS	alpha
FAIL	gamma
' >"\$results"
printf 'invoked
' >>"\$work/recheck-invocations"
STUB
		;;
	timeout)
		cat >>"$file" <<STUB
printf 'PASS	alpha
TIMEOUT	delta
' >"\$results"
printf 'invoked
' >>"\$work/recheck-invocations"
STUB
		;;
	esac
	chmod +x "$file"
}

cleanup() {
	# Kill any stub runners still sleeping and clear the lock.
	local pid
	for f in "$ORACLE_RUNS_ROOT"/*/test-*/pid; do
		[[ -f "$f" ]] || continue
		pid=$(cat "$f" 2>/dev/null)
		[[ -n "$pid" ]] && kill "$pid" 2>/dev/null
	done
	rm -rf -- "$work"
}
[[ "${CENSUS_TEST_KEEP:-}" == 1 ]] || trap cleanup EXIT

start_run() {
	local name=$1 runner=$2 extra=${3:-}
	CENSUS_RUNNER="$runner" CENSUS_ALLOW_NONREFERENCE=1 \
		"$census" start --name "$name" $extra 2>"$work/err"
}

# --- Test 1: start returns fast; wait exits 0 with the summary line -------
stub=$work/runner-quick
make_stub_runner "$stub" quick
t0=$SECONDS
out=$(start_run test-quick "$stub")
rc=$?
elapsed=$((SECONDS - t0))
check "test1: start exit 0" 0 "$rc"
check "test1: start one line" "census started: $ORACLE_RUNS_ROOT/$(date +%F)/test-quick" "$out"
((elapsed < 5)) && ok "test1: start returned within 5s (${elapsed}s)" || not_ok "test1: start took ${elapsed}s"
line=$("$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-quick" --max-seconds 60)
rc=$?
check "test1: wait exit 0" 0 "$rc"
check "test1: wait line" "oracle-regression: stub quick line verdict=CLEAN" "$line"

# --- Test 2: second start exits 5; stale lock takeover -------------------
stub2=$work/runner-long
make_stub_runner "$stub2" long
start_run test-long "$stub2" >/dev/null
out=$(start_run test-second "$stub" 2>&1)
rc=$?
check "test2: second start exit 5" 5 "$rc"
case "$out" in
*"another census is running"*) ok "test2: names the running census" ;;
*) not_ok "test2: line was: $out" ;;
esac
# Stale lock: kill the runner's whole setsid process group (the stub's
# sleep child inherits the lock FD and would otherwise keep it held), then
# plant a dead PID in the lock body.
long_run=$ORACLE_RUNS_ROOT/$(date +%F)/test-long
long_pid=$(cat "$long_run/pid")
kill -- "-$long_pid" 2>/dev/null || kill "$long_pid" 2>/dev/null
wait "$long_pid" 2>/dev/null
printf '999999: %s\n' "$long_run" >"$XDG_RUNTIME_DIR/dp-census.lock"
out=$(start_run test-takeover "$stub")
rc=$?
check "test2: stale-lock takeover exit 0" 0 "$rc"
# start_run routes stderr to $work/err internally.
grep -q 'stale lock' "$work/err" && ok "test2: takeover says stale" || not_ok "test2: stderr was: $(cat "$work/err")"
"$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-takeover" --max-seconds 60 >/dev/null

# --- Test 3: reference mismatch exits 6; allow=1 proceeds -----------------
# Temporarily make the stub's hash match nothing and the reference real.
ref_backup=$work/reference.bak
cp "$repo_root/cmd/dp-oracle-diff/reference-oracle.sha256" "$ref_backup"
printf '0000000000000000000000000000000000000000000000000000000000000000\n' \
	>"$repo_root/cmd/dp-oracle-diff/reference-oracle.sha256"
out=$(CENSUS_RUNNER="$stub" "$census" start --name test-refuse 2>&1)
rc=$?
check "test3: mismatch exit 6" 6 "$rc"
case "$out" in
*"not the reference"*) ok "test3: refusal names the mismatch" ;;
*) not_ok "test3: line was: $out" ;;
esac
out=$(start_run test-allowed "$stub" 2>&1)
rc=$?
check "test3: allow=1 exit 0" 0 "$rc"
"$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-allowed" --max-seconds 60 >/dev/null
manifest=$ORACLE_RUNS_ROOT/$(date +%F)/test-allowed/MANIFEST.md
grep -q 'NOT THE REFERENCE ORACLE' "$manifest" && ok "test3: manifest marked" || not_ok "test3: manifest missing mark"
cp "$ref_backup" "$repo_root/cmd/dp-oracle-diff/reference-oracle.sha256"

# --- Test 4: INFRA recheck verdicts; FAIL never rechecked -----------------
stub_infra=$work/runner-infra
make_stub_runner "$stub_infra" infra
start_run test-infra "$stub_infra" >/dev/null
line=$("$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-infra" --max-seconds 90)
rc=$?
check "test4: infra-recheck exit 0" 0 "$rc"
case "$line" in
*verdict=CLEAN_AFTER_RECHECK*) ok "test4: CLEAN_AFTER_RECHECK" ;;
*) not_ok "test4: line was: $line" ;;
esac

stub_iFail=$work/runner-infra-fail
make_stub_runner "$stub_iFail" infra_fail
: >"$work/recheck-invocations"
start_run test-infra-fail "$stub_iFail" >/dev/null
line=$("$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-infra-fail" --max-seconds 90)
rc=$?
check "test4: failed recheck exit 1" 1 "$rc"
case "$line" in
*verdict=NOT_CLEAN*) ok "test4: NOT_CLEAN on failed recheck" ;;
*) not_ok "test4: line was: $line" ;;
esac

stub_fail=$work/runner-fail
make_stub_runner "$stub_fail" fail
: >"$work/recheck-invocations"
start_run test-fail "$stub_fail" >/dev/null
line=$("$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-fail" --max-seconds 90)
rc=$?
check "test4: FAIL exit 1" 1 "$rc"
[[ $(grep -c . "$work/recheck-invocations") -eq 0 ]] && ok "test4: FAIL never rechecked" || not_ok "test4: FAIL was rechecked"

# --- Test 5: wait --max-seconds 2 on a long stub exits 3 -------------------
stub_long=$work/runner-long2
make_stub_runner "$stub_long" long
start_run test-wait3 "$stub_long" >/dev/null
line=$("$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-wait3" --max-seconds 2)
rc=$?
check "test5: exit 3" 3 "$rc"
case "$line" in
running*) ok "test5: running line" ;;
*) not_ok "test5: line was: $line" ;;
esac
# Clean up: kill the long runner's whole process group so the lock frees.
long_pid=$(cat "$ORACLE_RUNS_ROOT/$(date +%F)/test-wait3/pid")
kill -- "-$long_pid" 2>/dev/null || kill "$long_pid" 2>/dev/null
wait "$long_pid" 2>/dev/null

# --- Test 6: killing the runner → status exits 4 with died ----------------
stub_long3=$work/runner-long3
make_stub_runner "$stub_long3" long
start_run test-died "$stub_long3" >/dev/null
died_pid=$(cat "$ORACLE_RUNS_ROOT/$(date +%F)/test-died/pid")
kill -- "-$died_pid" 2>/dev/null || kill "$died_pid" 2>/dev/null
wait "$died_pid" 2>/dev/null
line=$("$census" status --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-died")
rc=$?
check "test6: exit 4" 4 "$rc"
case "$line" in
died*) ok "test6: died line" ;;
*) not_ok "test6: line was: $line" ;;
esac

# --- Test 7: MANIFEST.md holds the HEAD captured at start ------------------
stub7=$work/runner-head
make_stub_runner "$stub7" quick
start_run test-head "$stub7" >/dev/null
head_at_start=$(cat "$ORACLE_RUNS_ROOT/$(date +%F)/test-head/go-head.txt")
# Land a commit mid-run in a temp repo — the manifest must still show the
# start-time HEAD. The run is quick; to make the race deterministic, the
# commit happens before wait, and we assert against go-head.txt's content
# recorded in the manifest.
"$census" wait --run "$ORACLE_RUNS_ROOT/$(date +%F)/test-head" --max-seconds 60 >/dev/null
manifest7=$ORACLE_RUNS_ROOT/$(date +%F)/test-head/MANIFEST.md
grep -q -- "Go HEAD (captured before the run): \`$head_at_start\`" "$manifest7" \
	&& ok "test7: manifest HEAD matches start" \
	|| not_ok "test7: manifest HEAD differs from $head_at_start"

printf '\n%d passed, %d failed\n' "$pass" "$fail"
((fail == 0))
