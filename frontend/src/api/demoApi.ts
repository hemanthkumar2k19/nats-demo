export interface Job {
  job_id: string;
  type: string;
  payload: Record<string, any>;
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
}

export interface ServiceStatus {
  name: string;
  status: 'active' | 'connected' | 'disconnected' | 'unknown';
  details: string;
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

/**
 * Validates a job using Request/Reply.
 * Sends a validation request to the backend.
 */
export async function validateJob(job: Job): Promise<{ valid: boolean; message: string }> {
  const response = await fetch(`${API_BASE_URL}/jobs/validate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(job),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to validate job: ${response.status} ${response.statusText}. ${errorText}`);
  }

  return response.json();
}

/**
 * Retrieves the status/list of services.
 * Calls real GET /status from the Go backend.
 */
export async function getServiceStatus(): Promise<ServiceStatus[]> {
  const response = await fetch(`${API_BASE_URL}/status`);
  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error');
    throw new Error(`Failed to fetch service status: ${response.status} ${response.statusText}. ${errorText}`);
  }

  const data = await response.json();
  const result: ServiceStatus[] = [];

  // 1. Map NATS Server status
  const natsConnected = data.nats?.status === 'CONNECTED';
  result.push({
    name: 'NATS Server',
    status: natsConnected ? 'connected' : 'disconnected',
    details: natsConnected ? 'Connected to NATS broker' : 'Disconnected from NATS broker',
  });

  // 2. Map other services
  if (Array.isArray(data.services)) {
    data.services.forEach((svc: any) => {
      const isActive = svc.status === 'ACTIVE';
      result.push({
        name: svc.name,
        status: isActive ? 'active' : 'disconnected',
        details: isActive ? 'Service is operational' : 'Service is offline',
      });
    });
  }

  return result;
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


