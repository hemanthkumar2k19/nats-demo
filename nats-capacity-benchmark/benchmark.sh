#!/bin/bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CHECK_ONLY=false

if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=true
  shift
fi

# Usage:
#   ./benchmark.sh [--check] [WORKLOAD_CONFIG] [ENVIRONMENT_CONFIG] [TEST_CONFIG]
# Defaults preserve the original one-argument invocation.
WORKLOAD_CONFIG="${1:-${ROOT_DIR}/config/core-nats.conf}"
ENVIRONMENT_CONFIG="${2:-${ROOT_DIR}/config/environment.conf}"
TEST_CONFIG="${3:-${ROOT_DIR}/config/test.conf}"

[[ -f "$WORKLOAD_CONFIG" ]] || {
  echo "Workload config not found: $WORKLOAD_CONFIG" >&2
  exit 1
}

[[ -f "$ENVIRONMENT_CONFIG" ]] || {
  echo "Environment config not found: $ENVIRONMENT_CONFIG" >&2
  exit 1
}

# Capture caller-provided environment overrides before sourcing config files
_ENV_MESSAGING_TYPE="${MESSAGING_TYPE:-}"
_ENV_SCENARIO="${SCENARIO:-}"
_ENV_PUBLISHER_CONCURRENCY="${PUBLISHER_CONCURRENCY:-}"
_ENV_SUBSCRIBER_CONCURRENCY="${SUBSCRIBER_CONCURRENCY:-}"
_ENV_TARGET_MSGS_PER_SEC="${TARGET_MSGS_PER_SEC:-}"
_ENV_MESSAGE_SIZE="${MESSAGE_SIZE:-}"
_ENV_TEST_DURATION_SECONDS="${TEST_DURATION_SECONDS:-}"
_ENV_CONFIGURED_MESSAGE_COUNT="${CONFIGURED_MESSAGE_COUNT:-}"
_ENV_SUBSCRIPTION_PATTERN="${SUBSCRIPTION_PATTERN:-}"
_ENV_QUEUE_NAME="${QUEUE_NAME:-}"
_ENV_SUBJECT="${SUBJECT:-}"
_ENV_SUBJECT_DISTRIBUTION="${SUBJECT_DISTRIBUTION:-}"
_ENV_MULTI_SUBJECT_MAX="${MULTI_SUBJECT_MAX:-}"
_ENV_TEST_ID="${TEST_ID:-}"
_ENV_TEST_DESCRIPTION="${TEST_DESCRIPTION:-}"
_ENV_TIMESTAMP_SUBFOLDER="${TIMESTAMP_SUBFOLDER:-}"
_ENV_NATS_URL="${NATS_URL:-}"
_ENV_METRICS_BACKEND="${METRICS_BACKEND:-}"
_ENV_DISK_METRICS_BACKEND="${DISK_METRICS_BACKEND:-}"
_ENV_NATS_DATA_BASE_DIR="${NATS_DATA_BASE_DIR:-}"
_ENV_BASELINE_DURATION_SECONDS="${BASELINE_DURATION_SECONDS:-}"
_ENV_METRICS_SAMPLE_INTERVAL="${METRICS_SAMPLE_INTERVAL:-}"
_ENV_NATS_CONTAINERS="${NATS_CONTAINERS:-}"
_ENV_RESULTS_DIR="${RESULTS_DIR:-}"
_ENV_STREAM_NAME="${STREAM_NAME:-}"
_ENV_STREAM_SUBJECT="${STREAM_SUBJECT:-}"
_ENV_STORAGE_TYPE="${STORAGE_TYPE:-}"
_ENV_REPLICATION_FACTOR="${REPLICATION_FACTOR:-}"
_ENV_MAX_BYTES="${MAX_BYTES:-}"
_ENV_RETENTION_POLICY="${RETENTION_POLICY:-}"
_ENV_JS_PUB_MODE="${JS_PUB_MODE:-}"
_ENV_CONSUMER_NAME="${CONSUMER_NAME:-}"
_ENV_CONSUMER_ACK_POLICY="${CONSUMER_ACK_POLICY:-}"
_ENV_CONSUMER_BATCH_SIZE="${CONSUMER_BATCH_SIZE:-}"
_ENV_PREPOPULATE_COUNT="${PREPOPULATE_COUNT:-}"
_ENV_PUBLISHER_GRACE_SECONDS="${PUBLISHER_GRACE_SECONDS:-}"
_ENV_SUBSCRIBER_DRAIN_SECONDS="${SUBSCRIBER_DRAIN_SECONDS:-}"
_ENV_JS_SETUP_TIMEOUT_SECONDS="${JS_SETUP_TIMEOUT_SECONDS:-}"

# Load environment first, then workload-specific settings, then test identification.
source "$ENVIRONMENT_CONFIG"
source "$WORKLOAD_CONFIG"

if [[ -f "$TEST_CONFIG" ]]; then
  source "$TEST_CONFIG"
fi

# Re-apply explicit environment overrides
[[ -n "$_ENV_MESSAGING_TYPE" ]] && MESSAGING_TYPE="$_ENV_MESSAGING_TYPE"
[[ -n "$_ENV_SCENARIO" ]] && SCENARIO="$_ENV_SCENARIO"
[[ -n "$_ENV_PUBLISHER_CONCURRENCY" ]] && PUBLISHER_CONCURRENCY="$_ENV_PUBLISHER_CONCURRENCY"
[[ -n "$_ENV_SUBSCRIBER_CONCURRENCY" ]] && SUBSCRIBER_CONCURRENCY="$_ENV_SUBSCRIBER_CONCURRENCY"
[[ -n "$_ENV_TARGET_MSGS_PER_SEC" ]] && TARGET_MSGS_PER_SEC="$_ENV_TARGET_MSGS_PER_SEC"
[[ -n "$_ENV_MESSAGE_SIZE" ]] && MESSAGE_SIZE="$_ENV_MESSAGE_SIZE"
[[ -n "$_ENV_TEST_DURATION_SECONDS" ]] && TEST_DURATION_SECONDS="$_ENV_TEST_DURATION_SECONDS"
[[ -n "$_ENV_CONFIGURED_MESSAGE_COUNT" ]] && CONFIGURED_MESSAGE_COUNT="$_ENV_CONFIGURED_MESSAGE_COUNT"
[[ -n "$_ENV_SUBSCRIPTION_PATTERN" ]] && SUBSCRIPTION_PATTERN="$_ENV_SUBSCRIPTION_PATTERN"
[[ -n "$_ENV_QUEUE_NAME" ]] && QUEUE_NAME="$_ENV_QUEUE_NAME"
[[ -n "$_ENV_SUBJECT" ]] && SUBJECT="$_ENV_SUBJECT"
[[ -n "$_ENV_SUBJECT_DISTRIBUTION" ]] && SUBJECT_DISTRIBUTION="$_ENV_SUBJECT_DISTRIBUTION"
[[ -n "$_ENV_MULTI_SUBJECT_MAX" ]] && MULTI_SUBJECT_MAX="$_ENV_MULTI_SUBJECT_MAX"
[[ -n "$_ENV_TEST_ID" ]] && TEST_ID="$_ENV_TEST_ID"
[[ -n "$_ENV_TEST_DESCRIPTION" ]] && TEST_DESCRIPTION="$_ENV_TEST_DESCRIPTION"
[[ -n "$_ENV_TIMESTAMP_SUBFOLDER" ]] && TIMESTAMP_SUBFOLDER="$_ENV_TIMESTAMP_SUBFOLDER"
[[ -n "$_ENV_NATS_URL" ]] && NATS_URL="$_ENV_NATS_URL"
[[ -n "$_ENV_METRICS_BACKEND" ]] && METRICS_BACKEND="$_ENV_METRICS_BACKEND"
[[ -n "$_ENV_DISK_METRICS_BACKEND" ]] && DISK_METRICS_BACKEND="$_ENV_DISK_METRICS_BACKEND"
[[ -n "$_ENV_NATS_DATA_BASE_DIR" ]] && NATS_DATA_BASE_DIR="$_ENV_NATS_DATA_BASE_DIR"
[[ -n "$_ENV_BASELINE_DURATION_SECONDS" ]] && BASELINE_DURATION_SECONDS="$_ENV_BASELINE_DURATION_SECONDS"
[[ -n "$_ENV_METRICS_SAMPLE_INTERVAL" ]] && METRICS_SAMPLE_INTERVAL="$_ENV_METRICS_SAMPLE_INTERVAL"
[[ -n "$_ENV_NATS_CONTAINERS" ]] && NATS_CONTAINERS="$_ENV_NATS_CONTAINERS"
[[ -n "$_ENV_RESULTS_DIR" ]] && RESULTS_DIR="$_ENV_RESULTS_DIR"
[[ -n "$_ENV_STREAM_NAME" ]] && STREAM_NAME="$_ENV_STREAM_NAME"
[[ -n "$_ENV_STREAM_SUBJECT" ]] && STREAM_SUBJECT="$_ENV_STREAM_SUBJECT"
[[ -n "$_ENV_STORAGE_TYPE" ]] && STORAGE_TYPE="$_ENV_STORAGE_TYPE"
[[ -n "$_ENV_REPLICATION_FACTOR" ]] && REPLICATION_FACTOR="$_ENV_REPLICATION_FACTOR"
[[ -n "$_ENV_MAX_BYTES" ]] && MAX_BYTES="$_ENV_MAX_BYTES"
[[ -n "$_ENV_RETENTION_POLICY" ]] && RETENTION_POLICY="$_ENV_RETENTION_POLICY"
[[ -n "$_ENV_JS_PUB_MODE" ]] && JS_PUB_MODE="$_ENV_JS_PUB_MODE"
[[ -n "$_ENV_CONSUMER_NAME" ]] && CONSUMER_NAME="$_ENV_CONSUMER_NAME"
[[ -n "$_ENV_CONSUMER_ACK_POLICY" ]] && CONSUMER_ACK_POLICY="$_ENV_CONSUMER_ACK_POLICY"
[[ -n "$_ENV_CONSUMER_BATCH_SIZE" ]] && CONSUMER_BATCH_SIZE="$_ENV_CONSUMER_BATCH_SIZE"
[[ -n "$_ENV_PREPOPULATE_COUNT" ]] && PREPOPULATE_COUNT="$_ENV_PREPOPULATE_COUNT"
[[ -n "$_ENV_PUBLISHER_GRACE_SECONDS" ]] && PUBLISHER_GRACE_SECONDS="$_ENV_PUBLISHER_GRACE_SECONDS"
[[ -n "$_ENV_SUBSCRIBER_DRAIN_SECONDS" ]] && SUBSCRIBER_DRAIN_SECONDS="$_ENV_SUBSCRIBER_DRAIN_SECONDS"
[[ -n "$_ENV_JS_SETUP_TIMEOUT_SECONDS" ]] && JS_SETUP_TIMEOUT_SECONDS="$_ENV_JS_SETUP_TIMEOUT_SECONDS"

RUN_TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
: "${TEST_ID:=${RUN_TIMESTAMP}}"
: "${TEST_DESCRIPTION:=}"
: "${TIMESTAMP_SUBFOLDER:=false}"

source "${ROOT_DIR}/scripts/preflight.sh"

: "${MESSAGING_TYPE:=Core NATS}"
: "${NATS_URL:=nats://127.0.0.1:4222}"
: "${SCENARIO:=pubsub}"
: "${SUBSCRIPTION_PATTERN:=pubsub}"
: "${QUEUE_NAME:=benchmark-core-group}"
: "${SUBJECT_DISTRIBUTION:=single}"
: "${MULTI_SUBJECT_MAX:=10000}"
: "${SUBJECT:=benchmark.core.test}"
: "${TARGET_MSGS_PER_SEC:=1000}"
: "${CONFIGURED_MESSAGE_COUNT:=}"
: "${MESSAGE_SIZE:=1024}"
: "${PUBLISHER_CONCURRENCY:=1}"
: "${SUBSCRIBER_CONCURRENCY:=1}"
: "${TEST_DURATION_SECONDS:=60}"
: "${METRICS_SAMPLE_INTERVAL:=5}"
: "${BASELINE_DURATION_SECONDS:=15}"
: "${NATS_CONTAINERS:=nats-1 nats-2 nats-3}"
: "${RESULTS_DIR:=results}"
: "${METRICS_BACKEND:=podman}"
: "${DISK_METRICS_BACKEND:=local}"
: "${NATS_DATA_BASE_DIR:=${ROOT_DIR}/../deploy/data}"

if [[ "$NATS_DATA_BASE_DIR" != /* ]]; then
  NATS_DATA_BASE_DIR="$(cd "${ROOT_DIR}/${NATS_DATA_BASE_DIR}" 2>/dev/null && pwd || echo "${ROOT_DIR}/${NATS_DATA_BASE_DIR}")"
fi

export DISK_METRICS_BACKEND
export NATS_DATA_BASE_DIR
: "${EXPECTED_NATS_NODES:=3}"
: "${NATS_MONITOR_URLS:=http://127.0.0.1:8222 http://127.0.0.1:8223 http://127.0.0.1:8224}"
: "${EXPECTED_REPLICAS:=3}"
: "${STREAM_NAME:=BENCH_STREAM}"
: "${STREAM_SUBJECT:=$SUBJECT}"
: "${STORAGE_TYPE:=file}"
: "${REPLICATION_FACTOR:=3}"
: "${MAX_BYTES:=5GB}"
: "${RETENTION_POLICY:=limits}"
: "${JS_PUB_MODE:=sync}"
: "${CONSUMER_NAME:=bench-consumer}"
: "${CONSUMER_ACK_POLICY:=explicit}"
: "${CONSUMER_BATCH_SIZE:=500}"
: "${PREPOPULATE_COUNT:=0}"

if [[ "$MESSAGING_TYPE" == "JetStream" ]]; then
  : "${PUBLISHER_GRACE_SECONDS:=20}"
  : "${SUBSCRIBER_DRAIN_SECONDS:=25}"
  : "${JS_SETUP_TIMEOUT_SECONDS:=15}"
else
  : "${PUBLISHER_GRACE_SECONDS:=5}"
  : "${SUBSCRIBER_DRAIN_SECONDS:=5}"
  : "${JS_SETUP_TIMEOUT_SECONDS:=15}"
fi

export PUBLISHER_GRACE_SECONDS
export SUBSCRIBER_DRAIN_SECONDS
export JS_SETUP_TIMEOUT_SECONDS

if [[ "$CHECK_ONLY" == true ]]; then
  preflight_checks
  exit $?
fi

die() {
  echo "ERROR: $*" >&2
  exit 1
}

case "$SCENARIO" in
  pubsub|publish|subscribe_setup|consume) ;;
  *) die "Invalid SCENARIO: $SCENARIO. Must be pubsub, publish, subscribe_setup, or consume" ;;
esac

case "$SUBSCRIPTION_PATTERN" in
  pubsub|queue) ;;
  *) die "Invalid SUBSCRIPTION_PATTERN: $SUBSCRIPTION_PATTERN. Must be pubsub or queue" ;;
esac

case "$SUBJECT_DISTRIBUTION" in
  single|multi) ;;
  *) die "Invalid SUBJECT_DISTRIBUTION: $SUBJECT_DISTRIBUTION. Must be single or multi" ;;
esac

positive() {
  [[ "$2" =~ ^[1-9][0-9]*$ ]] || die "$1 must be a positive integer (>= 1)"
}

non_negative() {
  [[ "$2" =~ ^[0-9]+$ ]] || die "$1 must be a non-negative integer (>= 0)"
}

for x in \
  "MESSAGE_SIZE:$MESSAGE_SIZE" \
  "TEST_DURATION_SECONDS:$TEST_DURATION_SECONDS" \
  "METRICS_SAMPLE_INTERVAL:$METRICS_SAMPLE_INTERVAL" \
  "BASELINE_DURATION_SECONDS:$BASELINE_DURATION_SECONDS"; do
  positive "${x%%:*}" "${x#*:}"
done

non_negative "TARGET_MSGS_PER_SEC" "$TARGET_MSGS_PER_SEC"
non_negative "PUBLISHER_CONCURRENCY" "$PUBLISHER_CONCURRENCY"
non_negative "SUBSCRIBER_CONCURRENCY" "$SUBSCRIBER_CONCURRENCY"

if [[ "$PUBLISHER_CONCURRENCY" -eq 0 && "$SUBSCRIBER_CONCURRENCY" -eq 0 ]]; then
  die "PUBLISHER_CONCURRENCY and SUBSCRIBER_CONCURRENCY cannot both be 0"
fi

# Auto-align SCENARIO when concurrency defines the workload pattern
if [[ "$PUBLISHER_CONCURRENCY" -gt 0 && "$SUBSCRIBER_CONCURRENCY" -eq 0 ]]; then
  if [[ "$SCENARIO" == "pubsub" ]]; then
    SCENARIO="publish"
  fi
elif [[ "$PUBLISHER_CONCURRENCY" -eq 0 && "$SUBSCRIBER_CONCURRENCY" -gt 0 ]]; then
  if [[ "$SCENARIO" == "pubsub" ]]; then
    if [[ "$MESSAGING_TYPE" == "JetStream" ]]; then
      SCENARIO="consume"
    else
      SCENARIO="subscribe_setup"
    fi
  fi
fi

case "$SCENARIO" in
  publish)
    if [[ "$PUBLISHER_CONCURRENCY" -eq 0 ]]; then
      die "SCENARIO 'publish' requires PUBLISHER_CONCURRENCY >= 1"
    fi
    SUBSCRIBER_CONCURRENCY=0
    ;;
  consume)
    if [[ "$SUBSCRIBER_CONCURRENCY" -eq 0 ]]; then
      die "SCENARIO 'consume' requires SUBSCRIBER_CONCURRENCY >= 1"
    fi
    PUBLISHER_CONCURRENCY=0
    ;;
  subscribe_setup)
    if [[ "$SUBSCRIBER_CONCURRENCY" -eq 0 ]]; then
      die "SCENARIO 'subscribe_setup' requires SUBSCRIBER_CONCURRENCY >= 1"
    fi
    PUBLISHER_CONCURRENCY=0
    ;;
  pubsub)
    positive "PUBLISHER_CONCURRENCY" "$PUBLISHER_CONCURRENCY"
    positive "SUBSCRIBER_CONCURRENCY" "$SUBSCRIBER_CONCURRENCY"
    ;;
esac

if [[ "$TARGET_MSGS_PER_SEC" -eq 0 ]]; then
  TOTAL_MESSAGES="${CONFIGURED_MESSAGE_COUNT:-100000}"
else
  TOTAL_MESSAGES="${CONFIGURED_MESSAGE_COUNT:-$((TARGET_MSGS_PER_SEC * TEST_DURATION_SECONDS))}"
fi
positive "TOTAL_MESSAGES" "$TOTAL_MESSAGES"

command -v nats >/dev/null || die "nats CLI not found"
command -v python3 >/dev/null || die "python3 not found"

if [[ "$METRICS_BACKEND" == "podman" ]]; then
  command -v podman >/dev/null || die "podman not found"
  command -v jq >/dev/null || die "jq not found"
elif [[ "$METRICS_BACKEND" != "none" ]]; then
  die "METRICS_BACKEND must be podman or none"
fi

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

cat > "$RESULT_DIR/test-config.txt" <<EOF
test_id=${TEST_ID}
test_description=${TEST_DESCRIPTION}
run_timestamp=${RUN_TIMESTAMP}
messaging_type=${MESSAGING_TYPE}
scenario=${SCENARIO}
subscription_pattern=${SUBSCRIPTION_PATTERN}
queue_name=${QUEUE_NAME}
subject_distribution=${SUBJECT_DISTRIBUTION}
multi_subject_max=${MULTI_SUBJECT_MAX}
nats_url=${NATS_URL}
subject=${SUBJECT}
target_publish_msgs_per_sec=${TARGET_MSGS_PER_SEC}
message_size_bytes=${MESSAGE_SIZE}
publisher_concurrency=${PUBLISHER_CONCURRENCY}
subscriber_concurrency=${SUBSCRIBER_CONCURRENCY}
nominal_duration_seconds=${TEST_DURATION_SECONDS}
configured_message_count=${TOTAL_MESSAGES}
metrics_backend=${METRICS_BACKEND}
disk_metrics_backend=${DISK_METRICS_BACKEND}
nats_data_base_dir=${NATS_DATA_BASE_DIR}
metrics_sample_interval_seconds=${METRICS_SAMPLE_INTERVAL}
nats_containers=${NATS_CONTAINERS}
baseline_duration_seconds=${BASELINE_DURATION_SECONDS}
stream_name=${STREAM_NAME}
storage_type=${STORAGE_TYPE}
replication_factor=${REPLICATION_FACTOR}
max_bytes=${MAX_BYTES}
retention_policy=${RETENTION_POLICY}
js_pub_mode=${JS_PUB_MODE}
consumer_name=${CONSUMER_NAME}
consumer_ack_policy=${CONSUMER_ACK_POLICY}
consumer_batch_size=${CONSUMER_BATCH_SIZE}
prepopulate_count=${PREPOPULATE_COUNT}
publisher_grace_seconds=${PUBLISHER_GRACE_SECONDS}
subscriber_drain_seconds=${SUBSCRIBER_DRAIN_SECONDS}
js_setup_timeout_seconds=${JS_SETUP_TIMEOUT_SECONDS}
EOF

# JetStream Lifecycle Setup before baseline/workload
if [[ "$MESSAGING_TYPE" == "JetStream" ]]; then
  echo "Setting up JetStream stream $STREAM_NAME (storage=$STORAGE_TYPE, R=$REPLICATION_FACTOR)..."
  bash "${ROOT_DIR}/scripts/js_stream.sh" setup \
    "$NATS_URL" \
    "$STREAM_NAME" \
    "$SUBJECT" \
    "$STORAGE_TYPE" \
    "$REPLICATION_FACTOR" \
    "$MAX_BYTES" \
    "$RETENTION_POLICY" \
    "$CONSUMER_NAME" \
    "$CONSUMER_ACK_POLICY"

  if [[ "$SCENARIO" == "consume" && "$PREPOPULATE_COUNT" -gt 0 ]]; then
    bash "${ROOT_DIR}/scripts/js_stream.sh" prepopulate \
      "$NATS_URL" \
      "$STREAM_NAME" \
      "$SUBJECT" \
      "$PREPOPULATE_COUNT" \
      "$MESSAGE_SIZE"
  fi
fi

export MESSAGING_TYPE STREAM_NAME JS_PUB_MODE CONSUMER_NAME CONSUMER_ACK_POLICY CONSUMER_BATCH_SIZE

METRICS_PID=""

cleanup() {
  if [[ -n "$METRICS_PID" ]] && kill -0 "$METRICS_PID" 2>/dev/null; then
    kill -TERM "$METRICS_PID" 2>/dev/null || true
    wait "$METRICS_PID" 2>/dev/null || true
  fi
}

interrupted() {
  echo
  echo "Benchmark interrupted by user."
  if [[ -n "${RESULT_DIR:-}" && -d "$RESULT_DIR" ]]; then
    echo "INTERRUPTED" > "$RESULT_DIR/workload_outcome.txt"
  fi
  cleanup
  exit 130
}

trap interrupted INT TERM
trap cleanup EXIT

# Baseline collection
if [[ "$METRICS_BACKEND" == "podman" ]]; then
  "${ROOT_DIR}/scripts/metrics.sh" \
    "$RESULT_DIR/metrics.csv" \
    "$METRICS_SAMPLE_INTERVAL" \
    baseline $NATS_CONTAINERS &

  METRICS_PID=$!
  sleep "$BASELINE_DURATION_SECONDS"

  kill -TERM "$METRICS_PID" 2>/dev/null || true
  wait "$METRICS_PID" 2>/dev/null || true
  METRICS_PID=""

  # Workload metric collection
  "${ROOT_DIR}/scripts/metrics.sh" \
    "$RESULT_DIR/metrics.csv" \
    "$METRICS_SAMPLE_INTERVAL" \
    workload $NATS_CONTAINERS &

  METRICS_PID=$!
  sleep 2
fi

# Execute workload
START="$(date +%s)"

set +e
if [[ "$SCENARIO" == "subscribe_setup" ]]; then
  bash "${ROOT_DIR}/scripts/sub_setup.sh" \
    "$NATS_URL" \
    "$SUBJECT" \
    "$SUBSCRIBER_CONCURRENCY" \
    "$TEST_DURATION_SECONDS" \
    "$RESULT_DIR" \
    2>&1 | tee "$RESULT_DIR/workload.log"
  WORKLOAD_EXIT=${PIPESTATUS[0]}
else
  bash "${ROOT_DIR}/scripts/workload.sh" \
    "$NATS_URL" \
    "$SUBJECT" \
    "$PUBLISHER_CONCURRENCY" \
    "$SUBSCRIBER_CONCURRENCY" \
    "$TOTAL_MESSAGES" \
    "$MESSAGE_SIZE" \
    "$TARGET_MSGS_PER_SEC" \
    "$RESULT_DIR/subscriber.log" \
    "$SCENARIO" \
    "$SUBSCRIPTION_PATTERN" \
    "$QUEUE_NAME" \
    "$SUBJECT_DISTRIBUTION" \
    "$MULTI_SUBJECT_MAX" \
    "$TEST_DURATION_SECONDS" \
    "$RESULT_DIR" \
    2>&1 | tee "$RESULT_DIR/workload.log"
  WORKLOAD_EXIT=${PIPESTATUS[0]}
fi
set -e

END="$(date +%s)"
ACTUAL_DURATION=$((END - START))

(( ACTUAL_DURATION > 0 )) || ACTUAL_DURATION=1

# Stop metric collection
if [[ -n "$METRICS_PID" ]]; then
  sleep 1
  kill -TERM "$METRICS_PID" 2>/dev/null || true
  wait "$METRICS_PID" 2>/dev/null || true
  METRICS_PID=""
fi

# Determine formal test outcome: COMPLETED, TIMEOUT, FAILED, or INTERRUPTED
if [[ -f "$RESULT_DIR/workload_outcome.txt" ]]; then
  TEST_OUTCOME="$(cat "$RESULT_DIR/workload_outcome.txt" | tr -d '[:space:]')"
elif [[ "$WORKLOAD_EXIT" -eq 0 ]]; then
  TEST_OUTCOME="COMPLETED"
elif [[ "$WORKLOAD_EXIT" -eq 130 ]]; then
  TEST_OUTCOME="INTERRUPTED"
else
  TEST_OUTCOME="FAILED"
fi

# Aggregate results
python3 "${ROOT_DIR}/scripts/aggregate.py" \
  "$RESULT_DIR" \
  "$TEST_ID" \
  "$SUBJECT" \
  "$TARGET_MSGS_PER_SEC" \
  "$MESSAGE_SIZE" \
  "$PUBLISHER_CONCURRENCY" \
  "$SUBSCRIBER_CONCURRENCY" \
  "$TEST_DURATION_SECONDS" \
  "$ACTUAL_DURATION" \
  "$WORKLOAD_EXIT" \
  "$METRICS_BACKEND" \
  "$TEST_OUTCOME"

echo ""
echo "===================================="
echo "Benchmark Summary"
echo "===================================="
echo "Test ID     : $TEST_ID"
echo "Outcome     : $TEST_OUTCOME"
echo "Duration    : ${ACTUAL_DURATION}s (nominal: ${TEST_DURATION_SECONDS}s)"
echo "Exit Code   : $WORKLOAD_EXIT"
echo "Results     : $RESULT_DIR"
echo "Report HTML : $RESULT_DIR/report.html"
echo "===================================="

exit "$WORKLOAD_EXIT"