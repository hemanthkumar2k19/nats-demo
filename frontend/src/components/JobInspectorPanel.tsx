import React, { useEffect } from 'react';
import { JobDetailResponse } from '../api/demoApi';
import { JsonViewer } from './JsonViewer';

interface JobInspectorPanelProps {
  job: JobDetailResponse | null;
  isLoading: boolean;
  error: string | null;
  onClose: () => void;
  onShowInfo?: (key: string) => void;
}

export const JobInspectorPanel: React.FC<JobInspectorPanelProps> = ({
  job,
  isLoading,
  error,
  onClose,
  onShowInfo,
}) => {
  // Close modal on Escape key press
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const getBadgeClass = (status: string) => {
    switch (status.toUpperCase()) {
      case 'SUBMITTED':
      case 'PUBLISHED':
        return 'badge-submitted';
      case 'PROCESSING':
        return 'badge-processing';
      case 'COMPLETED':
      case 'ACKED':
        return 'badge-completed';
      case 'FAILED':
        return 'badge-failed';
      case 'DLQ_PUBLISHED':
        return 'badge-dlq';
      default:
        return '';
    }
  };

  const formatTimestamp = (isoString: string) => {
    try {
      const date = new Date(isoString);
      return date.toLocaleTimeString();
    } catch {
      return isoString;
    }
  };

  return (
    <div className="inspector-modal-backdrop" onClick={onClose}>
      <div className="inspector-modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="panel-header" style={{ marginBottom: '1rem', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.75rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <h2 className="panel-title" style={{ display: 'flex', alignItems: 'center', gap: '0.375rem', margin: 0 }}>
              <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
              Job Details Inspector
            </h2>
            {onShowInfo && (
              <button
                type="button"
                className="node-info-btn"
                onClick={() => onShowInfo('job-details')}
                title="Learn about Job Details"
              >
                (i)
              </button>
            )}
          </div>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={onClose}
            style={{ padding: '0.2rem 0.6rem', fontSize: '0.8125rem', fontWeight: 600 }}
            title="Close Inspector"
          >
            [X] Close
          </button>
        </div>

        {isLoading && (
          <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
            Loading job details from NATS...
          </div>
        )}

        {error && (
          <div className="alert-banner" style={{ margin: '1rem 0' }}>
            {error}
          </div>
        )}

        {!isLoading && !error && !job && (
          <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
            No details available for this job.
          </div>
        )}

        {!isLoading && !error && job && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
            {/* Metadata Summary Grid */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '0.5rem' }}>
              <div className="readonly-box" style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Job ID</span>
                <span className="mono-cell" style={{ color: 'var(--accent-cyan)', fontWeight: 700 }}>{job.job_id}</span>
              </div>
              <div className="readonly-box" style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Status</span>
                <div>
                  <span className={`badge ${getBadgeClass(job.status)}`}>{job.status}</span>
                </div>
              </div>
              <div className="readonly-box" style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Delivery Count</span>
                <span className="mono-cell font-bold">{job.delivery_count}</span>
              </div>
              {job.type && (
                <div className="readonly-box" style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                  <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Job Type</span>
                  <span className="mono-cell">{job.type}</span>
                </div>
              )}
              {job.delivery_mode && (
                <div className="readonly-box" style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                  <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Delivery Mode</span>
                  <span className="mono-cell" style={{ color: '#F59E0B' }}>{job.delivery_mode}</span>
                </div>
              )}
              {job.worker && (
                <div className="readonly-box" style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                  <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Worker</span>
                  <span className="mono-cell">{job.worker}</span>
                </div>
              )}
            </div>

            {/* Trace Context Link */}
            {job.trace_id && (
              <div className="readonly-box" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                  <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Distributed Trace ID</span>
                  <span className="mono-cell" style={{ color: '#818CF8' }}>{job.trace_id}</span>
                </div>
                <a
                  href={`http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22tempo%22,%7B%22query%22:%22${job.trace_id}%22%7D%5D`}
                  target="_blank"
                  rel="noreferrer"
                  className="btn btn-secondary"
                  style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem', color: '#818CF8', textDecoration: 'none' }}
                >
                  View in Tempo -&gt;
                </a>
              </div>
            )}

            {/* Status History Log */}
            {job.history && job.history.length > 0 && (
              <div>
                <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.4rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Status History Timeline
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem', padding: '0.5rem', background: 'rgba(0,0,0,0.2)', border: '1px solid var(--border-color)', borderRadius: '6px' }}>
                  {job.history.map((h, i) => (
                    <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '0.8125rem' }}>
                      <span className={`badge ${getBadgeClass(h.status)}`}>{h.status}</span>
                      <span className="mono-cell" style={{ color: 'var(--text-muted)' }}>{formatTimestamp(h.timestamp)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Raw JSON viewer */}
            <JsonViewer data={job as any} title="Raw Job Details Payload" />
          </div>
        )}
      </div>
    </div>
  );
};
