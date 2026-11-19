# Workload Estimation Guide: From Business TPS to Parameters

## 1. Estimation Context (Sample Scenario)
* **Application**: E-Commerce / Order Processing Platform
* **Business Scale**: 1,000 Average Business TPS (Orders/sec), 3,000 Peak TPS (3x diurnal surge)
* **Architecture**: 40 microservice pods across publishing and subscribing services, 24-hour retention

---

## 2. Parameter Estimation Cheat Sheet (1 Line per Parameter)

### 1. Core NATS (In-Memory Routing) [Moved to Top]
* **`x_core_avg`**: `Steady-state ephemeral ingress` -> `0 msgs/s` (or e.g. 2,500 msgs/s for live presence/metrics)
* **`k_peak_core`**: `Ephemeral burst multiplier` -> `2.0x` (derives `x_core_peak = x_core_avg * 2.0`)
* **`c_core`**: `Broadcast subscriber fan-out` -> `0 subs` (in-memory broadcast delivery multiplier)

### 2. JetStream Workload & Message Sizing
* **`x_js_avg`**: `Avg Business TPS * persistent events/tx` -> `1,000 TPS * 2 events = 2,000 msgs/s` (sizes cumulative disk volume)
* **`k_peak_js`**: `Traffic Peak Multiplier` -> `3.0x E-Commerce` (derives `x_js_peak = 2,000 * 3.0 = 6,000 msgs/s`)
* **`payload_avg`**: `Typical JSON payload size` -> `1,024 bytes` (standard microservice business record)
* **`k_payload_peak`**: `Burst payload multiplier` -> `1.5x` (derives `payload_peak = 1,024 * 1.5 = 1,536 bytes`)
* **`header`**: `Tracing and correlation metadata` -> `64 bytes` (W3C traceparent + span IDs)
* **`retention_hours`**: `Operational retention window` -> `72 hours` (or 24h for 1 day) before segment block deletion

### 3. Application Topology & Client Sockets
* **`n_pub_services`**: `Distinct publishing microservices` -> `5 services` (e.g. Order, Payment, Cart, User, Inventory)
* **`pub_replicas`**: `K8s pod replicas per publisher` -> `3 pods/svc` (15 publishing pods)
* **`n_sub_services`**: `Distinct consuming microservices` -> `10 services` (downstream consumer groups)
* **`sub_replicas`**: `K8s pod replicas per consumer` -> `3 pods/svc` (30 consuming pods)
* **`conn_per_pod`**: `NATS SDK multiplexed TCP connections` -> `1 socket/pod` (standard SDK connection multiplexing)
* **`n_conn_admin`**: `Ingress & Prometheus monitoring sockets` -> `5 sockets`
* **`n_conn_peak`**: `(15 pub + 30 sub) * 1 + 5 admin` -> `Derived: 50 client sockets` (~17 sockets/node via HAProxy)

### 4. Streams & Consumer Group Taxonomy
* **`n_stream`**: `Domain stream boundaries` -> `5 active streams` (Orders, Payments, Inventory, Audit, Invoices)
* **`c_js`**: `Consuming groups per stream` -> `2 groups/stream` (e.g. Payment + Inventory services reading Orders)
* **`n_consumer_peak`**: `n_stream * c_js` -> `Derived: 10 durable consumers` (~7.29 MB heap/node each)
* **`n_subj`**: `Taxonomy routing hierarchy leaves` -> `1,000 subjects` (`orders.<region>.<tenant>.<status>`, ~313 B heap/node each)

### 5. Model Constants & Standards
* **`batch_size`**: `Pull consumer batch fetch RPC size` -> `500 msgs` (standard SDK batch amortization)
* **`headroom`**: `Safety buffer for N-1 failover and Go GC` -> `30%` (targets <= 70% peak sustained utilization)
* **`mcpu_base`**: `Idle cluster route ping overhead` -> `50 mCPU` (fixed inter-node route floor, T_ROUTE = 2s)
* **`ram_base`**: `Startup Go runtime RSS allocator floor` -> `128 MB` (fixed runtime memory pools & spans)
* **`s_proto`**: `Wire and block record framing overhead` -> `32 bytes` (record header on wire & disk)

---

## 3. Resulting Sizing Deliverable (At a Glance)
Entering these 18 parameters into `docs/infra_calculator/infra_calc.html` yields:
* **Compute (CPU)**: **2 vCPU per node** (exact raw demand: 921 mCPU, provisioned with 30% headroom: 1,316 mCPU &rarr; 6 vCPU cluster total)
* **Memory (RAM)**: **~350 MB Go Process RSS per node** (paired with a standard 4 GB VM, leaves **>3.6 GB spare host RAM** for Linux Page Cache holding >22 mins of writes for Zero-Disk Read Replay)
* **Storage (Disk)**: **~270 GB persistent disk per node** (24h retention at 2,000 msgs/s avg &times; 1,120 B, provisioned with 30% headroom &rarr; ~810 GB cluster total)
