#!/bin/bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${1:-${ROOT_DIR}/config/core-nats.conf}"

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "ERROR: Configuration file not found: $CONFIG_FILE" >&2
  exit 1
fi

# shellcheck source=/dev/null
source "$CONFIG_FILE"

: "${NATS_URL:=nats://127.0.0.1:4222}"
: "${SUBJECT:=benchmark.core.test}"
: "${TARGET_MSGS_PER_SEC:=1000}"
: "${MESSAGE_SIZE:=1024}"
: "${PUBLISHER_CONCURRENCY:=1}"
: "${CONSUMER_CONCURRENCY:=1}"
: "${TEST_DURATION_SECONDS:=60}"
: "${METRICS_BACKEND:=podman}"
: "${METRICS_SAMPLE_INTERVAL:=5}"
: "${BASELINE_DURATION_SECONDS:=15}"
: "${NATS_CONTAINERS:=nats-1 nats-2 nats-3}"
: "${RESULTS_DIR:=results}"

fail() { echo "ERROR: $*" >&2; exit 1; }
require_command() { command -v "$1" >/dev/null 2>&1 || fail "Required command not found: $1"; }
require_positive_integer() {
  local name="$1" value="$2"
  [[ "$value" =~ ^[1-9][0-9]*$ ]] || fail "$name must be a positive integer (got: $value)"
}

require_command nats
require_command date
require_command awk
require_command tee
require_positive_integer TARGET_MSGS_PER_SEC "$TARGET_MSGS_PER_SEC"
require_positive_integer MESSAGE_SIZE "$MESSAGE_SIZE"
require_positive_integer PUBLISHER_CONCURRENCY "$PUBLISHER_CONCURRENCY"
require_positive_integer CONSUMER_CONCURRENCY "$CONSUMER_CONCURRENCY"
require_positive_integer TEST_DURATION_SECONDS "$TEST_DURATION_SECONDS"
require_positive_integer METRICS_SAMPLE_INTERVAL "$METRICS_SAMPLE_INTERVAL"
require_positive_integer BASELINE_DURATION_SECONDS "$BASELINE_DURATION_SECONDS"

if [[ "$METRICS_BACKEND" == "podman" ]]; then
  require_command podman
  require_command jq
elif [[ "$METRICS_BACKEND" != "none" ]]; then
  fail "Unsupported METRICS_BACKEND '$METRICS_BACKEND'. Supported values: podman, none"
fi

TOTAL_MESSAGES=$((TARGET_MSGS_PER_SEC * TEST_DURATION_SECONDS))
TEST_ID="$(date -u +%Y%m%dT%H%M%SZ)"
RESULT_DIR="${ROOT_DIR}/${RESULTS_DIR}/${TEST_ID}"
mkdir -p "$RESULT_DIR"

METRICS_FILE="${RESULT_DIR}/metrics.csv"
WORKLOAD_LOG="${RESULT_DIR}/workload.log"
SUMMARY_FILE="${RESULT_DIR}/summary.csv"
METRICS_PID=""

cleanup() {
  if [[ -n "$METRICS_PID" ]] && kill -0 "$METRICS_PID" 2>/dev/null; then
    kill -TERM "$METRICS_PID" 2>/dev/null || true
    wait "$METRICS_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

cat > "${RESULT_DIR}/test-config.txt" <<EOF_CONFIG
Test ID: ${TEST_ID}
Started UTC: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Messaging type: Core NATS
NATS URL: ${NATS_URL}
Subject: ${SUBJECT}
Target publish throughput (aggregate): ${TARGET_MSGS_PER_SEC} messages/sec
Message size: ${MESSAGE_SIZE} bytes
Publisher concurrency: ${PUBLISHER_CONCURRENCY}
Consumer concurrency: ${CONSUMER_CONCURRENCY}
Test duration target: ${TEST_DURATION_SECONDS} seconds
Total messages configured: ${TOTAL_MESSAGES}
Metrics backend: ${METRICS_BACKEND}
Metrics sample interval: ${METRICS_SAMPLE_INTERVAL} seconds
EOF_CONFIG

printf '\nNATS Capacity Benchmark: %s\n' "$TEST_ID"
printf 'Target: %s msg/s | Payload: %s bytes | Publishers: %s | Consumers: %s | Duration: %ss\n' \
  "$TARGET_MSGS_PER_SEC" "$MESSAGE_SIZE" "$PUBLISHER_CONCURRENCY" "$CONSUMER_CONCURRENCY" "$TEST_DURATION_SECONDS"
printf 'Results: %s\n\n' "$RESULT_DIR"

if [[ "$METRICS_BACKEND" == "podman" ]]; then
  echo "Starting baseline metric collection (${BASELINE_DURATION_SECONDS}s)..."
  "${ROOT_DIR}/scripts/metrics.sh" "$METRICS_FILE" "$METRICS_SAMPLE_INTERVAL" baseline $NATS_CONTAINERS &
  METRICS_PID=$!
  sleep "$BASELINE_DURATION_SECONDS"
  # Restart collector in workload phase, preserving the same CSV file.
  kill -TERM "$METRICS_PID" 2>/dev/null || true
  wait "$METRICS_PID" 2>/dev/null || true
  METRICS_PID=""
  "${ROOT_DIR}/scripts/metrics.sh" "$METRICS_FILE" "$METRICS_SAMPLE_INTERVAL" workload $NATS_CONTAINERS &
  METRICS_PID=$!
  sleep 2
fi

printf 'Starting NATS benchmark...\n'
WORKLOAD_SUBSCRIBER_LOG="${RESULT_DIR}/subscriber.log" \
  "${ROOT_DIR}/scripts/workload.sh" "$NATS_URL" "$SUBJECT" "$PUBLISHER_CONCURRENCY" \
  "$CONSUMER_CONCURRENCY" "$TOTAL_MESSAGES" "$MESSAGE_SIZE" "$TARGET_MSGS_PER_SEC" \
  2>&1 | tee "$WORKLOAD_LOG"

if [[ -n "$METRICS_PID" ]]; then
  sleep 1
  kill -TERM "$METRICS_PID" 2>/dev/null || true
  wait "$METRICS_PID" 2>/dev/null || true
  METRICS_PID=""
fi

"${ROOT_DIR}/scripts/results.sh" "$RESULT_DIR" "$TEST_ID" "$SUBJECT" \
  "$TARGET_MSGS_PER_SEC" "$MESSAGE_SIZE" "$PUBLISHER_CONCURRENCY" \
  "$CONSUMER_CONCURRENCY" "$TEST_DURATION_SECONDS" "$TOTAL_MESSAGES" "$METRICS_BACKEND"

printf '\nBenchmark completed.\n'
printf 'Workload log : %s\n' "$WORKLOAD_LOG"
printf 'Metrics CSV  : %s\n' "$METRICS_FILE"
printf 'Summary CSV  : %s\n' "$SUMMARY_FILE"
