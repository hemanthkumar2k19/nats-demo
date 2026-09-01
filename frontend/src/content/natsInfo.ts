export interface NatsComponentInfo {
  id: string;
  title: string;
  role: string;
  concepts: string[];
  demoUsage: string;
  trivia?: string;
}

export const NATS_COMPONENTS_INFO: Record<string, NatsComponentInfo> = {
  // Topology Components
  'demo-service': {
    id: 'demo-service',
    title: 'Demo Service',
    role: 'API Gateway and Message Publisher / Requester',
    concepts: [
      'NATS Go Client (nats.go)',
      'Publish / Subscribe (Core NATS)',
      'JetStream Publishing (PublishMsg with Ack)',
      'Request / Reply pattern (RequestMsg)',
      'Subject-based routing',
    ],
    demoUsage:
      'Receives HTTP requests from the React UI and uses the official NATS Go client to publish jobs, request synchronous validations, and listen to platform lifecycle events.',
    trivia:
      'The Demo Service uses subject wildcards (jobs.>) to observe all job lifecycle transitions across the system without coupling directly to processor implementations.',
  },

  'nats-server': {
    id: 'nats-server',
    title: 'NATS Server',
    role: 'High-Performance Messaging Backbone and JetStream Engine',
    concepts: [
      'Subject-based addressing',
      'In-memory messaging (sub-microsecond latency)',
      'Request / Reply with dynamic inbox subjects (_INBOX.>)',
      'JetStream persistence engine',
      'Publish deduplication window',
    ],
    demoUsage:
      'Acts as the central communication hub connecting the Demo Service and Processor instances, managing transient Core NATS topics as well as durable JetStream streams.',
    trivia:
      'NATS was built from the ground up to be lightweight and simple: a single Go binary with no external dependencies that can process tens of millions of messages per second.',
  },

  'jobs-stream': {
    id: 'jobs-stream',
    title: 'JOBS Stream',
    role: 'JetStream Persistent Message Store',
    concepts: [
      'Stream storage (File / Memory)',
      'Subject filtering (jobs.submitted)',
      'Global stream sequence numbering (Seq: 1, 2, 3...)',
      'Message deduplication (Nats-Msg-Id)',
      'Configurable retention and deduplication windows',
    ],
    demoUsage:
      'Durably captures all job submissions on subject jobs.submitted. Messages are assigned monotonically increasing stream sequences and retained even if all processors are offline.',
    trivia:
      'In this demo, the stream specifically captures jobs.submitted rather than jobs.> to prevent transient Request/Reply validation messages from being persisted.',
  },

  'consumer': {
    id: 'consumer',
    title: 'JetStream Consumer',
    role: 'Message Delivery and Consumption Manager',
    concepts: [
      'Durable vs Ephemeral consumers',
      'Pull-based consumption (Fetch / NextMsg)',
      'At-Least-Once delivery semantics',
      'Explicit Acknowledgement (ACK, NAK)',
      'Ordered Consumer delivery',
    ],
    demoUsage:
      'Controls how stream messages are delivered to workers. It tracks pending message counts, handles unacknowledged message redeliveries, and dispatches to competing workers.',
    trivia:
      'A Durable consumer retains its cursor position even if the processor service restarts or is toggled OFF, whereas an Ephemeral consumer is discarded when its subscription ends.',
  },

  'processor-service': {
    id: 'processor-service',
    title: 'Processor Service',
    role: 'Background Task Worker Pool',
    concepts: [
      'Competing Consumers pattern (Worker groups)',
      'Simulated task execution duration',
      'Failure simulation and NAK redelivery',
      'Dynamic processing state toggle (ON / OFF)',
      'Synchronous validation responder (jobs.validate)',
    ],
    demoUsage:
      'Runs concurrent worker routines (processor-1, processor-2) that pull jobs from the JetStream consumer, process them, and emit lifecycle tracking events back to NATS.',
    trivia:
      'Multiple worker routines compete for messages from the same JetStream pull consumer, allowing workloads to be shared naturally across workers without manual scheduling.',
  },

  // Dashboard Sections
  'platform-status': {
    id: 'platform-status',
    title: 'Platform Status',
    role: 'Real-Time System Health and Control Bar',
    concepts: [
      'NATS Server Connection Lifecycle (CONNECTING, CONNECTED, DISCONNECTED)',
      'Request / Reply Health Pings (status.processor)',
      'Dynamic JetStream Stream Metrics (StreamInfo / ConsumerInfo)',
      'Processor Service Runtime State Toggle',
    ],
    demoUsage:
      'Monitors NATS server connectivity, active worker availability, pending messages in the JOBS stream, and provides a one-click toggle to pause/resume background job consumption.',
    trivia:
      'When processing is toggled OFF, the processor unsubscribes or halts fetching, allowing messages to accumulate safely in JetStream without losing data.',
  },

  'submit-job': {
    id: 'submit-job',
    title: 'Submit Job',
    role: 'Message Publisher & Deduplication Testing',
    concepts: [
      'Core NATS (Transient) vs JetStream (Durable) Delivery Modes',
      'Message Payload Envelope & Metadata Headers',
      'Built-in JetStream Deduplication via Nats-Msg-Id',
      'Simulated Task Failure parameters for testing retries',
    ],
    demoUsage:
      'Allows submitting jobs with configurable payloads and delivery modes. Features a dedicated Deduplication section to publish first-time and identical duplicate messages to test deduplication windows.',
    trivia:
      'If a duplicate message ID is published within the configured 2-minute window, JetStream stores it zero times and acknowledges it with Duplicate: true.',
  },

  'consumer-lab': {
    id: 'consumer-lab',
    title: 'Consumer Lab',
    role: 'Interactive JetStream Consumer Controller',
    concepts: [
      'Durable Consumers (state survives client disconnects)',
      'Ephemeral Consumers (temporary lifecycle bound to connection)',
      'Competing Consumers (multiple workers pulling from same consumer)',
      'Ordered Consumers (strict sequential stream-to-delivery mapping)',
      'At-Least-Once Delivery (ACK, NAK, and Redeliveries)',
    ],
    demoUsage:
      'Lets you configure the processor consumer dynamically: switch between Durable and Ephemeral, adjust active worker pool between 1 and 2, and switch between Normal and Ordered delivery.',
    trivia:
      'When Ordered is selected, worker count is automatically locked to 1 because concurrent workers execute at varying speeds, which could cause processing completion order to drift.',
  },

  'request-reply': {
    id: 'request-reply',
    title: 'Request / Reply',
    role: 'Synchronous Two-Way Service Validation',
    concepts: [
      'NATS Request / Reply Pattern (RequestMsg)',
      'Dynamic Inbox Subjects (_INBOX.hostname.random.id)',
      'Direct Point-to-Point Reply Routing without Broker Queues',
      'Client-Side Timeout Handling (nats.ErrTimeout)',
    ],
    demoUsage:
      'Sends job validation requests to subject jobs.validate. The processor validates the payload and replies directly to the ephemeral inbox subject within the timeout window.',
    trivia:
      'NATS Request/Reply is purely subject-based: the publisher creates an ephemeral _INBOX subscription, includes it in the Reply header, and the respondent publishes directly back to it.',
  },

  'jetstream-replay': {
    id: 'jetstream-replay',
    title: 'JetStream Replay',
    role: 'Historical Stream Message Replayer',
    concepts: [
      'Replay Policy (Instant vs Original timing)',
      'Start Sequence / Start Time rewinds',
      'Non-Destructive Historical Auditing',
      'Stream Message Inspection',
    ],
    demoUsage:
      'Creates an ephemeral replay consumer starting from a past sequence or relative timestamp to replay previously stored stream messages without re-executing job side effects.',
    trivia:
      'Replaying messages from JetStream does not delete or alter original stream messages; multiple consumers can inspect the exact same historical stream independently.',
  },

  'activity-log': {
    id: 'activity-log',
    title: 'Activity Log',
    role: 'Live Event Observability and Lifecycle Tracing',
    concepts: [
      'Wildcard Event Subscriptions (jobs.>)',
      'Correlation IDs (X-Correlation-Id tracing)',
      'Lifecycle Event States (PUBLISHED, RECEIVED, REDELIVERED, ACKED, COMPLETED)',
      'Worker Attribution (processor-1 vs processor-2)',
    ],
    demoUsage:
      'Displays a real-time table of all lifecycle events published across the system, showing event type, sequence number, subject, delivery count, and worker assignment.',
    trivia:
      'By subscribing to the wildcard jobs.>, the observability panel captures events emitted by the Demo Service, NATS JetStream, and multiple processor workers in a unified stream.',
  },

  'job-details': {
    id: 'job-details',
    title: 'Job Details Inspector',
    role: 'Deep-Dive Job Lifecycle Inspector',
    concepts: [
      'State Machine Transitions (PENDING -> RECEIVED -> PROCESSING -> COMPLETED)',
      'Delivery Attempt Tracking (Attempt Count & Delivery Count)',
      'Payload and Header Inspection',
      'Chronological Timeline Audit',
    ],
    demoUsage:
      'Opens when clicking any job in the Activity Log or Job History. Shows complete payload data, error messages on failure, redelivery flags, and a step-by-step audit log.',
    trivia:
      'Each job transition preserves the original correlation ID, enabling end-to-end tracing from initial HTTP submission to final worker acknowledgement.',
  },

  'subject-addressing': {
    id: 'subject-addressing',
    title: 'NATS Subject Addressing',
    role: 'Subject Hierarchy and Wildcard Explorer',
    concepts: [
      'Token-based Subject Namespaces (dot-delimited tokens)',
      'Single-Token Wildcard (*) matches exactly one token',
      'Full Wildcard (>) matches one or more tokens at the end',
      'Exact vs Pattern Subscription Matching',
    ],
    demoUsage:
      'Shows all active NATS subject subscriptions in the demo and maps live incoming events to the specific subject and wildcards that matched them.',
    trivia:
      'In NATS, subject routing is handled entirely in memory with radical efficiency; wildcards are evaluated at line rate without regex overhead.',
  },

  'metrics-observability': {
    id: 'metrics-observability',
    title: 'Metrics Observability',
    role: 'OpenTelemetry Application Metrics & Prometheus NATS Exporter',
    concepts: [
      'OpenTelemetry (OTel) OTLP Metrics Export',
      'NATS Prometheus Exporter (-connz, -subz, -varz, -jsz)',
      'Grafana OTEL-LGTM Stack (Collector + Prometheus + Grafana)',
      'Low-cardinality metric dimensions (delivery_mode, worker, status)',
      'Real-time rate, counter, and latency percentile tracking (p50/p95)',
    ],
    demoUsage:
      'Application metrics (job counts, validation RPCs, latencies, redeliveries) are exported from demo-service and processor-service via OpenTelemetry OTLP to the local Grafana OTEL-LGTM stack. NATS Server and JetStream infrastructure metrics are scraped by Prometheus via the NATS Prometheus Exporter.',
    trivia:
      'Following best practices, metric labels use low-cardinality dimensions (e.g. delivery_mode, worker, status) rather than high-cardinality values like job_id or correlation_id, ensuring optimal Prometheus query performance.',
  },
};
