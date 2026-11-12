#!/usr/bin/env bash
set -Eeuo pipefail
set -m

# NATS Capacity Benchmark - Streamlined Workload Runner
# Invokes 'nats bench' subcommands directly with native --csv and --duration flags.
#
# Calling Convention (Positional Arguments):
#   $1  : NATS_URL      - NATS server URL (e.g. nats://127.0.0.1:4222)
#   $2  : SUBJECT       - NATS subject name (e.g. benchmark.core.test)
#   $3  : PUBS          - Number of concurrent publisher clients
#   $4  : SUBS          - Number of concurrent subscriber clients
#   $5  : TOTAL         - Total messages to publish or consume
#   $6  : SIZE          - Message payload size with unit (e.g. 1024B, 8KB)
#   $7  : RATE          - Target publish rate in msgs/sec (0 for unthrottled)
#   $8  : DURATION      - Nominal test duration (e.g. 30s)
#   $9  : RESULT_DIR    - Directory where pub.csv, sub.csv, and outcome are saved
#   $10 : SCENARIO      - Workload scenario: pubsub | publish | subscribe |
#                         js-pub-sync | js-pub-async | js-consume
#   $11 : QUEUE         - Queue group name (if non-empty, subscribers form a queue group)
#   $12 : MULTISUBJECT  - Whether to randomize target subject (true/false)
#   $13 : MULTI_MAX     - Maximum subject suffix when MULTISUBJECT=true
#   $14 : STREAM        - JetStream stream name
#   $15 : CONSUMER      - JetStream consumer name
#   $16 : BATCH         - JetStream consumer batch size
#   $17 : ACKS          - JetStream ack policy (explicit, none, all)

NATS_URL="${1:-nats://127.0.0.1:4222}"
SUBJECT="${2:-benchmark.core.test}"
PUBS="${3:-1}"
SUBS="${4:-1}"
TOTAL="${5:-100000}"
SIZE="${6:-1024B}"
RATE="${7:-1000}"
DURATION="${8:-30s}"
RESULT_DIR="${9:-.}"
SCENARIO="${10:-pubsub}"
QUEUE="${11:-}"
MULTISUBJECT="${12:-false}"
MULTI_MAX="${13:-100000}"
STREAM="${14:-BENCH_STREAM}"
CONSUMER="${15:-bench-consumer}"
BATCH="${16:-500}"
ACKS="${17:-explicit}"

: "${PUB_SUBJECT:=${SUBJECT}}"
: "${SUB_SUBJECT:=${SUBJECT}}"
: "${MULTISUBJECT_RANDOM:=false}"
: "${STREAM_COUNT:=1}"
: "${CONSUMER_COUNT:=0}"

PUB_CSV="${RESULT_DIR}/pub.csv"
SUB_CSV="${RESULT_DIR}/sub.csv"
OUTCOME_FILE="${RESULT_DIR}/workload_outcome.txt"

PUB_FLAGS=(--no-progress --csv="$PUB_CSV")
SUB_FLAGS=(--no-progress --csv="$SUB_CSV")

if [[ "$MULTISUBJECT" == "true" ]]; then
  PUB_FLAGS+=(--multisubject --multisubjectmax="$MULTI_MAX")
  if [[ "$MULTISUBJECT_RANDOM" == "true" ]]; then
    PUB_FLAGS+=(--multisubjectrandomize)
  fi
  if [[ "$SUB_SUBJECT" != *">"* && "$SUB_SUBJECT" != *"*"* ]]; then
    SUB_FLAGS+=(--multisubject)
  fi
fi

SUB_PIDS=()
cleanup() {
  for pid in ${SUB_PIDS[@]+"${SUB_PIDS[@]}"}; do
    if kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
}
trap cleanup EXIT

interrupted() {
  echo "INTERRUPTED" > "$OUTCOME_FILE"
  cleanup
  exit 130
}
trap interrupted INT TERM

echo "Starting workload: scenario=$SCENARIO, pubs=$PUBS, subs=$SUBS, rate=$RATE, duration=$DURATION, size=$SIZE"

WORKLOAD_OUTCOME="COMPLETED"
PUB_EXIT=0

case "$SCENARIO" in
  pubsub)
    # Start subscribers first to ensure no published messages are missed.
    if [[ -n "$QUEUE" ]]; then
      # Queue group load-balancing pattern:
      # In a queue group, messages are distributed across workers rather than copied
      # to all of them. Each subscriber process requests its proportional share:
      # SHARE = TOTAL / SUBS. The aggregate message count received across all workers
      # will equal TOTAL (1-to-1 delivery semantics).
      SHARE=$((TOTAL / SUBS))
      (( SHARE > 0 )) || SHARE=1
      for ((i=1; i<=SUBS; i++)); do
        nats --server "$NATS_URL" sub --queue="$QUEUE" --count="$SHARE" --raw "$SUB_SUBJECT" > /dev/null 2>&1 &
        SUB_PIDS+=($!)
      done
    else
      # Broadcast fan-out pattern:
      # Without a queue group, every subscriber receives a full copy of every message.
      # The aggregate messages received across all subscribers will equal TOTAL * SUBS.
      nats --server "$NATS_URL" bench sub "$SUB_SUBJECT" \
        --clients "$SUBS" \
        --msgs "$TOTAL" \
        --size "$SIZE" \
        ${SUB_FLAGS[@]+"${SUB_FLAGS[@]}"} > /dev/null 2>&1 &
      SUB_PIDS+=($!)
    fi
    sleep 1

    # Run publisher with native duration and csv
    PUB_ARGS=(
      nats --server "$NATS_URL" bench pub "$PUB_SUBJECT"
      --clients "$PUBS"
      --size "$SIZE"
      ${PUB_FLAGS[@]+"${PUB_FLAGS[@]}"}
    )
    if [[ "$RATE" -gt 0 ]]; then
      PUB_ARGS+=(--throughput "$RATE" --duration "$DURATION")
    else
      PUB_ARGS+=(--msgs "$TOTAL")
    fi

    set +e
    "${PUB_ARGS[@]}"
    PUB_EXIT=$?
    set -e
    ;;

  publish)
    PUB_ARGS=(
      nats --server "$NATS_URL" bench pub "$PUB_SUBJECT"
      --clients "$PUBS"
      --size "$SIZE"
      ${PUB_FLAGS[@]+"${PUB_FLAGS[@]}"}
    )
    if [[ "$RATE" -gt 0 ]]; then
      PUB_ARGS+=(--throughput "$RATE" --duration "$DURATION")
    else
      PUB_ARGS+=(--msgs "$TOTAL")
    fi

    set +e
    "${PUB_ARGS[@]}"
    PUB_EXIT=$?
    set -e
    ;;

  subscribe)
    nats --server "$NATS_URL" bench sub "$SUB_SUBJECT" \
      --clients "$SUBS" \
      --msgs "$TOTAL" \
      --size "$SIZE" \
      ${SUB_FLAGS[@]+"${SUB_FLAGS[@]}"}
    PUB_EXIT=$?
    ;;

  js-pub-sync|js-pub-async)
    MODE="sync"
    [[ "$SCENARIO" == "js-pub-async" ]] && MODE="async"

    # In consumer fan-out benchmark scenarios, launch background JetStream consumers
    # to acknowledge messages concurrently with the publisher, capturing real-time
    # disk write IOPS and Raft consensus tax of consumer fan-out.
    if [[ "$CONSUMER_COUNT" -gt 0 ]]; then
      echo "Launching $CONSUMER_COUNT concurrent background JetStream consumer(s)..."
      for ((j=1; j<=CONSUMER_COUNT; j++)); do
        c_name="${CONSUMER}-${j}"
        nats --server "$NATS_URL" bench js consume \
          --stream "$STREAM" \
          --consumer "$c_name" \
          --clients 1 \
          --batch "$BATCH" \
          --acks "$ACKS" \
          --msgs "$TOTAL" \
          --no-progress > /dev/null 2>&1 &
        SUB_PIDS+=($!)
      done
      sleep 1
    fi

    PUB_ARGS=(
      nats --server "$NATS_URL" bench js pub "$MODE" "$PUB_SUBJECT"
      --stream "$STREAM"
      --clients "$PUBS"
      --size "$SIZE"
      ${PUB_FLAGS[@]+"${PUB_FLAGS[@]}"}
    )
    if [[ "$RATE" -gt 0 ]]; then
      PUB_ARGS+=(--throughput "$RATE" --duration "$DURATION")
    else
      PUB_ARGS+=(--msgs "$TOTAL")
    fi

    set +e
    "${PUB_ARGS[@]}"
    PUB_EXIT=$?
    set -e
    ;;

  js-consume)
    # Consumer drain benchmark:
    # Measures how quickly JetStream consumers can pull and acknowledge messages.
    # Note: This scenario does not launch a publisher. Messages must already exist
    # in the target stream (e.g. pre-populated via js_stream.sh prepopulate or by
    # a preceding js-pub benchmark).
    set +e
    nats --server "$NATS_URL" bench js consume \
      --stream "$STREAM" \
      --consumer "$CONSUMER" \
      --clients "$SUBS" \
      --batch "$BATCH" \
      --acks "$ACKS" \
      --msgs "$TOTAL" \
      ${SUB_FLAGS[@]+"${SUB_FLAGS[@]}"}
    PUB_EXIT=$?
    set -e
    ;;

  *)
    echo "ERROR: Unknown scenario: $SCENARIO" >&2
    exit 1
    ;;
esac

# Gracefully wait for background subscribers to finish processing in-flight messages (up to 15s)
if [[ "${#SUB_PIDS[@]}" -gt 0 ]]; then
  echo "Waiting for subscribers to drain in-flight messages..."
  DRAIN_WAIT=0
  while [[ "$DRAIN_WAIT" -lt 15 ]]; do
    ALL_DONE=true
    for spid in "${SUB_PIDS[@]}"; do
      if kill -0 "$spid" 2>/dev/null; then
        ALL_DONE=false
        break
      fi
    done
    [[ "$ALL_DONE" == true ]] && break
    sleep 1
    (( DRAIN_WAIT++ ))
  done

  # Terminate any subscribers still hanging after drain deadline
  for spid in "${SUB_PIDS[@]}"; do
    if kill -0 "$spid" 2>/dev/null; then
      kill -TERM "$spid" 2>/dev/null || true
      wait "$spid" 2>/dev/null || true
    fi
  done
fi

if [[ "$PUB_EXIT" -eq 0 ]]; then
  WORKLOAD_OUTCOME="COMPLETED"
elif [[ "$PUB_EXIT" -eq 130 ]]; then
  WORKLOAD_OUTCOME="INTERRUPTED"
else
  WORKLOAD_OUTCOME="FAILED"
fi

echo "$WORKLOAD_OUTCOME" > "$OUTCOME_FILE"
exit "$PUB_EXIT"
