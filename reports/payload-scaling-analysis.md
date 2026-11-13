# NATS Behavior Analysis: Payload Dimension Scaling

## 1. Executive Summary

The Payload Dimension Scaling suite (`payload-scaling`, timestamp `20261112T085035Z`) investigates the behavioral and systems impact of message payload size across four orders of magnitude: 128 B, 1 KiB, 8 KiB, and 32 KiB. Tests were executed at a fixed target offered rate of 5,000 msgs/s using 4 concurrent publishers and a 2-worker Queue Group (`bench-queue`), which enforces a 1:1 load-balanced delivery semantic.

Key findings:
1. **Sub-Microsecond Invariance (128 B to 8 KiB)**: Across payloads from 128 bytes up to 8 KiB (39.06 MB/s network ingress), NATS maintains sub-microsecond median routing latency ($p50 \le 0.42 \ \mu\text{s}$) and sub-4-microsecond $p99$ tail latency with flat memory allocation (~56 MB).
2. **Network Interface Saturation Wall (32 KiB)**: At 32 KiB payloads, network throughput reached 103.61 MB/s (~869 Mbps), hitting the physical 1 Gbps interface bandwidth limit. The publisher throughput was throttled from offered 5,000 msgs/s down to 3,312 msgs/s, and $p99$ latency experienced a severe queueing rupture, exploding from $3.42 \ \mu\text{s}$ to $25,122 \ \mu\text{s}$ (25.1 ms).

---

## 2. Empirical Benchmark Data

All tests executed with 4 publishers, 2 subscribers in queue group `bench-queue`, over 30 seconds:

| Test ID | Payload Size | Offered Rate | Achieved Rate | Wire Bandwidth | p50 Latency | p99 Latency | p99.9 Latency | Max Latency | Cluster Avg CPU | Cluster Peak RAM | Memory Growth |
| :--- | ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:|
| **`TEST-PL-128B`** | 128 B | 5,000 /s | 4,996.0 /s | 0.61 MB/s | 0.24 us | 0.79 us | 8.71 us | 52.8 us | 9.26% | 55.8 MB | +7.8 MB |
| **`TEST-PL-1KB`** | 1,024 B | 5,000 /s | 4,996.0 /s | 4.88 MB/s | 0.29 us | 1.92 us | 9.62 us | 3,373 us | 6.86% | 55.6 MB | +2.5 MB |
| **`TEST-PL-8KB`** | 8,192 B | 5,000 /s | 4,996.0 /s | 39.06 MB/s | 0.42 us | 3.42 us | 12.96 us | 983.6 us | 10.61% | 56.2 MB | +3.0 MB |
| **`TEST-PL-32KB`**| 32,768 B | 5,000 /s | 3,312.0 /s | 103.61 MB/s | 5.84 us | 25,122 us | 28,679 us | 186,808 us | 15.43% | 55.8 MB | +2.1 MB |

---

## 3. Systems & Architectural Behavioral Analysis

### 3.1 Packet Fragmentation and TCP Buffer Dynamics (128 B to 8 KiB)
- At 128 bytes and 1,024 bytes, payload sizes fit comfortably within standard Ethernet MTU frames (1,500 bytes). A published message is encapsulated into a single TCP packet. Latency remains virtually identical ($p50 = 0.24 \ \mu\text{s}$ vs $0.29 \ \mu\text{s}$).
- At 8,192 bytes, the message exceeds standard MTU, requiring TCP segmentation into ~6 IP packets per message. Despite this serialization requirement, NATS routes 4,996 msgs/s (39.06 MB/s) with a median latency of only $0.42 \ \mu\text{s}$ and $p99$ of $3.42 \ \mu\text{s}$.
- Cluster memory remained rock-solid across 128 B, 1 KiB, and 8 KiB tests (55.6 MB to 56.2 MB), proving that the Go runtime pool allocator reuses network buffers without heap thrashing even when handling multi-megabyte aggregate streams.

### 3.2 The 1 Gbps Physical Bandwidth Wall (32 KiB)
When message size was increased to 32 KiB:
- **Bandwidth Saturation**: At 3,312 msgs/s, wire transfer rate reached 103.61 MB/s on ingress and ~124 MB/s aggregate across cluster nodes (`nats-1`: 35.7 MB/s tx, `nats-2`: 44.5 MB/s tx, `nats-3`: 44.3 MB/s tx). In bits-per-second, this represents:
  $$103.61 \text{ MB/s} \times 8 \times 1.048576 \approx 869.2 \text{ Mbps}$$
  Accounting for TCP frame headers, IP overhead, and container bridge routing, this fully saturates standard 1 Gbps physical network adapters.
- **TCP Socket Write Blocking**: When the network pipe saturated, the OS kernel TCP send buffer (`SO_SNDBUF`) filled up. The `write()` system calls in client publishers blocked waiting for TCP ACK window clearance.
- **Tail Latency Explosion**:
  - $p50$ latency rose modestly from $0.42 \ \mu\text{s}$ to $5.84 \ \mu\text{s}$.
  - $p99$ latency exploded from $3.42 \ \mu\text{s}$ to $25,122.21 \ \mu\text{s}$ (25.1 milliseconds) - an increase of over 7,300x.
  - Maximum observed latency spiked to $186,807.88 \ \mu\text{s}$ (186.8 ms).
- **Throughput Truncation**: Unable to inject the full 5,000 msgs/s through the saturated network interface, achieved throughput dropped by 33.8% to 3,312 msgs/s.

### 3.3 Queue Group Behavioral Impact
The use of `QUEUE=bench-queue` was pivotal in this evaluation:
- With a queue group, each message is delivered to exactly one worker ($1\times$ fan-out), keeping egress bandwidth equal to ingress bandwidth.
- Had standard broadcast fan-out been used with 2 subscribers, network egress would have reached $103.6 \times 2 = 207.2 \text{ MB/s}$ (~1.7 Gbps), which would have triggered network saturation much earlier at the 8 KiB tier. The queue group isolated payload serialization mechanics from subscriber egress multiplication.

---

## 4. Bottleneck Summary & Thresholds

| Metric | Safe Operating Zone | Saturation Knee | Limiting Factor |
| :--- | :--- | :--- | :--- |
| **Payload Size** | Up to 8 KiB | 32 KiB (at 5k msg/s) | MTU fragmentation and TCP buffer fill |
| **Wire Bandwidth** | Up to 70 MB/s (~600 Mbps) | > 100 MB/s (~850+ Mbps) | 1 Gbps network card / interface capacity |
| **p99 Tail Latency** | < 5 us | > 25,000 us | TCP send socket blocking on saturated wire |

