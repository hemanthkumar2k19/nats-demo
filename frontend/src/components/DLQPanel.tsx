import React, { useState, useEffect, useCallback } from 'react';
import { submitJob, getDLQStatus, getDLQMessages, DLQStatus, DLQMessage, Job } from '../api/demoApi';

interface DLQPanelProps {
  onShowInfo: (key: string) => void;
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
  onRefreshAll?: () => void;
}

export const DLQPanel: React.FC<DLQPanelProps> = ({
  onShowInfo,
  onAlert,
  onRefreshAll,
}) => {
  const [maxAttempts, setMaxAttempts] = useState<number>(3);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [isLoadingDLQ, setIsLoadingDLQ] = useState<boolean>(false);
  const [dlqStatus, setDlqStatus] = useState<DLQStatus | null>(null);
  const [dlqMessages, setDlqMessages] = useState<DLQMessage[]>([]);

  const fetchDLQData = useCallback(async () => {
    setIsLoadingDLQ(true);
    try {
      const [status, messages] = await Promise.all([
        getDLQStatus(),
        getDLQMessages(),
      ]);
      setDlqStatus(status);
      setDlqMessages(messages);
    } catch {
      // Background poll failure handled gracefully
    } finally {
      setIsLoadingDLQ(false);
    }
  }, []);

  useEffect(() => {
    fetchDLQData();
    const timer = setInterval(fetchDLQData, 3000);
    return () => clearInterval(timer);
  }, [fetchDLQData]);

  const handleSendFailingJob = async () => {
    setIsSubmitting(true);
    const randomSuffix = Math.floor(100 + Math.random() * 900);
    const jobId = `job-fail-${randomSuffix}`;

    const failingJob: Job = {
      job_id: jobId,
      type: 'image-processing',
      payload: {
        file: 'corrupt-image.dat',
        simulate_failure: true,
        simulate_failure_count: 0, // Fail every attempt until reaching max
        max_delivery_attempts: maxAttempts,
      },
      delivery_mode: 'JETSTREAM',
    };

    try {
      await submitJob(failingJob);
      if (onAlert) {
        onAlert('success', `Submitted failing job ${jobId} (Max delivery attempts: ${maxAttempts})`);
      }
      if (onRefreshAll) {
        onRefreshAll();
      }
      setTimeout(fetchDLQData, 1200);
    } catch (err: any) {
      if (onAlert) {
        onAlert('error', `Failed to submit failing job: ${err?.message || 'Unknown error'}`);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const storedCount = dlqStatus?.messages ?? 0;
  const pendingCount = dlqStatus?.pending ?? 0;

  return (
    <section className="panel dlq-panel">
      {/* Header with Title, Standard (i) Button, and Refresh */}
      <div className="panel-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title" style={{ display: 'flex', alignItems: 'center', gap: '0.375rem', margin: 0 }}>
            <svg style={{ width: '14px', height: '14px', color: '#EF4444' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            Dead Letter Queue
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('dead-letter-queue')}
              title="Learn about JetStream Dead Letter Queue"
            >
              (i)
            </button>
          )}
        </div>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={fetchDLQData}
          disabled={isLoadingDLQ}
          style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
          title="Refresh DLQ Stream and Consumer Status"
        >
          {isLoadingDLQ ? 'Checking...' : 'Refresh'}
        </button>
      </div>

      <div style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginBottom: '0.875rem', lineHeight: '1.4' }}>
        Isolates failed messages into stream{' '}
        <span className="mono-cell" style={{ color: '#F87171', fontWeight: 600 }}>JOBS_DLQ</span>{' '}
        after exhausting delivery attempts.
      </div>

      {/* DLQ Resources Summary (2-Column Card Grid) */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.5rem', marginBottom: '0.875rem' }}>
        <div className="readonly-box" style={{ padding: '0.5rem 0.65rem', display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
          <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 600 }}>
            DLQ Stream
          </span>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span className="mono-cell" style={{ color: '#F87171', fontWeight: 600, fontSize: '0.75rem' }}>
              JOBS_DLQ
            </span>
            <span
              className="node-badge"
              style={{
                background: storedCount > 0 ? 'rgba(239, 68, 68, 0.15)' : 'rgba(107, 114, 128, 0.15)',
                color: storedCount > 0 ? '#F87171' : 'var(--text-muted)',
                border: storedCount > 0 ? '1px solid rgba(239, 68, 68, 0.3)' : '1px solid rgba(107, 114, 128, 0.25)',
                fontSize: '0.6875rem',
                padding: '0.05rem 0.35rem',
                fontWeight: 600,
              }}
            >
              {storedCount} {storedCount === 1 ? 'msg' : 'msgs'}
            </span>
          </div>
        </div>

        <div className="readonly-box" style={{ padding: '0.5rem 0.65rem', display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
          <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 600 }}>
            Consumer
          </span>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span className="mono-cell" style={{ color: 'var(--text-secondary)', fontWeight: 600, fontSize: '0.75rem' }}>
              dlq-inspector
            </span>
            <span
              className="node-badge"
              style={{
                background: pendingCount > 0 ? 'rgba(245, 158, 11, 0.15)' : 'rgba(107, 114, 128, 0.15)',
                color: pendingCount > 0 ? '#FCD34D' : 'var(--text-muted)',
                border: pendingCount > 0 ? '1px solid rgba(245, 158, 11, 0.3)' : '1px solid rgba(107, 114, 128, 0.25)',
                fontSize: '0.6875rem',
                padding: '0.05rem 0.35rem',
                fontWeight: 600,
              }}
            >
              {pendingCount} pend
            </span>
          </div>
        </div>
      </div>

      {/* Action Box: Max Attempts Configuration & Simulation Button */}
      <div className="form-group" style={{ marginBottom: '0.875rem' }}>
        <label className="form-label" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
          <span>Max Delivery Attempts</span>
          <span style={{ fontSize: '0.6875rem', color: 'var(--text-muted)' }}>Threshold before DLQ</span>
        </label>
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
          <input
            type="number"
            min="1"
            max="10"
            className="input-field"
            style={{ width: '60px', textAlign: 'center', fontFamily: 'var(--font-mono)', fontWeight: 600 }}
            value={maxAttempts}
            onChange={(e) => setMaxAttempts(Math.max(1, parseInt(e.target.value) || 1))}
            disabled={isSubmitting}
          />
          <button
            type="button"
            className="btn btn-danger"
            style={{ flex: 1, padding: '0.45rem 0.75rem', fontSize: '0.8125rem', fontWeight: 600 }}
            onClick={handleSendFailingJob}
            disabled={isSubmitting}
          >
            {isSubmitting ? 'Simulating Failure...' : 'Send Failing Job'}
          </button>
        </div>
        <span style={{ display: 'block', fontSize: '0.6875rem', color: 'var(--text-muted)', marginTop: '0.35rem', lineHeight: '1.35' }}>
          Fails and NAKs on each delivery attempt. At attempt {maxAttempts}, the processor routes it to JOBS_DLQ and ACKs the original message.
        </span>
      </div>

      {/* Messages In DLQ List */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.25rem' }}>
          <span style={{ fontSize: '0.625rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            MESSAGES IN JOBS_DLQ ({dlqMessages.length})
          </span>
          {dlqMessages.length > 0 && (
            <span style={{ fontSize: '0.6875rem', color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
              {dlqMessages.length} captured
            </span>
          )}
        </div>

        {dlqMessages.length === 0 ? (
          <div
            style={{
              padding: '0.875rem',
              textAlign: 'center',
              fontSize: '0.75rem',
              color: 'var(--text-muted)',
              background: 'rgba(0, 0, 0, 0.2)',
              borderRadius: '6px',
              border: '1px dashed var(--border-color)',
            }}
          >
            No messages in JOBS_DLQ. Click "Send Failing Job" to observe redeliveries and DLQ isolation.
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.45rem', maxHeight: '200px', overflowY: 'auto' }}>
            {dlqMessages.map((msg, idx) => (
              <div
                key={`${msg.job_id}-${idx}`}
                style={{
                  background: 'rgba(0, 0, 0, 0.25)',
                  border: '1px solid rgba(239, 68, 68, 0.25)',
                  borderRadius: '6px',
                  padding: '0.45rem 0.6rem',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '0.25rem',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span className="mono-cell" style={{ color: '#F87171', fontWeight: 700, fontSize: '0.75rem' }}>
                    {msg.job_id}
                  </span>
                  <span style={{ fontSize: '0.625rem', fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>
                    {msg.timestamp ? msg.timestamp.substring(11, 19) : '-'}
                  </span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '0.5rem' }}>
                  <span
                    style={{
                      color: '#FCA5A5',
                      fontSize: '0.6875rem',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      flex: 1,
                    }}
                    title={msg.failure_reason}
                  >
                    {msg.failure_reason}
                  </span>
                  <span
                    className="node-badge"
                    style={{
                      background: 'rgba(239, 68, 68, 0.2)',
                      color: '#FCA5A5',
                      border: '1px solid rgba(239, 68, 68, 0.35)',
                      fontSize: '0.625rem',
                      padding: '0.05rem 0.3rem',
                      fontWeight: 700,
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {msg.delivery_attempts} att
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
};
