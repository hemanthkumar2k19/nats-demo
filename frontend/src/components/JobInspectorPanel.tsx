import React from 'react';
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
  const getBadgeClass = (status: string) => {
    switch (status.toUpperCase()) {
      case 'SUBMITTED':
        return 'badge-submitted';
      case 'PROCESSING':
        return 'badge-processing';
      case 'COMPLETED':
        return 'badge-completed';
      case 'FAILED':
        return 'badge-failed';
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
    <div className="panel" style={{ marginTop: '0.5rem' }}>
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
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
          className="btn btn-secondary" 
          onClick={onClose}
          style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
        >
          Close
        </button>
      </div>

      {isLoading && (
        <div style={{ padding: '2rem 0', textAlign: 'center', color: 'var(--text-secondary)', fontSize: '0.875rem' }}>
          Loading details...
        </div>
      )}

      {error && (
        <div style={{ color: 'var(--status-danger)', fontSize: '0.875rem', padding: '1rem 0', fontFamily: 'var(--font-mono)' }}>
          * Error: {error}
        </div>
      )}

      {!isLoading && !error && !job && (
        <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', padding: '1rem 0', textAlign: 'center' }}>
          Click on any Job ID in the Activity Log to inspect.
        </div>
      )}

      {!isLoading && !error && job && (
        <div>
          {/* Metadata list */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', marginBottom: '1rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.25rem', fontSize: '0.875rem' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Job ID</span>
              <span className="mono-cell" style={{ fontWeight: 600 }}>{job.job_id}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.25rem', fontSize: '0.875rem' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Status</span>
              <span className={`badge ${getBadgeClass(job.status)}`}>{job.status}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.25rem', fontSize: '0.875rem' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Correlation ID</span>
              <span className="mono-cell" style={{ color: 'var(--text-secondary)' }}>{job.correlation_id || '-'}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.25rem', fontSize: '0.875rem' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Delivery Count</span>
              <span className="mono-cell" style={{ color: job.delivery_count > 1 ? 'var(--status-warning)' : 'var(--text-muted)' }}>
                {job.delivery_count}
              </span>
            </div>
            {job.trace_id && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.25rem', fontSize: '0.875rem' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Trace ID</span>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                  <span className="mono-cell" style={{ color: '#818cf8', fontSize: '0.75rem' }}>{job.trace_id}</span>
                  <a
                    href={`http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22tempo%22,%7B%22query%22:%22${job.trace_id}%22%7D%5D`}
                    target="_blank"
                    rel="noreferrer"
                    className="btn btn-secondary"
                    style={{ padding: '0.125rem 0.375rem', fontSize: '0.7rem', color: '#818cf8', textDecoration: 'none' }}
                  >
                    View in Tempo -&gt;
                  </a>
                </div>
              </div>
            )}
          </div>

          {/* History log list */}
          <div style={{ marginBottom: '1rem' }}>
            <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.5rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              Status History Log
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', padding: '0.5rem', background: 'rgba(255,255,255,0.01)', border: '1px solid var(--border-color)', borderRadius: '6px' }}>
              {job.history.map((h, i) => (
                <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '0.8125rem' }}>
                  <span className={`badge ${getBadgeClass(h.status)}`} style={{ transform: 'scale(0.9)', transformOrigin: 'left center' }}>{h.status}</span>
                  <span className="mono-cell" style={{ color: 'var(--text-muted)' }}>{formatTimestamp(h.timestamp)}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Raw JSON viewer */}
          <JsonViewer data={job as any} title="Raw Job Details Payload" />
        </div>
      )}
    </div>
  );
};
