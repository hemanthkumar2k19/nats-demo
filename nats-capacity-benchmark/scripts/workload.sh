#!/usr/bin/env bash
set -Eeuo pipefail

[[ $# -ge 8 ]] || {
  echo "Usage: workload.sh NATS_URL SUBJECT PUBS SUBS TOTAL_MESSAGES MESSAGE_SIZE TARGET_RATE SUBSCRIBER_LOG [SCENARIO PATTERN QUEUE_NAME SUBJECT_DIST MULTI_MAX]" >&2
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
SCENARIO="${9:-pubsub}"
PATTERN="${10:-pubsub}"
QUEUE_NAME="${11:-benchmark-core-group}"
SUBJECT_DIST="${12:-single}"
MULTI_MAX="${13:-10000}"

PUB_EXTRA_FLAGS=()
if [[ "$SUBJECT_DIST" == "multi" ]]; then
  PUB_EXTRA_FLAGS+=(--multisubject --multisubjectmax="$MULTI_MAX")
fi

SUB_EXTRA_FLAGS=()
if [[ "$SUBJECT_DIST" == "multi" ]]; then
  SUB_EXTRA_FLAGS+=(--multisubject)
fi

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
trap cleanup EXIT INT TERM

if [[ "$SCENARIO" == "pubsub" ]]; then
  if [[ "$PATTERN" == "queue" ]]; then
    # Queue Group: Subscribers load-balance messages
    echo "Starting Queue Group subscribers [group=$QUEUE_NAME, clients=$SUBS, total_msgs=$TOTAL, subject=$SUBJECT]" > "$SUB_LOG"
    SUB_START=$(date +%s)
    SHARE=$((TOTAL / SUBS))
    (( SHARE > 0 )) || SHARE=1
    for ((i=1; i<=SUBS; i++)); do
      nats --server "$NATS_URL" sub --queue="$QUEUE_NAME" --count="$SHARE" --raw "$SUBJECT" > /dev/null 2>&1 &
      SUB_PIDS+=($!)
    done
  else
    # Normal Pub/Sub: Each subscriber receives every message (Fan-out = SUBS)
    nats --server "$NATS_URL" bench sub "$SUBJECT" \
      --msgs "$((TOTAL * SUBS))" \
      --clients "$SUBS" \
      --size "${SIZE}B" \
      ${SUB_EXTRA_FLAGS[@]+"${SUB_EXTRA_FLAGS[@]}"} \
      --no-progress \
      > "$SUB_LOG" 2>&1 &
    SUB_PIDS+=($!)
  fi

  # Allow subscriber clients to establish subscriptions
  sleep 2
else
  # Publish-only scenario
  echo "Publish-only scenario: no subscriber clients active." > "$SUB_LOG"
fi

# Start publishers
nats --server "$NATS_URL" bench pub "$SUBJECT" \
  --msgs "$TOTAL" \
  --clients "$PUBS" \
  --size "$SIZE" \
  --throughput "$RATE" \
  ${PUB_EXTRA_FLAGS[@]+"${PUB_EXTRA_FLAGS[@]}"} \
  --no-progress

PUB_EXIT=$?

# Wait for subscribers if any
if [[ ${#SUB_PIDS[@]} -gt 0 ]]; then
  for pid in ${SUB_PIDS[@]+"${SUB_PIDS[@]}"}; do
    wait "$pid" 2>/dev/null || true
  done
  if [[ "$PATTERN" == "queue" ]]; then
    SUB_END=$(date +%s)
    SUB_DUR=$((SUB_END - SUB_START))
    (( SUB_DUR > 0 )) || SUB_DUR=1
    ACTUAL_SUB_RATE=$((TOTAL / SUB_DUR))
    BW_KIB=$(( (TOTAL * SIZE) / SUB_DUR / 1024 ))
    echo "NATS Core NATS subscriber stats: $ACTUAL_SUB_RATE msgs/sec ~ $BW_KIB KiB/sec (queue group: $QUEUE_NAME)" >> "$SUB_LOG"
  fi
fi

trap - EXIT INT TERM
exit "$PUB_EXIT"
