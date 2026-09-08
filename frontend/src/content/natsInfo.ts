export interface InfoQnA {
  question: string;
  answer: string;
  formula?: string;
}

export interface NatsComponentInfo {
  id: string;
  title: string;
  role: string;
  concepts: string[];
  demoUsage: string;
  trivia?: string;
  diagram?: string;
  qna?: InfoQnA[];
}

export const NATS_COMPONENTS_INFO: Record<string, NatsComponentInfo> = {
  // ============================================================
  // TOPOLOGY & ARCHITECTURAL COMPONENTS
  // ============================================================

  "react-ui": {
    id: "react-ui",
    title: "React UI",
    role: "Web interface for interacting with the NATS demo and observing its runtime behavior.",
    concepts: [
      "Frontend: Browser-based UI used to submit jobs, control demos, and inspect NATS behavior.",
      "Demo Control API: The UI communicates with backend demo services over HTTP rather than connecting directly to NATS.",
      "Observability: Displays message activity, consumer state, worker behavior, and monitoring information.",
    ],
    demoUsage:
      "Use the UI to trigger NATS operations, change demo configuration, observe message flow, and inspect runtime state.",
  },

  "job-service": {
    id: "job-service",
    title: "Job Service",
    role: "Business service that publishes job messages and performs request/reply operations over NATS.",
    concepts: [
      "NATS Client: Application uses the NATS Go client (nats.go) to connect and communicate with NATS.",
      "Publisher: Sends messages to NATS subjects without knowing which consumers will process them.",
      "Request / Reply: Sends a request to a subject and receives the response through a reply subject.",
    ],
    demoUsage:
      "Submit jobs through the REST API and observe the Job Service publish messages to NATS or perform synchronous validation through request/reply.",
  },

  "demo-control-service": {
    id: "demo-control-service",
    title: "Demo Control Service",
    role: "Backend service that controls the demo and observes NATS activity without being part of the business processing path.",
    concepts: [
      "Observer: Subscribes to NATS subjects to observe activity without becoming the business consumer.",
      "Wildcard Subscription: Uses subjects such as jobs.> to observe multiple related subjects.",
      "Consumer Management: Creates and manages JetStream consumers for replay and inspection scenarios.",
      "Worker Control: Controls demo workers so failure, retry, and load-balancing scenarios can be reproduced.",
    ],
    demoUsage:
      "Use this service through the UI to control workers, run replay scenarios, and populate the live activity view.",
  },

  "demo-service": {
    id: "demo-control-service",
    title: "Demo Control Service",
    role: "Backend service that controls the demo and observes NATS activity without being part of the business processing path.",
    concepts: [
      "Observer: Subscribes to NATS subjects to observe activity without becoming the business consumer.",
      "Wildcard Subscription: Uses subjects such as jobs.> to observe multiple related subjects.",
      "Consumer Management: Creates and manages JetStream consumers for replay and inspection scenarios.",
      "Worker Control: Controls demo workers so failure, retry, and load-balancing scenarios can be reproduced.",
    ],
    demoUsage:
      "Use this service through the UI to control workers, run replay scenarios, and populate the live activity view.",
  },

  "nats-server": {
    id: "nats-server",
    title: "NATS Server",
    role: "Messaging server providing Core NATS for lightweight messaging and JetStream for persistent messaging.",
    concepts: [
      "Core NATS: In-memory publish/subscribe messaging with at-most-once delivery.",
      "JetStream: Persistence and streaming layer providing durable messages, consumers, acknowledgements, redelivery, and replay.",
      "Subject: Address used to route messages, such as jobs.submitted or jobs.queue.",
      "Publish / Subscribe: Publishers send messages to subjects and subscribers receive matching messages.",
      "Request / Reply: Requester publishes a request and receives the response through a reply subject.",
      "Stream: JetStream resource that stores messages published to configured subjects.",
      "Consumer: Stateful JetStream resource that controls message delivery and tracks acknowledgement/delivery state.",
    ],
    demoUsage:
      "The demo uses one NATS Server for Core NATS messaging and JetStream persistence, allowing transient and durable messaging patterns to be compared on the same platform.",
  },

  "jobs-stream": {
    id: "jobs-stream",
    title: "JOBS Stream",
    role: "JetStream Stream that persistently stores messages matching its configured subjects.",
    concepts: [
      "Stream: Persistent JetStream resource containing messages published to configured subjects.",
      "Subject Capture: Defines which subjects are stored by the Stream, such as jobs.submitted.",
      "Stream Sequence: Sequence number assigned to stored messages and used for positioning and replay.",
      "Retention Policy: Determines when messages are removed from the Stream according to the configured retention model.",
      "Persistence: Stored messages remain available after publishers or consumers disconnect, subject to retention and storage limits.",
    ],
    demoUsage:
      "Publish jobs, stop or pause workers, and inspect the JOBS Stream to demonstrate that persisted messages remain available for later consumption.",
  },

  "jetstream-engine": {
    id: "jetstream-engine",
    title: "Stage 2: NATS View & CLI (Stream Storage & Consumer Cursors)",
    role: "Central NATS JetStream broker engine managing persistent stream storage and tracking stateful consumer cursors.",
    diagram: [
      "Stream Log:      (pruned: 1, 2, 3) [ 4 ] [ 5 ] [ 6 ] [ 7 ] [ 8 ] [ 9 ] [ 10 ]",
      "First Sequence:  -----------------> ( 4 ) [Oldest Available]",
      "AckFloor:        -------------------------> ( 5 ) [Safety Watermark]",
      "Delivered Seq:   ---------------------------------------------> ( 8 ) [Handed to Workers]",
      "Last Sequence:   ---------------------------------------------------------> ( 10 ) [Latest Appended]",
      "",
      "Retained Storage Window: [ First Sequence (4)  to  Last Sequence (10) ] = 7 messages",
    ].join("\n"),
    qna: [
      {
        question: "1. What messages are currently retained in the stream?",
        answer: "First Sequence to Last Sequence. Shown in 'nats stream state JOBS' (or stream info). Represents the actual messages physically stored on disk. If older messages (1 to 3) were purged or expired via TTL/MaxAge/MaxMsgs, First Sequence advances. Only messages from First Sequence (4) to Last Sequence (10) are retained.",
        formula: "Retained Window = [First Sequence ... Last Sequence] (Messages count: 7)",
      },
      {
        question: "2. What is the total count / latest message published to the stream?",
        answer: "Stream Last Sequence (last_seq = 10). Represents the highest sequence number assigned to a published message in this stream's history.",
        formula: "Latest Appended = Last Sequence (10)",
      },
      {
        question: "3. What messages are guaranteed to be processed and completed?",
        answer: "Ack Floor (5). The safety watermark below which every single message is 100% completed and acknowledged. Even if all workers crash, messages up through AckFloor will never be redelivered.",
        formula: "Guaranteed Safe = 1 to AckFloor (5)",
      },
      {
        question: "4. What messages are buffered in the stream waiting to be pulled?",
        answer: "Stream Backlog. Messages that have been stored in the stream but have not yet been handed to any worker.",
        formula: "Stream Backlog = Last Sequence - Delivered Seq = 10 - 8 = 2 messages (9, 10)",
      },
      {
        question: "5. What is the total number of in-flight messages?",
        answer: "Outstanding ACKs (Ack Pending). Messages actively pulled by workers that have not yet sent an ACK back to the broker. Here, messages 6, 7, and 8 are currently being processed.",
        formula: "In-Flight = Delivered - Acked = 3 messages (in-flight: 6, 7, 8)",
      },
      {
        question: "6. What happens if ACKs arrive out of order?",
        answer: "NATS enforces the Contiguous Ack Floor Rule. If message 7 is ACKed before message 6 finishes, AckFloor remains at 5. NATS tracks 7 as an individual bit in its pending bitset. Once 6 is ACKed, the AckFloor leaps directly to 7.",
        formula: "AckFloor = Highest Contiguous Acknowledged Sequence",
      },
    ],
    concepts: [
      "Stream Retention: First Sequence marks the oldest available message. Once messages expire or are purged, First Sequence advances.",
      "Stream Persistence: App-side messages are committed to an immutable append-only log on disk before publishing acknowledges.",
      "Stateless Workers vs Stateful Cursors: Workers hold zero cluster state; NATS broker tracks cursors (Delivered, AckFloor, and In-Flight bitset).",
      "Competing Pull Consumers: Multiple workers pull from the shared consumer cursor without partition locking.",
      "Out-of-Order Safety: The AckFloor only advances when the earliest outstanding message in the sequence is acknowledged.",
    ],
    demoUsage:
      "Run 'nats stream state JOBS' to inspect First Sequence and Last Sequence, and 'nats consumer info JOBS job-processor' to observe Delivered Seq, AckFloor, and Outstanding ACKs in real-time.",
    trivia:
      "Unlike partition-bound streaming systems where a slow message stalls an entire partition queue, NATS JetStream tracks individual message ACK timeouts and out-of-order completions per-message with zero rebalance pauses.",
  },

  "dead-letter-queue": {
    id: "dead-letter-queue",
    title: "Dead Letter Queue (JOBS_DLQ)",
    role: "Application-level pattern for isolating messages that cannot be successfully processed after repeated delivery attempts.",
    concepts: [
      "Dead Letter Queue (DLQ): Destination where messages requiring manual investigation or special handling are moved.",
      "MaxDeliver: JetStream consumer setting that limits the number of delivery attempts for a message.",
      "Poison Message: Message that repeatedly fails processing and should no longer continue through normal processing.",
      "DLQ Routing: Application republishes the failed message to a dedicated DLQ subject/Stream after retry exhaustion.",
      "Original Message: Application acknowledges the original message after successfully routing it to the DLQ.",
    ],
    demoUsage:
      "Force a job to fail repeatedly, reach the configured delivery limit, and observe it being routed to jobs.dlq in the JOBS_DLQ Stream.",
    trivia:
      "A DLQ is an architectural pattern rather than a separate NATS messaging primitive. JetStream Streams and Consumers provide the building blocks for implementing it.",
  },

  "jobs-dlq": {
    id: "dead-letter-queue",
    title: "Dead Letter Queue (JOBS_DLQ)",
    role: "Application-level pattern for isolating messages that cannot be successfully processed after repeated delivery attempts.",
    concepts: [
      "Dead Letter Queue (DLQ): Destination where messages requiring manual investigation or special handling are moved.",
      "MaxDeliver: JetStream consumer setting that limits the number of delivery attempts for a message.",
      "Poison Message: Message that repeatedly fails processing and should no longer continue through normal processing.",
      "DLQ Routing: Application republishes the failed message to a dedicated DLQ subject/Stream after retry exhaustion.",
      "Original Message: Application acknowledges the original message after successfully routing it to the DLQ.",
    ],
    demoUsage:
      "Force a job to fail repeatedly, reach the configured delivery limit, and observe it being routed to jobs.dlq in the JOBS_DLQ Stream.",
    trivia:
      "A DLQ is an architectural pattern rather than a separate NATS messaging primitive. JetStream Streams and Consumers provide the building blocks for implementing it.",
  },

  "consumer": {
    id: "consumer",
    title: "JetStream Consumer",
    role: "Stateful JetStream resource that defines how messages are delivered from a Stream and tracks consumer progress.",
    concepts: [
      "Consumer: Stateful view over a Stream that tracks delivery position, acknowledgements, and redelivery state.",
      "Durable Consumer: Consumer whose identity and state are retained for reuse after a client disconnects.",
      "Ephemeral Consumer: Temporary consumer intended for short-lived consumption; its state is not retained as a durable consumer.",
      "Pull Consumer: Worker explicitly requests messages, optionally in batches. Useful for worker pools and controlled concurrency.",
      "Push Consumer: JetStream delivers messages to a configured subscriber subject.",
      "ACK: Consumer confirms successful processing of a delivered message.",
      "NAK: Consumer explicitly indicates processing failure and requests redelivery.",
      "AckWait: Time JetStream waits for an ACK after delivery before the message becomes eligible for redelivery.",
      "MaxAckPending: Maximum number of delivered but unacknowledged messages allowed for the consumer; provides backpressure.",
      "Deliver Policy: Defines where consumption starts, such as all, new, last, a sequence, or a time.",
    ],
    demoUsage:
      "Configure the job-processor consumer and observe how durability, pull delivery, acknowledgements, retries, and consumer state affect processing.",
  },

  "processor-service": {
    id: "processor-service",
    title: "Processor Service",
    role: "Worker application that consumes NATS messages and performs the actual job processing.",
    concepts: [
      "Worker: Application execution unit responsible for processing messages.",
      "Worker Pool: Multiple workers processing jobs concurrently.",
      "Competing Consumers: Multiple workers share available messages so each message is processed by one available worker.",
      "ACK: Worker confirms successful processing to JetStream.",
      "NAK: Worker reports processing failure and requests redelivery.",
      "Redelivery: JetStream delivers an unacknowledged or negatively acknowledged message again according to consumer configuration.",
    ],
    demoUsage:
      "Run multiple processor workers, pause or fail workers, and observe distribution, acknowledgement, retry, and redelivery behavior.",
  },

  // ============================================================
  // DASHBOARD / CAPABILITY PANELS
  // ============================================================

  "platform-status": {
    id: "platform-status",
    title: "Platform Status",
    role: "Runtime status view for NATS, JetStream, and the demo services.",
    concepts: [
      "NATS Connectivity: Indicates whether application clients can connect to the NATS Server.",
      "JetStream Health: Indicates availability of the JetStream subsystem used by Streams and Consumers.",
      "Service Health: Shows availability of the demo services participating in the messaging flow.",
      "Processing Control: Allows demo worker processing to be paused or resumed.",
    ],
    demoUsage:
      "Use this panel to verify the demo environment before running messaging scenarios and to intentionally pause processing for durability and retry demonstrations.",
  },

  "submit-job": {
    id: "submit-job",
    title: "Pub / Sub & Stream Publishing",
    role: "Demonstrates the difference between transient Core NATS publishing and persistent JetStream publishing.",
    concepts: [
      "Core NATS Publish: Message is delivered to currently matching subscribers and is not persisted by Core NATS.",
      "JetStream Publish: Message is stored in a matching Stream and receives a publish acknowledgement (PubAck).",
      "Subject: Message destination used by both Core NATS and JetStream.",
      "At-Most-Once: Core NATS provides best-effort delivery without persistent message storage or consumer redelivery.",
      "At-Least-Once: JetStream consumers can redeliver messages that are not successfully acknowledged.",
      "Message ID: Nats-Msg-Id can be used by JetStream for publish deduplication.",
    ],
    demoUsage:
      "Publish the same job through Core NATS and JetStream, then pause workers to observe the difference between transient delivery and persistent storage.",
  },

  "message-deduplication": {
    id: "message-deduplication",
    title: "Message Deduplication",
    role: "JetStream protection against storing duplicate publishes with the same message ID within the configured duplicate window.",
    concepts: [
      "Nats-Msg-Id: Publisher-supplied identifier used by JetStream to detect duplicate publishes.",
      "Duplicate Window: Time window during which JetStream remembers message IDs for deduplication.",
      "Duplicate Publish: A publish using an already-seen message ID within the window is treated as a duplicate.",
      "PubAck: JetStream publish acknowledgement indicates whether the publish was accepted as a new message or identified as a duplicate.",
      "Not Exactly-Once Processing: Publish deduplication does not guarantee exactly-once business processing; idempotent application handling may still be required.",
    ],
    demoUsage:
      "Publish two messages with the same Nats-Msg-Id and observe that JetStream detects the duplicate instead of storing another copy.",
  },

  "request-reply": {
    id: "request-reply",
    title: "Request / Reply",
    role: "Synchronous communication pattern built directly into NATS subject-based messaging.",
    concepts: [
      "Request: Client publishes a request to a service subject and provides a reply subject.",
      "Reply: Responder publishes the result to the supplied reply subject.",
      "Inbox (_INBOX.*): NATS convention for temporary reply subjects used by requesters.",
      "Timeout: Requester waits for a configured period; no response results in a timeout.",
      "Synchronous Semantics: Caller waits for the response even though the underlying transport is asynchronous messaging.",
    ],
    demoUsage:
      "Send a validation request to jobs.validate and observe the response. Pause the responder to demonstrate requester timeout behavior.",
  },

  "jetstream-replay": {
    id: "jetstream-replay",
    title: "Stream Replay",
    role: "Reads previously stored JetStream messages again by starting a consumer from a historical stream position.",
    concepts: [
      "Replay: Re-consumption of messages already stored in a Stream.",
      "Deliver Policy: Determines the starting position, including DeliverByStartSequence and DeliverByStartTime.",
      "Stream Sequence: Historical position assigned to each stored message.",
      "Replay Policy: Controls delivery timing. Instant replays as quickly as possible; Original attempts to preserve original message timing.",
      "Ephemeral Replay Consumer: Temporary consumer can be created specifically for historical replay without changing the existing processing consumer.",
    ],
    demoUsage:
      "Select a historical sequence or time and replay stored JOBS messages using a temporary consumer while leaving the active processing consumer unchanged.",
  },

  "activity-log": {
    id: "activity-log",
    title: "Activity Log",
    role: "Application-level view of message lifecycle events observed from NATS subjects.",
    concepts: [
      "Event Observation: Monitoring subscribers can observe matching NATS subjects without becoming the business consumer.",
      "Subject Wildcard: jobs.> matches the jobs subject hierarchy and allows related events to be observed together.",
      "Lifecycle Events: Shows application-published events such as submitted, delivered, processed, completed, failed, and DLQ transitions.",
      "Delivery Metadata: Events can include identifiers, timestamps, and sequence information useful for correlating processing activity.",
    ],
    demoUsage:
      "Watch jobs.> activity in real time and follow individual jobs as they move through publishing, processing, retry, and completion.",
  },

  "job-details": {
    id: "job-details",
    title: "Job Details Inspector",
    role: "Detailed view of a selected job's application payload, processing state, and delivery history.",
    concepts: [
      "Message Metadata: Subject, identifiers, timestamps, and other message attributes.",
      "Delivery Attempts: Number of observed processing attempts, useful for identifying retries and failures.",
      "Processing Timeline: Chronological application events associated with the job.",
      "Payload Inspection: Displays the message payload used by the demo.",
      "Trace Context: If the application is OpenTelemetry-instrumented, trace identifiers can correlate the job with application traces.",
    ],
    demoUsage:
      "Select a job from the Activity Log to inspect its payload, lifecycle, delivery attempts, and associated application telemetry.",
  },

  "subject-addressing": {
    id: "subject-addressing",
    title: "NATS Subject Addressing",
    role: "Explorer for NATS subjects, hierarchical tokens, and wildcard matching.",
    concepts: [
      "Subject: Dot-separated address used for routing NATS messages, for example jobs.submitted.",
      "Token: Individual section of a subject separated by a dot. jobs.submitted contains two tokens.",
      "Exact Match: Subscription receives only the specified subject.",
      "Single-Token Wildcard (*): Matches exactly one subject token. jobs.* matches jobs.submitted but not jobs.order.created.",
      "Multi-Token Wildcard (>): Matches one or more remaining tokens. jobs.> matches jobs.submitted and jobs.order.created.",
      "Subject-Based Routing: Publishers do not need to know which subscribers or services will receive a message.",
    ],
    demoUsage:
      "Publish subjects and observe which exact or wildcard subscriptions receive each message.",
  },

  "metrics-observability": {
    id: "metrics-observability",
    title: "Metrics & Observability",
    role: "NATS-centric observability view covering server metrics, logs, system events, and application telemetry integration.",
    concepts: [
      "NATS Metrics: Server, connection, subscription, cluster, account, and JetStream operational information can be exposed through NATS monitoring interfaces.",
      "Prometheus Exporter: nats-prometheus-exporter collects NATS monitoring information and exposes it as Prometheus metrics.",
      "NATS Logs: Server logs can be collected by Fluent Bit or OpenTelemetry Collector and sent to Loki.",
      "$SYS: NATS system subject namespace used for system-level requests, responses, and events.",
      "JetStream Advisories: Structured events published under JetStream advisory subjects provide information about JetStream operations and activity.",
      "Tracing: NATS Server is not a native OTLP span producer. Application OpenTelemetry instrumentation can create traces around operations that use NATS.",
    ],
    demoUsage:
      "Use the LGTM stack to view NATS metrics in Prometheus/Grafana, NATS logs in Loki, and selected NATS system/JetStream events through the event pipeline.",
  },

  "queue-groups": {
    id: "queue-groups",
    title: "Core NATS Queue Groups",
    role: "Core NATS mechanism for load-balancing messages across multiple subscribers in the same queue group.",
    concepts: [
      "Subject: jobs.queue is the message destination to which the publisher sends jobs.",
      "Queue Group: job-workers is a shared group name supplied when subscribers join the queue.",
      "Competing Consumers: Subscribers in the same queue group compete for messages; a given message is delivered to one eligible subscriber in that group.",
      "1-of-N Delivery: With N workers in the same queue group, each message is delivered to one worker rather than all N workers.",
      "At-Most-Once: Core NATS does not persist messages or provide ACK/NAK-based redelivery semantics.",
      "Multi-Group Fanout: Multiple distinct queue groups can subscribe to the same subject. Each group receives its own delivery while load-balancing within that group.",
      "Dynamic Membership: Subscribers can join or leave a queue group while the system is running.",
    ],
    demoUsage:
      "Publish jobs to jobs.queue and observe messages distributed across processor workers sharing the job-workers queue group. Add or remove workers to observe changing distribution.",
    trivia:
      "Core NATS queue groups are useful for lightweight work distribution where transient, at-most-once delivery is acceptable. Use JetStream when persistence, acknowledgements, or redelivery are required.",
  },

  "delayed-retry-delivery": {
    id: "delayed-retry-delivery",
    title: "Delayed & Retry Delivery",
    role: "Demonstrates explicit retry delay, acknowledgement timeout, and application-level scheduled publishing.",
    concepts: [
      "NAK with Delay: Worker explicitly rejects a message and requests redelivery after a specified delay. Used for controlled retry/backoff.",
      "AckWait: Timer starts after message delivery. If the consumer does not receive an ACK within AckWait, the message becomes eligible for redelivery.",
      "Scheduled Delivery: JetStream does not provide a general-purpose native 'deliver at timestamp' scheduler; delayed initial publishing can be implemented using an application scheduler/timer.",
      "Retry vs Scheduling: NAK with Delay and AckWait concern messages that have already been delivered; scheduling delays when a message is published or released for consumption.",
    ],
    demoUsage:
      "Run three scenarios: NAK a message with a delay, intentionally omit an ACK until AckWait expires, and use an application timer to publish a message at a future time.",
    trivia:
      "NAK with Delay is useful for transient failures such as rate limits or temporary downstream outages because retry timing can be controlled explicitly.",
  },
};