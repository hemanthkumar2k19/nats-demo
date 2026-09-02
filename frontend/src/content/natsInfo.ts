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

  'job-service': {
    id: 'job-service',
    title: 'Job Service',
    role: 'Application service that accepts jobs and interacts with NATS.',
    concepts: [
      'NATS Go Client (nats.go)',
      'Publish / Subscribe',
      'Request / Reply',
      'NATS Subjects',
      'Core NATS vs JetStream publishing',
    ],
    demoUsage:
      'Receives HTTP requests from the React UI and uses the NATS Go client to publish jobs, send validation requests, and observe job lifecycle events.',
    trivia:
      'A NATS client connects to the NATS Server and uses subjects to publish and subscribe to messages. The application does not need to know where another subscriber is running.',
  },

  // Backward compatibility alias
  'demo-service': {
    id: 'job-service',
    title: 'Job Service',
    role: 'Application service that accepts jobs and interacts with NATS.',
    concepts: [
      'NATS Go Client (nats.go)',
      'Publish / Subscribe',
      'Request / Reply',
      'NATS Subjects',
      'Core NATS vs JetStream publishing',
    ],
    demoUsage:
      'Receives HTTP requests from the React UI and uses the NATS Go client to publish jobs, send validation requests, and observe job lifecycle events.',
    trivia:
      'A NATS client connects to the NATS Server and uses subjects to publish and subscribe to messages. The application does not need to know where another subscriber is running.',
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
      'Correlation IDs',
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
      'Correlation',
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
