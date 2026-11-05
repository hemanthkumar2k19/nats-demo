#!/usr/bin/env bash
set -Eeuo pipefail
set -m

: "${JS_SETUP_TIMEOUT_SECONDS:=15}"

run_with_timeout() {
  local timeout_sec="$1"
  shift
  local cmd=("$@")

  "${cmd[@]}" &
  local pid=$!

  (
    sleep "$timeout_sec"
    if kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
      sleep 1
      kill -KILL "$pid" 2>/dev/null || true
    fi
  ) &
  local watchdog_pid=$!

  set +e
  wait "$pid"
  local exit_code=$?
  set -e

  kill -TERM "$watchdog_pid" 2>/dev/null || true
  wait "$watchdog_pid" 2>/dev/null || true

  return "$exit_code"
}

# JetStream Stream and Consumer Lifecycle Management Utility
# Usage:
#   js_stream.sh setup NATS_URL STREAM_NAME SUBJECT STORAGE REPLICAS MAX_BYTES RETENTION CONSUMER ACK_POLICY
#   js_stream.sh prepopulate NATS_URL STREAM_NAME SUBJECT COUNT SIZE
#   js_stream.sh purge NATS_URL STREAM_NAME
#   js_stream.sh cleanup NATS_URL STREAM_NAME

ACTION="${1:-}"
shift || true

case "$ACTION" in
  setup)
    NATS_URL="$1"
    STREAM_NAME="$2"
    SUBJECT="$3"
    STORAGE="${4:-file}"
    REPLICAS="${5:-3}"
    MAX_BYTES="${6:-5GB}"
    RETENTION="${7:-limits}"
    CONSUMER="${8:-bench-consumer}"
    ACK_POLICY="${9:-explicit}"

    echo "=== [JetStream Lifecycle: Setup] ==="
    echo "Stream Name  : $STREAM_NAME"
    echo "Subject      : $SUBJECT"
    echo "Storage      : $STORAGE"
    echo "Replicas     : $REPLICAS"
    echo "Max Bytes    : $MAX_BYTES"
    echo "Retention    : $RETENTION"
    echo "Consumer     : $CONSUMER ($ACK_POLICY)"

    # Remove existing stream to guarantee fresh benchmark parameters
    if run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream info "$STREAM_NAME" >/dev/null 2>&1; then
      echo "Purging/removing existing stream $STREAM_NAME..."
      run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream rm "$STREAM_NAME" -f >/dev/null 2>&1 || true
    fi

    echo "Creating JetStream stream $STREAM_NAME..."
    run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream add "$STREAM_NAME" \
      --subjects "$SUBJECT" \
      --storage "$STORAGE" \
      --replicas "$REPLICAS" \
      --max-bytes "$MAX_BYTES" \
      --retention "$RETENTION" \
      --discard old \
      --defaults

    if [[ -n "$CONSUMER" ]]; then
      echo "Creating durable pull consumer $CONSUMER on stream $STREAM_NAME..."
      run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" consumer add "$STREAM_NAME" "$CONSUMER" \
        --pull \
        --ack "$ACK_POLICY" \
        --deliver all \
        --defaults
    fi
    echo "JetStream setup completed successfully."
    ;;

  prepopulate)
    NATS_URL="$1"
    STREAM_NAME="$2"
    SUBJECT="$3"
    COUNT="$4"
    SIZE="$5"

    echo "=== [JetStream Lifecycle: Prepopulate] ==="
    echo "Prepopulating $COUNT messages (${SIZE}B) into $STREAM_NAME..."
    nats --server "$NATS_URL" bench js pub async "$SUBJECT" \
      --stream="$STREAM_NAME" \
      --msgs="$COUNT" \
      --size="${SIZE}B" \
      --no-progress
    echo "Prepopulation completed."
    ;;

  purge)
    NATS_URL="$1"
    STREAM_NAME="$2"
    echo "Purging messages from stream $STREAM_NAME..."
    nats --server "$NATS_URL" stream purge "$STREAM_NAME" -f >/dev/null 2>&1 || true
    ;;

  cleanup)
    NATS_URL="$1"
    STREAM_NAME="$2"
    echo "Removing stream $STREAM_NAME..."
    nats --server "$NATS_URL" stream rm "$STREAM_NAME" -f >/dev/null 2>&1 || true
    ;;

  *)
    echo "Usage: js_stream.sh {setup|prepopulate|purge|cleanup} ..." >&2
    exit 2
    ;;
esac

