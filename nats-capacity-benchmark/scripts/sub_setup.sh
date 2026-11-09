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

OUTCOME_FILE="${RESULT_DIR}/workload_outcome.txt"
SUB_PIDS=()

cleanup() {
  if [[ ${#SUB_PIDS[@]} -gt 0 ]]; then
    for pid in ${SUB_PIDS[@]+"${SUB_PIDS[@]}"}; do
      if kill -0 "$pid" 2>/dev/null; then
        kill -TERM "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
      fi
    done
  fi
}

interrupted() {
  echo "INTERRUPTED" > "$OUTCOME_FILE"
  cleanup
  exit 130
}

trap interrupted INT TERM
trap cleanup EXIT

echo "Starting Core NATS Subscribe Setup Benchmark"
echo "Target Connections : $CONCURRENCY"
echo "Target Subject     : $SUBJECT"
echo "Hold Duration      : ${DURATION}"

# Clean duration if it has 's' suffix
CLEAN_DUR="${DURATION%s}"

START_TIME=$(date +%s)

for ((i=1; i<=CONCURRENCY; i++)); do
  nats --server "$NATS_URL" sub "$SUBJECT" --raw > /dev/null 2>&1 &
  SUB_PIDS+=($!)
done

END_CONNECT=$(date +%s)
SETUP_DURATION=$((END_CONNECT - START_TIME))
(( SETUP_DURATION > 0 )) || SETUP_DURATION=1

CONNS_PER_SEC=$((CONCURRENCY / SETUP_DURATION))

echo "Subscribe Setup Completed: ${CONCURRENCY} connections in ${SETUP_DURATION}s (~${CONNS_PER_SEC} conns/sec)"

# Write structured sub.csv
cat > "${RESULT_DIR}/sub.csv" <<EOF
#RunID,ClientID,MsgCount,MsgBytes,MsgsPerSec,BytesPerSec,DurationSecs
SETUP,S0,${CONCURRENCY},0,${CONNS_PER_SEC},0,${SETUP_DURATION}
EOF

sleep "$CLEAN_DUR"

echo "COMPLETED" > "$OUTCOME_FILE"
trap - EXIT INT TERM
cleanup
exit 0
