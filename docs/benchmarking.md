# NATS Capacity Benchmarking: Architecture & Methodology Design

## 1. Executive Overview & Architectural Purpose

### 1.1 The Architectural Value Proposition
In enterprise event-driven architectures, messaging platforms serve as the nervous system connecting distributed microservices, telemetry ingestion pipelines, and transactional workflows. Sizing messaging infrastructure has historically relied on rough estimates, vendor claims, or painful trial-and-error in production.

This repository provides an empirical, reproducible benchmarking framework and reference standard for NATS (Core in-memory routing and JetStream persistence). It replaces sizing guesswork with **scientifically measured throughput ceilings, tail latency envelopes, and resource consumption coefficients**.

The benchmarking suite enables enterprise teams to:
* **Quantify Core NATS vs JetStream Durability Costs**: Measure the exact throughput and latency trade-offs between in-memory pub/sub, in-memory replicated streaming (Memory R=3), and synchronous disk persistence (File R=3).
* **Right-Size Node and Container Resources**: Derive deterministic requirements for vCPU quotas, memory limits (preventing Linux OOM killer invocations), and storage IOPS tiering.
* **Identify Saturation Knees & Bottlenecks**: Pinpoint the precise hardware limits (socket buffer bloat, rotational/SSD `fsync()` limits, multi-Raft heartbeat contention, consumer ACK amplification) where tail latency inflects.
* **Establish Enterprise NATS Reference Standards**: Enforce clean subject hierarchies, stream quotas, and consumer patterns aligned with production operations.

The framework adheres to the Linux Foundation's **OpenMessaging Benchmark (OMB)** streaming specification, **Brendan Gregg's USE Method** (Systems Performance), and **ISO/IEC 25010/25023 Performance Efficiency** standards.

---

## 2. Benchmarking Test Scenarios & Operational Workload Model

The benchmarking harness evaluates two distinct messaging engines within NATS: **Core NATS** (in-memory broadcast and load-balanced queueing) and **JetStream** (replicated, disk-backed streaming).

Across all benchmark suites, the harness concentrates on the **End-to-End Publish/Subscribe Operational Model**. Rather than evaluating isolated publisher ingress or idle subscriber connections in a vacuum, every test exercises full end-to-end traffic flows: publishers inject load, the server routes or persists messages, and concurrent subscribers or consumers receive and acknowledge deliveries in real time.

### 2.1 Core NATS Operational Scenarios

Core NATS operates strictly in-memory. Workload tests execute under the `pub-sub` scenario:

| Operational Pattern | Delivery Mechanism | Primary Objective |
| :--- | :--- | :--- |
| **Normal Pub/Sub (Broadcast)** | Every subscriber receives an independent copy of each message ($\text{Fan-Out} = N_{\text{subscribers}}$). | Measure end-to-end routing latency, multi-client fan-out overhead, and network egress bandwidth scaling. |
| **Queue Groups (Load Balancing)** | Messages published to a subject are distributed among worker group members ($\text{Fan-Out} = 1\times$). | Measure worker group distribution fairness, message balancing efficiency, and capped network egress. |

#### Architectural Considerations:
* **Strict In-Memory Semantics**: Core NATS routes messages exclusively in RAM; disk block I/O is zero.
* **Buffer Bloat Identification**: When publishers transmit faster than subscribers can consume, the server buffers pending data in memory. Rapid memory growth during a Core NATS test is a direct diagnostic for slow consumers and TCP socket buffer queueing.

### 2.2 JetStream Operational Scenarios

JetStream introduces persistence, Raft consensus replication, and consumer state tracking. Workload tests execute under the `js-pub-sync` scenario:

| Operational Pattern | Replication & Delivery Flow | Primary Objective |
| :--- | :--- | :--- |
| **Synchronous Replicated Publishing** | Publishers append messages and synchronously wait for Raft quorum commit acknowledgments (`PUB-ACK`). | Measure synchronous disk `fsync()` write batching, Raft commit latency, and server write bandwidth (MB/s). |
| **Concurrent Durable Consumer Fan-Out** | Publishers write to the stream while concurrent durable pull consumers retrieve and explicitly acknowledge messages (`+ACK`). | Measure simultaneous disk write contention, network egress multiplication, and ACK consensus commit amplification. |

#### Architectural Considerations:
* **Raft Consensus Quorum ($R=3$)**: Every message append requires physical persistence and acknowledgment from a quorum (2 of 3 nodes) before the server issues a `PUB-ACK`.
* **Stateful Consumer Progress**: With durable consumers and explicit ACK policies, consumer progress updates (sequence floor, pending list) are persisted via Raft consensus, multiplying disk write operations.
* **Resource Isolation & Quiescence**: Streams and consumers are managed programmatically with a mandatory 10-second settling window (`SETTLE_SECONDS=10`) between sequential runs to guarantee zero cross-test interference.

---

## 3. Infrastructure Baseline & Deployment Architecture

To guarantee scientific validity and run-over-run reproducibility, the underlying infrastructure topology remains strictly constant throughout benchmarking:

```
[ Benchmark Client (Host) ]
            |
            v  (Port :4222)
    [ HAProxy L4 LB ]  (0.5 vCPU, 512 MB RAM)
       /    |    \
      /     |     \
     v      v      v
 [nats-1] [nats-2] [nats-3]  (Each: 1.0 vCPU, 1 GB RAM, Disk Quotas)
    |<====== Full-Mesh Cluster Routes (Port :6222) ======>|
```

### Infrastructure Specifications:
* **Node Count**: 3-node NATS cluster running in full-mesh clustering (`nats-1`, `nats-2`, `nats-3`).
* **Resource Constraints**: Fixed CPU quotas (`cpus: 1.0`) and memory limits (`mem_limit: 1g`) enforced via container configuration.
* **Client Ingress**: Client connections access the cluster exclusively through an L4 TCP load balancer (HAProxy) using round-robin distribution to prevent single-node ingress bias.
* **Client Isolation**: Benchmark workload clients run on the host system outside the container runtime, preventing client-side CPU contention from starving server nodes.

---

## 4. Capacity Identification Methodology

The benchmark follows a **Time-Bound Fixed Infrastructure + Stepped Workload** methodology:

```
+---------------------------------------------------------------------------------------+
| 1. Preflight Validation (Cluster Quorum, Route Health, Storage Admission Check)       |
+---------------------------------------------------------------------------------------+
                                           |
                                           v
+---------------------------------------------------------------------------------------+
| 2. Idle Baseline (15s Sampling of Background CPU, RAM, and Disk Activity)            |
+---------------------------------------------------------------------------------------+
                                           |
                                           v
+---------------------------------------------------------------------------------------+
| 3. Stepped Workload Execution (Duration-Bound Window per Test Step)                  |
+---------------------------------------------------------------------------------------+
                                           |
                                           v
+---------------------------------------------------------------------------------------+
| 4. Capacity Boundary Identification (Detect Saturation Knee & Bottleneck Resource)    |
+---------------------------------------------------------------------------------------+
```

### Capacity Boundary Criteria:
Maximum sustainable throughput is defined as the highest operating rate where:
1. **Delivery Integrity is 100%**: Zero dropped messages, zero unhandled errors, and 100% acknowledgment rate.
2. **CPU Utilization Maintains Headroom**: Cluster peak CPU remains below 80-85% across all nodes.
3. **Tail Latency Remains Controlled**: Client P99 latency remains within acceptable thresholds (e.g. $<5\text{ ms}$ for Core NATS; $<15\text{ ms}$ for JetStream File R=3).
4. **Inter-Node Cluster Routes Remain Healthy**: No unbounded route buffer accumulation or slow-consumer drops.

### Formal Test Execution Outcomes:
Every test execution records its lifecycle into one of four standard states:
* **`COMPLETED`**: Workload executed as expected within the designated duration with exit code 0.
* **`TIMEOUT`**: Execution exceeded the allowed duration window (e.g. unresponsive process or subscriber drain timeout).
* **`FAILED`**: Publisher, consumer, or benchmark process exited with a runtime error or non-zero exit code (error phase and descriptive message preserved in `error.log`).
* **`INTERRUPTED`**: Workload was aborted or cancelled manually via `Ctrl+C` (`SIGINT`/`SIGTERM`).

---

## 5. The 7 Canonical Benchmark Suites (Executive Overview)

Workload parameter sweeps are partitioned into **7 Canonical Executable Suites**, each isolating one physical dimension to avoid confounding variables:

| Suite ID | Engine | Dimension Isolated | Parameter Range | Primary Architectural Question |
| :--- | :--- | :--- | :--- | :--- |
| **`core-ladder`** | Core NATS | Offered Throughput | 1k to 100k msgs/s | Where does in-memory pub/sub hit socket buffer bloat? |
| **`payload-scaling`** | Core NATS | Message Size | 128 B to 32 KiB | At what payload size does network bandwidth saturate? |
| **`subject-cardinality`**| Core NATS | Taxonomy Cardinality | 1 to 10,000 subjs, `>` | Does deep subject branching degrade routing lookup speed? |
| **`jetstream-ladder`** | JetStream | Sync File Throughput | 250 to 10,000 msgs/s | What is the rotational/SSD disk `fsync()` write ceiling? |
| **`jetstream-guarantees`**| Multi-Tier| Durability Levels | Core vs Mem R3 vs File | What is the exact CPU, latency, and disk tax of durability?|
| **`stream-scaling`** | JetStream | Multi-Raft Density | 1 to 50 streams | What is the control-plane CPU tax of multi-Raft streams? |
| **`consumer-fanout`** | JetStream | Fan-Out & ACKs | 1 to 20 consumers | How do explicit ACKs multiply disk write IOPS? |

> For the complete parameter matrices, test IDs, and execution instructions for all 7 suites, refer to the Test Inventory & Matrix Specification in `docs/tests.md`.

---

## 6. Telemetry & Results Artifact Architecture

Every benchmark execution produces a structured, immutable bundle of telemetry artifacts:

### 6.1 Single-Run Artifacts (`results/<TEST_ID>/`)

| Artifact | Format | Purpose | Primary Consumer |
| :--- | :--- | :--- | :--- |
| **`results.json`** | JSON | Canonical machine-readable metrics: test metadata, client latency percentiles, throughput, and per-node resource rollups. | Automation, CI/CD, Trend Analysis |
| **`report.html`** | HTML | Self-contained, zero-dependency offline dashboard featuring KPI cards, latency tables, per-node hardware telemetry, and run configuration. | Humans, Architects, Engineers |
| **`metrics.json`** | JSON | Time-series container resource telemetry (CPU, RAM, net RX/TX, disk block I/O, physical disk growth) sampled across baseline and workload phases. | Capacity Engineers |
| **`pub.csv`** | CSV | Native publisher benchmark output (MsgCount, MsgBytes, MsgsPerSec, BytesPerSec, full latency percentiles). | Statistical Audit Trail |
| **`sub.csv`** | CSV | Native subscriber benchmark output (MsgCount, Bytes, Delivery rates). | Delivery Verification |
| **`test-config.txt`** | Text | Snapshot of all 20+ runtime configuration parameters. | Repeatability Audit Trail |

### 6.2 Executive Suite Comparative Deliverables (`results/suites/<SUITE_ID>/<TIMESTAMP>/`)

When running automated benchmark suites (`./suite.sh <suite-name>`), individual runs are consolidated into an executive presentation bundle:
* **`suite-results.json`**: Rollup aggregating all child test runs with executive sizing recommendations (sustained capacity, saturation knee, cluster peak headroom).
* **`executive-report.html`** / **`suite-report.html`**: Executive dashboard featuring:
  1. Executive Sizing Recommendations cards.
  2. Inline SVG Throughput vs Latency saturation curve ("hockey stick" curve) with P50/P99 latency knee callouts.
  3. Comparative Ladder Matrix table with auto-detected variant badges.
  4. Interactive child-run drill-down drawer with complete configuration tables.

---

## 7. Phased Implementation Roadmap

* **Phase 1: Baseline (Completed)**: Establishing fundamental Core NATS and JetStream throughput, latency, and resource baselines.
* **Phase 2: Loading (Completed)**: Throughput saturation ladders measuring capacity ceilings and latency inflection knees.
* **Phase 3: Scaling (Completed)**: Dimensional parameter sweeps isolating message payload size, subject taxonomy, stream density, and consumer fan-out.
* **Phase 4: Analysis & Reporting (Completed)**: Technical evaluation reports published across Reports 1 through 7 with empirical constants.
* **Phase 5: Estimation Utility (Upcoming)**: Web-based interactive capacity calculator mapping empirical sizing principles to container and cluster infrastructure.

---

## 8. References

* **OpenMessaging Benchmark (OMB)**: https://openmessaging.cloud/
* **Brendan Gregg - Systems Performance & The USE Method**: https://www.brendangregg.com/usemethod.html
* **ISO/IEC 25010 / 25023 Systems and Software Quality Requirements**: https://iso25000.com/index.php/en/iso-25000-standards/iso-25010
* **Neil Gunther - Universal Scalability Law (USL)**: http://www.perfdynamics.com/Manifesto/USLscalability.html
* **Synadia / McKnight Consulting Group NATS TCO Study**: https://www.synadia.com/downloads/nats-kafka-tco-report.pdf
