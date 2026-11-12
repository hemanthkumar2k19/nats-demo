#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Usage:
#   ./benchmark.sh [--check] [--serve] [CONFIG_FILE]

if [[ "${1:-}" == "--serve" ]]; then
  exec python3 "${ROOT_DIR}/scripts/server.py"
fi

CHECK_ONLY=false
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=true
  shift
fi

# Locate configuration file
CONFIG_FILE="${1:-${ROOT_DIR}/benchmark.conf}"

# Save environment overrides prior to sourcing config.
# When benchmark.sh is invoked as part of a suite (via suite.sh), test-specific
# overrides (such as THROUGHPUT, PUBS, SUBS) are exported into the environment.
# Sourcing benchmark.conf would overwrite those exported variables with static defaults.
# By capturing them first in _ENV_* and reapplying them after sourcing, environment
# overrides reliably take precedence over configuration file defaults.
_ENV_TEST_ID="${TEST_ID:-}"
_ENV_TEST_DESC="${TEST_DESCRIPTION:-}"
_ENV_NATS_URL="${NATS_URL:-}"
_ENV_SCENARIO="${SCENARIO:-}"
_ENV_CLIENTS="${CLIENTS:-}"
_ENV_PUBS="${PUBS:-}"
_ENV_SUBS="${SUBS:-}"
_ENV_THROUGHPUT="${THROUGHPUT:-}"
_ENV_DURATION="${DURATION:-}"
_ENV_MSGS="${MSGS:-}"
_ENV_SIZE="${SIZE:-}"
_ENV_SUBJECT="${SUBJECT:-}"
_ENV_PUB_SUBJECT="${PUB_SUBJECT:-}"
_ENV_SUB_SUBJECT="${SUB_SUBJECT:-}"
_ENV_QUEUE="${QUEUE:-}"
_ENV_STREAM="${STREAM:-}"
_ENV_CONSUMER="${CONSUMER:-}"
_ENV_STORAGE="${STORAGE:-}"
_ENV_REPLICAS="${REPLICAS:-}"
_ENV_MAX_BYTES="${MAX_BYTES:-}"
_ENV_RETENTION="${RETENTION:-}"
_ENV_BATCH="${BATCH:-}"
_ENV_ACKS="${ACKS:-}"
_ENV_MULTISUBJECT="${MULTISUBJECT:-}"
_ENV_MULTISUBJECT_MAX="${MULTISUBJECT_MAX:-}"
_ENV_MULTISUBJECT_RANDOM="${MULTISUBJECT_RANDOM:-}"
_ENV_STREAM_COUNT="${STREAM_COUNT:-}"
_ENV_CONSUMER_COUNT="${CONSUMER_COUNT:-}"
_ENV_CONTAINERS="${CONTAINERS:-}"
_ENV_BASELINE_DURATION_SECONDS="${BASELINE_DURATION_SECONDS:-}"
_ENV_RESULTS_DIR="${RESULTS_DIR:-}"
_ENV_TIMESTAMP_SUBFOLDER="${TIMESTAMP_SUBFOLDER:-}"

# Source config if found
if [[ -f "$CONFIG_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$CONFIG_FILE"
fi

# Reapply environment overrides
[[ -n "$_ENV_TEST_ID" ]] && TEST_ID="$_ENV_TEST_ID"
[[ -n "$_ENV_TEST_DESC" ]] && TEST_DESCRIPTION="$_ENV_TEST_DESC"
[[ -n "$_ENV_NATS_URL" ]] && NATS_URL="$_ENV_NATS_URL"
[[ -n "$_ENV_SCENARIO" ]] && SCENARIO="$_ENV_SCENARIO"
[[ -n "$_ENV_CLIENTS" ]] && CLIENTS="$_ENV_CLIENTS"
[[ -n "$_ENV_PUBS" ]] && PUBS="$_ENV_PUBS"
[[ -n "$_ENV_SUBS" ]] && SUBS="$_ENV_SUBS"
[[ -n "$_ENV_THROUGHPUT" ]] && THROUGHPUT="$_ENV_THROUGHPUT"
[[ -n "$_ENV_DURATION" ]] && DURATION="$_ENV_DURATION"
[[ -n "$_ENV_MSGS" ]] && MSGS="$_ENV_MSGS"
[[ -n "$_ENV_SIZE" ]] && SIZE="$_ENV_SIZE"
[[ -n "$_ENV_SUBJECT" ]] && SUBJECT="$_ENV_SUBJECT"
[[ -n "$_ENV_PUB_SUBJECT" ]] && PUB_SUBJECT="$_ENV_PUB_SUBJECT"
[[ -n "$_ENV_SUB_SUBJECT" ]] && SUB_SUBJECT="$_ENV_SUB_SUBJECT"
[[ -n "$_ENV_QUEUE" ]] && QUEUE="$_ENV_QUEUE"
[[ -n "$_ENV_STREAM" ]] && STREAM="$_ENV_STREAM"
[[ -n "$_ENV_CONSUMER" ]] && CONSUMER="$_ENV_CONSUMER"
[[ -n "$_ENV_STORAGE" ]] && STORAGE="$_ENV_STORAGE"
[[ -n "$_ENV_REPLICAS" ]] && REPLICAS="$_ENV_REPLICAS"
[[ -n "$_ENV_MAX_BYTES" ]] && MAX_BYTES="$_ENV_MAX_BYTES"
[[ -n "$_ENV_RETENTION" ]] && RETENTION="$_ENV_RETENTION"
[[ -n "$_ENV_BATCH" ]] && BATCH="$_ENV_BATCH"
[[ -n "$_ENV_ACKS" ]] && ACKS="$_ENV_ACKS"
[[ -n "$_ENV_MULTISUBJECT" ]] && MULTISUBJECT="$_ENV_MULTISUBJECT"
[[ -n "$_ENV_MULTISUBJECT_MAX" ]] && MULTISUBJECT_MAX="$_ENV_MULTISUBJECT_MAX"
[[ -n "$_ENV_MULTISUBJECT_RANDOM" ]] && MULTISUBJECT_RANDOM="$_ENV_MULTISUBJECT_RANDOM"
[[ -n "$_ENV_STREAM_COUNT" ]] && STREAM_COUNT="$_ENV_STREAM_COUNT"
[[ -n "$_ENV_CONSUMER_COUNT" ]] && CONSUMER_COUNT="$_ENV_CONSUMER_COUNT"
[[ -n "$_ENV_CONTAINERS" ]] && CONTAINERS="$_ENV_CONTAINERS"
[[ -n "$_ENV_BASELINE_DURATION_SECONDS" ]] && BASELINE_DURATION_SECONDS="$_ENV_BASELINE_DURATION_SECONDS"
[[ -n "$_ENV_RESULTS_DIR" ]] && RESULTS_DIR="$_ENV_RESULTS_DIR"
[[ -n "$_ENV_TIMESTAMP_SUBFOLDER" ]] && TIMESTAMP_SUBFOLDER="$_ENV_TIMESTAMP_SUBFOLDER"

# -----------------------------------------------------------------------------
# Configuration Defaults (grouped by subsystem)
# -----------------------------------------------------------------------------

# 1. Run Identification & Cluster Endpoint
RUN_TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
: "${TEST_ID:=${RUN_TIMESTAMP}}"
: "${TEST_DESCRIPTION:=NATS Capacity Benchmark Run}"
: "${NATS_URL:=nats://127.0.0.1:4222}"
: "${RESULTS_DIR:=results}"
: "${TIMESTAMP_SUBFOLDER:=false}"

# 2. Workload & Rate Parameters
: "${SCENARIO:=pubsub}"
: "${CLIENTS:=4}"
: "${PUBS:=${CLIENTS}}"
: "${SUBS:=${CLIENTS}}"
: "${THROUGHPUT:=1000}"
: "${DURATION:=30s}"
: "${MSGS:=30000}"
: "${SIZE:=1024B}"

# Ensure duration and size have standard units
[[ "$DURATION" =~ ^[0-9]+$ ]] && DURATION="${DURATION}s"
[[ "$SIZE" =~ ^[0-9]+$ ]] && SIZE="${SIZE}B"

# Automatically derive expected message count from throughput and duration when throughput is set.
# Note: 'nats bench pub' operates in either count mode (--msgs) or rate/duration mode
# (--throughput + --duration). In rate/duration mode, deriving MSGS = THROUGHPUT * DURATION
# enables downstream aggregation scripts and report dashboards to compute target vs achieved
# delivery volumes and verify whether all messages were successfully delivered.
CLEAN_DUR="${DURATION%s}"
if [[ "$THROUGHPUT" -gt 0 && "$CLEAN_DUR" =~ ^[0-9]+$ ]]; then
  if [[ -z "$_ENV_MSGS" ]]; then
    MSGS=$(( THROUGHPUT * CLEAN_DUR ))
  fi
fi

# 3. Subject & Queue Grouping
: "${SUBJECT:=benchmark.core.test}"
: "${PUB_SUBJECT:=${SUBJECT}}"
: "${SUB_SUBJECT:=${SUBJECT}}"
: "${QUEUE:=}"
: "${MULTISUBJECT:=false}"
: "${MULTISUBJECT_MAX:=100000}"
: "${MULTISUBJECT_RANDOM:=false}"

# 4. JetStream Semantics
: "${STREAM:=BENCH_STREAM}"
: "${STREAM_COUNT:=1}"
: "${CONSUMER:=bench-consumer}"
: "${CONSUMER_COUNT:=0}"
: "${STORAGE:=file}"
: "${REPLICAS:=3}"
: "${MAX_BYTES:=5GB}"
: "${RETENTION:=limits}"
: "${BATCH:=500}"
: "${ACKS:=explicit}"
: "${PREPOPULATE_COUNT:=0}"

# 5. Telemetry & Hardware Monitoring
: "${METRICS_BACKEND:=podman}"
: "${CONTAINERS:=nats-1 nats-2 nats-3}"
: "${METRICS_SAMPLE_INTERVAL:=5}"
: "${BASELINE_DURATION_SECONDS:=15}"
: "${DISK_METRICS_BACKEND:=local}"
: "${NATS_DATA_BASE_DIR:=${ROOT_DIR}/../deploy/data}"

# Derived parameters for preflight
export NATS_URL METRICS_BACKEND DISK_METRICS_BACKEND NATS_DATA_BASE_DIR
: "${EXPECTED_NATS_NODES:=3}"
: "${NATS_MONITOR_URLS:=http://127.0.0.1:8222 http://127.0.0.1:8223 http://127.0.0.1:8224}"
: "${EXPECTED_REPLICAS:=3}"

MESSAGING_TYPE="Core NATS"
[[ "$SCENARIO" == js-* ]] && MESSAGING_TYPE="JetStream"
export MESSAGING_TYPE

source "${ROOT_DIR}/scripts/preflight.sh"

if [[ "$CHECK_ONLY" == true ]]; then
  preflight_checks
  exit $?
fi

# Prepare output directory
if [[ "$RESULTS_DIR" == /* ]]; then
  BASE_RESULTS_DIR="${RESULTS_DIR}"
else
  BASE_RESULTS_DIR="${ROOT_DIR}/${RESULTS_DIR}"
fi

if [[ "$TIMESTAMP_SUBFOLDER" == true ]]; then
  RESULT_DIR="${BASE_RESULTS_DIR}/${TEST_ID}/${RUN_TIMESTAMP}"
else
  RESULT_DIR="${BASE_RESULTS_DIR}/${TEST_ID}"
fi
mkdir -p "$RESULT_DIR"

# Write test-config.txt
cat > "$RESULT_DIR/test-config.txt" <<EOF
test_id=${TEST_ID}
test_description=${TEST_DESCRIPTION}
run_timestamp=${RUN_TIMESTAMP}
messaging_type=${MESSAGING_TYPE}
scenario=${SCENARIO}
nats_url=${NATS_URL}
subject=${SUBJECT}
pub_subject=${PUB_SUBJECT}
sub_subject=${SUB_SUBJECT}
queue_name=${QUEUE}
target_publish_msgs_per_sec=${THROUGHPUT}
message_size_bytes=${SIZE}
publisher_concurrency=${PUBS}
subscriber_concurrency=${SUBS}
nominal_duration_seconds=${DURATION}
configured_message_count=${MSGS}
stream_name=${STREAM}
storage_type=${STORAGE}
replication_factor=${REPLICAS}
max_bytes=${MAX_BYTES}
retention_policy=${RETENTION}
consumer_name=${CONSUMER}
consumer_ack_policy=${ACKS}
consumer_batch_size=${BATCH}
multisubject=${MULTISUBJECT}
multisubject_max=${MULTISUBJECT_MAX}
multisubject_random=${MULTISUBJECT_RANDOM}
stream_count=${STREAM_COUNT}
consumer_count=${CONSUMER_COUNT}
EOF

# JetStream setup if required
if [[ "$MESSAGING_TYPE" == "JetStream" ]]; then
  echo "Preparing JetStream stream: $STREAM (storage=$STORAGE, R=$REPLICAS, streams=$STREAM_COUNT, consumers=$CONSUMER_COUNT)..."
  set +e
  STREAM_COUNT="$STREAM_COUNT" CONSUMER_COUNT="$CONSUMER_COUNT" \
  bash "${ROOT_DIR}/scripts/js_stream.sh" setup \
    "$NATS_URL" "$STREAM" "$SUBJECT" "$STORAGE" "$REPLICAS" "$MAX_BYTES" "$RETENTION" "$CONSUMER" "$ACKS" 2>&1 | tee "$RESULT_DIR/setup.log"
  JS_SETUP_EXIT=${PIPESTATUS[0]}
  set -e

  if [[ "$JS_SETUP_EXIT" -ne 0 ]]; then
    echo "ERROR: JetStream stream setup failed with exit code $JS_SETUP_EXIT" >&2
    ERROR_MSG=$(grep -E "(error:|ERROR:|FAIL:)" "$RESULT_DIR/setup.log" | tail -n 1 | sed 's/^[ \t]*//' || echo "JetStream stream setup failed")
    cat > "$RESULT_DIR/error.log" <<EOF
error_phase=jetstream_setup
error_message=${ERROR_MSG}
exit_code=${JS_SETUP_EXIT}
EOF
    python3 "${ROOT_DIR}/scripts/aggregate.py" "$RESULT_DIR" "$TEST_ID" || true
    exit "$JS_SETUP_EXIT"
  fi

  if [[ "$SCENARIO" == "js-consume" && "$PREPOPULATE_COUNT" -gt 0 ]]; then
    set +e
    bash "${ROOT_DIR}/scripts/js_stream.sh" prepopulate \
      "$NATS_URL" "$STREAM" "$SUBJECT" "$PREPOPULATE_COUNT" "$SIZE" 2>&1 | tee -a "$RESULT_DIR/setup.log"
    JS_PREPOP_EXIT=${PIPESTATUS[0]}
    set -e

    if [[ "$JS_PREPOP_EXIT" -ne 0 ]]; then
      echo "ERROR: JetStream prepopulate failed with exit code $JS_PREPOP_EXIT" >&2
      ERROR_MSG=$(grep -E "(error:|ERROR:|FAIL:)" "$RESULT_DIR/setup.log" | tail -n 1 | sed 's/^[ \t]*//' || echo "JetStream prepopulate failed")
      cat > "$RESULT_DIR/error.log" <<EOF
error_phase=jetstream_prepopulate
error_message=${ERROR_MSG}
exit_code=${JS_PREPOP_EXIT}
EOF
      python3 "${ROOT_DIR}/scripts/aggregate.py" "$RESULT_DIR" "$TEST_ID" || true
      exit "$JS_PREPOP_EXIT"
    fi
  fi
fi

METRICS_PID=""
cleanup() {
  if [[ -n "$METRICS_PID" ]] && kill -0 "$METRICS_PID" 2>/dev/null; then
    kill -TERM "$METRICS_PID" 2>/dev/null || true
    wait "$METRICS_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# Baseline telemetry sampling
if [[ "$METRICS_BACKEND" == "podman" && "$BASELINE_DURATION_SECONDS" -gt 0 ]]; then
  echo "Sampling idle baseline for ${BASELINE_DURATION_SECONDS}s..."
  "${ROOT_DIR}/scripts/metrics.sh" "$RESULT_DIR/metrics.json" "$METRICS_SAMPLE_INTERVAL" baseline $CONTAINERS &
  METRICS_PID=$!
  sleep "$BASELINE_DURATION_SECONDS"
  kill -TERM "$METRICS_PID" 2>/dev/null || true
  wait "$METRICS_PID" 2>/dev/null || true
  METRICS_PID=""
fi

# Active workload telemetry sampling
if [[ "$METRICS_BACKEND" == "podman" ]]; then
  "${ROOT_DIR}/scripts/metrics.sh" "$RESULT_DIR/metrics.json" "$METRICS_SAMPLE_INTERVAL" workload $CONTAINERS &
  METRICS_PID=$!
  sleep 1
fi

# Execute workload
echo "Executing workload..."
set +e
MULTISUBJECT_RANDOM="$MULTISUBJECT_RANDOM" PUB_SUBJECT="$PUB_SUBJECT" SUB_SUBJECT="$SUB_SUBJECT" \
STREAM_COUNT="$STREAM_COUNT" CONSUMER_COUNT="$CONSUMER_COUNT" \
bash "${ROOT_DIR}/scripts/workload.sh" \
  "$NATS_URL" "$SUBJECT" "$PUBS" "$SUBS" "$MSGS" "$SIZE" "$THROUGHPUT" "$DURATION" \
  "$RESULT_DIR" "$SCENARIO" "$QUEUE" "$MULTISUBJECT" "$MULTISUBJECT_MAX" \
  "$STREAM" "$CONSUMER" "$BATCH" "$ACKS" 2>&1 | tee "$RESULT_DIR/workload.log"
WORKLOAD_EXIT=${PIPESTATUS[0]}
set -e

if [[ "$WORKLOAD_EXIT" -ne 0 ]]; then
  echo "WARNING: Workload finished with non-zero exit code: $WORKLOAD_EXIT" >&2
  ERROR_MSG="Workload execution returned exit code ${WORKLOAD_EXIT}"
  if [[ -f "$RESULT_DIR/workload_outcome.txt" ]]; then
    outcome_txt=$(cat "$RESULT_DIR/workload_outcome.txt" | head -n 1)
    [[ -n "$outcome_txt" ]] && ERROR_MSG="Workload failed: ${outcome_txt}"
  else
    extracted_err=$(grep -E "(error:|ERROR:|FAIL:)" "$RESULT_DIR/workload.log" | tail -n 1 | sed 's/^[ \t]*//' || true)
    [[ -n "$extracted_err" ]] && ERROR_MSG="${extracted_err}"
  fi
  cat > "$RESULT_DIR/error.log" <<EOF
error_phase=workload
error_message=${ERROR_MSG}
exit_code=${WORKLOAD_EXIT}
EOF
fi

# Stop telemetry
if [[ -n "$METRICS_PID" ]]; then
  sleep 1
  kill -TERM "$METRICS_PID" 2>/dev/null || true
  wait "$METRICS_PID" 2>/dev/null || true
  METRICS_PID=""
fi

# Aggregate results into canonical results.json and self-contained report.html
python3 "${ROOT_DIR}/scripts/aggregate.py" "$RESULT_DIR" "$TEST_ID"

echo ""
echo "===================================="
echo "Benchmark Finished"
echo "Test ID     : $TEST_ID"
echo "Exit Code   : $WORKLOAD_EXIT"
echo "Report HTML : $RESULT_DIR/report.html"
echo "Results     : $RESULT_DIR"
echo "===================================="

exit "$WORKLOAD_EXIT"
