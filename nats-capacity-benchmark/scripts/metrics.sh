#!/usr/bin/env bash
set -Eeuo pipefail

# NATS Capacity Benchmark - Container Telemetry Collector
#
# Calling Convention:
#   metrics.sh <OUTPUT_JSON> <INTERVAL_SECONDS> <PHASE> <CONTAINER_NAME...>
#
# Arguments:
#   OUTPUT_JSON      : Target path where telemetry samples are appended (e.g. metrics.json)
#   INTERVAL_SECONDS : Sampling interval between stats queries (e.g. 5)
#   PHASE            : Benchmark phase tag: 'baseline' (pre-workload idle) or 'workload' (active run)
#   CONTAINER_NAME...: One or more container names to poll (e.g. nats-1 nats-2 nats-3)

[[ $# -ge 4 ]] || {
  echo "Usage: metrics.sh OUTPUT_FILE INTERVAL_SECONDS PHASE CONTAINER..." >&2
  exit 2
}

OUT="$1"
INTERVAL="$2"
PHASE="$3"
shift 3

exec python3 -u - "$OUT" "$INTERVAL" "$PHASE" "$@" << 'EOF'
import sys, os, time, json, subprocess, signal, re

out_file = sys.argv[1]
interval = float(sys.argv[2])
phase = sys.argv[3]
containers = sys.argv[4:]

disk_backend = os.environ.get("DISK_METRICS_BACKEND", "local")
data_base_dir = os.environ.get("NATS_DATA_BASE_DIR", "")

running = True
def handle_term(signum, frame):
    global running
    running = False

signal.signal(signal.SIGTERM, handle_term)
signal.signal(signal.SIGINT, handle_term)

def get_dir_size(path):
    total = 0
    if not os.path.isdir(path):
        return 0
    try:
        with os.scandir(path) as it:
            for entry in it:
                try:
                    if entry.is_file(follow_symlinks=False):
                        total += entry.stat(follow_symlinks=False).st_size
                    elif entry.is_dir(follow_symlinks=False):
                        total += get_dir_size(entry.path)
                except OSError:
                    pass
    except OSError:
        pass
    return total

def parse_bytes(s):
    if not s or not isinstance(s, str): return 0
    s = s.strip()
    units = [
        ("kib", 1024), ("mib", 1024**2), ("gib", 1024**3),
        ("kb", 1000), ("mb", 1000**2), ("gb", 1000**3),
        ("b", 1)
    ]
    for u, mult in units:
        if s.lower().endswith(u):
            try:
                return float(s[:-len(u)].strip()) * mult
            except ValueError:
                return 0
    try:
        return float(s)
    except ValueError:
        return 0

def parse_duration_seconds(s):
    if not s: return 0.0
    # Multi-character units must precede single-character units to prevent 'ms' from matching 'm' (minutes)
    parts = re.findall(r"([0-9.]+)\s*(ms|us|ns|h|m|s)", str(s))
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
    except Exception:
        return 0.0

def query_stats(names):
    try:
        cmd = ["podman", "stats", "--no-stream", "--format=json"] + names
        res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, check=True)
        return json.loads(res.stdout)
    except Exception:
        return []

all_samples = []
if os.path.exists(out_file):
    try:
        with open(out_file, "r") as f:
            all_samples = json.load(f)
    except Exception:
        all_samples = []

prev_state = {}

while running:
    now = time.time()
    stats = query_stats(containers)
    for c in stats:
        name = c.get("name") or c.get("id", "unknown")
        cpu_time = parse_duration_seconds(c.get("cpu_time", ""))

        mem_usage_str = str(c.get("mem_usage", "0B / 0B")).split("/")[0].strip()
        mem_bytes = parse_bytes(mem_usage_str)

        net_io = str(c.get("net_io", "0B / 0B")).split("/")
        net_rx_cum = parse_bytes(net_io[0].strip()) if len(net_io) > 0 else 0
        net_tx_cum = parse_bytes(net_io[1].strip()) if len(net_io) > 1 else 0

        block_io = str(c.get("block_io", "0B / 0B")).split("/")
        blk_r_cum = parse_bytes(block_io[0].strip()) if len(block_io) > 0 else 0
        blk_w_cum = parse_bytes(block_io[1].strip()) if len(block_io) > 1 else 0

        disk_bytes = 0
        if disk_backend == "local" and data_base_dir:
            node_dir = os.path.join(data_base_dir, name)
            disk_bytes = get_dir_size(node_dir)

        # Compute interval delta metrics:
        # Note: 'podman stats --no-stream' returns 'cpu_time' (cumulative CPU nanoseconds
        # consumed since container startup) and a lifetime average 'cpu_percent'.
        # For a short benchmark window (e.g. 30s), lifetime average masks active workload
        # load. To obtain accurate instantaneous rates, we sample cumulative counters at each
        # interval and calculate: delta_metric / delta_time.
        interval_cpu_pct = 0.0
        net_rx_mb_sec = 0.0
        net_tx_mb_sec = 0.0
        blk_r_mb_sec = 0.0
        blk_w_mb_sec = 0.0

        if name in prev_state:
            prev = prev_state[name]
            dt = max(0.001, now - prev["time"])
            dcpu = max(0.0, cpu_time - prev["cpu_time"])
            interval_cpu_pct = round((dcpu / dt) * 100.0, 2)

            dn_rx = max(0.0, net_rx_cum - prev["net_rx"])
            dn_tx = max(0.0, net_tx_cum - prev["net_tx"])
            net_rx_mb_sec = round(dn_rx / (1024 * 1024 * dt), 2)
            net_tx_mb_sec = round(dn_tx / (1024 * 1024 * dt), 2)

            db_r = max(0.0, blk_r_cum - prev["blk_r"])
            db_w = max(0.0, blk_w_cum - prev["blk_w"])
            blk_r_mb_sec = round(db_r / (1024 * 1024 * dt), 2)
            blk_w_mb_sec = round(db_w / (1024 * 1024 * dt), 2)
        else:
            # First sample fallback: parse lifetime cpu_percent as baseline seed
            cpu_str = str(c.get("cpu_percent", "0%")).rstrip("%")
            interval_cpu_pct = float(cpu_str) if cpu_str else 0.0

        prev_state[name] = {
            "time": now,
            "cpu_time": cpu_time,
            "net_rx": net_rx_cum,
            "net_tx": net_tx_cum,
            "blk_r": blk_r_cum,
            "blk_w": blk_w_cum
        }

        sample = {
            "timestamp": now,
            "phase": phase,
            "container": name,
            "cpu_percent": interval_cpu_pct,
            "mem_bytes": mem_bytes,
            "net_rx_mb_sec": net_rx_mb_sec,
            "net_tx_mb_sec": net_tx_mb_sec,
            "blk_read_mb_sec": blk_r_mb_sec,
            "blk_write_mb_sec": blk_w_mb_sec,
            "disk_bytes": disk_bytes
        }
        all_samples.append(sample)

    try:
        with open(out_file, "w") as f:
            json.dump(all_samples, f, indent=2)
    except Exception:
        pass

    start_sleep = time.time()
    while running and (time.time() - start_sleep < interval):
        time.sleep(0.5)

EOF
