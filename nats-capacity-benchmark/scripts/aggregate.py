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
test_description = ""
run_timestamp = ""

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
            elif k == "test_description":
                test_description = v
            elif k == "run_timestamp":
                run_timestamp = v

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


def process_phase_samples(phase_rows):
    nodes = defaultdict(list)
    for row in phase_rows:
        nodes[row.get("container", "unknown")].append(row)

    results = []
    for node, samples in nodes.items():
        samples.sort(key=lambda row: num(row.get("timestamp_epoch")) or 0)
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

            memory_text = (row.get("memory_usage") or "").split("/", 1)[0].strip()
            memory = bytes_value(memory_text)
            if memory is not None:
                memory_values.append(memory)

            if previous:
                t0 = num(previous.get("timestamp_epoch"))
                t1 = num(row.get("timestamp_epoch"))
                elapsed = (t1 - t0 if t0 is not None and t1 is not None else 0)
                if elapsed > 0:
                    old_rx, old_tx = pair_bytes(previous.get("network_io"))
                    new_rx, new_tx = pair_bytes(row.get("network_io"))
                    old_read, old_write = pair_bytes(previous.get("block_io"))
                    new_read, new_write = pair_bytes(row.get("block_io"))

                    pairs = [
                        (old_rx, new_rx, network_rx_rates),
                        (old_tx, new_tx, network_tx_rates),
                        (old_read, new_read, disk_read_rates),
                        (old_write, new_write, disk_write_rates)
                    ]
                    for old, new, destination in pairs:
                        if old is not None and new is not None and new >= old:
                            mbps = (new - old) / elapsed / 1_000_000
                            destination.append(mbps)
            previous = row

        def average(values):
            return round(statistics.fmean(values), 3) if values else ""

        def peak(values):
            return round(max(values), 3) if values else ""

        results.append({
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

    samples_by_time = defaultdict(list)
    for row in phase_rows:
        samples_by_time[row.get("timestamp_epoch")].append(row)

    cluster_cpu = []
    cluster_memory = []
    for group in samples_by_time.values():
        cpu_vals = [num(row.get("cpu_percent")) for row in group if num(row.get("cpu_percent")) is not None]
        mem_vals = [bytes_value((row.get("memory_usage") or "").split("/", 1)[0].strip()) for row in group if bytes_value((row.get("memory_usage") or "").split("/", 1)[0].strip()) is not None]
        if cpu_vals:
            cluster_cpu.append(sum(cpu_vals))
        if mem_vals:
            cluster_memory.append(sum(mem_vals))

    return results, cluster_cpu, cluster_memory


baseline_rows = [
    row for row in rows
    if row.get("phase") == "baseline"
]

workload_rows = [
    row for row in rows
    if row.get("phase") == "workload"
]

baseline_results, baseline_cluster_cpu, baseline_cluster_memory = process_phase_samples(baseline_rows)
node_results, cluster_cpu, cluster_memory = process_phase_samples(workload_rows)

cluster_baseline_cpu_avg = round(statistics.fmean(baseline_cluster_cpu), 3) if baseline_cluster_cpu else ""
cluster_baseline_cpu_peak = round(max(baseline_cluster_cpu), 3) if baseline_cluster_cpu else ""
cluster_baseline_mem_avg = round(statistics.fmean(baseline_cluster_memory), 0) if baseline_cluster_memory else ""
cluster_baseline_mem_peak = round(max(baseline_cluster_memory), 0) if baseline_cluster_memory else ""

# -------------------------
# Write node summary (Workload Phase)
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
# Extract workload rates & publisher stats
# -------------------------

def extract_rate(path, label):

    if not path.exists():
        return ""

    content = path.read_text(errors="replace")

    # Match aggregated stats (multi-client) or standard stats (single-client)
    pattern = rf"(?:{label}[a-z]*\s+aggregated\s+stats|{label}[a-z]*\s+stats):\s*([\d,.]+)\s+msgs/sec"
    m = re.search(pattern, content, re.I)
    if m:
        return int(round(float(m.group(1).replace(",", ""))))

    # Fallback to any "stats: X msgs/sec" in the file
    matches = re.findall(r"stats:\s*([\d,.]+)\s+msgs/sec", content, re.I)
    if matches:
        return int(round(float(matches[-1].replace(",", ""))))

    return ""


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
        r"(?:publisher\s+(?:aggregated\s+)?stats|stats):\s*([0-9,.]+)\s*msgs/sec\s*~\s*([0-9,.]+\s*[KMGT]?i?B/sec)",
        content,
        re.I
    )
    if m_rate:
        stats["rate"] = m_rate.group(1).replace(",", "")
        stats["bandwidth"] = m_rate.group(2).strip()

    metrics = ["min", "avg", "max", "P50", "P90", "P99", "P99.9"]

    # In multi-client runs, latencies per operation reports aggregated metrics across all concurrent publishers
    m_agg_lat = re.search(r"latencies per operation\s+([^\n\r]+)", content, re.I)
    latency_source = m_agg_lat.group(1) if m_agg_lat else content

    for m in metrics:
        pattern = rf"\b{re.escape(m)}:?\s*([0-9,.]+)\s*([a-zA-Z]+)"
        match = re.search(pattern, latency_source, re.I)
        if match:
            stats[m] = f"{match.group(1)} {match.group(2)}"

    return stats


def extract_subscriber_stats(path):
    if not path.exists():
        return {}
    content = path.read_text(errors="replace")
    stats = {}

    m_rate = re.search(
        r"(?:subscriber\s+(?:aggregated\s+)?stats|stats):\s*([0-9,.]+)\s*msgs/sec\s*~\s*([0-9,.]+\s*[KMGT]?i?B/sec)",
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
    "test_description": test_description,
    "run_timestamp": run_timestamp,
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

    "cluster_baseline_cpu_avg_percent": cluster_baseline_cpu_avg,
    "cluster_baseline_cpu_peak_percent": cluster_baseline_cpu_peak,
    "cluster_baseline_memory_avg_bytes": cluster_baseline_mem_avg,
    "cluster_baseline_memory_peak_bytes": cluster_baseline_mem_peak,

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

    node_count = len(node_results) or len(baseline_results) or 1
    total_cluster_vcpu = float(node_count)

    def format_cpu_cell(cpu_pct, capacity_vcpu=1.0, sub_text=None):
        if cpu_pct is None or cpu_pct == "" or cpu_pct == "-":
            return "-"
        try:
            val = float(cpu_pct)
            vcpu = val / 100.0
            if capacity_vcpu > 1.0:
                vcpu_str = f"{vcpu:.3f} vCPU of {capacity_vcpu:.1f} vCPUs"
            else:
                vcpu_str = f"{vcpu:.3f} vCPU"

            if sub_text:
                sub = f"<span class='sub-metric'>{vcpu_str} &bull; {sub_text}</span>"
            else:
                sub = f"<span class='sub-metric'>{vcpu_str}</span>"
            return f"{val:.2f}%{sub}"
        except (ValueError, TypeError):
            return f"{cpu_pct}%"

    cluster_cpu_peak_val = summary["cluster_cpu_peak_percent"]
    if cluster_cpu_peak_val != "" and cluster_cpu_peak_val is not None:
        try:
            peak_pct = float(cluster_cpu_peak_val)
            peak_vcpu = peak_pct / 100.0
            cluster_cpu_peak_display = f"{peak_pct:.2f}% <span style='font-size:0.9rem; font-weight:400; color:var(--text-sec);'>({peak_vcpu:.3f} vCPU)</span>"
        except (ValueError, TypeError):
            cluster_cpu_peak_display = f"{cluster_cpu_peak_val}%"
    else:
        cluster_cpu_peak_display = "-"

    cluster_cpu_avg_val = summary["cluster_cpu_avg_percent"]
    if cluster_cpu_avg_val != "" and cluster_cpu_avg_val is not None:
        try:
            avg_pct = float(cluster_cpu_avg_val)
            avg_vcpu = avg_pct / 100.0
            cluster_cpu_avg_display = f"{avg_pct:.2f}% ({avg_vcpu:.3f} vCPU of {total_cluster_vcpu:.1f} vCPUs)"
        except (ValueError, TypeError):
            cluster_cpu_avg_display = f"{cluster_cpu_avg_val}% across cluster"
    else:
        cluster_cpu_avg_display = "-"

    cluster_mem_peak_display = format_bytes(summary["cluster_memory_peak_bytes"])
    cluster_mem_avg_display = format_bytes(summary["cluster_memory_avg_bytes"])

    def sum_metric(nodes, key):
        vals = [n.get(key) for n in nodes if n.get(key) is not None and n.get(key) != ""]
        return sum(vals) if vals else None

    baseline_map = {b.get("container"): b for b in baseline_results}

    baseline_rows_html = []
    for b in baseline_results:
        baseline_rows_html.append(f"""
        <tr>
          <td><strong>{html.escape(str(b.get('container', '')))}</strong></td>
          <td class="num">{b.get('samples', 0)}</td>
          <td class="num">{format_cpu_cell(b.get('cpu_avg_percent'), 1.0)}</td>
          <td class="num bold">{format_cpu_cell(b.get('cpu_peak_percent'), 1.0)}</td>
          <td class="num">{format_bytes(b.get('memory_avg_bytes'))}</td>
          <td class="num bold">{format_bytes(b.get('memory_peak_bytes'))}</td>
          <td class="num">{format_rate(b.get('network_rx_avg_mbps'))} / {format_rate(b.get('network_tx_avg_mbps'))}</td>
        </tr>""")

    baseline_net_rx = sum_metric(baseline_results, "network_rx_avg_mbps")
    baseline_net_tx = sum_metric(baseline_results, "network_tx_avg_mbps")
    baseline_net_display = (
        f"{format_rate(baseline_net_rx)} / {format_rate(baseline_net_tx)}"
        if baseline_net_rx is not None or baseline_net_tx is not None
        else "-"
    )
    baseline_cpu_avg_display = f"{cluster_baseline_cpu_avg}%" if cluster_baseline_cpu_avg != "" else "-"
    baseline_cpu_peak_display = f"{cluster_baseline_cpu_peak}%" if cluster_baseline_cpu_peak != "" else "-"
    baseline_mem_avg_display = format_bytes(cluster_baseline_mem_avg)
    baseline_mem_peak_display = format_bytes(cluster_baseline_mem_peak)

    if baseline_results:
        baseline_rows_html.append(f"""
        <tr class="total-row">
          <td><strong>CLUSTER IDLE TOTAL</strong></td>
          <td class="num">{len(baseline_rows)}</td>
          <td class="num">{format_cpu_cell(cluster_baseline_cpu_avg, total_cluster_vcpu)}</td>
          <td class="num bold">{format_cpu_cell(cluster_baseline_cpu_peak, total_cluster_vcpu)}</td>
          <td class="num">{baseline_mem_avg_display}</td>
          <td class="num bold">{baseline_mem_peak_display}</td>
          <td class="num">{baseline_net_display}</td>
        </tr>""")
        baseline_table_body = "\n".join(baseline_rows_html)
        baseline_section_html = f"""
  <!-- Pre-Workload Idle Baseline Breakdown -->
  <div class="card">
    <div class="card-title">
      <span>Pre-Workload Idle Baseline Telemetry</span>
      <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">{len(baseline_rows)} samples across {len(baseline_results)} nodes &bull; Baseline Phase Samples</span>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Container Name</th>
            <th style="text-align:right;">Samples</th>
            <th style="text-align:right;">Idle CPU Avg (% / vCPU)</th>
            <th style="text-align:right;">Idle CPU Peak (% / vCPU)</th>
            <th style="text-align:right;">Idle RAM Avg</th>
            <th style="text-align:right;">Idle RAM Peak</th>
            <th style="text-align:right;">Idle Net RX / TX</th>
          </tr>
        </thead>
        <tbody>
          {baseline_table_body}
        </tbody>
      </table>
    </div>
  </div>"""
    else:
        baseline_section_html = """
  <!-- Pre-Workload Idle Baseline Breakdown -->
  <div class="card">
    <div class="card-title">
      <span>Pre-Workload Idle Baseline Telemetry</span>
      <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">No samples collected</span>
    </div>
    <div style="padding:12px; color:var(--text-sec); font-size:0.8rem;">
      No idle baseline telemetry samples were captured (baseline duration was 0s or metrics backend was none).
    </div>
  </div>"""

    # Table rows for nodes
    node_rows_html = []
    for n in node_results:
        b = baseline_map.get(n.get("container"))
        base_cpu_text = f"idle: {b.get('cpu_avg_percent', '-')}%" if b else None
        base_mem_sub = f"<span class='sub-metric'>idle: {format_bytes(b.get('memory_avg_bytes'))}</span>" if b else ""

        node_rows_html.append(f"""
        <tr>
          <td><strong>{html.escape(str(n.get('container', '')))}</strong></td>
          <td class="num">{n.get('samples', 0)}</td>
          <td class="num">{format_cpu_cell(n.get('cpu_avg_percent'), 1.0, sub_text=base_cpu_text)}</td>
          <td class="num bold">{format_cpu_cell(n.get('cpu_peak_percent'), 1.0)}</td>
          <td class="num">{format_bytes(n.get('memory_avg_bytes'))}</td>
          <td class="num bold">{format_bytes(n.get('memory_peak_bytes'))}{base_mem_sub}</td>
          <td class="num">{format_rate(n.get('network_rx_avg_mbps'))} / {format_rate(n.get('network_tx_avg_mbps'))}</td>
          <td class="num">{format_rate(n.get('block_read_avg_mbps'))} / {format_rate(n.get('block_write_avg_mbps'))}</td>
        </tr>""")

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

    cluster_idle_cpu_text = f"idle: {baseline_cpu_avg_display}" if baseline_results else None
    cluster_base_mem_sub = f"<span class='sub-metric'>idle: {baseline_mem_avg_display}</span>" if baseline_results else ""

    # Append Cluster Aggregate row
    node_rows_html.append(f"""
    <tr class="total-row">
      <td><strong>CLUSTER TOTAL</strong></td>
      <td class="num">{len(workload_rows)}</td>
      <td class="num">{format_cpu_cell(summary['cluster_cpu_avg_percent'], total_cluster_vcpu, sub_text=cluster_idle_cpu_text)}</td>
      <td class="num bold">{format_cpu_cell(summary['cluster_cpu_peak_percent'], total_cluster_vcpu)}</td>
      <td class="num">{cluster_mem_avg_display}</td>
      <td class="num bold">{cluster_mem_peak_display}{cluster_base_mem_sub}</td>
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
      <div class="kpi-sub">Average: {cluster_cpu_avg_display}</div>
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
      <div class="kpi-sub">Average: {cluster_cpu_avg_display}</div>
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
      <div class="kpi-sub">Average: {cluster_cpu_avg_display}</div>
    </div>

    <div class="card-kpi">
      <div class="kpi-label">Cluster Peak RAM</div>
      <div class="kpi-value" style="color:var(--yellow);">{cluster_mem_peak_display}</div>
      <div class="kpi-sub">Average: {cluster_mem_avg_display} across cluster</div>
    </div>"""

    # Parse configuration items
    config_dict = {}
    if config_content:
        for line in config_content.strip().splitlines():
            if "=" in line:
                k, v = line.split("=", 1)
                config_dict[k.strip()] = v.strip()
            elif ":" in line:
                k, v = line.split(":", 1)
                config_dict[k.strip()] = v.strip()

    # Monitored containers fallback from node_results
    if "nats_containers" not in config_dict and node_results:
        node_names = [str(n.get("container", "")) for n in node_results if n.get("container")]
        if node_names:
            config_dict["nats_containers"] = ", ".join(node_names)

    # 1. Setup Details fields
    setup_fields = [
        ("nats_url", "NATS Endpoint", config_dict.get("nats_url", "-")),
        ("messaging_type", "Messaging Platform", config_dict.get("messaging_type", "Core NATS")),
        ("nats_containers", "Monitored Containers", config_dict.get("nats_containers", "-")),
        ("metrics_backend", "Metrics Backend", config_dict.get("metrics_backend", backend)),
        ("metrics_sample_interval_seconds", "Telemetry Sample Interval", f"{config_dict.get('metrics_sample_interval_seconds', '-')}s" if config_dict.get('metrics_sample_interval_seconds') else "-"),
        ("baseline_duration_seconds", "Baseline Phase Duration", f"{config_dict.get('baseline_duration_seconds', '-')}s" if config_dict.get('baseline_duration_seconds') else "-"),
    ]
    known_setup_keys = {"nats_url", "messaging_type", "nats_containers", "metrics_backend", "metrics_sample_interval_seconds", "baseline_duration_seconds"}
    for k, v in config_dict.items():
        if k not in known_setup_keys and any(sub in k.lower() for sub in ["endpoint", "monitor", "replica", "node", "cluster", "backend", "container", "baseline"]):
            setup_fields.append((k, k.replace("_", " ").title(), v))

    # 2. Test Details fields
    test_fields = [
        ("test_id", "Test Run ID", test_id),
    ]
    if test_description:
        test_fields.append(("test_description", "Test Description", test_description))
    if run_timestamp:
        test_fields.append(("run_timestamp", "Execution Timestamp", run_timestamp))
    test_fields.extend([
        ("scenario", "Workload Scenario", scenario_display),
        ("subscription_pattern", "Subscription Pattern", pattern_display),
    ])
    if subscription_pattern == "queue" and queue_name:
        test_fields.append(("queue_name", "Queue Group Name", queue_name))
    test_fields.extend([
        ("subject", "Target Subject", f"{subject} ({subject_dist_display})"),
        ("target_publish_msgs_per_sec", "Target Publish Rate", pub_target_desc if scenario != "subscribe_setup" else "N/A (Connection Setup)"),
        ("message_size_bytes", "Message Payload Size", f"{msg_size} B ({format_bytes(int(msg_size))})" if (msg_size and scenario != "subscribe_setup") else "N/A"),
        ("publisher_concurrency", "Publisher Concurrency", f"{pubs} client(s)" if scenario != "subscribe_setup" else "0 (N/A)"),
        ("subscriber_concurrency", "Subscriber Concurrency", f"{subs} client(s)" if scenario != "publish" else "0 (Publish-only)"),
        ("nominal_duration_seconds", "Nominal Duration", f"{nominal_duration}s (Actual: {actual_duration}s)"),
        ("configured_message_count", "Configured Messages", f"{config_dict.get('configured_message_count', '-')} msgs" if config_dict.get('configured_message_count') else "-"),
    ])
    known_test_keys = {"test_id", "test_description", "run_timestamp", "scenario", "subscription_pattern", "queue_name", "subject", "subject_distribution", "multi_subject_max", "target_publish_msgs_per_sec", "message_size_bytes", "publisher_concurrency", "subscriber_concurrency", "nominal_duration_seconds", "configured_message_count"}
    for k, v in config_dict.items():
        if k not in known_setup_keys and k not in known_test_keys and not any(sub in k.lower() for sub in ["endpoint", "monitor", "replica", "node", "cluster", "backend", "container", "baseline"]):
            test_fields.append((k, k.replace("_", " ").title(), v))

    def render_details_rows(fields):
        rows = []
        for key, label, val in fields:
            if val is None or str(val).strip() == "" or str(val).strip() == "-":
                continue
            rows.append(f"""
            <tr>
              <td class="config-key">{html.escape(label)}<span class="config-raw">{html.escape(key)}</span></td>
              <td class="config-val">{html.escape(str(val))}</td>
            </tr>""")
        return "\n".join(rows) if rows else "<tr><td colspan='2'>No configuration records found</td></tr>"

    setup_table_body = render_details_rows(setup_fields)
    test_table_body = render_details_rows(test_fields)

    doc_title = f"NATS Benchmark Report: {html.escape(test_id)}"
    if test_description:
        doc_title += f" - {html.escape(test_description)}"

    desc_subtitle = f" &bull; Description: <strong>{html.escape(test_description)}</strong>" if test_description else ""

    html_content = f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{doc_title}</title>
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
    .section-title {{
      font-size: 1.05rem;
      font-weight: 600;
      color: var(--text);
      margin: 28px 0 16px 0;
      display: flex;
      align-items: center;
      justify-content: space-between;
      border-bottom: 1px solid var(--border);
      padding-bottom: 8px;
    }}
    .grid-details {{
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 20px;
      margin-bottom: 24px;
      align-items: start;
    }}
    @media (max-width: 900px) {{
      .grid-details {{
        grid-template-columns: 1fr;
      }}
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
    .sub-metric {{
      display: block;
      font-size: 0.7rem;
      color: var(--text-sec);
      font-weight: normal;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      margin-top: 2px;
    }}
    .config-table {{
      width: 100%;
    }}
    .config-key {{
      color: var(--text-sec);
      width: 44%;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      font-size: 0.8rem;
      vertical-align: top;
    }}
    .config-val {{
      color: var(--text);
      font-weight: 600;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 0.82rem;
      word-break: break-all;
    }}
    .config-raw {{
      color: var(--text-sec);
      font-size: 0.7rem;
      font-family: SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-weight: normal;
      display: block;
      opacity: 0.65;
      margin-top: 2px;
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
        Test ID: <strong>{html.escape(test_id)}</strong>{desc_subtitle} &bull;
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

  <!-- 1. Setup Details & 2. Test Details -->
  <div class="grid-details">
    <!-- 1. Setup Details -->
    <div class="card" style="margin-bottom:0;">
      <div class="card-title">
        <span>1. Setup Details</span>
        <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">Environment &amp; Infrastructure</span>
      </div>
      <div class="table-wrap">
        <table class="config-table">
          <tbody>
            {setup_table_body}
          </tbody>
        </table>
      </div>
    </div>

    <!-- 2. Test Details -->
    <div class="card" style="margin-bottom:0;">
      <div class="card-title">
        <span>2. Test Details</span>
        <span style="font-size:0.75rem; color:var(--text-sec); font-weight:normal;">Workload &amp; Parameters</span>
      </div>
      <div class="table-wrap">
        <table class="config-table">
          <tbody>
            {test_table_body}
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <!-- 3. Test Results -->
  <div class="section-title">
    <span>3. Test Results</span>
    <span class="badge {status_badge_class}">{status_label}</span>
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

  <!-- Pre-Workload Idle Baseline Breakdown -->
  {baseline_section_html}

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
            <th style="text-align:right;">CPU Avg (% / vCPU)</th>
            <th style="text-align:right;">CPU Peak (% / vCPU)</th>
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
