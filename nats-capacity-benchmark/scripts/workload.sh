#!/usr/bin/env bash
set -Eeuo pipefail

[[ $# -eq 8 ]] || {
  echo "Usage: workload.sh NATS_URL SUBJECT PUBS SUBS TOTAL_MESSAGES MESSAGE_SIZE TARGET_RATE SUBSCRIBER_LOG" >&2
  exit 2
}

NATS_URL="$1"
SUBJECT="$2"
PUBS="$3"
SUBS="$4"
TOTAL="$5"
SIZE="$6"
RATE="$7"
SUB_LOG="$8"

# Start subscribers first
nats --server "$NATS_URL" bench sub "$SUBJECT" \
  --msgs "$TOTAL" \
  --clients "$SUBS" \
  --no-progress \
  > "$SUB_LOG" 2>&1 &

SUB_PID=$!

cleanup() {
  if kill -0 "$SUB_PID" 2>/dev/null; then
    kill -TERM "$SUB_PID" 2>/dev/null || true
    wait "$SUB_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

# Allow subscriber clients to establish subscriptions
sleep 2

# Start publishers
nats --server "$NATS_URL" bench pub "$SUBJECT" \
  --msgs "$TOTAL" \
  --clients "$PUBS" \
  --size "$SIZE" \
  --throughput "$RATE" \
  --no-progress

# Wait for subscribers to finish
wait "$SUB_PID"

trap - EXIT INT TERM