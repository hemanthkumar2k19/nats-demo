import React, { useState } from 'react';
import { Job } from '../api/demoApi';

interface JobPanelProps {
  onSubmitJob: (job: Job) => Promise<void>;
  onValidateJob: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  isValidating: boolean;
}

export const JobPanel: React.FC<JobPanelProps> = ({
  onSubmitJob,
  onValidateJob,
  isSubmitting,
  isValidating,
}) => {
  const [jobId, setJobId] = useState<string>(`job-${Math.floor(100 + Math.random() * 900)}`);
  const [jobType, setJobType] = useState<string>('image-processing');
  const [deliveryMode, setDeliveryMode] = useState<string>('CORE');
  const [payloadStr, setPayloadStr] = useState<string>(
    JSON.stringify({ file: 'image-101.jpg', simulate_failure: false, simulate_failure_count: 0 }, null, 2)
  );
  const [jsonError, setJsonError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setJsonError(null);

    if (!jobId.trim()) {
      setJsonError('Job ID is required');
      return;
    }

    try {
      const parsedPayload = JSON.parse(payloadStr);
      const job: Job = {
        job_id: jobId,
        type: jobType,
        payload: parsedPayload,
        delivery_mode: deliveryMode,
      };
      await onSubmitJob(job);
      // Auto-increment Job ID suffix for convenience in consecutive runs
      const match = jobId.match(/^(.*?)-(\d+)$/);
      if (match) {
        const prefix = match[1];
        const nextNum = parseInt(match[2], 10) + 1;
        setJobId(`${prefix}-${nextNum}`);
      } else {
        setJobId(`job-${Math.floor(100 + Math.random() * 900)}`);
      }
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        setJsonError(`Invalid JSON payload: ${err.message}`);
      } else {
        setJsonError(err.message || 'Failed to submit job');
      }
    }
  };

  const handleValidate = async () => {
    setJsonError(null);
    try {
      const parsedPayload = JSON.parse(payloadStr);
      const job: Job = {
        job_id: jobId,
        type: jobType,
        payload: parsedPayload,
        delivery_mode: deliveryMode,
      };
      await onValidateJob(job);
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        setJsonError(`Invalid JSON payload: ${err.message}`);
      } else {
        setJsonError(err.message || 'Validation request failed');
      }
    }
  };

  return (
    <div className="panel">
      <div className="panel-header">
        <h2 className="panel-title">
          <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          Submit Job
        </h2>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label">Delivery Mode</label>
          <div style={{ display: 'flex', gap: '1rem', marginTop: '0.25rem' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.375rem', cursor: 'pointer', fontSize: '0.8125rem' }}>
              <input
                type="radio"
                name="deliveryMode"
                value="CORE"
                checked={deliveryMode === 'CORE'}
                onChange={() => setDeliveryMode('CORE')}
                disabled={isSubmitting || isValidating}
              />
              Core NATS (Transient)
            </label>
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.375rem', cursor: 'pointer', fontSize: '0.8125rem' }}>
              <input
                type="radio"
                name="deliveryMode"
                value="JETSTREAM"
                checked={deliveryMode === 'JETSTREAM'}
                onChange={() => setDeliveryMode('JETSTREAM')}
                disabled={isSubmitting || isValidating}
              />
              JetStream (Durable)
            </label>
          </div>
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="jobId">Job ID</label>
          <input
            id="jobId"
            type="text"
            className="form-input code"
            value={jobId}
            onChange={(e) => setJobId(e.target.value)}
            disabled={isSubmitting || isValidating}
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="jobType">Job Type</label>
          <select
            id="jobType"
            className="form-select"
            value={jobType}
            onChange={(e) => setJobType(e.target.value)}
            disabled={isSubmitting || isValidating}
          >
            <option value="image-processing">image-processing</option>
            <option value="data-sync">data-sync</option>
            <option value="email-alert">email-alert</option>
          </select>
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="payload">Payload (JSON)</label>
          <textarea
            id="payload"
            rows={5}
            className="form-textarea code"
            value={payloadStr}
            onChange={(e) => setPayloadStr(e.target.value)}
            disabled={isSubmitting || isValidating}
          />
        </div>

        {jsonError && (
          <div style={{ color: 'var(--status-danger)', fontSize: '0.8125rem', marginTop: '0.5rem', fontFamily: 'var(--font-mono)' }}>
            * {jsonError}
          </div>
        )}

        <div className="btn-group">
          <button
            type="submit"
            className="btn btn-primary"
            disabled={isSubmitting || isValidating}
            style={{ flex: 1 }}
          >
            {isSubmitting ? 'Submitting...' : 'Submit Job'}
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={handleValidate}
            disabled={isSubmitting || isValidating}
            style={{ flex: 1 }}
          >
            {isValidating ? 'Validating...' : 'Validate Job'}
          </button>
        </div>
      </form>
    </div>
  );
};
