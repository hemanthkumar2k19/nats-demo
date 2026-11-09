#!/usr/bin/env python3
"""
Streamlined results aggregator for NATS Capacity Benchmark.
Ingests native CSV outputs from nats bench and container telemetry JSON,
producing canonical results.json and a standalone, portable report.html.
"""

import csv
import json
import os
import re
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent

def parse_config(config_path):
    cfg = {}
    if not config_path.exists():
        return cfg
    for line in config_path.read_text(errors="replace").splitlines():
        line = line.strip()
        if "=" in line and not line.startswith("#"):
            k, v = line.split("=", 1)
            cfg[k.strip()] = v.strip().strip('"').strip("'")
    return cfg

def parse_bench_csv(csv_path):
    """
    Parses native CSV output produced by 'nats bench --csv'.
    Expected column headers produced by the NATS bench CLI include:
      - MsgCount, MsgBytes, MsgsPerSec, BytesPerSec
      - MinLatencyMicroSecs, AvgLatencyMicroSecs, MaxLatencyMicroSecs
      - MedianLatencyMicroSecs, P50LatencyMicroSecs, P90LatencyMicroSecs
      - P99LatencyMicroSecs, P99.9LatencyMicroSecs, StdDevLatencyMicroSecs
    """
    if not csv_path.exists():
        return None
    rows = []
    with open(csv_path, "r", encoding="utf-8", errors="replace") as f:
        lines = [line.lstrip("#") for line in f if line.strip()]
        if not lines:
            return None
        reader = csv.DictReader(lines)
        for r in reader:
            rows.append({k.strip(): float(v.strip()) for k, v in r.items() if k and v and k.strip() not in ("RunID", "ClientID")})
    return rows

def aggregate_pub_stats(rows):
    if not rows:
        return {}
    tot_msgs = sum(r.get("MsgCount", 0) for r in rows)
    tot_bytes = sum(r.get("MsgBytes", 0) for r in rows)
    tot_rate = sum(r.get("MsgsPerSec", 0) for r in rows)
    tot_bw = sum(r.get("BytesPerSec", 0) for r in rows) / (1024 * 1024)
    min_lat = min(r.get("MinLatencyMicroSecs", 0) for r in rows)
    max_lat = max(r.get("MaxLatencyMicroSecs", 0) for r in rows)
    p50_lat = sum(r.get("P50LatencyMicroSecs", 0) for r in rows) / len(rows)
    p90_lat = sum(r.get("P90LatencyMicroSecs", 0) for r in rows) / len(rows)
    p99_lat = max(r.get("P99LatencyMicroSecs", 0) for r in rows)
    p999_lat = max(r.get("P99.9LatencyMicroSecs", 0) for r in rows)
    avg_lat = sum(r.get("AvgLatencyMicroSecs", 0) * r.get("MsgCount", 0) for r in rows) / (tot_msgs or 1)
    std_dev = sum(r.get("StdDevLatencyMicroSecs", 0) for r in rows) / len(rows)

    return {
        "published_msgs": int(tot_msgs),
        "total_bytes": int(tot_bytes),
        "throughput_msgs_sec": round(tot_rate, 2),
        "bandwidth_mb_sec": round(tot_bw, 2),
        "latency_us": {
            "min": round(min_lat, 2),
            "avg": round(avg_lat, 2),
            "p50": round(p50_lat, 2),
            "p90": round(p90_lat, 2),
            "p99": round(p99_lat, 2),
            "p999": round(p999_lat, 2),
            "max": round(max_lat, 2),
            "std_dev": round(std_dev, 2)
        }
    }

def aggregate_sub_stats(rows):
    if not rows:
        return {}
    tot_msgs = sum(r.get("MsgCount", 0) for r in rows)
    tot_rate = sum(r.get("MsgsPerSec", 0) for r in rows)
    return {
        "received_msgs": int(tot_msgs),
        "consumer_rate_msgs_sec": round(tot_rate, 2)
    }

def aggregate_metrics(metrics_json_path):
    if not metrics_json_path.exists():
        return []
    try:
        samples = json.loads(metrics_json_path.read_text())
    except Exception:
        return []

    by_container = {}
    for s in samples:
        c = s.get("container", "unknown")
        if c not in by_container:
            by_container[c] = {"baseline": [], "workload": []}
        phase = s.get("phase", "workload")
        if phase in by_container[c]:
            by_container[c][phase].append(s)

    nodes = []
    for c, data in sorted(by_container.items()):
        wl = data["workload"]
        bl = data["baseline"]
        if not wl:
            wl = bl
        if not wl:
            continue

        cpus = [s.get("cpu_percent", 0.0) for s in wl]
        mems = [s.get("mem_bytes", 0) / (1024 * 1024) for s in wl]
        bl_mems = [s.get("mem_bytes", 0) / (1024 * 1024) for s in bl] if bl else mems

        base_mem = round(sum(bl_mems) / len(bl_mems), 1) if bl_mems else 0.0
        peak_mem = round(max(mems), 1) if mems else 0.0
        avg_mem = round(sum(mems) / len(mems), 1) if mems else 0.0
        mem_growth = round(max(0.0, peak_mem - base_mem), 1)

        net_rx = [s.get("net_rx_mb_sec", 0.0) for s in wl]
        net_tx = [s.get("net_tx_mb_sec", 0.0) for s in wl]
        blk_r = [s.get("blk_read_mb_sec", 0.0) for s in wl]
        blk_w = [s.get("blk_write_mb_sec", 0.0) for s in wl]

        init_disk = wl[0].get("disk_bytes", 0)
        final_disk = wl[-1].get("disk_bytes", 0)
        disk_growth = max(0, final_disk - init_disk)

        nodes.append({
            "name": c,
            "avg_cpu_pct": round(sum(cpus) / len(cpus), 2),
            "peak_cpu_pct": round(max(cpus), 2),
            "baseline_mem_mb": base_mem,
            "avg_mem_mb": avg_mem,
            "peak_mem_mb": peak_mem,
            "mem_growth_mb": mem_growth,
            "net_rx_mb_sec": round(sum(net_rx) / len(net_rx), 2) if net_rx else 0.0,
            "net_tx_mb_sec": round(sum(net_tx) / len(net_tx), 2) if net_tx else 0.0,
            "disk_read_mb_sec": round(sum(blk_r) / len(blk_r), 2) if blk_r else 0.0,
            "disk_write_mb_sec": round(sum(blk_w) / len(blk_w), 2) if blk_w else 0.0,
            "disk_growth_bytes": int(disk_growth)
        })
    return nodes

def main():
    if len(sys.argv) < 2:
        print("Usage: aggregate.py RESULT_DIR [TEST_ID]", file=sys.stderr)
        sys.exit(1)

    result_dir = Path(sys.argv[1]).resolve()
    test_id = sys.argv[2] if len(sys.argv) > 2 else result_dir.name

    cfg = parse_config(result_dir / "test-config.txt")
    test_id = cfg.get("test_id", test_id)

    outcome = "COMPLETED"
    outcome_file = result_dir / "workload_outcome.txt"
    if outcome_file.exists():
        outcome = outcome_file.read_text(errors="replace").strip() or "COMPLETED"

    pub_rows = parse_bench_csv(result_dir / "pub.csv")
    sub_rows = parse_bench_csv(result_dir / "sub.csv")
    client_stats = aggregate_pub_stats(pub_rows)
    sub_stats = aggregate_sub_stats(sub_rows)

    if sub_stats:
        client_stats.update(sub_stats)
        pub_msgs = client_stats.get("published_msgs", 0)
        rec_msgs = client_stats.get("received_msgs", 0)
        subs = int(cfg.get("subscriber_concurrency", 1) or 1)
        queue = cfg.get("queue_name", "")

        # Delivery Efficiency Calculation:
        # - Queue group pattern: messages are load-balanced across workers; each message
        #   is delivered to exactly one subscriber. Expected total received = pub_msgs.
        # - Fan-out pattern (no queue group): every subscriber receives every message.
        #   Expected total received = pub_msgs * subscriber_count.
        expected_rec = pub_msgs if queue else (pub_msgs * subs)
        client_stats["delivery_efficiency_pct"] = round((rec_msgs / expected_rec) * 100, 2) if expected_rec else 100.0

    nodes = aggregate_metrics(result_dir / "metrics.json")

    # Cluster-wide rollups
    avg_cluster_cpu = round(sum(n["avg_cpu_pct"] for n in nodes) / len(nodes), 2) if nodes else 0.0
    peak_cluster_cpu = max((n["peak_cpu_pct"] for n in nodes), default=0.0)
    total_peak_mem = round(sum(n["peak_mem_mb"] for n in nodes), 1) if nodes else 0.0
    total_mem_growth = round(sum(n["mem_growth_mb"] for n in nodes), 1) if nodes else 0.0

    achieved_rate = client_stats.get("throughput_msgs_sec", 0.0)
    cpu_eff = round(achieved_rate / (avg_cluster_cpu / 100.0), 2) if avg_cluster_cpu > 0 else 0.0

    cluster_rollup = {
        "avg_cpu_pct": avg_cluster_cpu,
        "peak_cpu_pct": peak_cluster_cpu,
        "total_peak_mem_mb": total_peak_mem,
        "total_mem_growth_mb": total_mem_growth,
        "cpu_efficiency_msgs_per_core": cpu_eff
    }

    results_data = {
        "test_id": test_id,
        "status": outcome,
        "metadata": {
            "test_description": cfg.get("test_description", ""),
            "scenario": cfg.get("scenario", "pubsub"),
            "target_rate": int(cfg.get("target_publish_msgs_per_sec", 0) or 0),
            "payload_size": cfg.get("message_size_bytes", "1024B"),
            "publisher_concurrency": int(cfg.get("publisher_concurrency", 1) or 1),
            "subscriber_concurrency": int(cfg.get("subscriber_concurrency", 1) or 1),
            "duration": cfg.get("nominal_duration_seconds", "30s"),
            "messaging_type": cfg.get("messaging_type", "Core NATS")
        },
        "client": client_stats,
        "cluster": cluster_rollup,
        "nodes": nodes
    }

    results_json_path = result_dir / "results.json"
    results_json_path.write_text(json.dumps(results_data, indent=2))

    # Inject into constant static/report.html to create self-contained offline report.html
    template_path = ROOT_DIR / "static" / "report.html"
    if template_path.exists():
        template_content = template_path.read_text(encoding="utf-8")
        sentinel_pattern = re.compile(
            r'<script\s+id=["\']benchmark-data["\']\s*>\s*window\.BENCHMARK_DATA\s*=\s*null;\s*</script>',
            re.MULTILINE
        )
        embed_script = f'<script id="benchmark-data">\n    window.BENCHMARK_DATA = {json.dumps(results_data)};\n  </script>'

        if sentinel_pattern.search(template_content):
            report_content = sentinel_pattern.sub(embed_script, template_content, count=1)
            (result_dir / "report.html").write_text(report_content, encoding="utf-8")
        else:
            print(f"WARNING: Sentinel pattern was not found in {template_path}. Offline report.html was not created.", file=sys.stderr)

    print(f"Results aggregated successfully:")
    print(f"  JSON Data   : {results_json_path}")
    print(f"  HTML Report : {result_dir / 'report.html'}")

if __name__ == "__main__":
    main()
