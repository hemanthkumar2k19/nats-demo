#!/usr/bin/env python3

import csv
import html
import re
import statistics
import sys

from collections import defaultdict
from pathlib import Path


if len(sys.argv) == 2:
    result_dir = Path(sys.argv[1])
    test_id = result_dir.name
    subject = ""
    target_rate = "0"
    msg_size = "0"
    pubs = "1"
    subs = "1"
    nominal_duration = "0"
    actual_duration = "0"
    workload_exit = "0"
    backend = "podman"

    summary_path = result_dir / "summary.csv"
    config_path = result_dir / "test-config.txt"

    if summary_path.exists():
        with summary_path.open() as f:
            reader = csv.DictReader(f)
            for row in reader:
                test_id = row.get("test_id", test_id)
                subject = row.get("subject", subject)
                target_rate = row.get("target_publish_msgs_per_sec", target_rate)
                msg_size = row.get("message_size_bytes", msg_size)
                pubs = row.get("publisher_concurrency", pubs)
                subs = row.get("subscriber_concurrency", subs)
                nominal_duration = row.get("nominal_duration_seconds", nominal_duration)
                actual_duration = row.get("actual_duration_seconds", actual_duration)
                workload_exit = row.get("workload_exit_code", workload_exit)
                backend = row.get("metrics_backend", backend)
                break
    elif config_path.exists():
        cfg = {}
        for line in config_path.read_text(errors="replace").splitlines():
            if "=" in line:
                k, v = line.split("=", 1)
                cfg[k.strip()] = v.strip()
        test_id = cfg.get("test_id", test_id)
        subject = cfg.get("subject", subject)
        target_rate = cfg.get("target_publish_msgs_per_sec", target_rate)
        msg_size = cfg.get("message_size_bytes", msg_size)
        pubs = cfg.get("publisher_concurrency", pubs)
        subs = cfg.get("subscriber_concurrency", subs)
        nominal_duration = cfg.get("nominal_duration_seconds", nominal_duration)
        backend = cfg.get("metrics_backend", backend)
elif len(sys.argv) >= 12:
    (
        result_dir,
        test_id,
        subject,
        target_rate,
        msg_size,
        pubs,
        subs,
        nominal_duration,
        actual_duration,
        workload_exit,
        backend
    ) = sys.argv[1:12]
    result_dir = Path(result_dir)
else:
    print(
        "Usage: aggregate.py <result_dir> [test_id subject target_rate msg_size pubs subs nominal_dur actual_dur exit backend]",
        file=sys.stderr
    )
    sys.exit(1)

scenario = "pubsub"
subscription_pattern = "pubsub"
queue_name = "benchmark-core-group"
subject_distribution = "single"
multi_subject_max = "10000"

config_file_path = result_dir / "test-config.txt"
if config_file_path.exists():
    for line in config_file_path.read_text(errors="replace").splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            k = k.strip()
            v = v.strip()
            if k == "scenario":
                scenario = v
            elif k == "subscription_pattern":
                subscription_pattern = v
            elif k == "queue_name":
                queue_name = v
            elif k == "subject_distribution":
                subject_distribution = v
            elif k == "multi_subject_max":
                multi_subject_max = v

if scenario == "publish":
    fanout_ratio = 0
elif scenario == "subscribe_setup":
    fanout_ratio = 0
elif subscription_pattern == "queue":
    fanout_ratio = 1
else:
    try:
        fanout_ratio = int(subs)
    except (ValueError, TypeError):
        fanout_ratio = 1


# -------------------------
# Conversion helpers
# -------------------------

def num(value):
    try:
        return float(str(value).strip().rstrip("%"))
    except (ValueError, TypeError):
        return None


def bytes_value(value):
    if not value:
        return None

    match = re.match(
        r"^\s*([0-9.]+)\s*([KMGT]?i?B)?\s*$",
        str(value).replace(",", ""),
        re.I
    )

    if not match:
        return None

    unit = (match.group(2) or "B").upper()

    scale = {
        "B": 1,
        "KB": 1000,
        "KIB": 1024,
        "MB": 1000 ** 2,
        "MIB": 1024 ** 2,
        "GB": 1000 ** 3,
        "GIB": 1024 ** 3,
        "TB": 1000 ** 4,
        "TIB": 1024 ** 4
    }

    return float(match.group(1)) * scale.get(unit, 1)


def format_bytes(bytes_val):
    if bytes_val is None or bytes_val == "":
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
    if rate_val is None or rate_val == "":
        return "-"
    try:
        val = float(rate_val)
        if val == 0:
            return f"0.000 {unit}"
        return f"{val:.3f} {unit}"
    except (ValueError, TypeError):
        return str(rate_val)


def pair_bytes(value):
    if not value or "/" not in value:
        return None, None

    left, right = value.split("/", 1)

    return bytes_value(left), bytes_value(right)


# -------------------------
# Read metrics CSV
# -------------------------

metrics_file = result_dir / "metrics.csv"

rows = []

if metrics_file.exists():
    with metrics_file.open(newline="") as file:
        rows = list(csv.DictReader(file))


# Only workload samples are used for workload aggregation
workload_rows = [
    row for row in rows
    if row.get("phase") == "workload"
]


# -------------------------
# Group samples by node
# -------------------------

nodes = defaultdict(list)

for row in workload_rows:
    nodes[row.get("container", "unknown")].append(row)


node_results = []


for node, samples in nodes.items():

    samples.sort(
        key=lambda row: num(row.get("timestamp_epoch")) or 0
    )

    cpu_values = []
    memory_values = []

    network_rx_rates = []
    network_tx_rates = []

    disk_read_rates = []
    disk_write_rates = []

    previous = None

    for row in samples:

        cpu = num(row.get("cpu_percent"))

        if cpu is not None:
            cpu_values.append(cpu)

        memory_text = (
            row.get("memory_usage") or ""
        ).split("/", 1)[0].strip()

        memory = bytes_value(memory_text)

        if memory is not None:
            memory_values.append(memory)

        # Calculate throughput from cumulative counters
        if previous:

            t0 = num(previous.get("timestamp_epoch"))
            t1 = num(row.get("timestamp_epoch"))

            elapsed = (
                t1 - t0
                if t0 is not None and t1 is not None
                else 0
            )

            if elapsed > 0:

                old_rx, old_tx = pair_bytes(
                    previous.get("network_io")
                )

                new_rx, new_tx = pair_bytes(
                    row.get("network_io")
                )

                old_read, old_write = pair_bytes(
                    previous.get("block_io")
                )

                new_read, new_write = pair_bytes(
                    row.get("block_io")
                )

                pairs = [
                    (old_rx, new_rx, network_rx_rates),
                    (old_tx, new_tx, network_tx_rates),
                    (old_read, new_read, disk_read_rates),
                    (old_write, new_write, disk_write_rates)
                ]

                for old, new, destination in pairs:

                    if (
                        old is not None
                        and new is not None
                        and new >= old
                    ):
                        mbps = (new - old) / elapsed / 1_000_000
                        destination.append(mbps)

        previous = row

    def average(values):
        return (
            round(statistics.fmean(values), 3)
            if values else ""
        )

    def peak(values):
        return round(max(values), 3) if values else ""

    node_results.append({
        "test_id": test_id,
        "container": node,
        "samples": len(samples),

        "cpu_avg_percent": average(cpu_values),
        "cpu_peak_percent": peak(cpu_values),

        "memory_avg_bytes": average(memory_values),
        "memory_peak_bytes": peak(memory_values),

        "network_rx_avg_mbps": average(network_rx_rates),
        "network_rx_peak_mbps": peak(network_rx_rates),

        "network_tx_avg_mbps": average(network_tx_rates),
        "network_tx_peak_mbps": peak(network_tx_rates),

        "block_read_avg_mbps": average(disk_read_rates),
        "block_read_peak_mbps": peak(disk_read_rates),

        "block_write_avg_mbps": average(disk_write_rates),
        "block_write_peak_mbps": peak(disk_write_rates)
    })


# -------------------------
# Write node summary
# -------------------------

node_fields = [
    "test_id",
    "container",
    "samples",
    "cpu_avg_percent",
    "cpu_peak_percent",
    "memory_avg_bytes",
    "memory_peak_bytes",
    "network_rx_avg_mbps",
    "network_rx_peak_mbps",
    "network_tx_avg_mbps",
    "network_tx_peak_mbps",
    "block_read_avg_mbps",
    "block_read_peak_mbps",
    "block_write_avg_mbps",
    "block_write_peak_mbps"
]

if not node_results:
    node_results = []

with (result_dir / "node-summary.csv").open(
    "w",
    newline=""
) as file:

    writer = csv.DictWriter(
        file,
        fieldnames=node_fields
    )

    writer.writeheader()
    writer.writerows(node_results)


# -------------------------
# Cluster-level aggregation
# -------------------------

# Sum node CPU and memory at each timestamp.
# Do not sum independent node peaks.

samples_by_time = defaultdict(list)

for row in workload_rows:
    samples_by_time[row.get("timestamp_epoch")].append(row)


cluster_cpu = []
cluster_memory = []

for group in samples_by_time.values():

    cpu_values = [
        num(row.get("cpu_percent"))
        for row in group
    ]

    cpu_values = [
        value for value in cpu_values
        if value is not None
    ]

    memory_values = [
        bytes_value(
            (row.get("memory_usage") or "").split("/", 1)[0].strip()
        )
        for row in group
    ]

    memory_values = [
        value for value in memory_values
        if value is not None
    ]

    if cpu_values:
        cluster_cpu.append(sum(cpu_values))

    if memory_values:
        cluster_memory.append(sum(memory_values))


# -------------------------
# Extract workload rates & publisher stats
# -------------------------

def extract_rate(path, label):

    if not path.exists():
        return ""

    content = path.read_text(errors="replace")

    matches = re.findall(
        rf"{label}[a-z]*\s+stats:\s*([\d,]+)\s+msgs/sec",
        content,
        re.I
    )

    if not matches:
        return ""

    return int(matches[-1].replace(",", ""))


publish_rate = extract_rate(
    result_dir / "workload.log",
    "Pub"
)

subscriber_rate = extract_rate(
    result_dir / "subscriber.log",
    "Sub"
)


def extract_publisher_stats(path):
    if not path.exists():
        return {}
    content = path.read_text(errors="replace")
    stats = {}

    m_rate = re.search(
        r"stats:\s*([0-9,.]+)\s*msgs/sec\s*~\s*([0-9,.]+\s*[KMGT]?i?B/sec)",
        content,
        re.I
    )
    if m_rate:
        stats["rate"] = m_rate.group(1).replace(",", "")
        stats["bandwidth"] = m_rate.group(2).strip()

    metrics = ["min", "avg", "max", "P50", "P90", "P99", "P99.9"]
    for m in metrics:
        pattern = rf"\b{re.escape(m)}:\s*([0-9,.]+)\s*([a-zA-Z]+)"
        match = re.search(pattern, content, re.I)
        if match:
            stats[m] = f"{match.group(1)} {match.group(2)}"

    return stats


def extract_subscriber_stats(path):
    if not path.exists():
        return {}
    content = path.read_text(errors="replace")
    stats = {}

    m_rate = re.search(
        r"stats:\s*([0-9,.]+)\s*msgs/sec\s*~\s*([0-9,.]+\s*[KMGT]?i?B/sec)",
        content,
        re.I
    )
    if m_rate:
        stats["rate"] = m_rate.group(1).replace(",", "")
        stats["bandwidth"] = m_rate.group(2).strip()

    return stats


def extract_setup_stats(path):
    if not path.exists():
        return {}
    content = path.read_text(errors="replace")
    m = re.search(
        r"setup stats:\s*established=(\d+),\s*duration=(\d+)s,\s*rate=(\d+)\s*conns/sec",
        content,
        re.I
    )
    if m:
        return {
            "established": int(m.group(1)),
            "duration": int(m.group(2)),
            "rate": int(m.group(3))
        }
    return {}


publisher_stats = extract_publisher_stats(result_dir / "workload.log")
subscriber_stats = extract_subscriber_stats(result_dir / "subscriber.log")
setup_stats = extract_setup_stats(result_dir / "subscriber.log")


# -------------------------
# Build summary
# -------------------------

summary = {

    "test_id": test_id,
    "messaging_type": "Core NATS",
    "scenario": scenario,
    "subscription_pattern": subscription_pattern,
    "subject_distribution": subject_distribution,
    "fanout_ratio": fanout_ratio,
    "subject": subject,

    "target_publish_msgs_per_sec": target_rate if scenario != "subscribe_setup" else "",
    "actual_publish_msgs_per_sec": publish_rate if scenario != "subscribe_setup" else "",
    "actual_subscriber_msgs_per_sec": subscriber_rate if scenario == "pubsub" else "",

    "message_size_bytes": msg_size if scenario != "subscribe_setup" else "",

    "publisher_concurrency": pubs if scenario != "subscribe_setup" else 0,
    "subscriber_concurrency": subs,

    "nominal_duration_seconds": nominal_duration,
    "actual_duration_seconds": actual_duration,

    "workload_exit_code": workload_exit,
    "metrics_backend": backend,

    "cluster_cpu_avg_percent": (
        round(statistics.fmean(cluster_cpu), 3)
        if cluster_cpu else ""
    ),

    "cluster_cpu_peak_percent": (
        round(max(cluster_cpu), 3)
        if cluster_cpu else ""
    ),

    "cluster_memory_avg_bytes": (
        round(statistics.fmean(cluster_memory), 0)
        if cluster_memory else ""
    ),

    "cluster_memory_peak_bytes": (
        round(max(cluster_memory), 0)
        if cluster_memory else ""
    ),

    "node_summary_file": "node-summary.csv",
    "metrics_file": "metrics.csv",
    "workload_log": "workload.log",
    "subscriber_log": "subscriber.log",
    "report_file": "report.html"
}


# -------------------------
# Write summary CSV
# -------------------------

with (result_dir / "summary.csv").open(
    "w",
    newline=""
) as file:

    writer = csv.DictWriter(
        file,
        fieldnames=list(summary.keys())
    )

    writer.writeheader()
    writer.writerow(summary)


# -------------------------
# Read configuration and raw logs for HTML report
# -------------------------

def read_text_safe(path):
    if path.exists():
        return path.read_text(errors="replace")
    return ""


config_content = read_text_safe(result_dir / "test-config.txt")
workload_log_content = read_text_safe(result_dir / "workload.log")
subscriber_log_content = read_text_safe(result_dir / "subscriber.log")


# -------------------------
# Generate standalone report.html
# -------------------------

def generate_report_html():
    is_success = str(workload_exit) == "0"
    status_label = "PASS" if is_success else f"FAIL (exit {workload_exit})"
    status_badge_class = "badge-pass" if is_success else "badge-fail"

    target_num = num(target_rate) or 0
    actual_pub_num = num(publish_rate) or 0
    if target_num > 0:
        pub_achieved_pct = f"{(actual_pub_num / target_num * 100.0):.1f}%"
        pub_target_desc = f"Target: {target_rate} msgs/s ({pub_achieved_pct} achieved)"
    else:
        pub_achieved_pct = "unthrottled"
        pub_target_desc = "Target: unthrottled (maximum achievable)"

    cluster_cpu_peak_display = f"{summary['cluster_cpu_peak_percent']}%" if summary["cluster_cpu_peak_percent"] != "" else "-"
    cluster_cpu_avg_display = f"{summary['cluster_cpu_avg_percent']}%" if summary["cluster_cpu_avg_percent"] != "" else "-"

    cluster_mem_peak_display = format_bytes(summary["cluster_memory_peak_bytes"])
    cluster_mem_avg_display = format_bytes(summary["cluster_memory_avg_bytes"])

    # Table rows for nodes
    node_rows_html = []
    for n in node_results:
        node_rows_html.append(f"""
        <tr>
          <td><strong>{html.escape(str(n.get('container', '')))}</strong></td>
          <td class="num">{n.get('samples', 0)}</td>
          <td class="num">{n.get('cpu_avg_percent', '-')}%</td>
          <td class="num bold">{n.get('cpu_peak_percent', '-')}%</td>
          <td class="num">{format_bytes(n.get('memory_avg_bytes'))}</td>
          <td class="num bold">{format_bytes(n.get('memory_peak_bytes'))}</td>
          <td class="num">{format_rate(n.get('network_rx_avg_mbps'))} / {format_rate(n.get('network_tx_avg_mbps'))}</td>
          <td class="num">{format_rate(n.get('block_read_avg_mbps'))} / {format_rate(n.get('block_write_avg_mbps'))}</td>
        </tr>""")

    def sum_metric(nodes, key):
        vals = [n.get(key) for n in nodes if n.get(key) is not None and n.get(key) != ""]
        return sum(vals) if vals else None

    cluster_net_rx = sum_metric(node_results, "network_rx_avg_mbps")
    cluster_net_tx = sum_metric(node_results, "network_tx_avg_mbps")
    cluster_block_read = sum_metric(node_results, "block_read_avg_mbps")
    cluster_block_write = sum_metric(node_results, "block_write_avg_mbps")

    cluster_net_display = (
        f"{format_rate(cluster_net_rx)} / {format_rate(cluster_net_tx)}"
        if cluster_net_rx is not None or cluster_net_tx is not None
        else "-"
    )
    cluster_block_display = (
        f"{format_rate(cluster_block_read)} / {format_rate(cluster_block_write)}"
        if cluster_block_read is not None or cluster_block_write is not None
        else "-"
    )

    # Append Cluster Aggregate row
    node_rows_html.append(f"""
    <tr class="total-row">
      <td><strong>CLUSTER TOTAL</strong></td>
      <td class="num">{len(workload_rows)}</td>
      <td class="num">{cluster_cpu_avg_display}</td>
      <td class="num bold">{cluster_cpu_peak_display}</td>
      <td class="num">{cluster_mem_avg_display}</td>
      <td class="num bold">{cluster_mem_peak_display}</td>
      <td class="num">{cluster_net_display}</td>
      <td class="num">{cluster_block_display}</td>
    </tr>""")

    node_table_body = "\n".join(node_rows_html)

    # Latency cards
    min_lat = publisher_stats.get("min", "-")
    avg_lat = publisher_stats.get("avg", "-")
    p50_lat = publisher_stats.get("P50", "-")
    p90_lat = publisher_stats.get("P90", "-")
    p99_lat = publisher_stats.get("P99", "-")
    p999_lat = publisher_stats.get("P99.9", "-")
    max_lat = publisher_stats.get("max", "-")
    pub_bandwidth = publisher_stats.get("bandwidth", "-")
    sub_bandwidth = subscriber_stats.get("bandwidth", "-")

    actual_sub_num = num(subscriber_rate) or 0
    if actual_pub_num > 0 and actual_sub_num > 0:
        delivery_pct = f"{(actual_sub_num / actual_pub_num * 100.0):.1f}% delivered"
    else:
        delivery_pct = ""

    pub_bandwidth_sub = f" &bull; Bandwidth: {pub_bandwidth}" if pub_bandwidth != "-" else ""
    delivery_sub = f" &bull; {delivery_pct}" if delivery_pct else ""

    if scenario == "publish":
        scenario_display = "Core NATS (Publish-only)"
        pattern_display = "No Subscribers"
    elif scenario == "subscribe_setup":
        scenario_display = "Core NATS (Subscribe Setup)"
        pattern_display = f"Concurrent Connections ({subs} clients)"
    elif subscription_pattern == "queue":
        scenario_display = "Core NATS (Publish + Subscribe)"
        pattern_display = f"Queue Group '{queue_name}' ({subs} workers, 1x fan-out)"
    else:
        scenario_display = "Core NATS (Publish + Subscribe)"
        pattern_display = f"Normal Pub/Sub ({subs} subscribers, {fanout_ratio}x fan-out)"

    subject_dist_display = "Single Subject" if subject_distribution == "single" else f"Multi-subject (up to {multi_subject_max})"

    if scenario == "subscribe_setup":
        conns_val = setup_stats.get("established", subs)
        rate_val = setup_stats.get("rate", "-")
        dur_val = setup_stats.get("duration", actual_duration)
        kpi_cards_html = f"""
    <div class="card-kpi">
      <div class="kpi-label">Established Connections</div>
      <div class="kpi-value" style="color:var(--accent);">{conns_val} <span style="font-size:0.9rem; font-weight:400;">clients</span></div>
      <div class="kpi-sub">Hold duration: {actual_duration}s</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Connection Rate</div>
      <div class="kpi-value" style="color:var(--green);">{rate_val} <span style="font-size:0.9rem; font-weight:400;">conns/s</span></div>
      <div class="kpi-sub">Setup elapsed: {dur_val}s</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak CPU</div>
      <div class="kpi-value" style="color:var(--purple);">{cluster_cpu_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_cpu_avg_display} across cluster</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak RAM</div>
      <div class="kpi-value" style="color:var(--yellow);">{cluster_mem_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_mem_avg_display} across cluster</div>
    </div>"""
    elif scenario == "publish":
        kpi_cards_html = f"""
    <div class="card-kpi">
      <div class="kpi-label">Actual Publish Rate</div>
      <div class="kpi-value" style="color:var(--accent);">{publish_rate if publish_rate != "" else "-"} <span style="font-size:0.9rem; font-weight:400;">msgs/s</span></div>
      <div class="kpi-sub">{pub_target_desc}{pub_bandwidth_sub}</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Subscriber Delivery Rate</div>
      <div class="kpi-value" style="color:var(--text-sec);">N/A <span style="font-size:0.9rem; font-weight:400;">(Publish-only)</span></div>
      <div class="kpi-sub">No active subscribers in this scenario</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak CPU</div>
      <div class="kpi-value" style="color:var(--purple);">{cluster_cpu_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_cpu_avg_display} across cluster</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak RAM</div>
      <div class="kpi-value" style="color:var(--yellow);">{cluster_mem_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_mem_avg_display} across cluster</div>
    </div>"""
    else:
        sub_desc = f"Bandwidth: {sub_bandwidth} &bull; Fan-out: {fanout_ratio}x ({delivery_pct})" if delivery_pct else f"Bandwidth: {sub_bandwidth} &bull; Fan-out: {fanout_ratio}x"
        kpi_cards_html = f"""
    <div class="card-kpi">
      <div class="kpi-label">Actual Publish Rate</div>
      <div class="kpi-value" style="color:var(--accent);">{publish_rate if publish_rate != "" else "-"} <span style="font-size:0.9rem; font-weight:400;">msgs/s</span></div>
      <div class="kpi-sub">{pub_target_desc}{pub_bandwidth_sub}</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Subscriber Delivery Rate</div>
      <div class="kpi-value" style="color:var(--green);">{subscriber_rate if subscriber_rate != "" else "-"} <span style="font-size:0.9rem; font-weight:400;">msgs/s</span></div>
      <div class="kpi-sub">{sub_desc}</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak CPU</div>
      <div class="kpi-value" style="color:var(--purple);">{cluster_cpu_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_cpu_avg_display} across cluster</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak RAM</div>
      <div class="kpi-value" style="color:var(--yellow);">{cluster_mem_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_mem_avg_display} across cluster</div>
    </div>"""

    # Config rows
    config_items = []
    if config_content:
        for line in config_content.strip().splitlines():
            if "=" in line:
                k, v = line.split("=", 1)
                config_items.append((k.strip(), v.strip()))
            elif ":" in line:
                k, v = line.split(":", 1)
                config_items.append((k.strip(), v.strip()))

    config_rows_html = []
    for k, v in config_items:
        config_rows_html.append(f"""
        <tr>
          <td class="config-key">{html.escape(k)}</td>
          <td class="config-val">{html.escape(v)}</td>
        </tr>""")
    config_table_body = "\n".join(config_rows_html) if config_rows_html else "<tr><td colspan='2'>No configuration records found</td></tr>"

    html_content = f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>NATS Benchmark Report: {html.escape(test_id)}</title>
  <style>
    * {{
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }}
    :root {{
      --bg: #0d1117;
      --bg-card: #161b22;
      --bg-hover: #1c2128;
      --border: #30363d;
      --text: #e6edf3;
      --text-sec: #8b949e;
      --accent: #58a6ff;
      --green: #3fb950;
      --yellow: #d29922;
      --red: #f85149;
      --purple: #bc8cff;
      --radius: 6px;
    }}
    body {{
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 24px;
      font-size: 14px;
    }}
    .container {{
      max-width: 1200px;
      margin: 0 auto;
    }}
    .header {{
      background: var(--bg-card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 20px 24px;
      margin-bottom: 20px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 16px;
    }}
    .header-left h1 {{
      font-size: 1.3rem;
      font-weight: 600;
      color: var(--text);
      margin-bottom: 4px;
    }}
    .header-subtitle {{
      color: var(--text-sec);
      font-size: 0.85rem;
    }}
    .badge {{
      display: inline-block;
      padding: 4px 12px;
      border-radius: 12px;
      font-size: 0.8rem;
      font-weight: 700;
      letter-spacing: 0.5px;
    }}
    .badge-pass {{
      background: rgba(63, 185, 80, 0.15);
      color: var(--green);
      border: 1px solid rgba(63, 185, 80, 0.4);
    }}
    .badge-fail {{
      background: rgba(248, 81, 73, 0.15);
      color: var(--red);
      border: 1px solid rgba(248, 81, 73, 0.4);
    }}
    .grid-kpi {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 16px;
      margin-bottom: 20px;
    }}
    .card-kpi {{
      background: var(--bg-card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 16px;
    }}
    .kpi-label {{
      font-size: 0.72rem;
      text-transform: uppercase;
      color: var(--text-sec);
      letter-spacing: 0.5px;
      margin-bottom: 6px;
    }}
    .kpi-value {{
      font-size: 1.6rem;
      font-weight: 700;
      color: var(--text);
      font-variant-numeric: tabular-nums;
    }}
    .kpi-sub {{
      font-size: 0.75rem;
      color: var(--text-sec);
      margin-top: 4px;
    }}
    .card {{
      background: var(--bg-card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 20px;
      margin-bottom: 20px;
    }}
    .card-title {{
      font-size: 0.95rem;
      font-weight: 600;
      margin-bottom: 14px;
      color: var(--text);
      display: flex;
      justify-content: space-between;
      align-items: center;
    }}
    .grid-latencies {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
      gap: 12px;
    }}
    .lat-box {{
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 12px;
      text-align: center;
    }}
    .lat-box .lat-label {{
      font-size: 0.7rem;
      color: var(--text-sec);
      text-transform: uppercase;
      margin-bottom: 4px;
    }}
    .lat-box .lat-value {{
      font-size: 1.15rem;
      font-weight: 700;
      color: var(--accent);
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
    }}
    .table-wrap {{
      overflow-x: auto;
      border: 1px solid var(--border);
      border-radius: var(--radius);
    }}
    table {{
      width: 100%;
      border-collapse: collapse;
      font-size: 0.82rem;
      text-align: left;
    }}
    th {{
      background: #1c2128;
      color: var(--text-sec);
      font-weight: 600;
      padding: 10px 14px;
      border-bottom: 1px solid var(--border);
      white-space: nowrap;
    }}
    td {{
      padding: 9px 14px;
      border-bottom: 1px solid var(--border);
      white-space: nowrap;
    }}
    tr:last-child td {{
      border-bottom: none;
    }}
    tr:hover td {{
      background: var(--bg-hover);
    }}
    tr.total-row td {{
      background: #1c2128;
      font-weight: 700;
      border-top: 2px solid var(--border);
    }}
    td.num {{
      text-align: right;
      font-variant-numeric: tabular-nums;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
    }}
    td.bold {{
      font-weight: 700;
      color: var(--accent);
    }}
    .config-table {{
      width: 100%;
    }}
    .config-key {{
      color: var(--text-sec);
      width: 35%;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
    }}
    .config-val {{
      color: var(--text);
      font-weight: 500;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
    }}
    details {{
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      margin-bottom: 12px;
      overflow: hidden;
    }}
    summary {{
      padding: 10px 14px;
      font-weight: 600;
      cursor: pointer;
      user-select: none;
      background: #1c2128;
      color: var(--text);
      font-size: 0.85rem;
    }}
    summary:hover {{
      color: var(--accent);
    }}
    pre {{
      padding: 14px;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.78rem;
      overflow-x: auto;
      white-space: pre-wrap;
      color: var(--text);
      line-height: 1.45;
    }}
    .footer {{
      text-align: center;
      margin-top: 30px;
      font-size: 0.75rem;
      color: var(--text-sec);
    }}
    @media print {{
      body {{
        background: #ffffff;
        color: #000000;
        padding: 0;
      }}
      .card, .header, .card-kpi, .lat-box {{
        background: #ffffff !important;
        border: 1px solid #cccccc !important;
        box-shadow: none !important;
        color: #000000 !important;
      }}
      th {{
        background: #f0f0f0 !important;
        color: #000000 !important;
      }}
      td {{
        color: #000000 !important;
      }}
      details summary {{
        background: #f0f0f0 !important;
        color: #000000 !important;
      }}
      pre {{
        background: #f8f8f8 !important;
        color: #111111 !important;
      }}
    }}
  </style>
</head>
<body>

<div class="container">

  <!-- Header -->
  <div class="header">
    <div class="header-left">
      <h1>NATS Capacity Benchmark Report</h1>
      <div class="header-subtitle">
        Test ID: <strong>{html.escape(test_id)}</strong> &bull;
        Scenario: <strong>{html.escape(scenario_display)}</strong> &bull;
        Pattern: <strong>{html.escape(pattern_display)}</strong> &bull;
        Subject: <strong>{html.escape(subject)} ({subject_dist_display})</strong> &bull;
        Duration: <strong>{actual_duration}s</strong> (nominal: {nominal_duration}s)
      </div>
    </div>
    <div>
      <span class="badge {status_badge_class}">{status_label}</span>
    </div>
  </div>

  <!-- KPI Cards -->
  <div class="grid-kpi">
{kpi_cards_html}
  </div>

  <!-- Publisher Latency Section -->
  <div class="card">
    <div class="card-title">
      <span>Publisher Client Latency Distribution</span>
      <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">Throughput cap: {target_rate} msg/s &bull; Payload: {msg_size} B</span>
    </div>
    <div class="grid-latencies">
      <div class="lat-box">
        <div class="lat-label">Min</div>
        <div class="lat-value">{min_lat}</div>
      </div>
      <div class="lat-box">
        <div class="lat-label">Avg</div>
        <div class="lat-value">{avg_lat}</div>
      </div>
      <div class="lat-box">
        <div class="lat-label">P50</div>
        <div class="lat-value">{p50_lat}</div>
      </div>
      <div class="lat-box">
        <div class="lat-label">P90</div>
        <div class="lat-value">{p90_lat}</div>
      </div>
      <div class="lat-box">
        <div class="lat-label">P99</div>
        <div class="lat-value">{p99_lat}</div>
      </div>
      <div class="lat-box">
        <div class="lat-label">P99.9</div>
        <div class="lat-value">{p999_lat}</div>
      </div>
      <div class="lat-box">
        <div class="lat-label">Max</div>
        <div class="lat-value">{max_lat}</div>
      </div>
    </div>
  </div>

  <!-- Node Infrastructure Breakdown -->
  <div class="card">
    <div class="card-title">
      <span>Cluster Node Infrastructure Resource Summary</span>
      <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">From node-summary.csv &bull; Workload Phase Samples</span>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Container Name</th>
            <th style="text-align:right;">Samples</th>
            <th style="text-align:right;">CPU Avg</th>
            <th style="text-align:right;">CPU Peak</th>
            <th style="text-align:right;">Memory Avg</th>
            <th style="text-align:right;">Memory Peak</th>
            <th style="text-align:right;">Net RX / TX</th>
            <th style="text-align:right;">Block Read / Write</th>
          </tr>
        </thead>
        <tbody>
          {node_table_body}
        </tbody>
      </table>
    </div>
  </div>

  <!-- Test Configuration -->
  <div class="card">
    <div class="card-title">
      <span>Benchmark Configuration</span>
      <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">From test-config.txt &amp; runtime parameters</span>
    </div>
    <div class="table-wrap">
      <table class="config-table">
        <tbody>
          {config_table_body}
        </tbody>
      </table>
    </div>
  </div>

  <!-- Raw Logs (Collapsible) -->
  <div class="card">
    <div class="card-title">Execution Console Logs</div>

    <details open>
      <summary>Publisher Log (workload.log)</summary>
      <pre>{html.escape(workload_log_content) if workload_log_content else "No publisher log content available"}</pre>
    </details>

    <details open>
      <summary>Subscriber Log (subscriber.log)</summary>
      <pre>{html.escape(subscriber_log_content) if subscriber_log_content else "No subscriber log content available"}</pre>
    </details>
  </div>

  <div class="footer">
    NATS Platform Capacity Benchmark &bull; Raw time-series telemetry preserved in <code>metrics.csv</code> &bull; Offline Dashboard
  </div>

</div>

</body>
</html>
"""

    report_path = result_dir / "report.html"
    report_path.write_text(html_content, encoding="utf-8")


generate_report_html()
