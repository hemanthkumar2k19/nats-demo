#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 7 ]]; then
  echo "Usage: workload.sh NATS_URL SUBJECT PUBS SUBS TOTAL_MESSAGES MESSAGE_SIZE TARGET_MSGS_PER_SEC" >&2
  exit 2
fi

NATS_URL="$1"
SUBJECT="$2"
PUBS="$3"
SUBS="$4"
TOTAL_MESSAGES="$5"
MESSAGE_SIZE="$6"
TARGET_MSGS_PER_SEC="$7"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Start subscribers first so Core NATS messages are not published before
# subscriptions are active. Each subscriber receives the published messages
# (fan-out semantics); --clients controls concurrent subscriber clients.
SUBSCRIBER_LOG="${WORKLOAD_SUBSCRIBER_LOG:-${ROOT_DIR}/subscriber.log}"

nats --server "$NATS_URL" bench sub "$SUBJECT" \
  --msgs "$TOTAL_MESSAGES" \
  --clients "$SUBS" \
  --no-progress \
  > "$SUBSCRIBER_LOG" 2>&1 &
SUBSCRIBER_PID=$!

cleanup() {
  if kill -0 "$SUBSCRIBER_PID" 2>/dev/null; then
    kill -TERM "$SUBSCRIBER_PID" 2>/dev/null || true
    wait "$SUBSCRIBER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

# Allow the subscriber benchmark to establish all subscriptions.
sleep 2

# The target throughput is aggregate across all publisher clients.
nats --server "$NATS_URL" bench pub "$SUBJECT" \
  --msgs "$TOTAL_MESSAGES" \
  --clients "$PUBS" \
  --size "$MESSAGE_SIZE" \
  --throughput "$TARGET_MSGS_PER_SEC" \
  --no-progress

# Wait for all subscribers to receive the expected message count.
wait "$SUBSCRIBER_PID"
trap - EXIT INT TERM
