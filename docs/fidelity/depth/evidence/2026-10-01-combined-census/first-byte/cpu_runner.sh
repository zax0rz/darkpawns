#!/usr/bin/env bash
set -eu
repo_root=$CENSUS_REPO_ROOT
proof_dir=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
log_dir=$CENSUS_RUN_DIR/cpu-attempts
mkdir -p "$log_dir/results"
printf 'cpu-proof: first-byte-wait=%s trailing-silence=300ms seed=2\n' "$CPU_PROOF_FIRST_WAIT"
sha256sum "$repo_root/cmd/dp-oracle-diff/main.go" "$repo_root/internal/oraclediff/conn.go" "$repo_root/internal/oraclediff/wsconn.go" > "$CENSUS_RUN_DIR/source-sha256.txt"
go build -C "$repo_root" -o "$log_dir/harness-real" ./cmd/dp-oracle-diff
go build -C "$repo_root" -o "$log_dir/server-real" ./cmd/server
export CPU_PROOF_SERVER_REAL="$log_dir/server-real"
export ORACLE_REGRESSION_SERVER="$proof_dir/server_proxy.py"
cat > "$log_dir/harness" <<'WRAP'
#!/usr/bin/env bash
exec "$CPU_PROOF_HARNESS_REAL" --first-byte-wait="$CPU_PROOF_FIRST_WAIT" --show-go-log "$@"
WRAP
chmod +x "$log_dir/harness"
export CPU_PROOF_HARNESS_REAL="$log_dir/harness-real"
export result_dir="$log_dir/results" harness_bin="$log_dir/harness" oracle_bin="$CENSUS_ORACLE_BIN" seed=2 scenario_timeout=240s
export EXPECTED_DIVERGENCES_FILE="$repo_root/cmd/dp-oracle-diff/expected_divergences.tsv"
export EXPECTED_DIVERGENCE_PINS_FILE="$repo_root/cmd/dp-oracle-diff/expected_divergence_pins.tsv"
export repo_root log_dir
"$repo_root/scripts/oracle_regression_worker.sh" wizard-zreset-depth.txt
cp "$result_dir/wizard-zreset-depth" "$ORACLE_REGRESSION_RESULTS"
kind=$(cut -f1 "$ORACLE_REGRESSION_RESULTS")
printf 'oracle-regression: CPU-contention proof kind=%s\n' "$kind"
