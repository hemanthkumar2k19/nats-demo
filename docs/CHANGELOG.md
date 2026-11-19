# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased] - 2026-11-19

### Added
- **Topology-Driven Sizing & Traffic Peak Multiplier Engine (`docs/infra_calculator/infra_calc.html`)**:
  - Reordered input cards to place **Core NATS (In-Memory Ephemeral)** at the top, preceding **JetStream**, reflecting natural messaging taxonomy.
  - Upgraded Core NATS with Average Ingress Rate (`x_core_avg`) and Core Peak Multiplier (`k_peak_core`) with one-click presets (`[1.5x Normal]`, `[2.0x Standard]`, `[3.0x Burst]`) and manual float inputs, automatically deriving `x_core_peak`.
  - Upgraded JetStream with Traffic Peak Multipliers (`k_peak_js`) with presets (`[2.0x Banking]`, `[3.0x E-Commerce]`, `[5.0x Retail Peak]`, `[10.0x Flash Sale]`) and Peak Payload Multipliers (`k_payload_peak`), deriving `x_js_peak` and `payload_peak`.
  - Introduced the **Application Topology Card** taking publishing services, consuming services, and K8s pod replicas to mathematically derive total microservice pods and peak client sockets (`n_conn_peak`).
  - Derived peak durable consumers (`n_consumer_peak = n_stream * c_js`) directly from active streams and downstream consumer groups per stream.
  - Enhanced bidirectional YAML synchronization and updated documentation (`notes.md`, `example.md`).
- **Sizing Intuition Paradoxes and Architecture Defense Notes (`docs/infra_calculator/notes.md`)**:
  - Added Section 3 resolving counter-intuitive sizing ratios (1 vCPU, ~300 MB RAM, ~830 GB Disk) for architecture reviews and technical discussions.
  - Documented physical reasons why Go process memory footprint is low (~300 MB) without missing variables, contrasting user-space heap vs. direct append-only `.blk` disk persistence.
  - Articulated the distinction between process resident RSS and host VM RAM, showing how cloud VM packaging (1:2 and 1:4 ratios) naturally supplies 2 GB to 4 GB of RAM where spare memory is passively utilized by the Linux kernel as Page Cache for zero-disk consumer reads.
  - Detailed the physical mechanics behind high persistent disk capacity (time accumulator over 72 hours accumulating 518.4 million messages) and why the 30% Headroom buffer (829.4 GB provisioned) is mandatory to prevent ext4/xfs fragmentation, absorb rollover padding, and handle asynchronous retention cleanup.
  - Included a compact architecture review objection and rebuttal cheat sheet table.
- **Ultra-Compact Workload Estimation Cheat Sheet (`docs/infra_calculator/example.md`)**:
  - Authored a high-density, 46-line rough note cheat sheet mapping Business TPS to the 18 NATS capacity parameters with exactly 1 line per parameter.
  - Provided concise, practical estimation formulas and sample values for an e-commerce workload (1,000 Avg TPS, 3,000 Peak TPS, 40 pods, 24h retention).
  - Summarized the resulting sizing deliverables at a glance: 2 vCPU, ~350 MB Go process RAM (>3.6 GB Page Cache), and ~270 GB persistent disk per node.
- **Persistent Disk Model Integration into Calculator (`docs/infra_calculator/infra_calc.html`)**:
  - Expanded the calculator UI to size Compute (vCPU), Memory (RAM), and Storage (Persistent Disk) across all input, dashboard, breakdown, and YAML views.
  - Added workload controls for `x_js_avg` (average ingress rate), `payload_avg` (average payload size), `header` (message headers), `retention_hours` (retention time window `t`), and `s_proto` (block framing overhead `S_PROTO`).
  - Added **Hero Card 3 (Provisioned Disk)** displaying recommended provisioned disk volume per node (GB/TB), raw volume, 30% Headroom, and cluster total ($R=3$).
  - Implemented the **Per-Node Persistent Disk Breakdown Table** itemizing effective stored message size, sequential peak write bandwidth (Node & Cluster), raw volume, provisioned volume targeting $\le 70\%$ ext4/xfs utilization, storage controller specs ($\ge 3,000$ IOPS baseline), and zero live consumer read IOPS.
  - Updated both editable Input YAML and read-only Output Provisioning YAML specifications with full persistent disk metrics and bidirectional live sync.
- **Interactive Dual YAML Export & Live Two-Way Editing (`docs/infra_calculator/infra_calc.html`)**:
  - Implemented segmented view toggle controls across Inputs (`Form View` vs `YAML Config`) and Outputs (`Dashboard View` vs `Provisioning YAML`).
  - Added editable workload YAML configuration with real-time bidirectional synchronization: editing YAML values dynamically parses and updates form sliders, number controls, and derived provisioning models in real-time.
  - Added exportable read-only Provisioning YAML specification detailing computed vCPU, Memory (Go Process RSS), cloud VM tier pairing (1:2 and 1:4 ratios), component breakdowns, and operational guardrail statuses.
  - Implemented one-click clipboard copy buttons with visual "Copied!" feedback badges for both input and output YAML specifications.
- **Persistent Disk Storage Capacity Planning & Sizing Model (`docs/infra_calculator/disk.md`)**:
  - Authored the comprehensive Persistent Disk estimation model for the fixed 3-node JetStream cluster ($R=3$, File Store, HAProxy L4).
  - Audited all 16 workload input variables and 7 system constants with physical layer inclusion/exclusion justifications, demonstrating why consumer fan-out (`c_js`) is strictly excluded from storage volume (write-once semantics).
  - Deduced the mathematical storage volume formulation ($D_{\text{node}} = x_{\text{js\_avg}} \cdot S_{\text{avg}} \cdot (t \cdot 3,600)$) with 30% Headroom for filesystem safety ($D_{\text{provisioned}} = D_{\text{node}} / 0.70$) and cluster total ($3 \times D_{\text{node}}$).
  - Formalized the physical rationale for the 30% storage headroom buffer: preventing ext4/xfs fragmentation, absorbing 8 MB/16 MB segment roll-over padding, covering Linux 5% root block reservation, and buffering retention deletion batch delay.
  - Sized sustained peak write bandwidth ($\text{BW}_{\text{write, node}} = x_{\text{js\_peak}} \cdot S_{\text{peak}}$), cluster replicated write throughput, and physical write IOPS requirements ($\ge 3,000$ IOPS baseline), while explaining fsync group commit amortization under load.
  - Reaffirmed the Zero-Disk Read Replay Principle: live streaming consumers read directly from memory pages held in spare host RAM (Linux OS Page Cache), generating zero physical disk read IOPS.
- **Operational Boundaries & Physical Sizing Guardrails (`docs/infra_calculator/guardrails.md`)**:
  - Authored a centralized reference cataloging all 9 physical saturation knees, hardware ceilings, and architectural limitations across Compute, Consensus, Memory, Storage, Network, and Multi-Raft density.
  - Formatted each guardrail with its physical mechanism, empirical benchmark observation, behavioral symptom, and mitigation sizing rule.
  - Decoupled guardrails from `docs/infra_calculator/cpu.md` and `docs/infra_calculator/memory.md`, establishing clean modular separation of concerns.
- **Mathematical & Systems Engineering Principles Reference (`docs/infra_calculator/method.md`)**:
  - Consolidated all 8 foundational mathematical and systems engineering principles across CPU, Memory, Disk, and Network into a dedicated, centralized reference document.
  - Included formal definitions, mathematical proofs, and a dedicated `Applied In:` application scope section for each principle: OFAT Controlled Isolation, Superposition Principle, Conservative Upper Secant Envelope, Differential Slope Extraction, Two-Tier Memory Architecture, Mechanical Sympathy, USE Method, and M/M/1 Queueing Headroom.
  - Streamlined Section 3 in `docs/infra_calculator/cpu.md` and `docs/infra_calculator/memory.md` to reference this central document, eliminating documentation duplication.
- **Memory (RAM) Capacity Planning and Sizing Model (`docs/infra_calculator/memory.md`)**:
  - Authored a comprehensive memory sizing model establishing the two-tier physical memory architecture: Go Process Heap (RSS) and Linux OS Page Cache ($M_{\text{cache}}$), explaining why clubbing them into a unified Memory tier prevents real-time consumer disk read IOPS thrashing.
  - Audited all 16 workload input variables and 8 system constants with exhaustive inclusion/exclusion rationales and physical layer justifications.
  - Deduced the Go Process Heap formulation from empirical benchmark coefficients: base floor ($128\text{ MB}$), sublist prefix trie ($\beta_{\text{subj}} = 313\text{ B/subj/node}$), Multi-Raft streams ($\beta_{\text{stream}} = 234\text{ KiB/stream/node}$), durable consumers ($\beta_{\text{consumer}} = 7.29\text{ MB/consumer/node}$), and TCP socket buffers ($\beta_{\text{conn}} = 64\text{ KiB/conn}$).
  - Established the Linux OS Page Cache working set formula ($M_{\text{cache}} = x_{\text{js\_avg}} \times S_{\text{avg}} \times W$) for the 2-minute temporal window ($W = 120\text{ s}$), proving why consumer fan-out does not duplicate kernel page cache buffers.
  - Established 4 operational boundaries & non-linear guardrails: cold read disk thrashing (lag $>120\text{s}$), large payload socket buffer amplification ($>8\text{ KiB}$), in-memory stream traps, and ephemeral UUID subject memory leaks.
- **Dedicated CPU Capacity Sizing Calculator UI (`docs/infra_calculator/infra_calc.html`)**:
  - Built a clean, self-contained interactive calculator UI dedicated strictly to empirical CPU sizing on the fixed 3-node cluster ($R=3$, HAProxy L4).
  - Displays the Recommended Rounded CPU (whole vCPU) per node and cluster total, with exact provisioned mCPU, raw demand, and 70% target utilization metrics.
  - Implemented real-time dual controls (range sliders + direct numeric inputs with explicit unit labels) across JetStream workload (`x_js_peak`, `c_js`), Core NATS (`x_core_peak`, `c_core`), sockets (`n_conn_peak`), and streams (`n_stream`), with an uncollapsed standard card for model constants (`BATCH`, `HEADROOM`, `mCPU_base`).
  - Added an empirical superposition breakdown table and dynamic non-linear guardrail badges (7,000 msgs/s disk fsync ceiling and 10,000 ACKs/s consumer wall).
- **Capacity Planning Notes & Architectural Rationale (`docs/infra_calculator/notes.md`)**:
  - Authored a compact, high-density technical note establishing why Subject Cardinality does not impact routing latency or CPU due to NATS prefix trie $O(K)$ algorithmic invariance and ~313 B/node heap cost.
  - Documented why JetStream physical sizing is governed almost exclusively by data velocity ($\text{Throughput} \times \text{Payload Size}$) across storage volume, disk IOPS, network bandwidth, and Page Cache working set, noting the only two exceptions (`MaxMsgsPerSubject` KV buckets and unbounded ephemeral UUIDs).
- **Capacity Sizing Framework Executive Overview (`docs/infra_calculator/overview.md`)**:
  - Authored an architect-grade systems engineering overview framed around solving capacity planning in the absence of target production hardware through local baseline profiling and mathematical extrapolation.
  - Defined the Systems Engineering Principle separating Software-Intrinsic Mechanics (in-memory routing, trie lookups, Raft consensus) from Physical Hardware Ceilings (cloud VM, Persistent Disk IOPS, 10G NIC).
  - Established the 5 proofs of estimation certainty without target infrastructure: deterministic algorithmic complexity (O(1)/O(K)), physical memory model (Go heap vs Linux OS Page Cache for zero-disk replay), normalized hardware units (core-cycles and bytes), kernel mechanical sympathy (epoll batching and fsync group commits), and 30% headroom absorbing cloud virtualization variance.
  - Formulated the hardware sizing deliverables summary around the three true cloud provisioning knobs (Compute, unified Memory, Persistent Disk Storage) using plain-English systems descriptions, standardized storage terminology to cloud-agnostic Persistent Disk, un-clubbed running prose into clean bulleted structures, and kept the document strictly self-contained by removing external cross-document references.

### Changed
- **Interactive Calculator Compute & Memory Integration (`docs/infra_calculator/infra_calc.html`)**:
  - Expanded the standalone calculator UI to size both Compute (CPU) and Memory (RAM) simultaneously.
  - Added workload input controls for `n_consumer_peak` (durable consumers), `n_subj` (registered subjects), `payload_peak` (message payload in bytes), and `ram_base` (baseline Go RSS floor in MB).
  - Implemented dual hero recommendation cards for Provisioned Compute (vCPU per node & cluster total) and Provisioned Memory (MB/GB Go Process RSS per node & cluster total).
  - Added the empirical Go Process Resident Memory breakdown table itemizing baseline floor (128 MB), sublist tries (313 B/node), Multi-Raft streams (234 KiB/node), durable consumer progress (7.29 MB/node), and TCP socket buffers (64 KiB/node).
  - Built a Cloud VM Packaging and Linux Page Cache analysis card, evaluating standard 1:2 (Compute-Optimized) and 1:4 (General-Purpose) cloud VM tiers and displaying the real-time zero-disk replay cache duration based on spare host RAM and incoming write velocity.
  - Added non-linear guardrail badges for Large Payload Socket Buffer Amplification (>8 KiB) and Page Cache Cushion.
- **Elimination of Artificial Page Cache Window ($W = 120\text{ s}$) Across Framework**:
  - Completely purged the non-configurable Page Cache Window parameter ($W = 120\text{ s}$) and synthetic formula ($M_{\text{cache}} = x \times S \times W$) from all framework documentation (`operations.md`, `cpu.md`, `memory.md`, `method.md`, `guardrails.md`, `overview.md`, `notes.md`) and UI (`infra_calc.html`).
  - Reframed Memory sizing strictly as deterministic Go Process Resident Memory ($\text{RAM}_{\text{process}}$) covering runtime structures, sublist tries, stream state, consumer cursors, and socket buffers, sized with 30% Headroom for Go GC: $\text{RAM}_{\text{provisioned}} = \text{RAM}_{\text{process}} / 0.70$.
  - Grounded Linux Page Cache as an automatic, passive operational benefit of standard cloud VM compute profiles (1:2 and 1:4 vCPU:RAM packaging ratios), where spare host RAM (3.5 GB to 7.5 GB on a 2 vCPU instance) naturally caches 45+ minutes of recent ingress data with zero disk read IOPS.
  - Reduced the true System Constants count from 8 to 7 across the framework.
  - Re-anchored Guardrail 4 (Cold Read Disk Thrashing) threshold to backlog volume exceeding available spare host RAM rather than an arbitrary 120-second time boundary.
- **CPU Sizing Model & Methodology Refinements (`docs/infra_calculator/cpu.md`)**:
  - Added Section 3.5 (Differential Slope Method), formally documenting how taking rate deltas between benchmark steps mathematically eliminates containerization and hypervisor overhead ($C_{\text{overhead}}$) to isolate software-intrinsic NATS Go runtime execution costs.
  - Clarified Section 2.C to document that the empirical $\alpha_{\text{ack}}$ coefficient ($0.0330\text{ mCPU}$) encapsulates the entire end-to-end consumer lifecycle (outbound TCP push delivery plus inbound Raft explicit ACK state commit).
  - Standardized storage terminology to cloud-agnostic Persistent Disk across Section 1.1 and Section 4 Guardrail 2.
- **Operational Baseline & Sizing Invariants Streamlining (`docs/infra_calculator/operations.md`)**:
  - Streamlined document into a high-density, compact reference format (~85 lines vs 140 lines) without loss of technical signals or parameters.
  - Removed default values from Section 3.1 (Workload Inputs), reflecting their nature as purely dynamic user-provided variables rather than system defaults.
  - Added explicit `Configurability` classification across Section 2.2 (Heartbeats & Keepalives) and Section 3.2 (System Constants), distinguishing parameters fixed by NATS Code or Model Standards from those tunable via Server, Client SDK, or Consumer configs.
  - Standardized storage terminology from NVMe SSD to cloud-agnostic Persistent Disk across all operational descriptions.
  - Paired Average and Peak dimensions in Workload Inputs into consolidated, readable rows.
  - Structured Section 4 (Derived System Invariants) and Section 5 (Ignored Operations) into direct, authoritative reference tables with exact mathematical formulas and exclusion rationales.

## [Unreleased] - 2026-11-16

### Added
- **Operation-First Capacity Evaluation Model (`docs/operations.md`)**:
  - Established a dedicated operational inventory model for NATS JetStream capacity evaluation, decoupling message operations, cluster keepalives, and notation into clean reference tables.
  - Standardized Section 2.2 Heartbeats and Keepalives using uppercase symbolic notation (`T_RAFT`, `T_ROUTE`, `T_PING`, `T_HB`), decoupling mechanism flows from tuning constants.
  - Refined Section 2.1 Core NATS Write flow to isolate memory ingress from read delivery.
  - Adopted lowercase `n_` prefix convention for resource cardinality symbols (`n_conn`, `n_subj`, `n_stream`, `n_consumer`).
  - Standardized uppercase naming convention for all system constants (`BATCH`, `W`, `HEADROOM`, `S_PROTO`, `T_RAFT`, `T_ROUTE`, `T_PING`, `T_HB`) to visually separate fixed defaults from workload variables.
  - Relocated Monitoring/Telemetry and Subscription Interest Gossip into dedicated Section 5 (Ignored Operations), documenting four explicit reasons why they are excluded from core capacity formulas (negligible CPU, zero disk, steady-state silence, headroom absorption).
  - Eliminated the redundant Notation section, directly defining all symbols in Section 3 (Parameters: User Inputs & System Constants) and Section 4 (System Invariants).
  - Formatted Section 3 (Parameters) with Average and Peak dimensions across dynamic workload inputs (`n_conn_avg`/`n_conn_peak`, `payload_avg`/`payload_peak`, `x_core_avg`/`x_core_peak`, `x_js_avg`/`x_js_peak`, `n_consumer_avg`/`n_consumer_peak`), aligning baseline sizing for Kubernetes requests and storage volume with peak ceilings for limits and network bandwidth.
  - Consolidated Section 4 (System Invariants) establishing deterministic mathematical relations: Read Throughput ($y = x \times c$), Batch Request Rate ($y / \text{BATCH}$), Average & Peak Effective Message Sizes ($S_{\text{avg}}$, $S_{\text{peak}}$), $R=3$ Replication, 70% Target Utilization ($1 - \text{HEADROOM}$), and Page Cache working set RAM.
- **Mathematical Sizing Derivations (`docs/infra_calculator/cpu.md`)**:
  - Authored CPU Capacity Planning model structured into: Section 1 Impact of Variables & Constants (exhaustive audit across all 16 workload variables and 8 system constants explaining CPU inclusion/exclusion), Section 2 Deduction of Mathematical Model (empirical slopes from benchmark suites and complete superposition formula), and Section 3 Engineering Methodologies Used (OFAT, Superposition, USE Method, M/M/1 Queueing Headroom).
  - Cross-validated empirical slopes against all 7 canonical benchmark suites (`core-ladder`, `payload-scaling`, `jetstream-ladder`, `jetstream-guarantees`, `subject-cardinality`, `stream-scaling`, `consumer-fanout`), aligning observed data points with exact measured container telemetry. Streamlined Section 2 and Section 4 to summarize the empirical observations and their direct impact on modeling cleanly, removing noisy file path references.
  - Added Section 4 Operational Boundaries & Non-Linear Guardrails, establishing explicit physical operating ceilings where linear superposition ends: single-stream ACK amplification wall (<= 10k ACKs/s), single-stream disk fsync write ceiling (<= 7k msgs/s), 1 Gbps physical NIC bandwidth wall, and stream leader vs follower CPU asymmetry.

## [Unreleased] - 2026-11-13

### Fixed
- **Calculator DOM Reactivity (`capacity-calculator.html` & `static/report.html`)**:
  - Resolved a fatal JavaScript `SyntaxError: Identifier 'bAck' has already been declared` in `calculateInfraCapacity()`, which prevented the `<script>` block from executing and caused slider and parameter changes to silently fail to update right-side UI elements.
  - Disambiguated the disk IOPS ACK batching divisor (`batchAck`) from the consumer ACK consensus guardrail badge element (`bAck`), restoring immediate, reactive updates across all 5 resource metric cards, hero deployment cards, and guardrails.
  - Aligned initial static fallback HTML values with the exact mathematical output for the default 2,500 msgs/s profile.

### Added
- **Dual Slider + Direct Numeric Input Controls (`capacity-calculator.html` & `static/report.html`)**:
  - Implemented bidirectional numeric `<input type="number">` controls paired alongside every parameter slider (Ingress Rate, Payload Size, Subject Cardinality, Stream Count, Consumer Count, Retention Hours, Headroom Buffer).
  - Eliminates precision difficulties when tuning large numerical spans (e.g. 50k subjects, 64 KiB payloads, 168h retention), enabling users to either drag sliders for broad sweeps or type exact numerical values directly with immediate real-time synchronization.

- **Interactive Infrastructure Capacity Calculator (`static/report.html` & `capacity-calculator.html`)**:
  - Embedded an interactive client-side Infrastructure Capacity Calculator as a dedicated 4th tab (`Infra Capacity Calculator`) in `nats-capacity-benchmark/static/report.html`.
  - Built a standalone single-file HTML calculator `capacity-calculator.html` in the repository root modeled after `temporal-index.html`, requiring zero backend dependencies and running 100% in-browser with sub-millisecond reactive calculations.
  - Implemented the complete empirical mathematical sizing model from `docs/model.md` across both tools:
    - **Memory Sizing**: Baseline cluster/node footprint ($45 \text{ MB}$ Core, $240 \text{ MB}$ JetStream), subject prefix trie scaling (~940 B/subject), multi-Raft stream replica state (~720 KiB/stream), and durable consumer progress tracking (~21.88 MB/consumer).
    - **CPU Utilization**: Base cluster overhead (5%), Core NATS routing coefficient ($1.01 \times 10^{-4}\%$ per msg/s), JetStream File R=3 publish tax ($3.17 \times 10^{-3}\%$ per msg/s), subscriber egress fan-out, consumer ACK consensus tax ($3.25 \times 10^{-3}\%$ per ACK/s), and idle stream heartbeat ticks ($0.0165\%$ per stream).
    - **Latency Projection**: Group commit fsync amortization modeling $T_{p50}(\lambda) = 600 \ \mu\text{s} + \frac{3,800 \ \mu\text{s}}{1 + 0.0008 \cdot \lambda}$ for JetStream File R=3 workloads.
    - **Storage & Bandwidth**: Persistent disk write throughput and capacity projection across retention windows (hours/days), alongside ingress, egress, and inter-node cluster replication bandwidth.
    - **Fixed 3-Node JetStream File R=3 Deployment Architecture**: Fixed the calculator exclusively to the enterprise production standard: a 3-Node High-Availability Cluster with File Storage and Replication Factor R=3 (Raft quorum), eliminating distracting variable dropdowns (1-node standalone, Core NATS routing patterns, in-memory stream modes).
    - **Streamlined 3 Canonical FinOps Packaging Ratios (GCP GKE Reference Primary)**: Restricted instance selection to the three universal cloud packaging ratios: General Purpose (1:4 - GKE `n2-standard` / AWS `m6i` / Azure `Dsv5`), Compute Optimized (1:2 - GKE `c3-highcpu` / AWS `c6i` / Azure `Fsv2`), and Memory Optimized (1:8 - GKE `n2-highmem` / AWS `r6i` / Azure `Esv5`), with GCP GKE established as the primary reference baseline.
    - **Memory Breakdown (Process Heap vs. Linux OS Page Cache)**: Metric cards and sizing specs explicitly distinguish NATS Go process heap (~240-600 MB) from remaining node RAM utilized as Linux OS Page Cache for zero-disk message replays.
    - **Message Retention Window**: Interactive retention slider (24h to 168h / 7 days, 72h enterprise standard) governing persistent disk storage volume ($D_{\text{total}} = \lambda \cdot S_{\text{eff}} \cdot R \cdot T_{\text{retention}}$), complete with a dedicated metric card displaying total cluster and per-node retained storage.
    - **Stream Partitioning Advisory**: When offered ingestion rate exceeds single-stream disk fsync saturation (~7,234 msgs/s on File R=3), advises sharding traffic across $\ge \lceil \lambda / 7000 \rceil$ partitioned streams (`ORDERS.partition.*`) across the 3 nodes rather than inflating physical node counts.
    - **Workload Presets & Exports**: 6 pre-configured enterprise JetStream File R=3 sizing profiles (Microservices Event Log, Order Processing Pipeline, High Throughput Partitioned, Consumer Fan-Out Hub, IoT Large Payloads, Ultra-Scale Ingestion), comprehensive JSON export, CSV export, and Markdown sizing summary copy.
  - Preserved the template sentinel pattern (`window.BENCHMARK_DATA = null;`) in `static/report.html` to maintain 100% backward compatibility with `aggregate.py` and `aggregate_suite.py`.

### Changed
- **Unified Mathematical Model Consolidation (`docs/model.md`)**:
  - Reorganized `docs/model.md` into a single, cohesive, end-to-end mathematical capacity sizing framework:
    - Merged the 5 Methodological Pillars of Sufficiency (DOE Decadal Spans, Queueing Regimes, USL, Deterministic Invariants, Quantization) with the empirical parametric deductions.
    - Integrated the master empirical reference dataset across all 7 canonical suites (`20261112` runs).
    - Structured sequential mathematical derivations for Memory ($M_{\text{req}}$), CPU ($C_{\text{req}}$), Latency/USL ($\sigma, \kappa$), Disk I/O, and Network Bandwidth.
    - Added the discrete cloud infrastructure quantization step function with the 30% operational headroom buffer, followed by two complete worked sizing examples (Core NATS broadcast and Enterprise JetStream clustered ingestion).
    - Verified 100% strict ASCII compliance, proper KaTeX math rendering, and zero local file markdown links.

## [Unreleased] - 2026-11-12

### Added
- **Dedicated NATS Behavioral Analysis Reports**:
  - Authored comprehensive behavioral analysis reports across all 7 canonical suites under `reports/`:
    - `reports/core-ladder-analysis.md`: Analysis of in-memory socket batching ($p50 = 0.2 - 0.3 \ \mu\text{s}$), 83.7k msgs/s ceiling, and socket buffer bloat (+354 MB RAM) at 100k msgs/s.
    - `reports/payload-scaling-analysis.md`: Analysis of sub-microsecond routing up to 8 KiB and 1 Gbps physical network card saturation wall at 32 KiB (103.6 MB/s, $p99 = 25.1 \text{ ms}$).
    - `reports/subject-cardinality-analysis.md`: Verification of $O(K)$ sublist trie routing latency invariance ($0.32 \ \mu\text{s}$) across 1 to 10k subjects, wildcard parity, and ~940 B/subject memory slope.
    - `reports/jetstream-ladder-analysis.md`: Systems explanation of hyperbolic fsync group commit amortization ($4.4 \text{ ms} \to 0.97 \text{ ms}$) and physical disk saturation knee at 7,234 msgs/s.
    - `reports/jetstream-guarantees-analysis.md`: Controlled step-by-step isolation of the 700x distributed consensus latency tax (Core vs Mem R=3 vs File R=1 vs File R=3).
    - `reports/stream-scaling-analysis.md`: Demonstration of multi-Raft control-plane isolation (flat $1.9 \text{ ms}$ latency), negligible idle heartbeat tax (+0.81% CPU across 49 streams), and 720 KiB/stream memory footprint.
    - `reports/consumer-fanout-analysis.md`: Analysis of linear consumer memory scaling (~21.9 MB/consumer) and the ACK amplification consensus saturation wall at 20 consumers.
- **Mathematical Model Deduction Framework**:
  - Authored complete mathematical derivations connecting empirical data points to unified capacity sizing formulas for Cluster Memory, CPU, Latency/USL, Disk I/O, and Cloud Infrastructure Quantization with 30% Headroom Buffer.
- **README Technical Accuracy Alignment**:
  - **HAProxy Stats Endpoint**: Corrected Section 3 HAProxy stats command to open `http://127.0.0.1:8404/stats` (matching `stats uri /stats` in `deploy/haproxy.cfg`), resolving an HTTP 404 error.
  - **Exposed 5 Suite Parameters**: Updated Section 4 configuration guide to include `PUB_SUBJECT`, `SUB_SUBJECT`, `MULTISUBJECT_RANDOM`, `STREAM_COUNT`, and `CONSUMER_COUNT` from `benchmark.conf`.
  - **Scenario Flag Correction**: Corrected Section 6 table to reference `CONSUMER_COUNT>0` (aligning with `workload.sh` variable naming).
- **Documentation Deduplication, Scoping & Link Hygiene**:
  - **Single-Responsibility Realignment**: Streamlined `README.md` to focus strictly on Developer Quickstart & Configuration: condensed Section 1 (Topology) to a 2-line summary pointing to `docs/benchmarking.md#3`, converted Section 6 (Workload Scenarios) into a compact 4-row reference table pointing to `docs/benchmarking.md#2`, and condensed Section 7 (Artifacts) to a 4-item primary deliverables table pointing to `docs/benchmarking.md#6`.
  - **Redundant Table Elimination in `tests.md`**: Replaced the duplicate 8-row infrastructure environment table in `docs/tests.md` Section 1.2 with an authoritative relative link to `docs/benchmarking.md#3`.
  - **Link Hygiene & Renaming**: Updated cross-links in `README.md` and `docs/notes.md` to reference `model.md` (retiring stale `sufficiency.md` pointers).
  - **Zero Local File References Guarantee**: Verified complete elimination of all local file markdown links, absolute machine paths, and local machine URIs across all repository markdown files, adopting clean code spans and plain text references (e.g. `docs/benchmarking.md`).
- **Documentation Neutrality, Portability & Sizing Alignment**:
  - **Mathematical Modeling Sufficiency Documentation**: Authored `docs/sufficiency.md`, documenting the formal scientific, queueing theory, and statistical Design of Experiments (DOE) rationale explaining why 4 to 6 benchmark points per suite across logarithmic decadal spans are mathematically sufficient for capacity modeling. Neutralized all review rationale to standard engineering summary terminology.
  - **README Scope & Link Alignment**: Updated `README.md` to catalog all **7 Canonical Predefined Suites** (adding `subject-cardinality`, `stream-scaling`, `consumer-fanout`), aligned scenario descriptions with the End-to-End Pub/Sub operational model, eliminated broken/outdated documentation links, verified zero machine-specific local URIs, and updated cross-references to the documentation files (`benchmarking.md`, `tests.md`, `guide.md`, `notes.md`, `sufficiency.md`) and `reports/`.
  - **End-to-End Pub/Sub Model Realignment**: Refactored Section 2 of `docs/benchmarking.md` to concentrate on the active `pub-sub` and `js-pub-sync` operational models executed across all 7 canonical suites, eliminating misleading legacy references to standalone setup-only scenarios.
  - **Platform & Cloud Neutrality**: Removed all vendor-specific "GKE" branding across `docs/benchmarking.md`, `docs/guide.md`, and `docs/notes.md`. Adopted standard systems engineering tiers (**Baseline**, **Loading**, and **Scaling**) for the implementation roadmap.
  - **Portable Relative Links**: Eliminated absolute machine paths from repository markdown files in favor of portable relative links (`tests.md`).
  - **Premature Formula Removal**: Cleaned premature mathematical capacity equations from `docs/guide.md` (reserving dedicated formula finalization for the final estimation utility phase) while retaining all empirical constants, physical dynamics, and bottleneck thresholds.
- **Consolidated Documentation Architecture & 7-Suite Test Inventory Unification**:
  - **Clean 4-Document Decoupling**: Reorganized repository documentation into a clean four-tier hierarchy:
    1. `docs/benchmarking.md`: Overview to others covering executive value proposition, Core NATS / JetStream scenarios, 3-node HAProxy cluster deployment topology, time-bound execution lifecycle, and results artifact bundle.
    2. `docs/tests.md`: Single consolidated test catalog clubbing Core NATS and JetStream test matrices into a unified inventory centered on the **7 Canonical Executable Suites** (`core-ladder`, `payload-scaling`, `subject-cardinality`, `jetstream-ladder`, `jetstream-guarantees`, `stream-scaling`, `consumer-fanout`), retiring separate JetStream draft docs.
    3. `docs/guide.md`: Technical math reference covering socket/cgroup dynamics, calculation derivations, and certified empirical sizing constants from Reports 1-7 (550 B/subject, 312 KiB/stream, 19 MB/consumer, ACK saturation knee).
    4. `docs/notes.md`: R&D technical reference covering architectural & technical FAQs, OFAT scientific justification, formal standards citations (USE, ISO 25010/25023, USL, OMB), empirical capacity coefficients, and the Two-Tier calibration model for cluster extrapolation.
- **Consumer Fan-Out & ACK Amplification Benchmark Analysis (Report 7)**: Documented in `reports/report7.md` the comprehensive technical analysis of the `consumer-fanout` benchmark suite run `20261112T052934Z`. Evaluated Persistent Disk write multiplication, network egress amplification, and ACK consensus overhead across 1, 3, 10, and 20 concurrent durable consumers on a JetStream File R=3 stream. Discovered and quantified the ACK saturation knee (throughput throttled by 50% to 494 msgs/s at 20 consumers due to 20,000 ACKs/sec IOPS barrier), tail latency inflation (P50 1.31 ms -> 7.77 ms, P99 2.24 ms -> 17.70 ms), disk write amplification (+65.4% physical bytes written for consumer progress journals), memory growth (~19.0 MB cluster RAM per active durable consumer), formulated container Pod and Persistent Disk IOPS sizing standards (standard SSD vs provisioned high-IOPS SSD), and documented architectural guidance for high-fanout topologies.
- **Stream Density & Multi-Raft Scaling Benchmark Analysis (Report 6)**: Documented in `reports/report6.md` the comprehensive technical analysis of the `stream-scaling` benchmark suite run `20261112T051820Z`. Evaluated multi-Raft control-plane tax, heartbeat CPU overhead, and baseline RAM growth across 1, 5, 20, and 50 concurrent Raft streams (File R=3) under constant offered load (1,000 msgs/s, 1 KiB payloads). Validated near-zero CPU tax (+3.32% cluster CPU for 50 streams, ~0.066% per stream), flat latency envelope beyond initial multi-Raft step (1.52 ms -> 1.75-1.99 ms P50, 3.46-3.63 ms P99), quantifiable memory expansion (~312 KiB RAM per stream replica per node, or 0.94 MB cluster-wide), verified the strict storage reservation fix (`MAX_BYTES=100MB`), and formulated container Pod capacity sizing equations.

## [Unreleased] - 2026-11-11

### Added
- **Stream Scaling Sizing Fix & Preflight Storage Admission Validation**:
  - **Density Stream Sizing**: Configured `MAX_BYTES=100MB` across all test cases in `suites/stream-scaling.conf` ($50 \times 100\text{ MB} = 5.0\text{ GB}$, fitting within the $10.0\text{ GB}$ node limit) and `MAX_BYTES=1GB` in `suites/consumer-fanout.conf`, resolving NATS error `10005` (`insufficient storage`).
  - **Storage & Memory Admission Checks**: Added `check_storage_admission` to `scripts/js_stream.sh` (`admission` action) and `scripts/preflight.sh` to validate that total declared storage ($\text{STREAM\_COUNT} \times \text{MAX\_BYTES}$) does not exceed cluster `max_file_store` or `max_mem_store` limits, failing fast with actionable sizing advice prior to workload execution.
- **End-to-End Error Context Capture & Reporting**:
  - **Pipeline Stderr & Phase Logging**: Enhanced `benchmark.sh` to capture setup and workload console streams into `setup.log` and `workload.log`, extracting the most descriptive failure line into `error.log` with `error_phase` and `error_message`.
  - **Early-Exit Aggregation**: Upgraded `benchmark.sh` to invoke `aggregate.py` on setup failure so `results.json` and single-run `report.html` are always generated rather than leaving an empty output folder.
  - **Report UI Error Callouts**: Upgraded `static/report.html`, `scripts/aggregate.py`, and `scripts/aggregate_suite.py` to preserve and display error context in Tab 2 (Suite Matrix) with a prominent red `[FAILED: <phase>]` monospace banner, and in Tab 1 / drilldown drawers with an `#error-alert-card`.
- **Subject Cardinality & Taxonomy Scaling Benchmark Analysis (Report 5)**: Documented in `reports/report5.md` the comprehensive technical analysis of the `subject-cardinality` benchmark suite run `20261111T105527Z`. Evaluated Core NATS in-memory sublist prefix trie routing latencies (confirming constant-time $O(K)$ lookup across 1 to 10,000 subjects with 0.31 to 0.35 us P50), measured exact heap memory expansion (~550 bytes per subject per node, or 1.50 KiB cluster-wide), validated wildcard routing efficiency parity (0.33 us P50, 4.33 us P99), verified run-over-run reproducibility against `20261111T100555Z`, and established container Pod RAM and CPU sizing standards.
- **Exposed Test Parameters, Differentiating Factors, and Full Metadata Visibility**:
  - **Full Metadata Preservation**: Upgraded `scripts/aggregate.py` and `scripts/aggregate_suite.py` to preserve all 20+ configuration parameters from `test-config.txt` (including `subject`, `pub_subject`, `sub_subject`, `multisubject`, `multisubject_max`, `stream_count`, `consumer_count`, `storage_type`, `replication_factor`, and `queue_name`) into `results.json["metadata"]` and `suite-results.json`.
  - **Suite Comparison Matrix Enhancements**: Added a dedicated `Test Description & Focus` column in Tab 2 of `static/report.html` featuring color-coded variant badges (`getVariantBadge()`) that automatically identify and highlight the changing factor across test cases (e.g. `10k subjects`, `Wildcard (>)`, `50 Streams`, `20 Consumers`, `Memory R=3`, `32 KiB`, `100k msgs/s`).
  - **Dedicated Run Header & Run Configuration Overhaul**: Enhanced single-run dashboards and child run drilldown views with a prominent Test Focus card displaying the variant focus badge, parameter pills, and human-readable parameter mapping in the Run Configuration table.
  - **Canonical Test Suite Sequencing**: Upgraded `scripts/aggregate_suite.py` to order suite steps according to the canonical sequence defined in `suites/<suite_id>.conf`, resolving directory-level alphabetical sorting issues (e.g. `10K` preceding `1K`).
  - **Retroactive Enrichment**: Re-aggregated all 5 completed suites (`subject-cardinality`, `jetstream-guarantees`, `jetstream-ladder`, `payload-scaling`, `core-ladder`), immediately enriching existing reports with the new metadata columns and configuration tables.
- **Three Canonical Scaling & Sizing Suites Added**: Added 3 new benchmark suites tailored for cluster capacity planning and TCO calculations:
  - `suites/subject-cardinality.conf`: Evaluates Core NATS sublist prefix trie lookup latencies and RAM scaling across 1 to 10,000 subjects and wildcard subscriptions (`>`), informing container Pod memory request sizing.
  - `suites/stream-scaling.conf`: Evaluates JetStream multi-Raft control-plane tax, heartbeat CPU overhead, and file descriptor usage across 1 to 50 concurrent streams, informing container Pod vCPU sizing.
  - `suites/consumer-fanout.conf`: Evaluates JetStream Persistent Disk write multiplication and explicit ACK consensus amplification across 1 to 20 durable consumers, informing Persistent Disk tiering (standard SSD vs provisioned high-IOPS SSD).
- **Native `nats bench` Multi-Subject, Randomization, and Consumer Fan-Out Orchestration**:
  - Enhanced `scripts/workload.sh` to support `--multisubjectrandomize`, separate publisher/subscriber subject overrides (`PUB_SUBJECT`, `SUB_SUBJECT` for wildcard routing tests), and concurrent background `nats bench js consume` workers during JetStream publish benchmarks when `CONSUMER_COUNT > 0`.
  - Enhanced `scripts/js_stream.sh` to pre-create and manage multiple concurrent Raft streams (`BENCH_STREAM_2` .. `BENCH_STREAM_N`) when `STREAM_COUNT > 1` and durable consumers (`bench-consumer-1` .. `bench-consumer-N`) when `CONSUMER_COUNT >= 1`, with robust cleanup across test iterations.
  - Preserved complete non-breaking backward compatibility across all existing single-run and suite metrics schemas (`results.json`, `suite-results.json`).
- **Suite Identification Banner & Intelligent Metadata Discovery**: Added a dedicated Suite Identification Banner card in Tab 2 (Executive Suite & Comparison) of `static/report.html` displaying human-readable Suite Title, Suite ID badge, description, and execution Run ID. Upgraded `scripts/aggregate_suite.py` to auto-discover suite ID from directory hierarchy and parse canonical `SUITE_TITLE` and `SUITE_DESCRIPTION` from `suites/<id>.conf` when invoked without explicit CLI arguments.
- **Auto-Scaled Storage Units & Empirical Disk Write Rate Fallback**: Overhauled disk telemetry display across `static/report.html` (Comparative Ladder Matrix and Node Resource breakdowns):
  - Added dynamic `formatBytes()` helper to auto-scale disk growth across `B`, `KiB`, `MB`, and `GB`, resolving misleading displays of large writes in raw KiB (e.g. `950649.3 KiB` -> `+928.4 MB`).
  - Added empirical disk write rate fallback `disk_growth_bytes / (1024 * 1024 * duration_seconds)` when container cgroup `blkio` counters report `0.00 MB/s` due to host bind-mounted directories (`./data/nats-X:/data`) bypassing virtual block device accounting.

## [Unreleased] - 2026-11-09

### Fixed
- **CPU Percentage Calculation Duration Unit Regex**: Corrected duration unit parsing regex alternation in `scripts/metrics.sh` line 83 from `(h|m|s|ms|us|ns)` to `(ms|us|ns|h|m|s)`. Placing multi-character units before single characters prevents Podman sub-second durations (e.g. `877.67ms`) from matching `m` (minutes) instead of `ms` (milliseconds), which previously inflated CPU percentages by 60,000x (e.g. 474,646%).
- **Subscriber Premature Termination on 100k Saturation**: Added a 10-second subscriber backlog drain wait loop to `scripts/workload.sh` after the publisher process completes. Prevents `trap cleanup EXIT` from immediately terminating subscriber processes while messages are still draining from the server socket buffer, resolving missing `sub.csv` artifacts and zero-subscriber metrics under saturation.
- **JetStream Memory Stream Quota Exceeded & Parameter Preservation**:
  - Added `MAX_BYTES=256MB` override to `TEST-G2-JS-MEM-R3` in `suites/jetstream-guarantees.conf`. Prevents cluster stream creation rejections caused by exceeding the 512MB `max_mem_store` limit configured in `deploy/nats-1.conf`.
  - Added `_ENV_MAX_BYTES`, `_ENV_RETENTION`, `_ENV_BATCH`, and `_ENV_ACKS` to the environment preservation and reapplication blocks in `benchmark.sh`. Previously, sourcing `benchmark.conf` overrode caller/suite parameters back to configuration defaults (`5GB`), which caused memory stream allocation failures.
- **Natural Payload Size Sorting in Suite Aggregations**: Implemented `parse_payload_bytes()` in `scripts/aggregate_suite.py` and updated suite sorting to sort by both target rate and payload byte size `(target_rate, payload_bytes)`, ensuring payload test runs sort logically (128B -> 1KB -> 8KB -> 32KB) instead of lexicographically (128B -> 1KB -> 32KB -> 8KB).
- **Suite Executive Report Generation**: Resolved bug preventing `suite.sh` from producing or rendering `executive-report.html` and `suite-report.html` on external systems:
  - Exported caller `NATS_URL` into child test subshells so remote endpoints are not clobbered by defaults in `benchmark.conf`.
  - Preserved `TIMESTAMP_SUBFOLDER` across `benchmark.sh` configuration sourcing, preventing unexpected nested test output directories.
  - Replaced literal sentinel string matching with multi-line regex substitution in `aggregate_suite.py` and `aggregate.py`, eliminating sensitivity to whitespace or CRLF line endings across different environments.
  - Hardened `compute_recommendations()` and rate parsing in `aggregate_suite.py` with `safe_float` and `safe_int` coercions, preventing unhandled exceptions on unthrottled (zero rate), missing, or malformed metrics.
  - Implemented recursive child result discovery (`child.rglob("results.json")`) and synthetic fallback run records from `test-config.txt` so `executive-report.html`, `suite-report.html`, and `suite-results.json` are always generated even during partial or failed runs.
  - Updated `static/report.html` suite rendering: dynamically sets `#status-badge` (`COMPLETED`, `PARTIAL`, `FAILED`, or `NO DATA`) and `#sub-header`, adds responsive SVG `viewBox="0 0 800 360"`, and activates the comparison tab prior to SVG curve calculation.

### Added
- **Telemetry Dictionary & Infrastructure Capacity Planning Guide**: Added `docs/METRICS_AND_SIZING_GUIDE.md` providing comprehensive reference documentation on metric definitions, calculation formulas, and infrastructure capacity sizing rules. Answers core architectural questions regarding Target vs Achieved Rate throttling, Delivery Efficiency % mechanics, CPU Efficiency formulas, Peak RAM vs Average RAM for Kubernetes/OpenShift OOM prevention, Network Egress (TX) broadcast fan-out amplification, JetStream inverted latency batching curves, and storage sizing standards.
- **Interactive SVG Saturation Curve Dashboard Overhaul**: Comprehensive upgrade to the Throughput vs Latency Saturation Curve in `static/report.html`:
  - Visual Legend: Added a responsive top legend displaying Median Latency (P50 Accent Blue), Tail Latency (P99 Red), and Saturation Knee (Amber dashed marker).
  - Explicit Point Callouts & Tooltips: Rendered direct visible numerical callouts above each P99 point for instant comprehension without requiring user interaction, accompanied by native SVG `<title>` tooltips providing Test ID, Offered Load, Delivered Throughput, P50 Median, and P99 Tail latencies on hover.
  - Calibrated X/Y Grid & Numerical Tick Values: Generated 5 calibrated horizontal grid levels from 0 to peak latency with auto-scaling units (`us` or `ms`), paired with vertical guide columns displaying both target offered load and delivered throughput.
  - Saturation Knee Marker: Dynamically plots an amber dashed vertical line with a labeled knee badge at the identified capacity boundary rate.
  - Re-aggregated all 4 benchmark suites (`core-ladder`, `payload-scaling`, `jetstream-ladder`, `jetstream-guarantees`) with the enhanced chart.
- **Developer Ergonomics and Inline Architectural Documentation**: Added comprehensive argument contract headers (e.g. 17-parameter positional table in `workload.sh`, invocation signature in `metrics.sh`, subshell isolation documentation in `suite.sh`), inline rationale for the environment-save/restore pattern in `benchmark.sh`, explanation of the Podman cumulative `cpu_time` delta calculation, JetStream consumer setup rationale, and delivery efficiency calculation formulas for broadcast fan-out vs queue groups.
- **Template Injection Safeguards**: Added explicit warning comments and programmatic verification in `aggregate.py` and `aggregate_suite.py` to check for the `<script id="benchmark-data">` sentinel tag before attempting HTML report generation, preventing silent failures.
- **Structured Dashboard Code Sections**: Added numbered ASCII section dividers in `static/report.html` organizing the client-side logic into discrete modules (State & Utilities, Single-Run KPI/Latency/Node Rendering, Suite Comparison Table, Child Run Drill-Down, SVG S-Curve, Configurator, and File Drop/DOM Initialization).

### Changed
- **Latency Distribution Table & Global Telemetry Redesign**: Overhauled the Latency Distribution table in `nats-capacity-benchmark/static/report.html` across both single-run dashboard and suite test accordion views:
  - Extracted global distribution metrics (`Total Samples`, `Mean Latency`, and `Std Dev (sigma)`) into a responsive summary header bar above the table, eliminating sparse, empty dashes (`-`) in rows 1 through 5.
  - Replaced the sparse table columns with a fully populated quantile breakdown (`Percentile`, `Operation Latency`, `Cumulative Deliveries`, `Volume Slice`), dynamically computing cumulative message volumes (Min: 1 msg, P50: 50%, P90: 90%, P99: 99%, P99.9: 99.9%, Max: 100%) for complete statistical transparency.
- **README.md architecture and execution modernization**: Updated root `README.md` to align with the unified configuration and reporting architecture:
  - Replaced the outdated 4-file decoupled configuration guide (`config/*.conf`) with the canonical `benchmark.conf` parameters and environment variable override semantics.
  - Modernized the Execution Guide to include single-configuration invocations, runtime environment overrides, and the interactive Web UI runner (`./benchmark.sh --serve`).
  - Corrected Section 6 (Supported Workload Scenarios) to match current engine capabilities: Core NATS (`pubsub`, `publish`, `subscribe` with `QUEUE` fan-out controls) and JetStream (`js-pub-sync`, `js-pub-async`, `js-consume`).
  - Aligned Section 7 (Results & Output Artifacts) with the canonical artifact inventory (`results.json`, `report.html`, `metrics.json`, `pub.csv`, `sub.csv`, `test-config.txt`, `suite-results.json`, `executive-report.html`), eliminating deprecated file references.
  - Updated cross-reference links in Section 9 to point to `docs/tests.md`, `docs/tests-jetstream.md`, and `docs/benchmarking.md`.
- **Documentation synchronized with unified architecture**: Updated `docs/benchmarking.md`, `docs/tests.md`, and `docs/tests-jetstream.md` to remove legacy artifact references (`summary.csv`, `node-summary.csv`, `metrics.csv`, `workload.log`, `subscriber.log`, `executive-summary.md`, and `config/*.conf`). Updated workflow examples and artifact inventories to reference canonical files (`benchmark.conf`, `results.json`, `metrics.json`, `pub.csv`, `sub.csv`, `suite-results.json`, `report.html`, and `executive-report.html`).
- **Suite `.conf` files updated to canonical variable names**: All four suite configuration files (`core-ladder.conf`, `jetstream-guarantees.conf`, `jetstream-ladder.conf`, `payload-scaling.conf`) updated from legacy override names (`TARGET_MSGS_PER_SEC`, `PUBLISHER_CONCURRENCY`, `SUBSCRIBER_CONCURRENCY`, `TEST_DURATION_SECONDS`, `MESSAGE_SIZE`, `STORAGE_TYPE`, `REPLICATION_FACTOR`, `SUBSCRIPTION_PATTERN`) to current canonical names (`THROUGHPUT`, `PUBS`, `SUBS`, `DURATION`, `SIZE`, `STORAGE`, `REPLICAS`, `QUEUE`). Stale config file references (`config/core-nats.conf`, `config/jetstream.conf`) replaced with `benchmark.conf` in all suite entries.
- **JetStream suite entries now specify `SCENARIO` explicitly**: `jetstream-guarantees.conf` and `jetstream-ladder.conf` entries now include `SCENARIO=js-pub-sync` and `STORAGE`/`REPLICAS` overrides, making each test case self-describing without relying on a deleted config file to carry those settings.

### Removed
- **Dead fallback paths in `benchmark.sh`**: Removed unreachable fallback lookup for `config/benchmark.conf` and `config/core-nats.conf` (the `config/` directory was deleted in the prior refactoring session).
- **`summary.csv` backwards-compatibility write in `aggregate.py`**: Removed the `summary.csv` output that was marked "backwards compatibility". The canonical output is `results.json`. Nothing in the current pipeline reads `summary.csv`.

## [Unreleased] - 2026-11-05

### Added
- **Unified Configuration Model (`benchmark.conf`)**: Consolidated disparate configuration files into a single configuration matching `nats bench` CLI flags (`--server`, `--clients`, `--throughput`, `--duration`, `--size`, `--stream`, `--consumer`, `--batch`, `--acks`).
- **Constant Static HTML Dashboard (`static/report.html`)**: Introduced a permanent, self-contained, zero-dependency HTML dashboard with single-run KPI cards, latency tables, dynamic SVG throughput-vs-latency curves, and a visual configurator with "Copy Command" and "Download benchmark.conf" capabilities.
- **Dual Execution Modes (CLI + Web UI)**: Added optional runner server (`scripts/server.py` invoked via `./benchmark.sh --serve`) allowing users to configure and trigger benchmark runs directly from their browser, while terminal execution via `./benchmark.sh` remains standard.
- **Direct CSV Ingestion**: Integrated native `nats bench --csv` flags across all pub, sub, and JetStream operations, eliminating regex log scraping.

### Removed
- **Runtime Python HTML & SVG Generation**: Removed 2,000+ lines of Python string concatenation in `scripts/aggregate.py` and `scripts/aggregate_suite.py`.
- **Runtime Markdown Summaries**: Permanently removed `executive-summary.md` generation.
- **Redundant Scripts**: Removed obsolete `scripts/results.sh` and split config files in `config/`.

### Changed
- **Massive Codebase Reduction**: Reduced overall benchmark harness lines of code from 3,643 to 1,467 lines (~60% net reduction).
- **Streamlined Aggregators**: Slashed `scripts/aggregate.py` from 1,635 to 208 lines and `scripts/aggregate_suite.py` from 578 to 66 lines.
- **Local Environment Hardening & Resource Enforcements (`deploy/`)**:
  - Configured `max_mem_store: 512MB` and `max_file_store: 10GB` in `nats-{1,2,3}.conf` to prevent unbounded memory or host disk exhaustion under saturation workloads.
  - Added connection buffers and write deadlines (`write_deadline: "5s"`, `max_pending: 32MB`, `max_payload: 8MB`) to eliminate slow-consumer buffer accumulation in container RAM.
  - Hardened `deploy/compose.yaml` with explicit swap isolation (`memswap_limit`), `restart: unless-stopped`, and `stop_grace_period: 15s`.
  - Configured `maxconn 4096` and `timeout queue 10s` in `deploy/haproxy.cfg` to protect Podman's user-mode network proxy (`gvproxy`) from ephemeral socket starvation.
  - Documented Podman Machine sizing guidelines (>= 5 vCPUs, >= 6 GiB RAM) in `README.md`.
- **Executive Benchmark Suite Runner (`suite.sh`)**: Added automated multi-test execution harness supporting sequential test matrices with cluster isolation (`nats stream rm -f`), mandatory 10-second quiescence cooldowns between runs, and automatic executive aggregation.
- **Predefined Executive Suites (`suites/*.conf`)**:
  - `jetstream-guarantees.conf`: Compares Core NATS, JetStream Memory R=3, JetStream File R=1, and JetStream File R=3.
  - `core-ladder.conf`: Core NATS throughput saturation ladder (1k to 100k msgs/s).
  - `jetstream-ladder.conf`: JetStream sync saturation ladder (250 to 10,000 msgs/s).
  - `payload-scaling.conf`: Payload dimension scaling from 128 B to 32 KiB.
- **Executive Comparative Aggregator (`scripts/aggregate_suite.py`)**:
  - Computes executive cluster sizing recommendations (sustained capacity, peak rate, hardware bottlenecks, storage delta).
  - Generates zero-dependency inline SVG Throughput vs Latency saturation curve ("hockey stick" curve) with P50/P99 latency knee callout.
  - Compiles the "Cost of Guarantees" comparative matrix table with deep-links to individual test reports.
  - Produces standalone `executive-report.html` and portable `executive-summary.md`.
- **Empirical Host Directory Disk Sizing**: Host filesystem scanner (`os.scandir`) measuring physical JetStream storage growth per node under `deploy/data/nats-{1,2,3}` to ensure accurate storage telemetry on platforms where container cgroup blkio is suppressed.
- **Strict POSIX Monitor Mode (`set -m`)**: Enforced monitor mode in `scripts/workload.sh` to prevent POSIX `SIG_IGN` background signal masking and ensure reliable graceful shutdown of `nats bench`.

### Changed
- **Suite Matrix Calibration & Single-Variable Parity**:
  - Calibrated `suites/jetstream-guarantees.conf` with `SUBSCRIPTION_PATTERN=queue` on Core NATS baseline for exact 1x load-balanced delivery matching JetStream pull consumer semantics.
  - Standardized `suites/jetstream-ladder.conf` with constant 8 publishers and 2 consumers across all 6 steps (250 to 10k msg/s) to isolate offered load as the sole independent variable.
  - Standardized `suites/payload-scaling.conf` at fixed 5,000 msg/s with 1x queue delivery, isolating message size (128 B to 32 KiB) without network broadcast multiplication.
  - Added post-suite stream cleanup in `suite.sh`.
- **Publisher Watchdog & Subscriber Drain Refinement**:
  - Expanded publisher watchdog in `scripts/workload.sh` to `DURATION + PUBLISHER_GRACE_SECONDS` to guarantee healthy workloads write their full stats tables without premature interruption.
  - Eliminated premature `kill -INT` signal broadcast on subscribers, allowing clean drain completion.
- Refactored `benchmark.sh` timeout handling to distinguish between nominal duration, grace window, and hard watchdog termination.
- Upgraded report visual hierarchy with dedicated setup details, test details, and results cards.

