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

  'react-ui': {
    id: 'react-ui',
    title: 'React UI',
    role: 'Developer demonstration dashboard providing interactive controls and topology visualization.',
    concepts: [
      'Decoupled Frontend Architecture',
      'HTTP API Client (Fetch API)',
      'Real-Time Polling & State Synchronization',
      'Zero Direct NATS Coupling',
    ],
    demoUsage:
      'Runs locally on port 5173. Calls Demo Control Service (:8080) for UI state, activity logs, replay, and worker toggles, and Job Service (:8081) for business job submissions.',
    trivia:
      'In a well-designed NATS architecture, web browsers do not need direct NATS credentials or socket connections when HTTP gateway services provide clean boundary abstraction.',
  },

  'job-service': {
    id: 'job-service',
    title: 'Job Service',
    role: 'Pure business microservice handling job submissions and domain validation.',
    concepts: [
      'Clean Business Architecture',
      'NATS Go Client (nats.go)',
      'Publish / Subscribe (jobs.submitted)',
      'Request / Reply (jobs.validate)',
      'OpenTelemetry W3C Trace Context Injection',
    ],
    demoUsage:
      'Exposes standard HTTP endpoints (POST /jobs, POST /jobs/validate) on port 8081. Decoupled from demo UI concerns - only executes business logic and communicates via NATS.',
    trivia:
      'A pure business service publishes events and lets interested parties (like consumers, audit loggers, or the demo harness) react asynchronously through subjects.',
  },

  'demo-control-service': {
    id: 'demo-control-service',
    title: 'Demo Control Service',
    role: 'Dedicated UI gateway and demo controller for the demonstration dashboard.',
    concepts: [
      'Separation of Concerns (Decoupled Demo vs Business Logic)',
      'NATS Wildcard Passive Observation (jobs.>)',
      'Ephemeral Replay Consumer Management',
      'Remote Worker State & Configuration Control',
    ],
    demoUsage:
      'Passively observes NATS events to populate the live activity log, powers the wildcard subject addressing comparison, triggers JetStream message replay, and relays consumer configuration commands without polluting the production job-service.',
    trivia:
      'Because NATS is an open publish/subscribe broker, any authorized service can attach a wildcard subscription (jobs.>) to observe events in real time without the publisher or consumer ever knowing or modifying their code.',
  },

  // Backward compatibility alias
  'demo-service': {
    id: 'demo-control-service',
    title: 'Demo Control Service',
    role: 'Dedicated UI gateway and demo controller for the demonstration dashboard.',
    concepts: [
      'Separation of Concerns',
      'NATS Wildcard Observation',
      'UI Dashboard Gateway',
    ],
    demoUsage: 'Serves the React dashboard and provides activity taps into NATS.',
  },

  'nats-server': {
    id: 'nats-server',
    title: 'NATS Server',
    role: 'Deployed messaging server that provides NATS messaging capabilities.',
    concepts: [
      'Core NATS',
      'JetStream',
      'Subject-based Routing',
      'Publish / Subscribe',
      'Request / Reply',
      'Streams and Consumers',
    ],
    demoUsage:
      'The demo runs a NATS Server that handles Core NATS messaging and hosts the JetStream resources used by the job-processing flow.',
    trivia:
      'Core NATS provides lightweight messaging, while JetStream adds persistence, replay, acknowledgements, and durable consumption. Streams and Consumers are logical resources managed by the NATS Server rather than separate deployments.',
  },

  'jobs-stream': {
    id: 'jobs-stream',
    title: 'JOBS Stream',
    role: 'Logical JetStream resource that persistently stores messages.',
    concepts: [
      'JetStream Stream',
      'Message Persistence',
      'Subject Capture',
      'Stream Sequence Numbers',
      'Message IDs and Deduplication',
      'Retention Policies',
    ],
    demoUsage:
      'Stores job submission messages published on jobs.submitted so they remain available for JetStream consumers even when processors are temporarily unavailable.',
    trivia:
      'A Stream is a persistent message store managed by JetStream. A stream can capture one or many subjects, and every stored message receives a monotonically increasing stream sequence number.',
  },

  'dead-letter-queue': {
    id: 'dead-letter-queue',
    title: 'Dead Letter Queue (JOBS_DLQ)',
    role: 'Persistent JetStream stream used to store messages that have exhausted their processing attempts.',
    concepts: [
      'Dead Letter Queue Pattern',
      'Max Delivery Attempts',
      'Redelivery',
      'JetStream Stream',
      'Failed Message Isolation',
    ],
    demoUsage:
      'Stores jobs that fail processing after the configured maximum delivery attempts.',
    trivia:
      'NATS does not require a DLQ to be a special server component. A DLQ can be implemented using a separate JetStream Stream and application-controlled failure routing.',
  },

  'jobs-dlq': {
    id: 'dead-letter-queue',
    title: 'Dead Letter Queue (JOBS_DLQ)',
    role: 'Persistent JetStream stream used to store messages that have exhausted their processing attempts.',
    concepts: [
      'Dead Letter Queue Pattern',
      'Max Delivery Attempts',
      'Redelivery',
      'JetStream Stream',
      'Failed Message Isolation',
    ],
    demoUsage:
      'Stores jobs that fail processing after the configured maximum delivery attempts.',
    trivia:
      'NATS does not require a DLQ to be a special server component. A DLQ can be implemented using a separate JetStream Stream and application-controlled failure routing.',
  },

  'consumer': {
    id: 'consumer',
    title: 'JetStream Consumer',
    role: 'Logical JetStream resource that controls and tracks message delivery from a Stream.',
    concepts: [
      'JetStream Consumer',
      'Durable vs Ephemeral Consumers',
      'Pull vs Push Consumption',
      'ACK / NAK',
      'At-Least-Once Delivery',
      'Redelivery',
      'Pending and Ack Pending',
      'Normal vs Ordered Consumers',
    ],
    demoUsage:
      'The job-processor consumer delivers messages from the JOBS Stream to Processor Service workers and tracks delivery, acknowledgements, pending messages, and redeliveries.',
    trivia:
      'A durable consumer keeps its delivery state so consumption can continue after a client disconnects. An ephemeral consumer is intended for temporary consumption and does not provide the same durable consumer lifecycle.',
  },

  'processor-service': {
    id: 'processor-service',
    title: 'Processor Service',
    role: 'Deployed application service that processes jobs received from NATS.',
    concepts: [
      'NATS Client Consumer',
      'Worker Pool',
      'Competing Consumers',
      'Message Acknowledgement',
      'At-Least-Once Processing',
      'Redelivery on Failure',
    ],
    demoUsage:
      'Runs one or more workers that pull jobs from the JetStream consumer, execute the simulated task, acknowledge successful processing, and NAK failed processing for redelivery.',
    trivia:
      'Multiple application workers can consume from the same JetStream consumer. This allows workers to share the workload while the consumer maintains the delivery state.',
  },

  // Dashboard Sections

  'platform-status': {
    id: 'platform-status',
    title: 'Platform Status',
    role: 'Real-time view of NATS connectivity and processing state.',
    concepts: [
      'NATS Connection State',
      'Request / Reply Health Check',
      'JetStream Stream and Consumer State',
      'Consumer Backlog',
    ],
    demoUsage:
      'Shows NATS connectivity, Processor Service availability, JetStream state, pending messages, and provides the control for pausing or resuming job processing.',
    trivia:
      'JetStream keeps stored messages available even when a consumer temporarily stops processing them. This allows the backlog to build without losing the persisted messages.',
  },

  'submit-job': {
    id: 'submit-job',
    title: 'Pub / Sub',
    role: 'Interactive job publisher for demonstrating NATS messaging and delivery behaviour.',
    concepts: [
      'Core NATS Publish / Subscribe',
      'JetStream Publishing',
      'Transient vs Persistent Messaging',
      'NATS Subjects',
      'Message IDs',
      'Duplicate Message Detection',
    ],
    demoUsage:
      'Allows jobs to be published using Core NATS or JetStream and provides controls for generating duplicate messages and simulated processing failures.',
    trivia:
      'Core NATS is transient: messages are delivered to active subscribers but are not persisted for future delivery. JetStream can persist published messages and make them available to consumers later.',
  },

  'message-deduplication': {
    id: 'message-deduplication',
    title: 'Message Deduplication',
    role: 'JetStream provides server-side, Stream-scoped, time-windowed deduplication based on an optional Nats-Msg-Id; it is not a global exactly-once mechanism.',
    concepts: [
      'Stream-Level Feature: Belongs to the Stream, not the Consumer',
      'Deduplication Window: Default 2m (configured via duplicate_window; no universal maximum)',
      'Optional Header: Triggered by Nats-Msg-Id (absent header opts out; payload is never inspected)',
      'Scope & Clustering: Scoped to the Stream (replicated across stream replicas; not global across regions)',
      'Server Overhead: Lightweight in-memory ID tracking and lookup on publish (no second pipeline)',
      'PubAck Response: Duplicate publish returns PubAck { duplicate: true, sequence: originalSeq }',
    ],
    demoUsage:
      'The JOBS stream tracks Nats-Msg-Id within the configured 2-minute window. Publishing an identical ID within 120s causes JetStream to suppress duplicate storage and emit a DEDUPLICATED event without re-delivering to workers.',
    trivia:
      'Architecture Summary (Stream vs Global):\n- Default Window: 2 minutes (server can set maximum limits)\n- Scope: Per-Stream only (follows stream replicas; not global cross-region)\n- Header: Optional (omit Nats-Msg-Id to opt out)\n- Overhead: In-memory sliding window ID lookup; keep window sized to publisher retry needs',
  },

  'consumer-lab': {
    id: 'consumer-lab',
    title: 'Consumer Lab',
    role: 'Interactive controller for experimenting with JetStream Consumer configurations.',
    concepts: [
      'Durable vs Ephemeral Consumers',
      'Pull Consumers',
      'Competing Consumers',
      'Ordered Consumers',
      'ACK / NAK',
      'Redelivery',
      'At-Least-Once Delivery',
    ],
    demoUsage:
      'Allows the Processor Service consumer to be configured as Durable or Ephemeral, changes the number of competing workers, and switches between Normal and Ordered consumption.',
    trivia:
      'Durable and Ephemeral describe the consumer lifecycle, not whether messages are persisted. Message persistence is provided by the Stream; the Consumer maintains delivery and acknowledgement state.',
  },

  'request-reply': {
    id: 'request-reply',
    title: 'Request / Reply',
    role: 'Interactive demonstration of synchronous request-response messaging over NATS.',
    concepts: [
      'NATS Request / Reply',
      'Subjects',
      'Reply Subjects',
      'Inbox Subjects',
      'Timeout Handling',
      'Point-to-Point Response Routing',
    ],
    demoUsage:
      'Sends a validation request to jobs.validate and waits for the Processor Service to return a response within the configured timeout.',
    trivia:
      'NATS Request / Reply is built on the same subject-based messaging model as publish/subscribe. A requester publishes a request with a reply subject, and the responder publishes the response to that subject.',
  },

  'jetstream-replay': {
    id: 'jetstream-replay',
    title: 'JetStream Replay',
    role: 'Interactive view for replaying historical messages stored in a JetStream Stream.',
    concepts: [
      'Message Replay',
      'Replay Policy',
      'Start Sequence',
      'Start Time',
      'Historical Consumption',
      'Multiple Consumers',
    ],
    demoUsage:
      'Creates a temporary replay consumer that starts from a selected stream position or time and displays previously stored messages without modifying the original Stream.',
    trivia:
      'JetStream allows different consumers to read the same stored messages independently. Replaying a message does not remove it from the Stream.',
  },

  'activity-log': {
    id: 'activity-log',
    title: 'Activity Log',
    role: 'Real-time view of job and messaging lifecycle events.',
    concepts: [
      'Event Observation',
      'NATS Subject Subscriptions',
      'Wildcard Subscriptions',
      'W3C Trace Context',
      'Message Delivery Events',
    ],
    demoUsage:
      'Displays lifecycle events generated during job submission and processing, including publish, receive, redelivery, acknowledgement, and completion events.',
    trivia:
      'NATS subjects can be subscribed to using wildcards, allowing an observer to monitor groups of related subjects without subscribing to every subject individually.',
  },

  'job-details': {
    id: 'job-details',
    title: 'Job Details Inspector',
    role: 'Detailed view of an individual job and its processing lifecycle.',
    concepts: [
      'Message Metadata',
      'Delivery Attempts',
      'Acknowledgement State',
      'Distributed Tracing',
      'Processing Lifecycle',
    ],
    demoUsage:
      'Shows the selected job payload, metadata, processing attempts, delivery information, errors, and chronological lifecycle events.',
    trivia:
      'With at-least-once delivery, a message can be delivered more than once when processing does not result in a successful acknowledgement. Delivery information helps identify such retries.',
  },

  'subject-addressing': {
    id: 'subject-addressing',
    title: 'NATS Subject Addressing',
    role: 'Interactive explorer for NATS subject hierarchy and wildcard matching.',
    concepts: [
      'Dot-delimited Subject Tokens',
      'Exact Subject Matching',
      'Single-Token Wildcard (*)',
      'Multi-Token Wildcard (>)',
      'Publish / Subscribe Routing',
    ],
    demoUsage:
      'Shows active subscriptions and demonstrates which exact or wildcard subscriptions match incoming NATS subjects.',
    trivia:
      'NATS subjects are hierarchical names made from dot-separated tokens. The * wildcard matches one token, while > matches one or more trailing tokens.',
  },

  'metrics-observability': {
    id: 'metrics-observability',
    title: 'Metrics Observability',
    role: 'Monitoring view for application and NATS infrastructure metrics.',
    concepts: [
      'OpenTelemetry Metrics',
      'Prometheus Metrics',
      'NATS Server Metrics',
      'JetStream Metrics',
      'Counters, Rates and Latencies',
      'Metric Cardinality',
    ],
    demoUsage:
      'Displays application metrics from the Job Service and Processor Service together with NATS and JetStream infrastructure metrics collected through the observability stack.',
    trivia:
      'Metrics should generally use low-cardinality dimensions such as operation, status, or worker. High-cardinality values such as job IDs are better suited to logs or traces than Prometheus metric labels.',
  },

};
