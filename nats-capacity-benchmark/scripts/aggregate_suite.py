#!/usr/bin/env python3
"""
Streamlined suite aggregator for NATS Capacity Benchmark.
Aggregates child test runs into suite-results.json, computes executive sizing recommendations,
and generates canonical executive-report.html (and suite-report.html).

Output Schema (suite-results.json):
  - suite_id: Identifier for the benchmark suite (e.g. core-ladder)
  - suite_title: Human-readable title
  - suite_description: Purpose and test matrix details
  - recommendations: Sizing boundaries (sustained capacity, saturation knee, peak throughput, headroom)
  - tests: Array of individual test results.json objects, ordered by ascending target rate
"""

import json
import os
import re
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent

def safe_float(v, default=0.0):
    try:
        if v is None:
            return default
        return float(str(v).strip().rstrip("%"))
    except (ValueError, TypeError):
        return default

def safe_int(v, default=0):
    try:
        if v is None:
            return default
        return int(float(str(v).strip()))
    except (ValueError, TypeError):
        return default

def parse_payload_bytes(size_str):
    if not size_str:
        return 0
    s = str(size_str).strip().upper()
    mult = 1
    if s.endswith("KB") or s.endswith("KIB"):
        mult = 1024
        s = s.replace("KIB", "").replace("KB", "").strip()
    elif s.endswith("MB") or s.endswith("MIB"):
        mult = 1024 * 1024
        s = s.replace("MIB", "").replace("MB", "").strip()
    elif s.endswith("B"):
        s = s[:-1].strip()
    try:
        return int(float(s) * mult)
    except (ValueError, TypeError):
        return 0

def compute_recommendations(runs):
    if not runs:
        return {}

    peak_rate = max((safe_float((r.get("client") or {}).get("throughput_msgs_sec", 0.0)) for r in runs), default=0.0)
    cluster_peak_cpu = max((safe_float((r.get("cluster") or {}).get("peak_cpu_pct", 0.0)) for r in runs), default=0.0)
    cluster_peak_mem = max((safe_float((r.get("cluster") or {}).get("total_peak_mem_mb", 0.0)) for r in runs), default=0.0)

    total_disk_growth = 0
    for r in runs:
        for n in (r.get("nodes") or []):
            total_disk_growth += safe_int(n.get("disk_growth_bytes", 0))

    # Sustained boundary: highest achieved rate where delivery >= 99%, P99 < 15ms (15000us), CPU < 80%
    sustained = 0.0
    for r in runs:
        client = r.get("client") or {}
        cluster = r.get("cluster") or {}
        lat = client.get("latency_us") or {}
        p99 = safe_float(lat.get("p99", 0.0))
        eff = safe_float(client.get("delivery_efficiency_pct", 100.0), 100.0)
        cpu = safe_float(cluster.get("peak_cpu_pct", 0.0))
        rate = safe_float(client.get("throughput_msgs_sec", 0.0))
        if eff >= 99.0 and p99 < 15000.0 and cpu < 80.0:
            if rate > sustained:
                sustained = rate

    # Saturation knee: first step where p99 > 15ms or delivery < 99% or throughput drops below offered
    saturation_knee = peak_rate
    for r in runs:
        client = r.get("client") or {}
        lat = client.get("latency_us") or {}
        p99 = safe_float(lat.get("p99", 0.0))
        eff = safe_float(client.get("delivery_efficiency_pct", 100.0), 100.0)
        target = safe_int((r.get("metadata") or {}).get("target_rate", 0))
        achieved = safe_float(client.get("throughput_msgs_sec", 0.0))
        if eff < 99.0 or p99 >= 15000.0 or (target > 0 and achieved < target * 0.95):
            saturation_knee = achieved
            break

    headroom = round(max(0.0, 100.0 - cluster_peak_cpu), 1)

    return {
        "sustained_capacity_msgs_sec": round(sustained, 2),
        "peak_throughput_msgs_sec": round(peak_rate, 2),
        "saturation_knee_msgs_sec": round(saturation_knee, 2),
        "cluster_peak_cpu_pct": round(cluster_peak_cpu, 2),
        "cluster_peak_mem_mb": round(cluster_peak_mem, 1),
        "total_disk_growth_bytes": int(total_disk_growth),
        "cpu_headroom_pct": headroom
    }

def main():
    if len(sys.argv) < 2:
        print("Usage: aggregate_suite.py SUITE_DIR [SUITE_ID] [SUITE_TITLE] [SUITE_DESCRIPTION]", file=sys.stderr)
        sys.exit(1)

    suite_dir = Path(sys.argv[1]).resolve()
    suite_id = sys.argv[2] if len(sys.argv) > 2 else suite_dir.name
    suite_title = sys.argv[3] if len(sys.argv) > 3 else f"Benchmark Suite: {suite_id}"
    suite_desc = sys.argv[4] if len(sys.argv) > 4 else ""

    runs = []
    for child in sorted(suite_dir.iterdir()):
        if not child.is_dir():
            continue

        res_file = child / "results.json"
        if not res_file.exists():
            nested_res = next(child.rglob("results.json"), None)
            if nested_res and nested_res.exists():
                res_file = nested_res

        if res_file.exists():
            try:
                data = json.loads(res_file.read_text(encoding="utf-8"))
                runs.append(data)
                continue
            except Exception:
                pass

        # Fallback: synthesize failed run record if test-config.txt exists
        cfg_file = child / "test-config.txt"
        if not cfg_file.exists():
            cfg_file = next(child.rglob("test-config.txt"), None)
        if cfg_file and cfg_file.exists():
            meta = {}
            try:
                for line in cfg_file.read_text(encoding="utf-8").splitlines():
                    if "=" in line:
                        k, v = line.split("=", 1)
                        meta[k.strip()] = v.strip()
            except Exception:
                pass

            runs.append({
                "test_id": child.name,
                "status": "FAILED",
                "metadata": {
                    "test_description": meta.get("test_description", child.name),
                    "scenario": meta.get("scenario", "unknown"),
                    "target_rate": safe_int(meta.get("target_publish_msgs_per_sec", 0)),
                    "payload_size": meta.get("message_size_bytes", "1024B"),
                    "publisher_concurrency": safe_int(meta.get("publisher_concurrency", 1), 1),
                    "subscriber_concurrency": safe_int(meta.get("subscriber_concurrency", 1), 1),
                    "duration": meta.get("nominal_duration_seconds", "0s"),
                    "messaging_type": meta.get("messaging_type", "Core NATS")
                },
                "client": {
                    "published_msgs": 0,
                    "received_msgs": 0,
                    "throughput_msgs_sec": 0.0,
                    "throughput_mb_sec": 0.0,
                    "delivery_efficiency_pct": 0.0,
                    "latency_us": {"min": 0.0, "avg": 0.0, "p50": 0.0, "p90": 0.0, "p99": 0.0, "max": 0.0}
                },
                "cluster": {
                    "avg_cpu_pct": 0.0,
                    "peak_cpu_pct": 0.0,
                    "total_peak_mem_mb": 0.0,
                    "total_mem_growth_mb": 0.0,
                    "cpu_efficiency_msgs_per_core": 0.0
                },
                "nodes": []
            })

    # Sort runs ascending by target rate and payload size for proper sequential ladders
    runs.sort(key=lambda r: (
        safe_int((r.get("metadata") or {}).get("target_rate", 0)),
        parse_payload_bytes((r.get("metadata") or {}).get("payload_size", "0B"))
    ))

    recs = compute_recommendations(runs)

    suite_data = {
        "suite_id": suite_id,
        "suite_title": suite_title,
        "suite_description": suite_desc,
        "recommendations": recs,
        "tests": runs
    }

    # Write suite-results.json
    suite_json_path = suite_dir / "suite-results.json"
    suite_json_path.write_text(json.dumps(suite_data, indent=2), encoding="utf-8")

    # Inject into static/report.html to create self-contained offline reports
    template_path = ROOT_DIR / "static" / "report.html"
    if template_path.exists():
        template_content = template_path.read_text(encoding="utf-8")
        sentinel_pattern = re.compile(
            r'<script\s+id=["\']benchmark-data["\']\s*>\s*window\.BENCHMARK_DATA\s*=\s*null;\s*</script>',
            re.MULTILINE
        )
        embed_script = f'<script id="benchmark-data">\n    window.BENCHMARK_DATA = {json.dumps(suite_data)};\n  </script>'

        if sentinel_pattern.search(template_content):
            report_content = sentinel_pattern.sub(embed_script, template_content, count=1)
            exec_report = suite_dir / "executive-report.html"
            suite_report = suite_dir / "suite-report.html"
            exec_report.write_text(report_content, encoding="utf-8")
            suite_report.write_text(report_content, encoding="utf-8")
        else:
            print(f"WARNING: Sentinel pattern was not found in {template_path}. Suite reports were not created.", file=sys.stderr)
    else:
        print(f"WARNING: Template {template_path} not found.", file=sys.stderr)

    print(f"Executive suite aggregated successfully ({len(runs)} runs):")
    print(f"  Suite JSON       : {suite_json_path}")
    print(f"  Executive Report : {suite_dir / 'executive-report.html'}")
    print(f"  Suite Report     : {suite_dir / 'suite-report.html'}")

if __name__ == "__main__":
    main()
