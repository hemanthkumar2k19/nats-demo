#!/usr/bin/env bash
set -Eeuo pipefail
set -m

[[ $# -ge 8 ]] || {
  echo "Usage: workload.sh NATS_URL SUBJECT PUBS SUBS TOTAL_MESSAGES MESSAGE_SIZE TARGET_RATE SUBSCRIBER_LOG [SCENARIO PATTERN QUEUE_NAME SUBJECT_DIST MULTI_MAX DURATION RESULT_DIR]" >&2
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
DURATION="${14:-60}"
RESULT_DIR="${15:-.}"

OUTCOME_FILE="${RESULT_DIR}/workload_outcome.txt"
PUB_LOG="${RESULT_DIR}/workload.log"

PUB_EXTRA_FLAGS=()
if [[ "$SUBJECT_DIST" == "multi" ]]; then
  PUB_EXTRA_FLAGS+=(--multisubject --multisubjectmax="$MULTI_MAX")
fi

SUB_EXTRA_FLAGS=()
if [[ "$SUBJECT_DIST" == "multi" ]]; then
  SUB_EXTRA_FLAGS+=(--multisubject)
fi

PUB_PID=""
WATCHDOG_PID=""
SUB_PIDS=()
WORKLOAD_OUTCOME="COMPLETED"

cleanup() {
  if [[ -n "$WATCHDOG_PID" ]] && kill -0 "$WATCHDOG_PID" 2>/dev/null; then
    kill -TERM "$WATCHDOG_PID" 2>/dev/null || true
    wait "$WATCHDOG_PID" 2>/dev/null || true
  fi
  if [[ -n "$PUB_PID" ]] && kill -0 "$PUB_PID" 2>/dev/null; then
    kill -TERM "$PUB_PID" 2>/dev/null || true
    wait "$PUB_PID" 2>/dev/null || true
  fi
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

: "${MESSAGING_TYPE:=Core NATS}"
: "${STREAM_NAME:=BENCH_STREAM}"
: "${JS_PUB_MODE:=sync}"
: "${CONSUMER_NAME:=bench-consumer}"
: "${CONSUMER_ACK_POLICY:=explicit}"
: "${CONSUMER_BATCH_SIZE:=500}"
: "${PUBLISHER_GRACE_SECONDS:=20}"
: "${SUBSCRIBER_DRAIN_SECONDS:=25}"

# Helper to launch and supervise time-bound publisher
run_time_bound_publisher() {
  local cmd=("$@")

  "${cmd[@]}" &
  PUB_PID=$!

  # Asynchronous Time-Bound Watchdog
  (
    sleep "$DURATION"
    if [[ -n "$PUB_PID" ]] && kill -0 "$PUB_PID" 2>/dev/null; then
      # Phase 1: Signal publisher gracefully via SIGINT to prompt stats finalization
      kill -INT "$PUB_PID" 2>/dev/null || true

      # Poll for graceful exit during first half of grace period
      HALF_GRACE=$(( PUBLISHER_GRACE_SECONDS / 2 ))
      (( HALF_GRACE > 2 )) || HALF_GRACE=2
      for ((k=0; k < HALF_GRACE * 2; k++)); do
        if ! kill -0 "$PUB_PID" 2>/dev/null; then
          break
        fi
        sleep 0.5
      done

      # Phase 2: If still running, send SIGTERM
      if kill -0 "$PUB_PID" 2>/dev/null; then
        kill -TERM "$PUB_PID" 2>/dev/null || true
        REM_GRACE=$(( PUBLISHER_GRACE_SECONDS - HALF_GRACE ))
        (( REM_GRACE > 2 )) || REM_GRACE=2
        for ((k=0; k < REM_GRACE * 2; k++)); do
          if ! kill -0 "$PUB_PID" 2>/dev/null; then
            break
          fi
          sleep 0.5
        done
      fi

      # Phase 3: Final escalation to SIGKILL only if process is completely unresponsive
      if kill -0 "$PUB_PID" 2>/dev/null; then
        echo "TIMEOUT" > "$OUTCOME_FILE"
        kill -KILL "$PUB_PID" 2>/dev/null || true
      fi
    fi
  ) &
  WATCHDOG_PID=$!

  set +e
  wait "$PUB_PID"
  PUB_EXIT=$?
  set -e

  # Clean up watchdog
  if [[ -n "$WATCHDOG_PID" ]] && kill -0 "$WATCHDOG_PID" 2>/dev/null; then
    kill -TERM "$WATCHDOG_PID" 2>/dev/null || true
    wait "$WATCHDOG_PID" 2>/dev/null || true
  fi
  WATCHDOG_PID=""
  PUB_PID=""

  # Exit code 130 (SIGINT) or 143 (SIGTERM) or 0 indicates time-bound completion
  if [[ "$PUB_EXIT" -eq 130 || "$PUB_EXIT" -eq 143 || "$PUB_EXIT" -eq 0 ]]; then
    PUB_EXIT=0
  else
    if [[ ! -f "$OUTCOME_FILE" || "$(cat "$OUTCOME_FILE")" != "TIMEOUT" ]]; then
      WORKLOAD_OUTCOME="FAILED"
    fi
  fi
}

if [[ "$MESSAGING_TYPE" == "JetStream" ]]; then
  # ----------------------------------------------------
  # JetStream Execution Path
  # ----------------------------------------------------
  if [[ "$SCENARIO" == "pubsub" || "$SCENARIO" == "consume" ]]; then
    echo "Starting JetStream durable pull consumers [stream=$STREAM_NAME, consumer=$CONSUMER_NAME, clients=$SUBS, batch=$CONSUMER_BATCH_SIZE, acks=$CONSUMER_ACK_POLICY]" > "$SUB_LOG"
    nats --server "$NATS_URL" bench js consume \
      --stream "$STREAM_NAME" \
      --consumer "$CONSUMER_NAME" \
      --clients "$SUBS" \
      --msgs "$TOTAL" \
      --batch "$CONSUMER_BATCH_SIZE" \
      --acks "$CONSUMER_ACK_POLICY" \
      --no-progress \
      >> "$SUB_LOG" 2>&1 &
    SUB_PIDS+=($!)
    sleep 2
  else
    echo "Publish-only scenario: no subscriber/consumer clients active." > "$SUB_LOG"
  fi

  if [[ "$SCENARIO" == "consume" ]]; then
    echo "Consume-only scenario: replaying from pre-populated stream."
    PUB_EXIT=0
  else
    run_time_bound_publisher \
      nats --server "$NATS_URL" bench js pub "$JS_PUB_MODE" "$SUBJECT" \
        --stream="$STREAM_NAME" \
        --msgs="$TOTAL" \
        --clients="$PUBS" \
        --size="$SIZE" \
        --throughput="$RATE" \
        ${PUB_EXTRA_FLAGS[@]+"${PUB_EXTRA_FLAGS[@]}"} \
        --no-progress
  fi
else
  # ----------------------------------------------------
  # Core NATS Execution Path
  # ----------------------------------------------------
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
        --msgs "$TOTAL" \
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

  # Start time-bound publisher
  run_time_bound_publisher \
    nats --server "$NATS_URL" bench pub "$SUBJECT" \
      --msgs "$TOTAL" \
      --clients "$PUBS" \
      --size "$SIZE" \
      --throughput "$RATE" \
      ${PUB_EXTRA_FLAGS[@]+"${PUB_EXTRA_FLAGS[@]}"} \
      --no-progress
fi

# Drain subscribers gracefully
if [[ ${#SUB_PIDS[@]} -gt 0 ]]; then
  # Instruct subscriber clients to finalize and output stats
  for pid in ${SUB_PIDS[@]+"${SUB_PIDS[@]}"}; do
    if kill -0 "$pid" 2>/dev/null; then
      kill -INT "$pid" 2>/dev/null || true
    fi
  done

  DRAIN_START=$(date +%s)
  TERM_THRESHOLD=$(( SUBSCRIBER_DRAIN_SECONDS * 3 / 4 ))
  (( TERM_THRESHOLD > 3 )) || TERM_THRESHOLD=3

  for pid in ${SUB_PIDS[@]+"${SUB_PIDS[@]}"}; do
    while kill -0 "$pid" 2>/dev/null; do
      NOW=$(date +%s)
      ELAPSED=$(( NOW - DRAIN_START ))
      if (( ELAPSED >= SUBSCRIBER_DRAIN_SECONDS )); then
        echo "Subscriber drain timeout reached (${SUBSCRIBER_DRAIN_SECONDS}s); terminating subscriber PID $pid" >> "$SUB_LOG"
        if [[ "$WORKLOAD_OUTCOME" == "COMPLETED" ]]; then
          WORKLOAD_OUTCOME="TIMEOUT"
        fi
        kill -KILL "$pid" 2>/dev/null || true
        break
      elif (( ELAPSED >= TERM_THRESHOLD )); then
        kill -TERM "$pid" 2>/dev/null || true
      fi
      sleep 0.5
    done
    wait "$pid" 2>/dev/null || true
  done

  if [[ "$MESSAGING_TYPE" != "JetStream" && "$PATTERN" == "queue" ]]; then
    SUB_END=$(date +%s)
    SUB_DUR=$((SUB_END - SUB_START))
    (( SUB_DUR > 0 )) || SUB_DUR=1
    ACTUAL_SUB_RATE=$((TOTAL / SUB_DUR))
    BW_KIB=$(( (TOTAL * SIZE) / SUB_DUR / 1024 ))
    echo "NATS Core NATS subscriber stats: $ACTUAL_SUB_RATE msgs/sec ~ $BW_KIB KiB/sec (queue group: $QUEUE_NAME)" >> "$SUB_LOG"
  fi
fi

# Write final outcome if not already set (e.g. by watchdog TIMEOUT or trap INTERRUPTED)
if [[ ! -f "$OUTCOME_FILE" ]]; then
  echo "$WORKLOAD_OUTCOME" > "$OUTCOME_FILE"
fi

trap - EXIT INT TERM
exit "$PUB_EXIT"
