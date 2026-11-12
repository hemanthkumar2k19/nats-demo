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

check_storage_admission() {
  local storage_type="$1"
  local stream_count="$2"
  local max_bytes_str="$3"

  local check_code='
import sys, re, urllib.request, json

storage_type = sys.argv[1].lower()
stream_count = int(sys.argv[2])
max_bytes_str = sys.argv[3].strip()

mults = {"gb": 1024**3, "g": 1024**3, "mb": 1024**2, "m": 1024**2, "kb": 1024, "k": 1024, "b": 1}
m = re.match(r"^([0-9.]+)\s*([a-zA-Z]*)$", max_bytes_str)
if not m:
    sys.exit(0)
val = float(m.group(1))
unit = m.group(2).lower()
mult = mults.get(unit, 1)
per_stream_bytes = int(val * mult)
total_bytes = per_stream_bytes * stream_count

mon_hosts = ["http://127.0.0.1:8222", "http://localhost:8222"]
data = None
for h in mon_hosts:
    try:
        req = urllib.request.Request(h + "/jsz", headers={"User-Agent": "nats-capacity-bench"})
        with urllib.request.urlopen(req, timeout=2) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            break
    except Exception:
        continue

if not data or "config" not in data:
    sys.exit(0)

cfg = data["config"]
max_allowed = cfg.get("max_storage", 0) if storage_type == "file" else cfg.get("max_memory", 0)

if max_allowed > 0 and total_bytes > max_allowed:
    req_mb = total_bytes / (1024 * 1024)
    limit_mb = max_allowed / (1024 * 1024)
    print(f"FAIL: Storage admission check failed for {stream_count} stream(s)!", file=sys.stderr)
    print(f"  Quota per stream : {max_bytes_str} ({per_stream_bytes:,} bytes)", file=sys.stderr)
    print(f"  Total required   : {req_mb:,.1f} MB ({total_bytes:,} bytes)", file=sys.stderr)
    print(f"  Cluster node limit: {limit_mb:,.1f} MB ({storage_type})", file=sys.stderr)
    print(f"  Failure reason   : Requested reservation ({req_mb:,.1f} MB) exceeds cluster node limit ({limit_mb:,.1f} MB).", file=sys.stderr)
    print(f"  Remediation      : Set MAX_BYTES in suite config (e.g. MAX_BYTES=100MB) or increase max_{storage_type}_store in nats.conf.", file=sys.stderr)
    sys.exit(1)
'
  python3 -c "$check_code" "$storage_type" "$stream_count" "$max_bytes_str"
}

# JetStream Stream and Consumer Lifecycle Management Utility
# Usage:
#   js_stream.sh setup NATS_URL STREAM_NAME SUBJECT STORAGE REPLICAS MAX_BYTES RETENTION CONSUMER ACK_POLICY
#   js_stream.sh prepopulate NATS_URL STREAM_NAME SUBJECT COUNT SIZE
#   js_stream.sh purge NATS_URL STREAM_NAME
#   js_stream.sh cleanup NATS_URL STREAM_NAME
#   js_stream.sh admission STORAGE STREAM_COUNT MAX_BYTES

ACTION="${1:-}"
shift || true

case "$ACTION" in
  admission)
    STORAGE="${1:-file}"
    STREAM_COUNT="${2:-1}"
    MAX_BYTES="${3:-5GB}"
    check_storage_admission "$STORAGE" "$STREAM_COUNT" "$MAX_BYTES"
    ;;

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

    : "${STREAM_COUNT:=1}"
    : "${CONSUMER_COUNT:=0}"

    echo "=== [JetStream Lifecycle: Setup] ==="
    echo "Stream Name  : $STREAM_NAME"
    echo "Subject      : $SUBJECT"
    echo "Storage      : $STORAGE"
    echo "Replicas     : $REPLICAS"
    echo "Max Bytes    : $MAX_BYTES"
    echo "Retention    : $RETENTION"
    echo "Consumer     : $CONSUMER ($ACK_POLICY)"
    echo "Stream Count : $STREAM_COUNT"
    echo "Consumer Cnt : $CONSUMER_COUNT"

    # Preflight Storage and Memory Admission Validation
    if ! check_storage_admission "$STORAGE" "$STREAM_COUNT" "$MAX_BYTES"; then
      exit 1
    fi

    # Remove existing stream to guarantee fresh benchmark parameters
    if run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream info "$STREAM_NAME" >/dev/null 2>&1; then
      echo "Purging/removing existing stream $STREAM_NAME..."
      run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream rm "$STREAM_NAME" -f >/dev/null 2>&1 || true
    fi

    # Remove any leftover auxiliary streams from previous runs matching prefix
    base_prefix="${STREAM_NAME%_1}"
    existing_streams=$(nats --server "$NATS_URL" stream ls --names 2>/dev/null || true)
    for s in $existing_streams; do
      if [[ "$s" =~ ^${base_prefix}_[0-9]+$ ]]; then
        echo "Removing leftover stream $s..."
        run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream rm "$s" -f >/dev/null 2>&1 || true
      fi
    done

    echo "Creating JetStream stream $STREAM_NAME..."
    run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream add "$STREAM_NAME" \
      --subjects "$SUBJECT" \
      --storage "$STORAGE" \
      --replicas "$REPLICAS" \
      --max-bytes "$MAX_BYTES" \
      --retention "$RETENTION" \
      --discard old \
      --defaults

    # Multi-Stream Density Scaling:
    # If STREAM_COUNT > 1, create auxiliary streams (BENCH_STREAM_2 .. BENCH_STREAM_N)
    # to measure multi-Raft control-plane tax and idle cluster memory/CPU overhead.
    if [[ "$STREAM_COUNT" -gt 1 ]]; then
      echo "Creating $((STREAM_COUNT - 1)) auxiliary JetStream streams (${base_prefix}_2 .. ${base_prefix}_${STREAM_COUNT})..."
      for ((i=2; i<=STREAM_COUNT; i++)); do
        aux_stream="${base_prefix}_${i}"
        if ! run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" stream add "$aux_stream" \
          --subjects "bench.stream${i}.>" \
          --storage "$STORAGE" \
          --replicas "$REPLICAS" \
          --max-bytes "$MAX_BYTES" \
          --retention "$RETENTION" \
          --discard old \
          --defaults; then
          echo "ERROR: Failed to create auxiliary stream $aux_stream" >&2
          exit 1
        fi
      done
    fi

    # Consumer Setup:
    # If CONSUMER_COUNT >= 1 (e.g. consumer-fanout suite), create N durable pull consumers
    # (bench-consumer-1 .. bench-consumer-N) on the primary stream.
    # Otherwise, if CONSUMER is non-empty, create the default single durable consumer.
    if [[ "$CONSUMER_COUNT" -ge 1 ]]; then
      echo "Creating $CONSUMER_COUNT durable pull consumers on stream $STREAM_NAME..."
      for ((j=1; j<=CONSUMER_COUNT; j++)); do
        c_name="${CONSUMER}-${j}"
        run_with_timeout "$JS_SETUP_TIMEOUT_SECONDS" nats --server "$NATS_URL" consumer add "$STREAM_NAME" "$c_name" \
          --pull \
          --ack "$ACK_POLICY" \
          --deliver all \
          --defaults
      done
    elif [[ -n "$CONSUMER" ]]; then
      # Create a durable pull consumer for consumer benchmark scenarios (js-consume).
      # Pre-creating the durable consumer binds it to the stream with the specified
      # ack policy and delivery policy ('deliver all'), allowing 'nats bench js consume'
      # to immediately pull batches without requiring runtime consumer negotiation.
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

    # Remove any auxiliary streams matching base prefix
    base_prefix="${STREAM_NAME%_1}"
    existing_streams=$(nats --server "$NATS_URL" stream ls --names 2>/dev/null || true)
    for s in $existing_streams; do
      if [[ "$s" =~ ^${base_prefix}_[0-9]+$ ]]; then
        echo "Removing auxiliary stream $s..."
        nats --server "$NATS_URL" stream rm "$s" -f >/dev/null 2>&1 || true
      fi
    done
    ;;

  *)
    echo "Usage: js_stream.sh {setup|prepopulate|purge|cleanup} ..." >&2
    exit 2
    ;;
esac

