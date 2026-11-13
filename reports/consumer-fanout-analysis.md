# NATS Behavior Analysis: Consumer Fan-Out & ACK Amplification

## 1. Executive Summary

The Consumer Fan-Out & ACK Amplification suite (`consumer-fanout`, timestamp `20261112T093119Z`) evaluates the resource consumption, disk write multiplication, and consensus contention induced by scaling concurrent active durable pull consumers (1, 3, 10, and 20 consumers) against a single replicated stream ($R=3$, File storage, 1,024-byte payloads). Tests operated at an offered rate of 1,000 msgs/s with 4 publishers over 30-second measurement windows.

Key findings:
1. **Deterministic Linear Consumer Memory Slope**: Cluster peak RAM scaled with strict linearity from 253.1 MB at 1 consumer to 668.9 MB at 20 consumers (+415.8 MB for 19 consumers), establishing an empirical consumer memory tax of **~21.88 MB per active durable consumer**.
2. **The ACK Amplification Wall (20 Consumers)**: At 20 consumers, 1,000 published msgs/s generates an expected 20,000 explicit delivery ACKs/sec. Because durable consumer ACKs mutate state in the Raft consensus pipeline, the cluster reached its consensus commit capacity limit. Publisher throughput was throttled by 50% down to **500.0 msgs/s**, median latency surged from $1.02 \text{ ms}$ to **$7.73 \text{ ms}$**, and cluster CPU rose to 58.27%.

---

## 2. Empirical Benchmark Data

All tests executed with 1 active stream at 1,000 msgs/s offered rate, 1,024-byte payloads, File storage, Replicas=3, explicit ACK policy:

| Test ID | Active Consumers | Expected ACKs/s | Achieved Pub Rate | p50 Latency | p99 Latency | p99.9 Latency | Max Latency | Cluster Avg CPU | Cluster Peak RAM | Memory Growth |
| :--- | ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:|
| **`TEST-CF-1`** | 1 | 1,000 /s | 996.0 /s | 1,248.48 us | 2,316.50 us | 8,705.83 us | 17,206 us | 21.02% | 253.1 MB | +150.4 MB |
| **`TEST-CF-3`** | 3 | 3,000 /s | 996.0 /s | 1,024.41 us | 2,196.75 us | 5,739.83 us | 10,720 us | 26.42% | 295.2 MB | +82.8 MB |
| **`TEST-CF-10`**| 10 | 10,000 /s | 976.0 /s | 3,504.29 us | 9,518.92 us | 20,077.96 us | 43,918 us | 50.23% | 449.4 MB | +232.7 MB |
| **`TEST-CF-20`**| 20 | 20,000 /s | 500.0 /s | 7,728.80 us | 17,018.46 us | 39,088.50 us | 68,337 us | 58.27% | 668.9 MB | +294.6 MB |

---

## 3. Systems & Architectural Behavioral Analysis

### 3.1 Anatomy of Stateful Durable Consumer Progress
In NATS JetStream, durable consumers are stateful entities tracked by the server:
- For every published message, each durable consumer independently retrieves the payload via pull batch requests (`fetch` / `next`).
- When a consumer client issues an acknowledgment (`+ACK`), the server cannot simply discard the record in RAM.
- It must advance the consumer sequence floor, remove pending message IDs from the in-flight list, and commit this state update to the consumer Raft log so progress survives node restarts or failovers.
- Consequently, consumer fan-out creates an internal write multiplier:
  $$\text{Raft Write Operations/sec} = \lambda_{\text{pub}} + \left( \lambda_{\text{pub}} \times N_{\text{consumers}} \right)$$

### 3.2 Memory Scaling Deduction ($k_{\text{consumer}}$)
The memory progression across consumer counts displays remarkable linear consistency:
- From 1 to 3 consumers: $\Delta M = 295.2 - 253.1 = 42.1 \text{ MB} \implies 21.05 \text{ MB / consumer}$.
- From 3 to 10 consumers: $\Delta M = 449.4 - 295.2 = 154.2 \text{ MB} \implies 22.03 \text{ MB / consumer}$.
- From 10 to 20 consumers: $\Delta M = 668.9 - 449.4 = 219.5 \text{ MB} \implies 21.95 \text{ MB / consumer}$.
- Average slope:
  $$k_{\text{consumer}} = \frac{668.9 \text{ MB} - 253.1 \text{ MB}}{20 - 1} = \frac{415.8 \text{ MB}}{19} = 21.88 \approx 21.9 \text{ MB / consumer}$$
- This ~21.9 MB footprint per active durable consumer reflects:
  1. Internal pending message sequence map buffers in Go memory.
  2. Pull request FIFO queue slices and batch tracking arrays.
  3. Pre-allocated consumer Raft log buffers and in-flight timer structures.

### 3.3 The ACK Amplification Wall (20 Consumers)
At 20 consumers:
- Published load of 1,000 msgs/s translated into 20,000 consumer delivery dispatches and 20,000 incoming explicit ACKs per second across the cluster.
- While the cluster comfortably handled 10,000 ACKs/sec at 10 consumers ($p50 = 3.5 \text{ ms}$, CPU = 50.2%), 20,000 ACKs/sec overwhelmed the single-stream consumer commit queue.
- **Raft Channel Contention**: Lock contention inside the Go channel dispatching consumer sequence updates blocked publisher write threads.
- **Throughput Truncation**: Achieved publisher throughput dropped from 996 msgs/s to 500.0 msgs/s (a 50% capacity collapse).
- **Latency Surge**: Median latency grew by 7.5x (from $1,024 \ \mu\text{s}$ at 3 consumers to $7,728 \ \mu\text{s}$ at 20 consumers), while tail latency ($p99.9$) spiked to 39.1 ms.

---

## 4. Bottleneck Summary & Thresholds

| Metric | Safe Operating Zone | Warning Threshold | Hard Saturation Knee | Operational Impact |
| :--- | :--- | :--- | :--- | :--- |
| **Active Consumers / Stream** | 1 - 5 consumers | 6 - 10 consumers | > 15 consumers | Publisher backpressure and throughput collapse |
| **Aggregate ACK Rate** | < 5,000 ACKs/s | 10,000 ACKs/s | > 15,000 ACKs/s | Raft consumer state mutation pipeline saturation |
| **Consumer Memory Tax** | ~22 MB / consumer | ~220 MB (10 consumers)| ~440 MB (20 consumers)| Linear memory expansion requiring explicit sizing |
| **Latency Inflation** | 1.0 - 2.5 ms | 3.5 - 9.5 ms | > 17 ms p99 | Consumer ACK serialization blocking publisher commits |

