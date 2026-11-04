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
_ENV_BASELINE_DURATION_SECONDS="${BASELINE_DURATION_SECONDS:-}"
_ENV_METRICS_SAMPLE_INTERVAL="${METRICS_SAMPLE_INTERVAL:-}"
_ENV_NATS_CONTAINERS="${NATS_CONTAINERS:-}"

# Load environment first, then workload-specific settings, then test identification.
source "$ENVIRONMENT_CONFIG"
source "$WORKLOAD_CONFIG"

if [[ -f "$TEST_CONFIG" ]]; then
  source "$TEST_CONFIG"
fi

# Re-apply explicit environment overrides
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
[[ -n "$_ENV_BASELINE_DURATION_SECONDS" ]] && BASELINE_DURATION_SECONDS="$_ENV_BASELINE_DURATION_SECONDS"
[[ -n "$_ENV_METRICS_SAMPLE_INTERVAL" ]] && METRICS_SAMPLE_INTERVAL="$_ENV_METRICS_SAMPLE_INTERVAL"
[[ -n "$_ENV_NATS_CONTAINERS" ]] && NATS_CONTAINERS="$_ENV_NATS_CONTAINERS"

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
: "${EXPECTED_NATS_NODES:=3}"
: "${NATS_MONITOR_URLS:=http://127.0.0.1:8222 http://127.0.0.1:8223 http://127.0.0.1:8224}"
: "${EXPECTED_REPLICAS:=3}"

if [[ "$CHECK_ONLY" == true ]]; then
  preflight_checks
  exit $?
fi

die() {
  echo "ERROR: $*" >&2
  exit 1
}

case "$SCENARIO" in
  pubsub|publish|subscribe_setup) ;;
  *) die "Invalid SCENARIO: $SCENARIO. Must be pubsub, publish, or subscribe_setup" ;;
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
    SCENARIO="subscribe_setup"
  fi
fi

case "$SCENARIO" in
  publish)
    if [[ "$PUBLISHER_CONCURRENCY" -eq 0 ]]; then
      die "SCENARIO 'publish' requires PUBLISHER_CONCURRENCY >= 1"
    fi
    SUBSCRIBER_CONCURRENCY=0
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

if [[ "$TIMESTAMP_SUBFOLDER" == true ]]; then
  RESULT_DIR="${ROOT_DIR}/${RESULTS_DIR}/${TEST_ID}/${RUN_TIMESTAMP}"
else
  RESULT_DIR="${ROOT_DIR}/${RESULTS_DIR}/${TEST_ID}"
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
metrics_sample_interval_seconds=${METRICS_SAMPLE_INTERVAL}
nats_containers=${NATS_CONTAINERS}
baseline_duration_seconds=${BASELINE_DURATION_SECONDS}
EOF

METRICS_PID=""

cleanup() {
  if [[ -n "$METRICS_PID" ]] && kill -0 "$METRICS_PID" 2>/dev/null; then
    kill -TERM "$METRICS_PID" 2>/dev/null || true
    wait "$METRICS_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

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
  "$METRICS_BACKEND"

echo "Results     : $RESULT_DIR"
echo "Report HTML : $RESULT_DIR/report.html"

exit "$WORKLOAD_EXIT"