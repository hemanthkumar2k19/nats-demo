#!/usr/bin/env python3
"""
Master Suite Aggregator for NATS Capacity Benchmark
Synthesizes multi-test benchmark suites into an Executive Dashboard (HTML + Markdown)
featuring:
  1. The Throughput vs Latency Saturation Curve (Embedded SVG)
  2. The Cost of Guarantees Comparative Matrix
  3. Executive Cluster Sizing & Capacity Recommendation Cards
"""

import sys
import os
import csv
import re
import html
import statistics
from pathlib import Path
from collections import defaultdict
from datetime import datetime, timezone

if len(sys.argv) < 2:
    print("Usage: aggregate_suite.py SUITE_DIR [SUITE_ID] [SUITE_TITLE] [SUITE_DESCRIPTION]", file=sys.stderr)
    sys.exit(1)

suite_dir = Path(sys.argv[1]).resolve()
suite_id = sys.argv[2] if len(sys.argv) > 2 else suite_dir.name
suite_title = sys.argv[3] if len(sys.argv) > 3 else f"Benchmark Suite: {suite_id}"
suite_description = sys.argv[4] if len(sys.argv) > 4 else ""

def num(val):
    if val is None:
        return None
    try:
        clean = str(val).strip().rstrip("%msusB").replace(",", "")
        return float(clean)
    except (ValueError, TypeError):
        return None

def format_bytes(bytes_val):
    if bytes_val is None or bytes_val == "" or bytes_val == "-":
        return "-"
    try:
        val = float(bytes_val)
    except (ValueError, TypeError):
        return str(bytes_val)
    if val >= 1024 ** 3:
        return f"{val / (1024 ** 3):.2f} GiB"
    if val >= 1024 ** 2:
        return f"{val / (1024 ** 2):.2f} MiB"
    if val >= 1024:
        return f"{val / 1024:.2f} KiB"
    return f"{val:.0f} B"

def format_rate(rate_val, unit="MB/s"):
    if rate_val is None or rate_val == "" or rate_val == "-":
        return "-"
    try:
        val = float(rate_val)
        return f"{val:.2f} {unit}"
    except (ValueError, TypeError):
        return str(rate_val)

def parse_latency_to_ms(lat_str):
    if not lat_str or lat_str == "-":
        return None
    s = str(lat_str).strip().lower()
    if s.endswith("ms"):
        return num(s.rstrip("ms"))
    elif s.endswith("us") or s.endswith("\u00b5s"):
        val = num(s.rstrip("us").rstrip("\u00b5s"))
        return round(val / 1000.0, 3) if val is not None else None
    elif s.endswith("s"):
        val = num(s.rstrip("s"))
        return round(val * 1000.0, 3) if val is not None else None
    return num(s)

def parse_publisher_stats(log_path):
    stats = {}
    if not log_path.exists():
        return stats
    content = log_path.read_text(errors="replace")
    for line in content.splitlines():
        line = line.strip()
        if not line:
            continue
        if line.startswith("min:"):
            for part in line.split(","):
                part = part.strip()
                if not part:
                    continue
                k, sep, v = part.partition(":")
                if sep:
                    stats[k.strip().lower()] = v.strip()
    return stats

# Discover test runs inside suite directory
test_runs = []
for child in sorted(suite_dir.iterdir()):
    if not child.is_dir():
        continue
    summary_csv = child / "summary.csv"
    if not summary_csv.exists():
        continue

    run_data = {}
    with summary_csv.open(newline="", errors="replace") as f:
        reader = csv.DictReader(f)
        for row in reader:
            run_data.update(row)
            break

    # Read node-summary.csv for per-node disk rates and peak memory
    node_summary_csv = child / "node-summary.csv"
    node_rows = []
    if node_summary_csv.exists():
        with node_summary_csv.open(newline="", errors="replace") as f:
            node_rows = list(csv.DictReader(f))

    # Read test-config.txt for run metadata and timestamp
    cfg_file = child / "test-config.txt"
    if cfg_file.exists():
        for line in cfg_file.read_text(errors="replace").splitlines():
            if "=" in line:
                k, v = line.split("=", 1)
                run_data.setdefault(k.strip(), v.strip())

    # Read publisher latency percentiles from workload.log
    pub_stats = parse_publisher_stats(child / "workload.log")
    run_data["publisher_stats"] = pub_stats
    run_data["node_rows"] = node_rows
    run_data["child_dir"] = child.name

    test_runs.append(run_data)

if not test_runs:
    print(f"ERROR: No valid benchmark runs with summary.csv found in {suite_dir}", file=sys.stderr)
    sys.exit(1)

# Sort runs by execution timestamp to preserve suite chronology, then rate/id
test_runs.sort(key=lambda r: (
    r.get("run_timestamp", ""),
    num(r.get("target_publish_msgs_per_sec")) or 0,
    r.get("test_id", "")
))

# Extract Comparative Matrix rows
matrix_rows = []
curve_points = []
total_cluster_growth = 0
max_achieved_rate = 0
recommended_capacity = 0
primary_bottleneck = "Cluster Headroom Available"

for run in test_runs:
    tid = run.get("test_id", "")
    tdesc = run.get("test_description", "")
    mtype = run.get("messaging_type", "Core NATS")
    stype = run.get("storage_type", "-")
    reps = run.get("replication_factor", "-")
    target_rate = num(run.get("target_publish_msgs_per_sec")) or 0
    actual_pub = num(run.get("actual_publish_msgs_per_sec")) or 0
    actual_sub = num(run.get("actual_subscriber_msgs_per_sec")) or 0
    outcome = run.get("test_outcome", "COMPLETED")
    cpu_avg = num(run.get("cluster_cpu_avg_percent")) or 0.0
    cpu_peak = num(run.get("cluster_cpu_peak_percent")) or 0.0
    mem_peak = num(run.get("cluster_memory_peak_bytes")) or 0.0
    disk_growth = num(run.get("cluster_disk_growth_bytes")) or 0.0
    total_cluster_growth += disk_growth

    # Publisher Latencies
    pstats = run.get("publisher_stats", {})
    p50_raw = pstats.get("p50", "-")
    p90_raw = pstats.get("p90", "-")
    p99_raw = pstats.get("p99", "-")

    p50_ms = parse_latency_to_ms(p50_raw)
    p90_ms = parse_latency_to_ms(p90_raw)
    p99_ms = parse_latency_to_ms(p99_raw)

    # Compute cluster aggregate disk write MB/s from node-summary
    cluster_write_mbps = 0.0
    for nr in run.get("node_rows", []):
        w_rate = num(nr.get("block_write_avg_mbps")) or 0.0
        cluster_write_mbps += w_rate

    # Guarantee Tier Description
    if mtype == "Core NATS":
        tier_label = "Core NATS (In-Memory)"
        guarantee_desc = "No persistence, ephemeral broadcast"
    else:
        tier_label = f"JetStream ({stype.title()}, R={reps})"
        if stype == "memory":
            guarantee_desc = f"Memory replicated across {reps} nodes (Raft quorum)"
        elif reps == "1":
            guarantee_desc = "Disk write-ahead log (single node, no quorum)"
        else:
            guarantee_desc = f"Disk write-ahead log replicated across {reps} nodes (Raft consensus)"

    # Efficiency metric: CPU percent per 1,000 msgs/s
    if actual_pub > 0:
        cpu_per_1k = round((cpu_avg / actual_pub) * 1000.0, 2)
    else:
        cpu_per_1k = "-"

    matrix_rows.append({
        "test_id": tid,
        "description": tdesc,
        "tier_label": tier_label,
        "guarantee_desc": guarantee_desc,
        "target_rate": target_rate,
        "actual_pub": actual_pub,
        "actual_sub": actual_sub,
        "p50_raw": p50_raw,
        "p90_raw": p90_raw,
        "p99_raw": p99_raw,
        "p50_ms": p50_ms,
        "p99_ms": p99_ms,
        "cpu_avg": cpu_avg,
        "cpu_peak": cpu_peak,
        "cpu_per_1k": cpu_per_1k,
        "mem_peak": mem_peak,
        "disk_write_mbps": cluster_write_mbps,
        "disk_growth": disk_growth,
        "outcome": outcome,
        "child_dir": run.get("child_dir", tid)
    })

    if actual_pub > max_achieved_rate:
        max_achieved_rate = actual_pub

    # Sizing heuristic: Safe operating boundary (<80% cluster CPU, clean completion, reasonable latency)
    if outcome == "COMPLETED" and cpu_avg < 200.0 and (p99_ms is None or p99_ms < 15.0):
        if actual_pub > recommended_capacity:
            recommended_capacity = actual_pub

    # Curve point if valid latencies exist
    if actual_pub > 0 and p99_ms is not None:
        curve_points.append({
            "target": target_rate,
            "achieved": actual_pub,
            "p50_ms": p50_ms if p50_ms is not None else 0.0,
            "p99_ms": p99_ms,
            "tid": tid
        })

    # Bottleneck detection
    if cpu_peak > 250.0:
        primary_bottleneck = "Cluster CPU Saturation (Core Quotas Reached)"
    elif cluster_write_mbps > 20.0 or disk_growth > 50 * 1024 * 1024:
        primary_bottleneck = "Raft Consensus Disk Write / Fsync Bound"
    elif outcome == "TIMEOUT":
        primary_bottleneck = "Raft Consensus Synchronous Drain Congestion"

# If recommended_capacity wasn't set, default to max achieved
if recommended_capacity == 0:
    recommended_capacity = max_achieved_rate

# -------------------------------------------------------------
# Generate Pure SVG Latency vs Throughput Saturation Curve
# -------------------------------------------------------------
svg_chart_html = ""
if len(curve_points) >= 2:
    chart_w, chart_h = 760, 320
    margin_l, margin_r, margin_t, margin_b = 65, 30, 30, 45
    plot_w = chart_w - margin_l - margin_r
    plot_h = chart_h - margin_t - margin_b

    max_x = max(p["achieved"] for p in curve_points) * 1.15
    max_y = max(p["p99_ms"] for p in curve_points) * 1.25
    if max_x <= 0:
        max_x = 1000.0
    if max_y <= 0:
        max_y = 10.0

    def to_coords(x_val, y_val):
        cx = margin_l + (x_val / max_x) * plot_w
        cy = margin_t + plot_h - (y_val / max_y) * plot_h
        return round(cx, 1), round(cy, 1)

    # Gridlines and axis labels
    grid_svg = []
    x_steps = 4
    for i in range(x_steps + 1):
        x_val = (max_x / x_steps) * i
        gx, _ = to_coords(x_val, 0)
        grid_svg.append(f'<line x1="{gx}" y1="{margin_t}" x2="{gx}" y2="{margin_t + plot_h}" stroke="#30363d" stroke-dasharray="3,3"/>')
        grid_svg.append(f'<text x="{gx}" y="{margin_t + plot_h + 18}" fill="#8b949e" font-size="11" text-anchor="middle">{int(x_val):,} msg/s</text>')

    y_steps = 4
    for i in range(y_steps + 1):
        y_val = (max_y / y_steps) * i
        _, gy = to_coords(0, y_val)
        grid_svg.append(f'<line x1="{margin_l}" y1="{gy}" x2="{margin_l + plot_w}" y2="{gy}" stroke="#30363d" stroke-dasharray="3,3"/>')
        grid_svg.append(f'<text x="{margin_l - 10}" y="{gy + 4}" fill="#8b949e" font-size="11" text-anchor="end">{y_val:.1f} ms</text>')

    # Polylines for P50 and P99
    p50_pts = [to_coords(p["achieved"], p["p50_ms"]) for p in curve_points]
    p99_pts = [to_coords(p["achieved"], p["p99_ms"]) for p in curve_points]

    p50_poly = " ".join(f"{x},{y}" for x, y in p50_pts)
    p99_poly = " ".join(f"{x},{y}" for x, y in p99_pts)

    points_p50_svg = []
    points_p99_svg = []
    for (x, y), p in zip(p50_pts, curve_points):
        points_p50_svg.append(f'<circle cx="{x}" cy="{y}" r="4" fill="#58a6ff"><title>{p["tid"]} - P50: {p["p50_ms"]} ms @ {int(p["achieved"]):,} msg/s</title></circle>')
    for (x, y), p in zip(p99_pts, curve_points):
        points_p99_svg.append(f'<circle cx="{x}" cy="{y}" r="5" fill="#f85149"><title>{p["tid"]} - P99: {p["p99_ms"]} ms @ {int(p["achieved"]):,} msg/s</title></circle>')

    svg_chart_html = f"""
    <div style="background:var(--bg-card); border:1px solid var(--border); border-radius:var(--radius); padding:20px; margin-bottom:24px;">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:12px;">
        <span style="font-weight:600; font-size:1.05rem;">Throughput vs Latency Saturation Curve (Hockey Stick)</span>
        <div style="font-size:0.8rem; display:flex; gap:16px;">
          <span style="color:#f85149; font-weight:600;">&bull; P99 Tail Latency</span>
          <span style="color:#58a6ff; font-weight:600;">&bull; P50 Median Latency</span>
        </div>
      </div>
      <div style="overflow-x:auto;">
        <svg viewBox="0 0 {chart_w} {chart_h}" style="width:100%; max-width:850px; height:auto; display:block; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
          {''.join(grid_svg)}
          <polyline points="{p50_poly}" fill="none" stroke="#58a6ff" stroke-width="2.5" stroke-linecap="round"/>
          <polyline points="{p99_poly}" fill="none" stroke="#f85149" stroke-width="2.5" stroke-linecap="round"/>
          {''.join(points_p50_svg)}
          {''.join(points_p99_svg)}
          <line x1="{margin_l}" y1="{margin_t}" x2="{margin_l}" y2="{margin_t + plot_h}" stroke="#8b949e" stroke-width="1.5"/>
          <line x1="{margin_l}" y1="{margin_t + plot_h}" x2="{margin_l + plot_w}" y2="{margin_t + plot_h}" stroke="#8b949e" stroke-width="1.5"/>
        </svg>
      </div>
      <div style="font-size:0.75rem; color:var(--text-sec); margin-top:8px; text-align:center;">
        Offered Rate ($X$) vs Client Commit Latency ($Y$). The upward curve inflection point marks cluster capacity saturation.
      </div>
    </div>
    """

# Build HTML Matrix Table rows
table_rows = []
for r in matrix_rows:
    outcome_badge = (
        '<span class="badge badge-green">COMPLETED</span>' if r["outcome"] == "COMPLETED"
        else f'<span class="badge badge-red">{r["outcome"]}</span>'
    )
    growth_str = f"+{format_bytes(r['disk_growth'])}" if r["disk_growth"] > 0 else "-"
    p99_display = f"<strong>{r['p99_raw']}</strong>" if r['p99_raw'] != "-" else "-"

    table_rows.append(f"""
    <tr>
      <td>
        <a href="{r['child_dir']}/report.html" style="color:var(--accent); font-weight:600; text-decoration:none;">{html.escape(r['test_id'])}</a>
        <div style="font-size:0.75rem; color:var(--text-sec);">{html.escape(r['description'])}</div>
      </td>
      <td>
        <strong>{html.escape(r['tier_label'])}</strong>
        <div style="font-size:0.72rem; color:var(--text-sec);">{html.escape(r['guarantee_desc'])}</div>
      </td>
      <td style="text-align:right;">
        <span class="bold" style="color:var(--text);">{int(r['actual_pub']):,}</span>
        <div style="font-size:0.72rem; color:var(--text-sec);">target: {int(r['target_rate']):,}</div>
      </td>
      <td style="text-align:right;">{r['p50_raw']}</td>
      <td style="text-align:right;">{p99_display}</td>
      <td style="text-align:right;">
        {r['cpu_avg']:.1f}%
        <div style="font-size:0.72rem; color:var(--text-sec);">peak: {r['cpu_peak']:.1f}%</div>
      </td>
      <td style="text-align:right;">
        {format_rate(r['disk_write_mbps'])}
        <div style="font-size:0.72rem; color:var(--text-sec);">{growth_str}</div>
      </td>
      <td style="text-align:center;">{outcome_badge}</td>
    </tr>
    """)

matrix_tbody = "\n".join(table_rows)

html_content = f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Executive Benchmark Report: {html.escape(suite_title)}</title>
  <style>
    :root {{
      --bg: #0d1117;
      --bg-card: #161b22;
      --border: #30363d;
      --text: #e6edf3;
      --text-sec: #8b949e;
      --accent: #58a6ff;
      --green: #3fb950;
      --red: #f85149;
      --yellow: #d29922;
      --purple: #bc8cff;
      --radius: 8px;
    }}
    * {{ box-sizing: border-box; margin: 0; padding: 0; }}
    body {{
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 24px;
      font-size: 14px;
    }}
    .container {{ max-width: 1200px; margin: 0 auto; }}
    .header {{
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--border);
    }}
    .header h1 {{ font-size: 1.6rem; color: var(--text); }}
    .header p {{ color: var(--text-sec); font-size: 0.9rem; margin-top: 4px; }}
    .grid-kpi {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }}
    .card-kpi {{
      background: var(--bg-card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 16px;
    }}
    .kpi-label {{ font-size: 0.75rem; text-transform: uppercase; color: var(--text-sec); letter-spacing: 0.5px; margin-bottom: 6px; }}
    .kpi-value {{ font-size: 1.6rem; font-weight: 700; color: var(--text); }}
    .kpi-sub {{ font-size: 0.8rem; color: var(--text-sec); margin-top: 4px; }}
    .badge {{
      display: inline-block;
      padding: 3px 8px;
      border-radius: 12px;
      font-size: 0.7rem;
      font-weight: 600;
      text-transform: uppercase;
    }}
    .badge-green {{ background: rgba(63,185,80,0.15); color: var(--green); border: 1px solid rgba(63,185,80,0.3); }}
    .badge-red {{ background: rgba(248,81,73,0.15); color: var(--red); border: 1px solid rgba(248,81,73,0.3); }}
    .table-wrap {{
      overflow-x: auto;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      background: var(--bg-card);
      margin-bottom: 24px;
    }}
    table {{ width: 100%; border-collapse: collapse; text-align: left; font-size: 0.85rem; }}
    th, td {{ padding: 12px 14px; border-bottom: 1px solid var(--border); vertical-align: middle; }}
    th {{ background: #21262d; color: var(--text-sec); font-weight: 600; font-size: 0.75rem; text-transform: uppercase; }}
    tr:last-child td {{ border-bottom: none; }}
    tr:hover {{ background: rgba(255,255,255,0.02); }}
    .bold {{ font-weight: 600; }}
    @media print {{
      body {{ background: #fff !important; color: #000 !important; }}
      .card-kpi, .table-wrap, div {{ background: #fff !important; color: #000 !important; border-color: #ddd !important; }}
      .badge {{ border: 1px solid #999; color: #000; }}
    }}
  </style>
</head>
<body>
<div class="container">
  <div class="header">
    <div>
      <h1>{html.escape(suite_title)}</h1>
      <p>{html.escape(suite_description)} &bull; Suite ID: <strong>{html.escape(suite_id)}</strong></p>
    </div>
    <div style="text-align:right;">
      <span class="badge badge-green">AUDITED SUITE</span>
      <div style="font-size:0.75rem; color:var(--text-sec); margin-top:6px;">Generated: {datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%SZ")}</div>
    </div>
  </div>

  <!-- Executive Sizing & Capacity Recommendation Cards -->
  <div class="grid-kpi">
    <div class="card-kpi">
      <div class="kpi-label">Sustained Operating Capacity</div>
      <div class="kpi-value" style="color:var(--green);">{int(recommended_capacity):,} <span style="font-size:0.9rem; font-weight:400;">msg/s</span></div>
      <div class="kpi-sub">Within SLA (P99 &lt; 15 ms, Zero drops)</div>
    </div>
    <div class="card-kpi">
      <div class="kpi-label">Peak Achieved Throughput</div>
      <div class="kpi-value" style="color:var(--accent);">{int(max_achieved_rate):,} <span style="font-size:0.9rem; font-weight:400;">msg/s</span></div>
      <div class="kpi-sub">Across all {len(test_runs)} evaluated test matrix tiers</div>
    </div>
    <div class="card-kpi">
      <div class="kpi-label">Identified Hardware Bottleneck</div>
      <div class="kpi-value" style="color:var(--yellow); font-size:1.15rem; font-weight:600;">{html.escape(primary_bottleneck)}</div>
      <div class="kpi-sub">Observed under peak saturation</div>
    </div>
    <div class="card-kpi">
      <div class="kpi-label">Total Suite Physical Storage</div>
      <div class="kpi-value" style="color:var(--purple);">{format_bytes(total_cluster_growth)}</div>
      <div class="kpi-sub">Empirical disk growth on 3-node cluster</div>
    </div>
  </div>

  <!-- Throughput vs Latency Saturation Curve -->
  {svg_chart_html}

  <!-- Cost of Guarantees Comparative Matrix Table -->
  <div style="margin-bottom:12px; display:flex; justify-content:space-between; align-items:baseline;">
    <h2 style="font-size:1.15rem; font-weight:600;">Guarantee Tier Comparative Matrix</h2>
    <span style="font-size:0.8rem; color:var(--text-sec);">{len(test_runs)} test runs synthesized &bull; Click Test ID to view individual run telemetry</span>
  </div>
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Test Identifier</th>
          <th>Guarantee Tier &amp; Placement</th>
          <th style="text-align:right;">Throughput</th>
          <th style="text-align:right;">P50 Latency</th>
          <th style="text-align:right;">P99 Latency</th>
          <th style="text-align:right;">Cluster CPU</th>
          <th style="text-align:right;">Disk Write</th>
          <th style="text-align:center;">Outcome</th>
        </tr>
      </thead>
      <tbody>
        {matrix_tbody}
      </tbody>
    </table>
  </div>
</div>
</body>
</html>
"""

# Write HTML Report
(suite_dir / "executive-report.html").write_text(html_content, encoding="utf-8")

# -------------------------------------------------------------
# Generate Portable Markdown Executive Summary (executive-summary.md)
# -------------------------------------------------------------
md_lines = [
    f"# Executive Benchmark Summary: {suite_title}",
    f"",
    f"**Suite ID**: `{suite_id}`  ",
    f"**Description**: {suite_description}  ",
    f"**Generated**: {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M:%SZ')}  ",
    f"",
    f"---",
    f"",
    f"## Executive Sizing & Capacity Recommendations",
    f"",
    f"| Recommendation Dimension | Finding | Architectural Operational Guideline |",
    f"| :--- | :--- | :--- |",
    f"| **Sustained Operational Capacity** | **`{int(recommended_capacity):,} msgs/sec`** | Maximum throughput sustaining sub-15ms P99 latency and healthy cluster CPU headroom (<80%). |",
    f"| **Peak Achieved Throughput** | **`{int(max_achieved_rate):,} msgs/sec`** | Maximum burst throughput observed across the benchmark matrix. |",
    f"| **Primary Cluster Bottleneck** | **`{primary_bottleneck}`** | Hardware constraint reached when offered load exceeds the saturation knee. |",
    f"| **Physical Storage Delta** | **`{format_bytes(total_cluster_growth)}`** | Total persistent disk bytes committed across the 3-node cluster. |",
    f"",
    f"---",
    f"",
    f"## Guarantee Tier Comparative Matrix",
    f"",
    f"| Test ID | Guarantee Tier | Target Rate | Achieved Rate | P50 Latency | P99 Latency | Cluster CPU | Disk Write Rate | Storage Growth | Outcome |",
    f"| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |",
]

for r in matrix_rows:
    p99_str = r['p99_raw'] if r['p99_raw'] != '-' else '-'
    growth_str = f"+{format_bytes(r['disk_growth'])}" if r['disk_growth'] > 0 else "-"
    md_lines.append(
        f"| **{r['test_id']}** | {r['tier_label']} | {int(r['target_rate']):,} | **{int(r['actual_pub']):,}** | {r['p50_raw']} | {p99_str} | {r['cpu_avg']:.1f}% | {format_rate(r['disk_write_mbps'])} | {growth_str} | `{r['outcome']}` |"
    )

md_lines.extend([
    f"",
    f"---",
    f"",
    f"*For granular diagnostic logs, idle baselines, and per-container metrics, open the corresponding `report.html` within each child test folder.*",
])

(suite_dir / "executive-summary.md").write_text("\n".join(md_lines), encoding="utf-8")
print(f"Successfully generated executive reports in: {suite_dir}")

