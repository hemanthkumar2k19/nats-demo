#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -lt 4 ]]; then
  echo "Usage: metrics.sh OUTPUT_CSV INTERVAL_SECONDS PHASE CONTAINER [CONTAINER ...]" >&2
  exit 2
fi

OUTPUT_CSV="$1"
INTERVAL="$2"
PHASE="$3"
shift 3
CONTAINERS=("$@")

command -v podman >/dev/null 2>&1 || { echo "ERROR: podman is required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required" >&2; exit 1; }

if [[ ! -f "$OUTPUT_CSV" ]]; then
  echo 'timestamp_utc,phase,container,cpu_percent,memory_usage,memory_percent,network_io,block_io' > "$OUTPUT_CSV"
fi

write_sample() {
  local container="$1" timestamp json
  timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  if ! json="$(podman stats --no-stream --format=json "$container" 2>/dev/null)"; then
    printf 'WARN: Unable to collect Podman stats for %s\n' "$container" >&2
    return 0
  fi

  # Podman emits a JSON array. Values are retained in their original units/format.
  printf '%s\n' "$json" | jq -r --arg ts "$timestamp" --arg phase "$PHASE" \
    '.[0] | [ $ts, $phase, (.name // .Name // ""), (.cpu_percent // .CPUPerc // ""), (.mem_usage // .MemUsage // ""), (.mem_percent // .MemPerc // ""), (.netio // .NetIO // ""), (.blocki // .BlockIO // "") ] | @csv' \
    >> "$OUTPUT_CSV"
}

trap 'exit 0' TERM INT
while true; do
  for container in "${CONTAINERS[@]}"; do
    write_sample "$container"
  done
  sleep "$INTERVAL"
done
