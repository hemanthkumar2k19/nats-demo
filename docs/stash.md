## OCP
- 3 NATS Cluster Deployments
  - Cluster communication through routes on port `6222`.
  - JetStream enabled.
    - `max_memory_store`: 512 MB
    - `max_file_store`: 5 GB
  - Client port `4222` exposed.
  - Monitoring port `8222` exposed.
- Kubernetes Service
- Kubernetes NodePort
- NATS Box
  - Connects through NodePort: `{OCP_WORKER_HOST_IP:30222}`.
  - Includes NATS CLI and benchmarking utilities.
  - Reference: [NATS Box](https://github.com/nats-io/nats-box).





Multi-Dimensional Architecture Comparison


Area of Consideration	Option 1: Raw NATS SDK	Option 2: Custom Enterprise SDK (Library)	Option 3: Service Proxy (Sidecar / DaemonSet)	Option 4: Interface-Only Thin Wrapper
Initial Platform Effort	🟢 None
Teams pull community drivers directly.	🔴 Very High
Must build, test, and maintain feature-parity across 3+ languages.	🟡 Medium
Build once in Go. Requires Kubernetes deployment configuration.	🟢 Low to Medium
Write minimal interface contracts in target languages.
Long-Term Maintenance	🔴 High
Platform team acts as an endless support desk for developer misconfigurations.	🔴 High
Every NATS cluster update or bug fix requires patching and releasing 3 libraries.	🟢 Very Low
Single Go codebase. Upgrades are deployed instantly to proxy containers.	🟡 Medium
Low code footprint, but requires occasional updates across language contracts.
Developer Experience (DevEx)	🔴 Poor
High cognitive load. App teams must master NATS subjects, JetStream, and Raft.	🟢 Excellent
Simple, idiomatic code abstractions tailored to your business.	🟢 Excellent
Developers use ubiquitous, standard gRPC/HTTP clients.	🟡 Moderate
Simplifies logic, but teams still need to pull specific internal dependencies.
Governance & Policy Enforcement	🔴 Impossible to Enforce
Teams can type arbitrary subjects, bypass schemas, and leak consumers.	🟡 Soft Enforcement
Enforced via code, but rogue teams can still bypass the library or skip updates.	━━━━ Absolute Guardrails ━━━━
Proxy intercepts all network traffic. Blocks non-compliant data at the edge.	🟡 Soft Enforcement
Enforced via interface compliance, but rely on code discipline.
Coupling & Migration Risk	🔴 Total Lock-In
NATS imports leak everywhere. Moving to Kafka/SQS requires massive app rewrites.	🟢 Decoupled
Abstracted behind the library. Swap broker code inside the dependency.	━━━━ Total Decoupling ━━━━
Apps only know gRPC/HTTP. Broker can change without app awareness.	🟢 Decoupled
The implementation behind the interface can be swapped cleanly.
Resource Efficiency	🟢 Maximum
Direct memory-to-wire execution with zero translation layers.	🟢 High
Minor language-specific reflection/serialization overhead.	🔴 Variable
Sidecars waste idle RAM cluster-wide. DaemonSets (Node Proxy) optimize this.	🟢 Maximum
Direct execution using the raw SDK underneath the interface.
Blast Radius & Reliability	🟢 Isolated
App errors or connection crashes are entirely self-contained.	🟢 Isolated
Dependency runs entirely within the individual application process.	🟡 Infrastructure Risk
Central proxy is a deadly SPOF. Sidecar/DaemonSet limits risk to pod/node.	🟢 Isolated
Errors are bound directly to the application process.

