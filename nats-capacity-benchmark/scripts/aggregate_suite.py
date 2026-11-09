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
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent

def compute_recommendations(runs):
    if not runs:
        return {}

    peak_rate = max((r.get("client", {}).get("throughput_msgs_sec", 0.0) for r in runs), default=0.0)
    cluster_peak_cpu = max((r.get("cluster", {}).get("peak_cpu_pct", 0.0) for r in runs), default=0.0)
    cluster_peak_mem = max((r.get("cluster", {}).get("total_peak_mem_mb", 0.0) for r in runs), default=0.0)
    total_disk_growth = sum((sum(n.get("disk_growth_bytes", 0) for n in r.get("nodes", [])) for r in runs), 0)

    # Sustained boundary: highest achieved rate where delivery >= 99%, P99 < 15ms (15000us), CPU < 80%
    sustained = 0.0
    for r in runs:
        client = r.get("client", {})
        cluster = r.get("cluster", {})
        p99 = client.get("latency_us", {}).get("p99", 0.0)
        eff = client.get("delivery_efficiency_pct", 100.0)
        cpu = cluster.get("peak_cpu_pct", 0.0)
        rate = client.get("throughput_msgs_sec", 0.0)
        if eff >= 99.0 and p99 < 15000.0 and cpu < 80.0:
            if rate > sustained:
                sustained = rate

    # Saturation knee: first step where p99 > 15ms or delivery < 99% or throughput drops below offered
    saturation_knee = peak_rate
    for r in runs:
        client = r.get("client", {})
        p99 = client.get("latency_us", {}).get("p99", 0.0)
        eff = client.get("delivery_efficiency_pct", 100.0)
        target = r.get("metadata", {}).get("target_rate", 0)
        achieved = client.get("throughput_msgs_sec", 0.0)
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
        if child.is_dir():
            res_file = child / "results.json"
            if res_file.exists():
                try:
                    data = json.loads(res_file.read_text())
                    runs.append(data)
                except Exception:
                    pass

    if not runs:
        print(f"No results.json runs found in {suite_dir}", file=sys.stderr)
        return

    # Sort runs ascending by target rate for proper sequential ladders
    runs.sort(key=lambda r: int(r.get("metadata", {}).get("target_rate", 0)))

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
    suite_json_path.write_text(json.dumps(suite_data, indent=2))

    # Inject into static/report.html to create self-contained offline reports
    template_path = ROOT_DIR / "static" / "report.html"
    if template_path.exists():
        template_content = template_path.read_text(encoding="utf-8")
        sentinel_tag = '<script id="benchmark-data">\n    window.BENCHMARK_DATA = null;\n  </script>'
        embed_script = f'<script id="benchmark-data">\n    window.BENCHMARK_DATA = {json.dumps(suite_data)};\n  </script>'

        if sentinel_tag in template_content:
            report_content = template_content.replace(sentinel_tag, embed_script, 1)
            exec_report = suite_dir / "executive-report.html"
            suite_report = suite_dir / "suite-report.html"
            exec_report.write_text(report_content, encoding="utf-8")
            suite_report.write_text(report_content, encoding="utf-8")
        else:
            print(f"WARNING: Sentinel tag was not found in {template_path}. Suite reports were not created.", file=sys.stderr)

    print(f"Executive suite aggregated successfully ({len(runs)} runs):")
    print(f"  Suite JSON       : {suite_json_path}")
    print(f"  Executive Report : {suite_dir / 'executive-report.html'}")
    print(f"  Suite Report     : {suite_dir / 'suite-report.html'}")

if __name__ == "__main__":
    main()
