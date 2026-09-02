import React, { useState } from 'react';
import { Job } from '../api/demoApi';

interface DeduplicationPanelProps {
  onSubmitJob: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  onShowInfo?: (key: string) => void;
}

export const DeduplicationPanel: React.FC<DeduplicationPanelProps> = ({
  onSubmitJob,
  isSubmitting,
  onShowInfo,
}) => {
  const [msgId, setMsgId] = useState<string>('dedup-msg-101');
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handlePublish = async (isDuplicateAttempt: boolean) => {
    setErrorMessage(null);
    setStatusMessage(null);

    if (!msgId.trim()) {
      setErrorMessage('Message ID is required');
      return;
    }

    try {
      const job: Job = {
        job_id: msgId.trim(),
        type: 'deduplication-test',
        payload: {
          file: 'dedup-sample.dat',
          simulate_failure: false,
          note: isDuplicateAttempt ? 'Duplicate retry simulation' : 'First message write',
        },
        delivery_mode: 'JETSTREAM',
      };

      await onSubmitJob(job);

      if (isDuplicateAttempt) {
        setStatusMessage(`Duplicate message published with ID "${msgId.trim()}". Check Activity Log for [DEDUPLICATED] event.`);
      } else {
        setStatusMessage(`1st message stored with ID "${msgId.trim()}". Click "Publish Duplicate" to verify deduplication.`);
      }
    } catch (err: any) {
      setErrorMessage(err.message || 'Publish failed');
    }
  };

  return (
    <div className="panel">
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            Message Deduplication
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('message-deduplication')}
              title="Learn about Message Deduplication"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      <div style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginBottom: '0.875rem' }}>
        Demonstrates JetStream server-side message deduplication using Nats-Msg-Id.
      </div>

      {/* Non-editable Current Config Parameters */}
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '0.5rem',
        marginBottom: '1rem',
        padding: '0.625rem 0.75rem',
        background: 'rgba(0, 0, 0, 0.2)',
        border: '1px solid var(--border-color)',
        borderRadius: '6px',
      }}>
        <div>
          <div style={{ fontSize: '0.6875rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>
            Stream Config:
          </div>
          <div className="mono-cell" style={{ color: 'var(--accent-cyan)', fontWeight: 600, fontSize: '0.75rem', wordBreak: 'break-all' }}>
            Duplicates = 2m 0s (120s window)
          </div>
        </div>

        <div>
          <div style={{ fontSize: '0.6875rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>
            Message ID Header:
          </div>
          <div className="mono-cell" style={{ color: 'var(--text-primary)', fontWeight: 600, fontSize: '0.75rem', wordBreak: 'break-all' }}>
            Nats-Msg-Id
          </div>
        </div>

        <div>
          <div style={{ fontSize: '0.6875rem', color: 'var(--text-secondary)', marginBottom: '0.15rem' }}>
            Target Stream:
          </div>
          <div className="mono-cell" style={{ color: 'var(--text-primary)', fontWeight: 600, fontSize: '0.75rem', wordBreak: 'break-all' }}>
            JOBS (Subject: jobs.submitted)
          </div>
        </div>
      </div>

      <form onSubmit={(e) => { e.preventDefault(); handlePublish(false); }}>
        <div className="form-group">
          <label className="form-label" htmlFor="dedupMsgId">Message ID (Nats-Msg-Id)</label>
          <input
            id="dedupMsgId"
            type="text"
            className="form-input code"
            value={msgId}
            onChange={(e) => setMsgId(e.target.value)}
            placeholder="e.g. dedup-msg-101"
            disabled={isSubmitting}
          />
        </div>

        {errorMessage && (
          <div style={{ color: 'var(--status-danger)', fontSize: '0.8125rem', marginTop: '0.5rem', fontFamily: 'var(--font-mono)' }}>
            * {errorMessage}
          </div>
        )}

        {statusMessage && (
          <div style={{ color: 'var(--accent-cyan)', fontSize: '0.8125rem', marginTop: '0.5rem', fontFamily: 'var(--font-mono)' }}>
            {statusMessage}
          </div>
        )}

        <div className="btn-group">
          <button
            type="button"
            className="btn btn-primary"
            disabled={isSubmitting}
            onClick={() => handlePublish(false)}
            style={{ flex: 1 }}
          >
            {isSubmitting ? 'Publishing...' : 'Publish (1st)'}
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            disabled={isSubmitting}
            onClick={() => handlePublish(true)}
            style={{ flex: 1 }}
          >
            {isSubmitting ? 'Publishing...' : 'Publish Duplicate'}
          </button>
        </div>
      </form>
    </div>
  );
};
