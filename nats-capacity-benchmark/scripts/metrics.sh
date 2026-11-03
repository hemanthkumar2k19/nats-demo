#!/usr/bin/env bash
set -Eeuo pipefail

[[ $# -ge 4 ]] || {
  echo "Usage: metrics.sh OUTPUT_CSV INTERVAL_SECONDS PHASE CONTAINER..." >&2
  exit 2
}

OUT="$1"
INTERVAL="$2"
PHASE="$3"

shift 3

command -v podman >/dev/null || {
  echo "podman is required" >&2
  exit 1
}

command -v jq >/dev/null || {
  echo "jq is required" >&2
  exit 1
}

# CSV header
if [[ ! -f "$OUT" ]]; then
  echo 'timestamp_epoch,timestamp_utc,phase,container,cpu_percent,memory_usage,memory_percent,network_io,block_io' > "$OUT"
fi

trap 'exit 0' TERM INT

while true; do
  for name in "$@"; do

    TS="$(date +%s)"
    UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

    JSON="$(podman stats --no-stream --format=json "$name" 2>/dev/null || true)"

    [[ -n "$JSON" ]] || continue

    printf '%s\n' "$JSON" |
      jq -r \
        --arg ts "$TS" \
        --arg utc "$UTC" \
        --arg phase "$PHASE" \
        '
        .[0] |
        [
          $ts,
          $utc,
          $phase,
          (.name // .Name // ""),
          (.cpu_percent // .CPUPerc // ""),
          (.mem_usage // .MemUsage // ""),
          (.mem_percent // .MemPerc // ""),
          (.netio // .NetIO // ""),
          (.blocki // .BlockIO // "")
        ] | @csv
        ' >> "$OUT"

  done

  sleep "$INTERVAL"
done