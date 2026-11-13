# NATS Behavior Analysis: Core NATS Throughput Saturation Ladder

## 1. Executive Summary

The Core NATS Throughput Saturation Ladder (`core-ladder`, timestamp `20261112T073441Z`) evaluates in-memory message routing across ascending offered load rates from 1,000 msgs/s to 100,000 msgs/s with a 1:4 broadcast fan-out topology (4 publishers, 4 subscribers).

The cluster exhibited three distinct behavioral regimes:
1. **Unbatched Microsecond Routing (1,000 msgs/s)**: Low latency floor where messages are processed without kernel socket batching.
2. **Linear Scalability & Zero-Allocation Steady State (10,000 - 50,000 msgs/s)**: Sub-microsecond median routing ($p50 \approx 0.2 - 0.3 \ \mu\text{s}$), sub-3-microsecond $p99$, completely flat memory footprint (~44 MB cluster-wide), and 100% delivery efficiency.
3. **Saturation Knee & Socket Buffer Bloat (100,000 msgs/s offered)**: Cluster throughput saturated at 83,706 msgs/s. Subscribers fell behind, causing the NATS Go server to buffer pending unconsumed messages in memory (+354.3 MB growth to 396.1 MB peak), resulting in severe socket buffer bloat and tail latency explosion ($p99.9 = 22.0 \text{ ms}$, $\text{Max} = 1.705 \text{ seconds}$).

---

## 2. Empirical Benchmark Data

All tests executed with 1,024-byte payloads over a 30-second measurement window:

| Test ID | Offered Rate | Achieved Rate | Ingress MB/s | p50 Latency | p99 Latency | p99.9 Latency | Max Latency | Cluster Avg CPU | Cluster Peak RAM | Memory Growth | Delivery Eff. |
| :--- | ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:| ---:|
| **`TEST-CL-1K`** | 1,000 /s | 996.0 /s | 0.98 MB/s | 1.99 us | 10.04 us | 51.08 us | 1,223 us | 6.63% | 42.5 MB | +13.1 MB | 100.0% |
| **`TEST-CL-10K`** | 10,000 /s | 9,996.0 /s | 9.77 MB/s | 0.33 us | 3.08 us | 11.21 us | 67.5 us | 25.53% | 44.0 MB | +2.5 MB | 100.0% |
| **`TEST-CL-25K`** | 25,000 /s | 24,996.0 /s | 24.41 MB/s | 0.25 us | 2.25 us | 13.04 us | 1,089 us | 29.39% | 44.3 MB | +2.7 MB | 100.0% |
| **`TEST-CL-50K`** | 50,000 /s | 49,992.0 /s | 48.82 MB/s | 0.28 us | 2.67 us | 16.25 us | 5,599 us | 31.41% | 45.4 MB | +2.3 MB | 100.0% |
| **`TEST-CL-100K`**| 100,000 /s | 83,706.0 /s | 81.75 MB/s | 0.23 us | 9.00 us | 21,999 us | 1,705,053 us | 21.51% | 396.1 MB | +354.3 MB | 0.0% (Unconsumed)|

---

## 3. Systems & Architectural Behavioral Analysis

### 3.1 Kernel Socket Batching Dynamics ($1\text{k} \to 10\text{k}$ msgs/s)
At 1,000 msgs/s, median latency was $1.99 \ \mu\text{s}$. Counterintuitively, as offered load increased to 10,000 msgs/s, median latency dropped by an order of magnitude to $0.33 \ \mu\text{s}$, and at 25,000 msgs/s reached $0.25 \ \mu\text{s}$.

This inversion is governed by the Go runtime network poller (`epoll`/`kqueue`) and TCP socket write buffering:
- At 1,000 msgs/s, inter-arrival time between messages is ~1 millisecond. Each message arrives on an idle socket, requiring individual context switching, system call execution (`read`/`write`), and thread unparking.
- At 10,000 to 50,000 msgs/s, messages arrive in rapid succession. The NATS client and server read and write multiple messages per system call loop, amortizing per-packet syscall overhead and triggering optimal CPU L1/L2 cache locality.

### 3.2 In-Memory Zero-Allocation Steady State ($10\text{k} \to 50\text{k}$ msgs/s)
Between 10,000 and 50,000 msgs/s:
- Total cluster RAM remained flat at 44.0 MB to 45.4 MB across all 3 nodes (~15 MB per node).
- Cluster CPU scaled from 25.5% to 31.4%.
- 100% of published messages were delivered to all 4 subscribers ($6,000,000$ delivered messages during the 50k test).

This demonstrates that Core NATS operates without accumulating message buffers in RAM when subscribers keep pace with publishers. The Go garbage collector runs minimally because NATS reuses pre-allocated byte slices from internal pools for message parsing and dispatching.

### 3.3 The Saturation Knee & Socket Buffer Bloat (100,000 msgs/s)
When offered load was elevated to 100,000 msgs/s:
- **Throughput Saturation**: The 3-node cluster hit a hard processing barrier at 83,706 msgs/s (representing 81.75 MB/s ingress and ~327 MB/s aggregate broadcast egress across 4 subscribers).
- **Socket Buffer Bloat**: Because subscribers could not consume at the offered rate, outbound TCP buffers filled up. NATS server connection write queues began buffering outbound messages in the Go heap.
- **Memory Explosion**: Total cluster memory spiked from 45.4 MB to 396.1 MB (+354.3 MB net growth).
- **Latency Collapse**: While the median publisher latency remained low ($0.23 \ \mu\text{s}$) because writes initially entered local kernel socket buffers, tail latency ruptured completely:
  - $p99.9$ latency jumped from $16.25 \ \mu\text{s}$ at 50k to $21,999.46 \ \mu\text{s}$ (22 ms) at 100k.
  - Maximum latency spiked to $1,705,053 \ \mu\text{s}$ (1.705 seconds).
- **Subscriber Backpressure**: Subscribers reported 0% delivery efficiency at the end of the 30-second window because slow consumer write deadlines were exceeded, demonstrating the classic queue rupture phenomenon.

---

## 4. Bottleneck Summary & Thresholds

| Metric | Safe Operating Zone | Saturation Knee | Failure Mode |
| :--- | :--- | :--- | :--- |
| **Ingress Rate** | Up to 50,000 msgs/s | 83,700 msgs/s | Throughput truncation |
| **Cluster RAM** | 42 - 46 MB (Flat) | > 50 MB rapid rise | Outbound slow-consumer buffering |
| **p99 Latency** | < 3 us | > 20,000 us | TCP socket queueing and slow consumer drops |
| **Egress Bandwidth** | Up to 200 MB/s | ~330 MB/s | Host socket write loop saturation |

