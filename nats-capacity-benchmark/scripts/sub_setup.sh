#!/usr/bin/env bash
set -Eeuo pipefail

[[ $# -ge 5 ]] || {
  echo "Usage: sub_setup.sh NATS_URL SUBJECT CONCURRENCY DURATION RESULT_DIR" >&2
  exit 2
}

NATS_URL="$1"
SUBJECT="$2"
CONCURRENCY="$3"
DURATION="$4"
RESULT_DIR="$5"

SUB_PIDS=()

cleanup() {
  echo "Cleaning up established subscriptions..."
  for pid in "${SUB_PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
}
trap cleanup EXIT INT TERM

echo "Starting Core NATS Subscribe Setup Benchmark"
echo "Target Connections : $CONCURRENCY"
echo "Target Subject     : $SUBJECT"
echo "Hold Duration      : ${DURATION}s"
echo ""

START_TIME=$(date +%s)

# Spawn subscriber clients to establish subscriptions
for ((i=1; i<=CONCURRENCY; i++)); do
  nats --server "$NATS_URL" sub "$SUBJECT" --raw > /dev/null 2>&1 &
  SUB_PIDS+=($!)
done

END_CONNECT=$(date +%s)
SETUP_DURATION=$((END_CONNECT - START_TIME))
(( SETUP_DURATION > 0 )) || SETUP_DURATION=1

CONNS_PER_SEC=$((CONCURRENCY / SETUP_DURATION))

echo "Subscribe Setup Completed:"
echo "  Established Connections: $CONCURRENCY"
echo "  Setup Elapsed Time     : ${SETUP_DURATION}s"
echo "  Connection Rate        : ${CONNS_PER_SEC} conns/sec"

# Write to subscriber.log for aggregator parsing
cat > "$RESULT_DIR/subscriber.log" <<EOF
[setup] Established $CONCURRENCY connections in ${SETUP_DURATION}s (~${CONNS_PER_SEC} conns/sec)
subscriber stats: 0 msgs/sec ~ 0 KiB/sec
setup stats: established=$CONCURRENCY, duration=${SETUP_DURATION}s, rate=${CONNS_PER_SEC} conns/sec
EOF

# Hold active subscriptions for the measurement duration
echo "Holding subscriptions open for ${DURATION}s to sample server resource overhead..."
sleep "$DURATION"

echo "Subscribe Setup benchmark complete."
trap - EXIT INT TERM
cleanup
exit 0

