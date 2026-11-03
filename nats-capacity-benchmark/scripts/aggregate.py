#!/usr/bin/env python3

import csv
import re
import statistics
import sys

from collections import defaultdict
from pathlib import Path


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
) = sys.argv[1:]

result_dir = Path(result_dir)


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
# Extract workload rates
# -------------------------

def extract_rate(path, label):

    if not path.exists():
        return ""

    content = path.read_text(errors="replace")

    matches = re.findall(
        rf"{label}\s+stats:\s*([\d,]+)\s+msgs/sec",
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


# -------------------------
# Build summary
# -------------------------

summary = {

    "test_id": test_id,
    "messaging_type": "Core NATS",
    "subject": subject,

    "target_publish_msgs_per_sec": target_rate,
    "actual_publish_msgs_per_sec": publish_rate,
    "actual_subscriber_msgs_per_sec": subscriber_rate,

    "message_size_bytes": msg_size,

    "publisher_concurrency": pubs,
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
    "subscriber_log": "subscriber.log"
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