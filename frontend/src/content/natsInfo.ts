export interface NatsComponentInfo {
  id: string;
  title: string;
  role: string;
  concepts: string[];
  demoUsage: string;
  trivia?: string;
}

export const NATS_COMPONENTS_INFO: Record<string, NatsComponentInfo> = {

  // Topology & Architectural Components

  'react-ui': {
    id: 'react-ui',
    title: 'React UI',
    role: 'Interactive demonstration dashboard for observing and controlling NATS capabilities.',
    concepts: [
      'Decoupled Frontend Architecture',
      'HTTP Client Gateway',
      'Real-Time State Polling',
      'Zero Direct NATS Coupling',
    ],
    demoUsage:
      'Communicates via HTTP with the Demo Control Service (:8080) for UI state and telemetry, and Job Service (:8081) for business job submissions.',
    trivia:
      'In a secure NATS architecture, browser clients do not require direct NATS credentials or WebSocket connections when boundary gateways mediate access.',
  },

  'job-service': {
    id: 'job-service',
    title: 'Job Service',
    role: 'Pure business microservice handling job submissions and domain validation.',
    concepts: [
      'Service Decoupling',
      'NATS Go Client (nats.go)',
      'Event Publishing',
      'Synchronous Request / Reply',
      'W3C Trace Context Propagation',
    ],
    demoUsage:
      'Exposes REST endpoints (POST /jobs, POST /jobs/validate) on port 8081. Decoupled from demo UI concerns - only executes business logic and communicates via NATS.',
    trivia:
      'A pure business service publishes events to subjects without coupling to consumers, audit loggers, or downstream orchestrators.',
  },

  'demo-control-service': {
    id: 'demo-control-service',
    title: 'Demo Control Service',
    role: 'Dedicated UI gateway and observer for the demonstration platform.',
    concepts: [
      'Passive Event Observation',
      'Wildcard Subscriptions (jobs.>)',
      'Replay Consumer Management',
      'Remote Worker Control',
    ],
    demoUsage:
      'Passively subscribes to jobs.> to populate the live activity log, executes historical JetStream replays, and controls worker pools without altering business code.',
    trivia:
      'Because NATS uses open publish/subscribe routing, authorized services can attach wildcard observers (jobs.>) without existing publishers or workers knowing.',
  },

  // Backward compatibility alias for topology diagram
  'demo-service': {
    id: 'demo-control-service',
    title: 'Demo Control Service',
    role: 'Dedicated UI gateway and observer for the demonstration platform.',
    concepts: [
      'Passive Event Observation',
      'Wildcard Subscriptions (jobs.>)',
      'Replay Consumer Management',
      'Remote Worker Control',
    ],
    demoUsage:
      'Passively subscribes to jobs.> to populate the live activity log, executes historical JetStream replays, and controls worker pools without altering business code.',
    trivia:
      'Because NATS uses open publish/subscribe routing, authorized services can attach wildcard observers (jobs.>) without existing publishers or workers knowing.',
  },

  'nats-server': {
    id: 'nats-server',
    title: 'NATS Server',
    role: 'High-performance messaging system providing Core NATS and JetStream capabilities.',
    concepts: [
      'Core NATS Engine',
      'JetStream Engine',
      'Subject-Based Addressing',
      'Publish / Subscribe',
      'Request / Reply',
      'Streams and Consumers',
    ],
    demoUsage:
      'Single deployed NATS server process handling transient pub/sub messaging while hosting the persistent JOBS and JOBS_DLQ streams.',
    trivia:
      'Core NATS provides ultra-low latency in-memory messaging, while JetStream provides persistence, replay, deduplication, and durable consumption within the same server binary.',
  },

  'jobs-stream': {
    id: 'jobs-stream',
    title: 'JOBS Stream',
    role: 'Persistent write-ahead log that captures and stores messages on defined subjects.',
    concepts: [
      'JetStream Stream',
      'Message Persistence',
      'Subject Capture (jobs.submitted)',
      'Stream Sequence Numbers',
      'Retention Policies',
      'Storage Footprint',
    ],
    demoUsage:
      'Persists messages published to jobs.submitted so they remain available for consumer workers even when the processor service is offline or paused.',
    trivia:
      'Streams decouple ingestion from consumption. Publishers append to the stream at wire speed without coordinating with consumer groups or waiting for ACKs from workers.',
  },

  'dead-letter-queue': {
    id: 'dead-letter-queue',
    title: 'Dead Letter Queue (JOBS_DLQ)',
    role: 'Isolated JetStream stream for persistent storage of poison messages that exceed retry limits.',
    concepts: [
      'Dead Letter Queue Pattern',
      'Max Deliveries (MaxDeliver)',
      'Redelivery Exhaustion',
      'Poison Message Isolation',
      'DLQ Stream Routing',
    ],
    demoUsage:
      'When a worker NAKs a job repeatedly until reaching 3 attempts, it routes the message to jobs.dlq in stream JOBS_DLQ, ACKs the original JOBS stream message, and notifies dlq-inspector.',
    trivia:
      'A DLQ in NATS does not require a proprietary server feature; it is an architectural pattern implemented cleanly using another JetStream stream and application routing.',
  },

  // Backward compatibility alias for DLQ
  'jobs-dlq': {
    id: 'dead-letter-queue',
    title: 'Dead Letter Queue (JOBS_DLQ)',
    role: 'Isolated JetStream stream for persistent storage of poison messages that exceed retry limits.',
    concepts: [
      'Dead Letter Queue Pattern',
      'Max Deliveries (MaxDeliver)',
      'Redelivery Exhaustion',
      'Poison Message Isolation',
      'DLQ Stream Routing',
    ],
    demoUsage:
      'When a worker NAKs a job repeatedly until reaching 3 attempts, it routes the message to jobs.dlq in stream JOBS_DLQ, ACKs the original JOBS stream message, and notifies dlq-inspector.',
    trivia:
      'A DLQ in NATS does not require a proprietary server feature; it is an architectural pattern implemented cleanly using another JetStream stream and application routing.',
  },

  'consumer': {
    id: 'consumer',
    title: 'JetStream Consumer',
    role: 'Logical JetStream resource that controls and tracks message delivery state from a Stream.',
    concepts: [
      'Consumer State Tracking',
      'Durable vs Ephemeral Lifecycle',
      'Push vs Pull Consumption',
      'ACK / NAK Semantics',
      'AckWait & MaxDeliver',
      'Pending vs Ack Pending Backlog',
    ],
    demoUsage:
      'The job-processor consumer reads from the JOBS stream, manages message dispatch to workers, and tracks redelivery attempts upon processing failure.',
    trivia:
      'Durable and Ephemeral describe Consumer lifecycle, while Push and Pull describe delivery method. Stream persistence stores messages; Durable describes persistence of Consumer state.',
  },

  'processor-service': {
    id: 'processor-service',
    title: 'Processor Service',
    role: 'Application service that executes business workloads delivered by NATS consumers.',
    concepts: [
      'Worker Pool',
      'Competing Consumers',
      'Work Distribution',
      'Message Acknowledgement (msg.Ack)',
      'Negative Acknowledgement (msg.Nak)',
      'Failure Redelivery',
    ],
    demoUsage:
      'Runs a pool of worker goroutines (processor-1, processor-2) that pull jobs from the JetStream consumer, simulate execution, and emit lifecycle events.',
    trivia:
      'Multiple workers can consume from the same JetStream consumer to share processing load without requiring application-managed message partitioning.',
  },

  // Dashboard Sections & Capability Panels

  'platform-status': {
    id: 'platform-status',
    title: 'Platform Status',
    role: 'Global operational health and control bar for services, connectivity, and JetStream availability.',
    concepts: [
      'NATS Server Connectivity',
      'JetStream Subsystem Health',
      'Microservice Discovery',
      'Global Processing Toggle',
    ],
    demoUsage:
      'Displays real-time connectivity for NATS Server, JetStream, Job Service, and Processor Service, and provides a toggle to pause or resume processing.',
    trivia:
      'NATS clients maintain persistent lightweight TCP connections and can reconnect automatically while buffering in-flight requests.',
  },

  'submit-job': {
    id: 'submit-job',
    title: 'Pub / Sub & Stream Publishing',
    role: 'Interactive publisher demonstrating transient Core NATS and persistent JetStream message delivery.',
    concepts: [
      'Core NATS Pub/Sub',
      'JetStream Publishing',
      'Transient vs Persistent Delivery',
      'Subject Addressing',
      'Delivery Mode Semantics',
      'Message ID Stamping',
    ],
    demoUsage:
      'Publishes job requests using Core NATS (at-most-once) or JetStream (at-least-once), allowing immediate comparison of offline behavior and durability.',
    trivia:
      'JetStream allows the same NATS platform to support both lightweight transient messaging and durable streaming without running separate message brokers.',
  },

  'message-deduplication': {
    id: 'message-deduplication',
    title: 'Message Deduplication',
    role: 'Server-side, stream-scoped deduplication that discards duplicate publishes within a configured time window.',
    concepts: [
      'Stream-Level Deduplication',
      'Deduplication Window (duplicate_window)',
      'Nats-Msg-Id Header',
      'PubAck Duplicate Detection',
      'Effectively-Once Business Semantics',
    ],
    demoUsage:
      'The JOBS stream tracks Nats-Msg-Id within a 2-minute sliding window. Re-publishing the same ID returns a duplicate PubAck and emits DEDUPLICATED without storing a second entry.',
    trivia:
      'JetStream combines acknowledgement-based delivery with publisher message-ID deduplication. Deduplication is not the same as exactly-once business processing; exactly-once business effects require combining messaging deduplication with idempotent application handlers.',
  },

  'consumer-lab': {
    id: 'consumer-lab',
    title: 'Consumer Lab',
    role: 'Interactive test bench for configuring JetStream Consumer parameters and observing delivery behavior.',
    concepts: [
      'Durable vs Ephemeral Consumers',
      'Pull Consumption Model',
      'Competing Consumers',
      'Normal vs Ordered Consumers',
      'Flow Control & Backpressure',
      'Worker Concurrency',
    ],
    demoUsage:
      'Dynamically configures consumer durability, adjusts worker pool concurrency (1-3 workers), and toggles between Normal and Ordered consumption.',
    trivia:
      'Multiple workers can share processing load without application-managed message partitioning. NATS also provides ordered-consumption capabilities without requiring Kafka-style partition-count planning.',
  },

  'request-reply': {
    id: 'request-reply',
    title: 'Request / Reply',
    role: 'Synchronous RPC pattern executed natively over subject-based messaging.',
    concepts: [
      'Request / Reply Pattern',
      'Dynamic Inbox Subjects (_INBOX.)',
      'Point-to-Point Response Routing',
      'Requester Timeout Handling',
      'Synchronous Semantics over NATS',
    ],
    demoUsage:
      'Dispatches job validation requests to jobs.validate and waits synchronously for a response from processor-service, demonstrating natural timeouts when the processor is paused.',
    trivia:
      'NATS provides Request / Reply natively through its subject-based messaging model, avoiding the need to build RPC semantics over a persistent log.',
  },

  'jetstream-replay': {
    id: 'jetstream-replay',
    title: 'Stream Replay',
    role: 'Time-window and sequence rewind controls for reading historical stream data.',
    concepts: [
      'Stream Replay',
      'Deliver Policy (DeliverByStartSequence, DeliverByStartTime)',
      'Replay Policy (Instant vs Original)',
      'Historical Backfill',
      'Ephemeral Replay Consumers',
    ],
    demoUsage:
      'Creates an ephemeral consumer starting from a selected sequence number or timestamp to replay previously stored JOBS messages without altering stream state or disrupting active workers.',
    trivia:
      'Consumers can start from historical Stream positions, enabling replay and backfill scenarios without modifying or removing original stored messages.',
  },

  'activity-log': {
    id: 'activity-log',
    title: 'Activity Log',
    role: 'Live chronological stream of message deliveries, worker milestones, and lifecycle transitions.',
    concepts: [
      'Lifecycle Telemetry',
      'Event Observation',
      'Multi-Subject Monitoring',
      'W3C Trace Context Association',
      'Delivery Sequence Tracking',
    ],
    demoUsage:
      'Displays real-time events published across jobs.> subjects (published, stored, delivered, processed, completed, failed, dlq_published), with full-text search and filtering.',
    trivia:
      'Subject wildcards allow monitoring services to observe entire message hierarchies with a single subscription, making system observability lightweight and non-invasive.',
  },

  'job-details': {
    id: 'job-details',
    title: 'Job Details Inspector',
    role: 'Modal inspection dialog displaying granular metadata, delivery attempts, and history for a selected job.',
    concepts: [
      'Message Metadata Inspection',
      'Delivery Attempt Counters',
      'Processing History Timeline',
      'Distributed Trace Context',
      'Raw Payload Verification',
    ],
    demoUsage:
      'Opens when clicking any Job ID in the Activity Log, displaying delivery counts, status milestones, raw JSON payloads, and direct links to Grafana Tempo traces.',
    trivia:
      'With at-least-once delivery, messages can be redelivered if workers crash or fail before ACKing. Inspecting delivery counts helps identify intermittent worker failures or poison payloads.',
  },

  'subject-addressing': {
    id: 'subject-addressing',
    title: 'NATS Subject Addressing',
    role: 'Interactive explorer for NATS subject hierarchies, tokens, and wildcard matching patterns.',
    concepts: [
      'Subject Hierarchy',
      'Dot-Delimited Tokens',
      'Exact Matching',
      'Single-Token Wildcard (*)',
      'Multi-Token Wildcard (>)',
      'Subject-Based Routing',
    ],
    demoUsage:
      'Shows active subscriptions alongside incoming subjects, visualizing how exact subjects (jobs.submitted), single-token wildcards (jobs.*), and multi-token wildcards (jobs.>) match events.',
    trivia:
      'Subject-based routing allows applications to communicate without provisioning a queue or broker configuration for every routing relationship.',
  },

  'metrics-observability': {
    id: 'metrics-observability',
    title: 'Metrics & Observability',
    role: 'Integrated dashboard views for application metrics, NATS operational stats, and distributed traces.',
    concepts: [
      'Prometheus Metrics Exporter',
      'NATS Server Monitoring',
      'JetStream Operational Metrics',
      'OpenTelemetry OTLP Instrumentation',
      'W3C Distributed Tracing',
    ],
    demoUsage:
      'Correlates NATS server and JetStream counters with application-level job latencies and error rates, with direct links to Grafana and Tempo.',
    trivia:
      'NATS exposes operational information through native monitoring events and integrates with standard Prometheus and OpenTelemetry observability stacks.',
  },

  'queue-groups': {
    id: 'queue-groups',
    title: 'Core NATS Queue Groups',
    role: 'Native server-side load balancing distributing messages across dynamic pools of subscribers with zero broker state.',
    concepts: [
      'Queue Group (job-workers): A logical pool identifier supplied at subscription time (nc.QueueSubscribe). All subscribers sharing the same queue name form a single cooperative processing pool.',
      'Subject (jobs.queue): The destination address where producers publish messages. NATS transparently balances deliveries among subscribers registered to the queue group on this subject.',
      'Load-Balanced Delivery (1 of N): For each published message, the NATS server randomly routes the message to exactly one connected subscriber in the queue group, eliminating application-level partition management.',
      'At-Most-Once Delivery (Best-Effort): Transient, in-memory delivery semantics. Messages are not written to disk; if no worker is connected when a message is published, the message is discarded.',
      'Stateless Messaging (No ACK/NAK): Unlike JetStream, Core NATS Queue Groups do not require acknowledgements, cursors, or redelivery timers, achieving sub-millisecond wire latencies.',
      'Multi-Group Fanout: If multiple distinct queue groups subscribe to the same subject (e.g. workers and auditors), NATS delivers the message to 1 member of each group, plus all non-queue subscribers.',
      'Dynamic Membership (1 to 5 Workers): Workers can join or leave the group on the fly without cluster-wide partition rebalances or stop-the-world pauses.',
      'Worker Distribution: Real-time telemetry tracking how the NATS server balances incoming message volume across all connected worker instances.',
      'Reset Worker Distribution: Clears accumulated distribution counters to baseline comparative load-balancing benchmarks.',
    ],
    demoUsage:
      'Publish test message bursts to jobs.queue and observe the NATS server distribute messages across 1 to 5 active workers (processor-1 through processor-5) sharing the job-workers queue group, contrasting in-memory 1-of-N load balancing with JetStream persistent consumers.',
    trivia:
      'Unlike Kafka consumer groups which require partition assignments and rebalance protocols, or RabbitMQ which relies on consumer prefetch buffers, Core NATS queue groups balance messages directly in the server socket layer with zero coordination overhead.',
  },

};

