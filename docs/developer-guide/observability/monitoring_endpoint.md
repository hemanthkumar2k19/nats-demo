## 1. HTTP Monitoring Endpoints

HTTP Monitoring Endpoints are exposed on the HTTP monitoring port (default: `8222`).

| Endpoint | Path | Primary Focus |
|---|---|---|
| General Server Info | `/varz` | Server metadata, runtime resource usage, traffic totals, JetStream summary, and operational stats |
| JetStream Engine | `/jsz` | JetStream storage, memory limits, account asset counts, and Raft meta-cluster consensus |
| Client Connections | `/connz` | Active client connection details, traffic statistics, buffer states, and client metadata |
| Account Directory | `/accountz` | Configured accounts directory and system account association |
| Account Telemetry | `/accstatz` | Per-account message, byte, connection, and JetStream usage breakdown |
| Subscriptions Table | `/subsz` | Subscription routing table size, cache hit rate, insertion/removal counts, and fanout metrics |
| Cluster Routes | `/routez` | Inter-server cluster route topology, peer node connections, RTT latency, and route bandwidth |
| LeafNodes | `/leafz` | Remote LeafNode bridge topology, connection status, and traffic metrics |
| Gateways | `/gatewayz` | Multi-cluster gateway link state, inbound/outbound gateways, and cross-cluster traffic |
| Raft Groups | `/raftz` | Raft consensus group states, leader election, quorum, WAL indices, and peer liveness |
| Health Probe | `/healthz` | Lightweight health check status endpoint for load balancers and orchestrators |
| Go Runtime Vars | `/debug/vars` | Go expvar metrics including Go GC pauses, memory allocations, heap objects, and goroutine stats |

---

### 1.2 Endpoint Metric Tables

#### `/varz` - General Server Statistics Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Metadata | Server Identifiers & System Build | Unique server ID, human-readable name, version string, protocol version, Git commit SHA, Go runtime version, host IP, client/cluster/http ports, configuration load timestamp, tags, and active feature flags. |
| Configuration | Operational Limits & Timeouts | Max allowed client connections, max payload byte size, auth handshake timeout, ping interval, ping max missed count, max control line size, pending connection limits, and designated system account. |
| System Resources | CPU & Memory Consumption | Resident Set Size (RSS) memory usage in bytes, available CPU cores, GOMAXPROCS setting, current CPU load percentage, uptime duration, and server start timestamp. |
| Connections | Client & Connection Health | Total active client connections, cumulative historical connections, slow consumer count, stale connection count, and stalled clients count. |
| Traffic & Throughput | Network Bytes & Messages | Inbound/outbound message counts, inbound/outbound payload bytes, and client-specific message/byte counters. |
| Topology | Node Mesh & Cluster Counts | Cluster name, cluster bind address/port, remote route count, active cluster route links, connected leaf nodes count, and gateway connections count. |
| JetStream Summary | Storage & Engine Snapshot | Configured max memory/storage limits, store directory path, current RAM/disk bytes used, reserved RAM/disk bytes, total accounts with JetStream, HA assets count, and Raft meta-cluster leader/peer IDs. |
| Diagnostics | Internal Operational Stats | HTTP monitoring endpoint request counts (`/`, `/varz`, `/jsz`), slow consumer breakdown by entity type (clients, routes, gateways, leafs), stale connection breakdown by entity type, and disk I/O wait stats. |

---

#### `/jsz` - JetStream Statistics Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and metric generation timestamp. |
| Configuration | Global Storage & Limit Rules | Max memory allocation limit, max disk storage limit, storage directory path, file sync interval duration, and strict configuration flag. |
| Resource Usage | Memory & Storage Utilization | Active RAM bytes used by streams, active disk storage bytes used by streams, reserved memory bytes, and reserved storage bytes across all accounts. |
| Inventory | Asset & Data Volume Summary | Count of accounts with JetStream enabled, high-availability assets count, total streams count, total consumers count, total messages stored across streams, and total payload bytes stored. |
| Engine Telemetry | API Operations & Errors | JetStream API protocol level, total JetStream API requests processed, and total JetStream API error responses returned. |
| Consensus | Raft Meta-Cluster Topology | Cluster name, active Raft meta-cluster leader node ID, local peer node ID, total cluster node count, pending Raft log entries/requests/infos, and Raft snapshot state (pending entries, pending byte size, last snapshot timestamp). |

---

#### `/connz` - Connection Statistics Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and response timestamp. |
| Summary | Connection Counts & Query Bounds | Total active client connections count, matching connections count, pagination offset, and query limit. |
| Connection Detail | Client Identification & Network | Connection ID (`cid`), remote client IP address, client ephemeral port, connection start timestamp, uptime duration, last activity timestamp, and idle duration. |
| Connection Detail | Authentication & Account Scope | Client name/type, authenticated user identity, account scope name, TLS protocol version, and TLS cipher suite. |
| Connection Detail | Traffic & Buffer Telemetry | Pending output byte buffer size, inbound message count, outbound message count, inbound byte count, outbound byte count, and active subscription count. |
| Connection Detail | Health & Drop Counters | Slow consumer flag, dropped message count, and connection state indicators. |

---

#### `/accountz` - Account Provisioning Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and response timestamp. |
| System | System Account Identity | Designated system account name used for internal server telemetry and administration (`$SYS`). |
| Inventory | Registered Account Directory | Complete list of all configured account names present on the server (such as `$G` global default account, `$SYS` system account, and custom application accounts). |

---

#### `/accstatz` - Account Telemetry Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and response timestamp. |
| Account Traffic | Message & Byte Bandwidth | Per-account inbound/outbound message counters and payload byte totals across all client connections associated with the account. |
| Account Resources | Connections & Subscriptions | Active client connection count per account and total active subject subscription count per account. |
| Account JetStream | Storage & Asset Limits | Per-account JetStream memory storage bytes used, disk storage bytes used, stream count, consumer count, and JetStream API request/error rates. |
| Account Health | Anomalies & Bridge Telemetry | Per-account slow consumer count, leaf node bridge connection count, and account-level connection drop counters. |

---

#### `/subsz` - Subscription Routing Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and response timestamp. |
| Table State | Active Subscriptions & Cache | Total active subject subscriptions registered across all local clients and system listeners (`num_subscriptions`), and total routing cache entries (`num_cache`). |
| Engine Operations | Table Mutations & Matches | Total subscription insertions (`num_inserts`), total subscription removals (`num_removes`), and total subject match evaluations (`num_matches`). |
| Performance | Routing Cache & Fanout Efficiency | Routing cache hit rate percentage (`cache_hit_rate`), maximum subscription fanout count (`max_fanout`), and average subscription fanout count (`avg_fanout`). |
| Pagination | Query Limits & Bounds | Total result count, pagination offset, and query result limit. |

---

#### `/routez` - Cluster Routing Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server & Topology Identification | Local server ID, local server name, and response timestamp. |
| Summary | Cluster Topology Summary | Total count of active cluster route connections (`num_routes`). |
| Peer Detail | Remote Server Identification | Route connection ID (`rid`), remote server ID, remote server human-readable name, remote server IP address, and remote route port. |
| Route State | Connection Lifecycle & Health | Route solicitation direction (`did_solicit`, `is_configured`), connection start timestamp, uptime duration, last activity timestamp, round-trip time latency (`rtt`), and idle duration. |
| Route Traffic | Message & Byte Bandwidth | Pending write buffer byte size, inbound message count, outbound message count, inbound byte count, outbound byte count, active subscriptions count bridged over the route, account scope (`$SYS`), and wire payload compression status. |

---

#### `/leafz` - LeafNode Topology Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and response timestamp. |
| Summary | LeafNode Topology Summary | Total active LeafNode connection count (`leafnodes`). |
| Bridge Detail | Remote Leaf Identity & Link State | Remote server ID/name, remote IP address and port, connection initiation direction (solicited outgoing vs incoming), TLS handshake state, and account bindings. |
| Bridge Traffic | Message & Byte Bandwidth | Inbound message and byte counts, outbound message and byte counts, active subject subscriptions bridged across the LeafNode, round-trip time latency (`rtt`), and link idle duration. |

---

#### `/gatewayz` - Gateway Topology Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Context | Server Identification | Unique server ID and response timestamp. |
| Summary | Gateway Topology Summary | Map of inbound gateway connections and map of outbound gateway connections across multi-cluster superclusters. |
| Link Detail | Remote Cluster & Node State | Remote gateway cluster name, remote gateway server ID/name, link connection IP address and port, connection state, and round-trip time latency (`rtt`). |
| Link Traffic | Cross-Cluster Bandwidth | Inbound and outbound message counts, inbound and outbound byte totals, configured vs auto-discovered gateway state, and cross-cluster subscription interest bridging stats. |

---

#### `/raftz` - Raft Group Consensus Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Scope | Raft Group Identification | Account scope and group identifier (e.g., `$SYS` meta-cluster `_meta_`, or individual stream Raft group ID). |
| State | Consensus & Leadership | Current Raft node state (`LEADER`, `FOLLOWER`, `CANDIDATE`), group node size, quorum count required for commits, current election term number, voted-for candidate ID, and ever-had-leader boolean flag. |
| Log Index | Replication & Commit Indices | Committed Raft log index, applied log index, previous term number (`pterm`), previous log index (`pindex`), and pending queue lengths (proposals, entries, responses, applications). |
| WAL | Write-Ahead Log Storage | Stored WAL message count, stored WAL byte size, first log entry sequence number and timestamp, last log entry sequence number and timestamp, and active WAL consumer count. |
| Peer Mesh | Peer Liveness & Discovery | Map of peer group node IDs to server names, peer discovery status (`known`), and peer last-seen duration latency. |

---

#### `/healthz` - Health Probe Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Status | Health Response | JSON object indicating overall server operational state (`{"status": "ok"}`). Used by load balancers, Kubernetes probes, and monitoring agents. |
| Probes | Health Check Filters | Optional query parameters (`?js-enabled-only=true`, `?js-server-only=true`) to enforce deep validation of JetStream storage health, Raft meta-cluster quorum state, or write capability. |

---

#### `/debug/vars` - Go Runtime Statistics Endpoint

| Category | Metrics Group | Description |
|---|---|---|
| Execution | Command Line Arguments | Server launch execution command line array (`cmdline`). |
| Go Runtime | Garbage Collector Performance | GC CPU utilization fraction, total cumulative GC pause duration, recent GC pause durations array (`PauseNs`), GC pause completion timestamps (`PauseEnd`), total GC run count (`NumGC`), forced GC count, and GC enabled status. |
| Go Runtime | Memory Allocator Telemetry | Total memory allocated from OS, active heap bytes allocated, total heap bytes from OS, heap idle bytes, heap in-use bytes, heap released bytes to OS, heap object count, stack in-use/system bytes, MSpan/MCache bytes, and GC metadata bytes. |
| Go Runtime | Size-Class Byte Allocation | Detailed table of memory allocation counts and free counts broken down by Go runtime memory block byte sizes (`BySize`). |

---