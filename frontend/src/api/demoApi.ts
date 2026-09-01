export interface Job {
  job_id: string;
  type: string;
  payload: Record<string, any>;
  delivery_mode?: string;
}

export interface JobStatusResponse {
  job_id: string;
  status: string;
  correlation_id?: string;
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
  pending: number;
}

export interface SystemStatusResponse {
  services: ServiceStatus[];
  jetstream?: JetStreamInfo;
}

const API_BASE_URL = 'http://localhost:8080';

/**
 * Submits a job to the backend API.
 * Calls real POST /jobs.
 */
export async function submitJob(job: Job): Promise<JobStatusResponse> {
  const response = await fetch(`${API_BASE_URL}/jobs`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Correlation-Id': `corr-${job.job_id}-${Date.now().toString(36)}`,
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
  correlationId?: string;
}

/**
 * Validates a job using Request/Reply.
 * Returns a ValidationResult. On processor timeout (HTTP 504), timedOut is set to true
 * rather than throwing so the UI can display the timeout scenario cleanly.
 */
export async function validateJob(job: Job): Promise<ValidationResult> {
  const correlationId = `corr-val-${job.job_id}-${Date.now().toString(36)}`;
  const response = await fetch(`${API_BASE_URL}/jobs/validate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Correlation-Id': correlationId,
    },
    body: JSON.stringify(job),
  });

  // 504 means the NATS request timed out - no responder was available.
  // Return a structured result rather than throwing so the UI can display the timeout panel.
  if (response.status === 504) {
    return { timedOut: true, error: 'No response received from processor service', correlationId };
  }

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to validate job: ${response.status} ${response.statusText}. ${errorText}`);
  }

  const data = await response.json();
  return { ...data, correlationId };
}

/**
 * Retrieves the status/list of services.
 * Calls real GET /status from the Go backend.
 */
export async function getServiceStatus(): Promise<SystemStatusResponse> {
  const response = await fetch(`${API_BASE_URL}/status`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch service status: ${response.status} ${response.statusText}. ${errorText}`);
  }

  const data = await response.json();
  const services: ServiceStatus[] = [];

  // 1. Map NATS Server status
  const natsConnected = data.nats?.status === 'CONNECTED';
  services.push({
    name: 'NATS Server',
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
          name: svc.name,
          status: isActive ? (isProcessing ? 'active' : 'stopped') : 'disconnected',
          details: isActive 
            ? (isProcessing ? 'Processor is active and processing messages' : 'Processor is paused')
            : 'Service is offline',
          processing: isProcessing,
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
    pending: data.jetstream.pending,
  } : undefined;

  return { services, jetstream };
}

/**
 * Retrieves the list of recent job activities.
 * Calls real GET /activities from the Go backend.
 */
export async function getActivity(): Promise<Activity[]> {
  const response = await fetch(`${API_BASE_URL}/activities`);
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
  correlation_id: string;
  history: JobHistoryItem[];
}

export interface ReplayRequest {
  from_sequence?: number;
  to_sequence?: number;
  from_time?: string;
  to_time?: string;
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
  const response = await fetch(`${API_BASE_URL}/jobs/${jobId}`);
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
  const response = await fetch(`${API_BASE_URL}/jobs/replay`, {
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
  const response = await fetch(`${API_BASE_URL}/messaging/subscriptions`);
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
  const response = await fetch(`${API_BASE_URL}/messaging/activity`);
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
  const response = await fetch(`${API_BASE_URL}/processor/state`, {
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
}

/**
 * Retrieves the consumer configuration and live status metrics.
 * Calls GET /consumer.
 */
export async function getConsumerStatus(): Promise<ConsumerStatus> {
  const response = await fetch(`${API_BASE_URL}/consumer`);
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
  const response = await fetch(`${API_BASE_URL}/consumer`, {
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


