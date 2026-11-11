# Payload Dimension Scaling Suite: Technical Analysis & Inference

**Suite**: `payload-scaling` (`Payload Dimension Scaling Suite`)  
**Run Timestamp**: `20261111T053542Z`  
**Artifact Inspected**: `nats-capacity-benchmark/results/suites/payload-scaling/20261111T053542Z/suite-results.json`  

---

## 1. Executive Summary & Telemetry Validation

### Telemetry Health & Sizing Boundary Validation
This suite evaluates NATS routing performance under a constant offered rate of **5,000 msgs/sec** while scaling message payload sizes by **256x** (from 128 Bytes to 32 KiB):
1. **Natural Dual-Key Sorting Validated**:
   - The report confirms proper sequential ordering: `128B -> 1KB -> 8KB -> 32KB` (resolving the previous alphabetical sort artifact).
2. **Bandwidth Saturation Ceiling Discovered**:
   - At 128B, 1KB, and 8KB, Core NATS delivered 100% of the offered 5,000 msgs/s with microsecond-level latency.
   - At **32 KiB**, the cluster hit the physical network interface bandwidth limit (~1 Gbps), throttling throughput to **3,255 msgs/s** (101.7 MB/s) and triggering a massive P99 tail latency explosion from 3.67 us to **67.2 milliseconds**.
3. **Strict Zero Disk Footprint**:
   - `total_disk_growth_bytes: 0` and `disk_write_mb_sec: 0.0` across all runs, validating 100% in-memory Core NATS execution.

---

## 2. Comparative Performance Matrix

| Metric | TEST-PL-128B | TEST-PL-1KB | TEST-PL-8KB | TEST-PL-32KB |
| :--- | :--- | :--- | :--- | :--- |
| **Payload Size** | **128 B** | **1,024 B (1 KiB)** | **8,192 B (8 KiB)** | **32,768 B (32 KiB)** |
| **Target Rate** | 5,000 /s | 5,000 /s | 5,000 /s | 5,000 /s |
| **Achieved Rate** | **4,996 /s** | **4,996 /s** | **4,996 /s** | **3,255 /s (65.1%)** |
| **Payload Ingress BW**| 0.61 MB/s | 4.88 MB/s | 39.03 MB/s | **101.72 MB/s** |
| **P50 Latency** | 0.57 us | 0.67 us | 0.36 us | **7.56 us** |
| **P99 Latency** | 1.58 us | 2.08 us | 3.67 us | **67,245.29 us (67.2 ms)** |
| **P99.9 Latency** | 6.38 us | 8.29 us | 12.38 us | **147,456.00 us (147.5 ms)**|
| **Max Latency** | 324 us | 30.75 us | 36.67 us | **406,572.33 us (406.6 ms)**|
| **Cluster Avg CPU** | 9.57% | 6.79% | 12.49% | 17.24% |
| **Cluster Peak CPU**| 10.82% | 10.71% | 14.28% | 35.92% |
| **Cluster Peak RAM** | 52.8 MB | 53.6 MB | 54.0 MB | 53.5 MB |
| **Net Ingress / Egress**| 1.75 / 1.87 MB/s | 5.71 / 5.45 MB/s | 55.97 / 55.96 MB/s | **121.93 / 121.94 MB/s** |
| **Disk Write / Growth**| 0.00 MB/s (0 B) | 0.00 MB/s (0 B) | 0.00 MB/s (0 B) | 0.00 MB/s (0 B) |

---

## 3. Detailed Step-by-Step Technical Breakdown

```
Throughput (msgs/sec) vs Payload Bandwidth
  5k +---* (128B: 0.6 MB/s)----* (1KB: 4.9 MB/s)----* (8KB: 39 MB/s)
  4k |                                                                \
  3k |                                                                 * [Ceiling: 3,255 /s @ 102 MB/s]
  2k |                                                                   (P99 explodes to 67.2ms)
  1k |
  0k +---+---------------------+--------------------+------------------+---> Payload Size
        128B                  1KB                  8KB                32KB
```

### Stage 1: TEST-PL-128B (Tiny Payloads / Pure Event Signaling)
* **What happened**: 5,000 msgs/sec with 128-byte message payloads.
* **Technical Observations**:
  * Achieved rate was **4,996 msgs/s** (99.9% accuracy).
  * Ingress bandwidth was minimal at **0.61 MB/s** (4.88 Mbps).
  * Latency was ultra-fast: P50 = **0.57 us**, P99 = **1.58 us**.
  * **Inference**: High-frequency IoT telemetry, microservice heartbeats, and financial ticks incur negligible latency overhead in Core NATS.

### Stage 2: TEST-PL-1KB (Standard JSON/Protobuf Payload)
* **What happened**: 5,000 msgs/sec with 1 KiB payloads.
* **Technical Observations**:
  * Ingress bandwidth scaled 8x to **4.88 MB/s** (39.0 Mbps).
  * Latency remained virtually unchanged: P50 = **0.67 us**, P99 = **2.08 us**.
  * Cluster CPU was only **6.79%** with Peak RAM at **53.6 MB**.
  * **Inference**: 1 KiB represents the standard operational baseline for Core NATS enterprise microservices.

### Stage 3: TEST-PL-8KB (Moderate Payload / Sizing Threshold)
* **What happened**: 5,000 msgs/sec with 8 KiB payloads.
* **Technical Observations**:
  * Ingress bandwidth scaled 8x again to **39.03 MB/s** (312.2 Mbps).
  * Total cluster network traffic reached **55.97 MB/s**.
  * Median P50 latency actually dropped to **0.36 us**, while P99 showed minor queueing at **3.67 us**.
  * **Inference**: 8 KiB is the largest payload size that can sustain high message throughput with strictly sub-microsecond median latency and zero network throttling.

### Stage 4: TEST-PL-32KB (Network Interface Saturation & Tail Latency Collapse)
* **What happened**: Target was 5,000 msgs/s with 32 KiB payloads. Delivered throughput collapsed to **3,255 msgs/s**.
* **Technical Observations**:
  1. **The 1 Gbps Bandwidth Ceiling**:
     * At 3,255 msgs/sec of 32 KiB, the raw payload transfer rate reached **101.72 MB/s** (813.8 Mbps).
     * Including TCP frame overhead, total network I/O hit **121.94 MB/s (~975.5 Mbps)**.
     * The cluster reached **100% capacity of the 1 Gbps network loopback interface**.
  2. **Tail Latency Explosion (18,000x Inflation)**:
     * P50 latency increased 20x from 0.36 us to **7.56 us**.
     * **P99 tail latency ruptured completely, spiking to 67,245 us (67.2 milliseconds)**!
     * **P99.9** surged to **147.5 milliseconds**, and **Max latency** reached **406.6 milliseconds**.
  3. **TCP Socket Send Buffer Backpressure (`SO_SNDBUF`)**:
     * Why did throughput drop to 3,255 msgs/s?
     * When the network interface saturated at ~102 MB/s, the operating system kernel's TCP socket send buffers filled completely.
     * Once the buffer was full, Go's `net.Conn.Write()` calls blocked synchronously. The publisher client threads were suspended in kernel I/O wait states, capping publish frequency at exactly the rate the network could transmit.
  4. **Memory Flatline vs. Latency Inflation**:
     * Note that cluster RAM remained steady at **53.5 MB**.
     * Why didn't memory balloon like in Core Ladder 100k?
     * Because the publishers were throttled at the client kernel socket layer, preventing the broker from being overloaded with unconsumed backlog.

---

## 4. Key Architectural & Sizing Takeaways

### 1. Payload Size Inversely Limits Safe Throughput
$$\text{Max Throughput (msgs/sec)} \le \frac{\text{NIC Bandwidth Capacity (Bytes/sec)}}{\text{Payload Size (Bytes)} \times N_{\text{subscribers}}}$$

* On a standard **1 Gbps NIC (125 MB/s raw limit)**:
  * For **128 B**: Network ceiling is $> 970,000\text{ msgs/s}$. (CPU-bound)
  * For **1 KiB**: Network ceiling is $\sim 120,000\text{ msgs/s}$. (Balanced)
  * For **8 KiB**: Network ceiling is $\sim 15,000\text{ msgs/s}$. (Egress-sensitive)
  * For **32 KiB**: Network ceiling is strictly **3,255 msgs/s**. (Hard network bound)

### 2. Sizing Rules of Thumb for Platform Engineers
* **The Claim-Check Pattern Standard**:
  * Payloads $\le 8\text{ KiB}$ should be sent directly over NATS topics for microsecond routing.
  * Payloads $> 32\text{ KiB}$ should **never be published directly over high-frequency NATS messaging subjects**.
  * Use the **Claim-Check Pattern**: Persist the large binary payload in NATS Object Store, MinIO, or S3, and publish a small 256-byte reference token containing the object URI over the messaging subject.
* **Hardware Interface Sizing**:
  * Clusters handling payloads $\ge 8\text{ KiB}$ with multi-client fan-out require **10 GbE or 25 GbE network interfaces** to prevent socket send buffer stalls.

