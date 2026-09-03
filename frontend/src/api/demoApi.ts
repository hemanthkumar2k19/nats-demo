export interface Job {
  job_id: string;
  type: string;
  payload: Record<string, any>;
  delivery_mode?: string;
  trace_id?: string;
}

export interface JobStatusResponse {
  job_id: string;
  status: string;
  trace_id?: string;
}

export interface Activity {
  timestamp: string;
  job_id: string;
  event: string;
  subject: string;
  worker: string;
  delivery_count: number;
  delivery_mode?: string;
  sequence?: number;
  msg_id?: string;
  job_type?: string;
  trace_id?: string;
}

export interface ServiceStatus {
  name: string;
  status: 'active' | 'connected' | 'disconnected' | 'unknown' | 'running' | 'stopped';
  details: string;
  processing?: boolean;
  workers?: number;
}

export interface JetStreamInfo {
  stream: string;
  messages?: number;
  bytes?: number;
  first_seq?: number;
  last_seq?: number;
  pending: number;
}

export interface SystemStatusResponse {
  services: ServiceStatus[];
  jetstream?: JetStreamInfo;
}

export interface DLQStatus {
  stream: string;
  messages: number;
  bytes: number;
  first_seq: number;
  last_seq: number;
  consumer: string;
  pending: number;
  ack_pending?: number;
}

export interface DLQMessage {
  sequence: number;
  job_id: string;
  type: string;
  original_subject: string;
  delivery_attempts: number;
  failure_reason: string;
  timestamp: string;
  worker?: string;
  payload?: Record<string, any>;
}

const DEMO_CONTROL_URL = 'http://localhost:8080';
const JOB_SERVICE_URL = 'http://localhost:8081';

/**
 * Submits a job to the pure business job-service API.
 * Calls real POST /jobs on :8081.
 */
export async function submitJob(job: Job): Promise<JobStatusResponse> {
  const response = await fetch(`${JOB_SERVICE_URL}/jobs`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(job),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to submit job: ${response.status} ${response.statusText}. ${errorText}`);
  }

  return response.json();
}

export interface ValidationResult {
  valid?: boolean;
  message?: string;
  error?: string;
  timedOut?: boolean;
}

/**
 * Validates a job using Request/Reply on the pure business job-service API.
 * Calls real POST /jobs/validate on :8081.
 * Returns a ValidationResult. On processor timeout (HTTP 504), timedOut is set to true
 * rather than throwing so the UI can display the timeout scenario cleanly.
 */
export async function validateJob(job: Job): Promise<ValidationResult> {
  const response = await fetch(`${JOB_SERVICE_URL}/jobs/validate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(job),
  });

  // 504 means the NATS request timed out - no responder was available.
  // Return a structured result rather than throwing so the UI can display the timeout panel.
  if (response.status === 504) {
    return { timedOut: true, error: 'No response received from processor service' };
  }

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to validate job: ${response.status} ${response.statusText}. ${errorText}`);
  }

  return response.json();
}

/**
 * Retrieves the status/list of services from demo-control-service.
 * Calls real GET /status from demo-control-service on :8080.
 */
export async function getServiceStatus(): Promise<SystemStatusResponse> {
  const response = await fetch(`${DEMO_CONTROL_URL}/status`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch service status: ${response.status} ${response.statusText}. ${errorText}`);
  }

  const data = await response.json();
  const services: ServiceStatus[] = [];

  // 1. Map NATS Server status
  const natsConnected = data.nats?.status === 'CONNECTED';
  services.push({
    name: 'NATS Server (4222)',
    status: natsConnected ? 'connected' : 'disconnected',
    details: natsConnected ? 'Connected to NATS broker' : 'Disconnected from NATS broker',
  });

  // 2. Map other services
  if (Array.isArray(data.services)) {
    data.services.forEach((svc: any) => {
      const isActive = svc.status === 'ACTIVE';
      if (svc.name === 'processor-service') {
        const isProcessing = svc.processing === true;
        services.push({
          name: 'processor-service',
          status: isActive ? (isProcessing ? 'active' : 'stopped') : 'disconnected',
          details: isActive 
            ? (isProcessing ? 'Processor is active and processing messages' : 'Processor is paused')
            : 'Service is offline',
          processing: isProcessing,
          workers: svc.workers,
        });
      } else if (svc.name === 'demo-control-service') {
        services.push({
          name: 'Demo Control (8080)',
          status: isActive ? 'active' : 'disconnected',
          details: isActive ? 'UI control gateway operational' : 'Demo control service is offline',
        });
      } else if (svc.name === 'job-service') {
        services.push({
          name: 'Job Service (8081)',
          status: isActive ? 'active' : 'disconnected',
          details: isActive ? 'Job business API operational' : 'Job service is offline',
        });
      } else {
        services.push({
          name: svc.name,
          status: isActive ? 'active' : 'disconnected',
          details: isActive ? 'Service is operational' : 'Service is offline',
        });
      }
    });
  }

  const jetstream = data.jetstream ? {
    stream: data.jetstream.stream,
    messages: data.jetstream.messages,
    bytes: data.jetstream.bytes,
    first_seq: data.jetstream.first_seq,
    last_seq: data.jetstream.last_seq,
    pending: data.jetstream.pending,
  } : undefined;

  return { services, jetstream };
}

/**
 * Retrieves the list of recent job activities.
 * Calls real GET /activities from the Go backend.
 */
export async function getActivity(): Promise<Activity[]> {
  const response = await fetch(`${DEMO_CONTROL_URL}/activities`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch activity logs: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

export interface JobHistoryItem {
  status: string;
  timestamp: string;
}

export interface JobDetailResponse {
  job_id: string;
  status: string;
  delivery_count: number;
  trace_id?: string;
  type?: string;
  delivery_mode?: string;
  worker?: string;
  history: JobHistoryItem[];
}

export interface ReplayRequest {
  start_sequence?: number;
  end_sequence?: number;
  from_sequence?: number;
  to_sequence?: number;
  start_time?: string;
  end_time?: string;
  from_time?: string;
  to_time?: string;
  replay_mode?: 'instant' | 'original';
  replay_from?: 'sequence' | 'time';
}

export interface ReplayResponse {
  status: string;
  consumer: string;
}

/**
 * Retrieves detailed status of a specific job.
 * Calls GET /jobs/{job_id} from the Go backend.
 */
export async function getJobDetail(jobId: string): Promise<JobDetailResponse> {
  const response = await fetch(`${JOB_SERVICE_URL}/jobs/${jobId}`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch job detail for "${jobId}": ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Triggers a NATS replay stream consumer.
 * Calls POST /jobs/replay from the Go backend.
 */
export async function replayJobs(req: ReplayRequest): Promise<ReplayResponse> {
  const response = await fetch(`${DEMO_CONTROL_URL}/jobs/replay`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to trigger replay: ${response.status} ${response.statusText}. ${errorText}`);
  }

  return response.json();
}

export interface AddressingSubscription {
  name: string;
  subject: string;
}

export interface AddressingEvent {
  job_id: string;
  subject: string;
  received_by: string[];
  timestamp: string;
}

/**
 * Retrieves the active addressing subscriptions from the backend.
 * Calls GET /messaging/subscriptions.
 */
export async function getAddressingSubscriptions(): Promise<AddressingSubscription[]> {
  const response = await fetch(`${DEMO_CONTROL_URL}/messaging/subscriptions`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch addressing subscriptions: ${response.status} ${response.statusText}. ${errorText}`);
  }
  const data = await response.json();
  return data.subscriptions || [];
}

/**
 * Retrieves observed addressing events from the backend.
 * Calls GET /messaging/activity.
 */
export async function getAddressingActivity(): Promise<AddressingEvent[]> {
  const response = await fetch(`${DEMO_CONTROL_URL}/messaging/activity`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch addressing activity: ${response.status} ${response.statusText}. ${errorText}`);
  }
  const data = await response.json();
  return data.events || [];
}

export interface ProcessorStateResponse {
  enabled: boolean;
  status: string;
}

/**
 * Updates the processor state (ON/OFF).
 * Calls PUT /processor/state.
 */
export async function updateProcessorState(enabled: boolean): Promise<ProcessorStateResponse> {
  const response = await fetch(`${DEMO_CONTROL_URL}/processor/state`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ enabled }),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to update processor state: ${response.status} ${response.statusText}. ${errorText}`);
  }

  return response.json();
}

export interface ConsumerConfig {
  type: 'durable' | 'ephemeral';
  workers: number;
  ordering: 'normal' | 'ordered';
}

export interface ConsumerStatus {
  name: string;
  type: string;
  workers: number;
  ordering: string;
  delivery: string;
  status: string;
  pending: number;
  ack_pending: number;
  redelivered: number;
  distribution?: Record<string, number>;
}

/**
 * Retrieves the consumer configuration and live status metrics.
 * Calls GET /consumer.
 */
export async function getConsumerStatus(): Promise<ConsumerStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/consumer`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch consumer status: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Updates the consumer configuration (durable/ephemeral, workers, ordering).
 * Calls PUT /consumer.
 */
export async function updateConsumerConfig(config: ConsumerConfig): Promise<ConsumerStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/consumer`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(config),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to update consumer config: ${response.status} ${response.statusText}. ${errorText}`);
  }

  return response.json();
}

/**
 * Fetches current DLQ stream and consumer status from demo-control-service.
 * Calls GET /dlq/status.
 */
export async function getDLQStatus(): Promise<DLQStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/dlq/status`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch DLQ status: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Fetches all failed messages persisted in the JOBS_DLQ stream.
 * Calls GET /dlq/messages.
 */
export async function getDLQMessages(): Promise<DLQMessage[]> {
  const response = await fetch(`${DEMO_CONTROL_URL}/dlq/messages`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch DLQ messages: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Reprocesses failed messages from JOBS_DLQ back into the active JOBS stream.
 * Calls POST /dlq/reprocess on demo-control-service.
 */
export async function reprocessDLQMessages(jobId?: string): Promise<{ reprocessed: number; jobs: string[]; message: string }> {
  const response = await fetch(`${DEMO_CONTROL_URL}/dlq/reprocess`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(jobId ? { job_id: jobId } : {}),
  });
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to reprocess DLQ messages: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Purges all messages from the JOBS_DLQ stream.
 * Calls POST /dlq/purge on demo-control-service.
 */
export async function purgeDLQ(): Promise<{ purged: boolean; stream: string; message: string }> {
  const response = await fetch(`${DEMO_CONTROL_URL}/dlq/purge`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
  });
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to purge DLQ stream: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

export interface QueueGroupStatus {
  subject: string;
  queue_group: string;
  workers: number;
  distribution: Record<string, number>;
}

export interface QueuePublishResponse {
  published: number;
  subject: string;
  jobs: string[];
}

/**
 * Fetches current Core NATS Queue Group status and message distribution.
 * Calls GET /queue-group on demo-control-service.
 */
export async function getQueueGroupStatus(): Promise<QueueGroupStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/queue-group`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch queue group status: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Updates Core NATS Queue Group active workers (1 or 2).
 * Calls PUT /queue-group on demo-control-service.
 */
export async function updateQueueGroupWorkers(workers: number): Promise<QueueGroupStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/queue-group`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ workers }),
  });
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to update queue group workers: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Publishes a batch of test messages to jobs.queue for Core NATS Queue Group distribution.
 * Calls POST /jobs/queue on job-service.
 */
export async function sendQueueTestMessages(count: number = 10): Promise<QueuePublishResponse> {
  const response = await fetch(`${JOB_SERVICE_URL}/jobs/queue`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ count, type: 'queue-job' }),
  });
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to send queue test messages: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Resets the worker distribution counters for the Core NATS Queue Group.
 * Calls POST /queue-group/reset on demo-control-service.
 */
export async function resetQueueGroupDistribution(): Promise<QueueGroupStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/queue-group/reset`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
  });
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to reset queue group distribution: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Resets the worker distribution counters for the JetStream Consumer.
 * Calls POST /consumer/reset on demo-control-service.
 */
export async function resetConsumerDistribution(): Promise<ConsumerStatus> {
  const response = await fetch(`${DEMO_CONTROL_URL}/consumer/reset`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
  });
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to reset consumer distribution: ${response.status} ${response.statusText}. ${errorText}`);
  }
  return response.json();
}

/**
 * Publishes a batch of test messages to the JOBS stream via Job Service POST /jobs with delivery_mode=JETSTREAM.
 */
export async function sendJetStreamTestMessages(count: number = 10): Promise<{ published: number; jobs: string[] }> {
  try {
    const response = await fetch(`${DEMO_CONTROL_URL}/jobs/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ count, type: 'image-processing' }),
    });
    if (response.ok) {
      return await response.json();
    }
  } catch {
    // Fallback to individual submitJob if endpoint temporarily unavailable
  }

  const jobIds: string[] = [];
  const baseNum = Math.floor(100 + Math.random() * 900);
  for (let i = 0; i < count; i++) {
    const jId = `job-js-${baseNum + i}`;
    jobIds.push(jId);
    await submitJob({
      job_id: jId,
      type: 'image-processing',
      delivery_mode: 'JETSTREAM',
      payload: { file: `img-${baseNum + i}.jpg`, batch: true },
    });
  }
  return { published: count, jobs: jobIds };
}





