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

exec python3 -u - "$OUT" "$INTERVAL" "$PHASE" "$@" << 'EOF'
import sys
import os
import time
import json
import subprocess
import signal
import re
import csv
from datetime import datetime, timezone

out_file = sys.argv[1]
interval = float(sys.argv[2])
phase = sys.argv[3]
containers = sys.argv[4:]

running = True
def handle_term(signum, frame):
    global running
    running = False

signal.signal(signal.SIGTERM, handle_term)
signal.signal(signal.SIGINT, handle_term)

def parse_time(s):
    if not s:
        return 0.0
    parts = re.findall(r"([0-9.]+)\s*(h|ms|us|ns|m|s)", str(s))
    if parts:
        tot = 0.0
        for v, u in parts:
            f = float(v)
            if u == "h": tot += f * 3600.0
            elif u == "m": tot += f * 60.0
            elif u == "s": tot += f
            elif u == "ms": tot += f / 1000.0
            elif u == "us": tot += f / 1e6
            elif u == "ns": tot += f / 1e9
        return tot
    try:
        return float(str(s).rstrip("s"))
    except (ValueError, TypeError):
        return 0.0

def query_stats(names):
    try:
        cmd = ["podman", "stats", "--no-stream", "--format=json"] + names
        res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, check=True)
        return json.loads(res.stdout)
    except Exception:
        return []

header = [
    "timestamp_epoch",
    "timestamp_utc",
    "phase",
    "container",
    "cpu_percent",
    "memory_usage",
    "memory_percent",
    "network_io",
    "block_io"
]

if not os.path.exists(out_file) or os.path.getsize(out_file) == 0:
    with open(out_file, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(header)

prev_samples = {}
init_stats = query_stats(containers)
t_init = time.time()
for item in init_stats:
    c_name = item.get("name") or item.get("Name") or ""
    if c_name:
        prev_samples[c_name] = (t_init, parse_time(item.get("cpu_time") or item.get("CPUTime")))

while running:
    elapsed_sleep = 0.0
    while running and elapsed_sleep < interval:
        step = min(0.5, interval - elapsed_sleep)
        time.sleep(step)
        elapsed_sleep += step

    if not running:
        break

    now_epoch = int(time.time())
    now_utc = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    now_perf = time.time()

    curr_stats = query_stats(containers)
    rows_to_write = []

    for item in curr_stats:
        c_name = item.get("name") or item.get("Name") or ""
        if not c_name:
            continue

        cpu_time_sec = parse_time(item.get("cpu_time") or item.get("CPUTime"))

        if c_name in prev_samples:
            t_prev, cpu_prev = prev_samples[c_name]
            dt = now_perf - t_prev
            dc = cpu_time_sec - cpu_prev
            if dt > 0 and dc >= 0:
                interval_cpu = (dc / dt) * 100.0
            else:
                interval_cpu = 0.0
        else:
            interval_cpu = 0.0

        prev_samples[c_name] = (now_perf, cpu_time_sec)

        cpu_str = f"{interval_cpu:.2f}%"
        mem_usage = item.get("mem_usage") or item.get("MemUsage") or ""
        mem_pct = item.get("mem_percent") or item.get("MemPerc") or ""
        net_io = item.get("net_io") or item.get("NetIO") or ""
        block_io = item.get("block_io") or item.get("BlockIO") or ""

        rows_to_write.append([
            now_epoch,
            now_utc,
            phase,
            c_name,
            cpu_str,
            mem_usage,
            mem_pct,
            net_io,
            block_io
        ])

    if rows_to_write:
        with open(out_file, "a", newline="") as f:
            writer = csv.writer(f)
            writer.writerows(rows_to_write)

EOF
