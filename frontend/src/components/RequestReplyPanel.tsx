import React, { useState } from 'react';
import { Job, ValidationResult, Activity, validateJob } from '../api/demoApi';
import { JsonViewer } from './JsonViewer';

interface RequestReplyPanelProps {
  activities: Activity[];
  onValidated: () => void;
  onShowInfo?: (key: string) => void;
}

// Events that belong to the Request/Reply interaction timeline.
const RR_EVENTS = new Set([
  'REQUEST_SENT',
  'REQUEST_RECEIVED',
  'REPLY_SENT',
  'REPLY_RECEIVED',
  'REQUEST_TIMEOUT',
]);

export const RequestReplyPanel: React.FC<RequestReplyPanelProps> = ({ activities, onValidated, onShowInfo }) => {
  const [jobId, setJobId] = useState<string>(`job-val-${Math.floor(100 + Math.random() * 900)}`);
  const [jobType, setJobType] = useState<string>('image-processing');
  const [payloadStr, setPayloadStr] = useState<string>(
    JSON.stringify({ file: 'image-101.jpg' }, null, 2)
  );
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [isSending, setIsSending] = useState<boolean>(false);
  const [result, setResult] = useState<ValidationResult | null>(null);
  const [lastJobId, setLastJobId] = useState<string>('');

  const handleSend = async () => {
    setJsonError(null);
    setResult(null);

    let parsedPayload: Record<string, any>;
    try {
      parsedPayload = JSON.parse(payloadStr);
    } catch (e: any) {
      setJsonError(`Invalid JSON payload: ${e.message}`);
      return;
    }

    const job: Job = { job_id: jobId, type: jobType, payload: parsedPayload };

    setIsSending(true);
    setLastJobId(jobId);
    try {
      const res = await validateJob(job);
      setResult(res);
      onValidated();
    } catch (err: any) {
      setJsonError(err.message || 'Request failed');
    } finally {
      setIsSending(false);
      // Auto-increment the job ID so consecutive sends use distinct IDs,
      // matching the behaviour of the Job Submission panel.
      setJobId(`job-val-${Math.floor(100 + Math.random() * 900)}`);
    }
  };

  // Filter activity log for request/reply events that match the last job ID sent.
  const timelineEvents = activities.filter(
    (a) => RR_EVENTS.has(a.event) && a.job_id === lastJobId
  );

  const getStatusLabel = (): { label: string; cls: string } => {
    if (!result) return { label: '', cls: '' };
    if (result.timedOut) return { label: 'TIMEOUT', cls: 'badge-failed' };
    if (result.valid === false) return { label: 'INVALID', cls: 'badge-processing' };
    return { label: 'SUCCESS', cls: 'badge-completed' };
  };

  const getTimelineBadgeClass = (event: string): string => {
    switch (event) {
      case 'REQUEST_SENT':     return 'badge-submitted';
      case 'REQUEST_RECEIVED': return 'badge-delivered';
      case 'REPLY_SENT':       return 'badge-processing';
      case 'REPLY_RECEIVED':   return 'badge-completed';
      case 'REQUEST_TIMEOUT':  return 'badge-failed';
      default:                 return '';
    }
  };

  const { label: statusLabel, cls: statusCls } = getStatusLabel();

  return (
    <div className="panel">
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
            </svg>
            Request / Reply
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('request-reply')}
              title="Learn about Request / Reply"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      <div style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginBottom: '1rem' }}>
        Demonstrates native NATS Request/Reply on{' '}
        <span className="mono-cell" style={{ color: 'var(--accent-cyan)' }}>jobs.validate</span>.
        Toggle the Processor OFF to observe a natural timeout.
      </div>

      <div className="form-group">
        <label className="form-label" htmlFor="rrJobId">Job ID</label>
        <input
          id="rrJobId"
          type="text"
          className="form-input code"
          value={jobId}
          onChange={(e) => setJobId(e.target.value)}
          disabled={isSending}
        />
      </div>

      <div className="form-group">
        <label className="form-label" htmlFor="rrJobType">Job Type</label>
        <select
          id="rrJobType"
          className="form-select"
          value={jobType}
          onChange={(e) => setJobType(e.target.value)}
          disabled={isSending}
        >
          <option value="image-processing">image-processing</option>
          <option value="data-sync">data-sync</option>
          <option value="email-alert">email-alert</option>
          <option value="unknown-type">unknown-type (invalid)</option>
        </select>
      </div>

      <div className="form-group">
        <label className="form-label" htmlFor="rrPayload">Payload (JSON)</label>
        <textarea
          id="rrPayload"
          rows={3}
          className="form-textarea code"
          value={payloadStr}
          onChange={(e) => setPayloadStr(e.target.value)}
          disabled={isSending}
        />
      </div>

      {jsonError && (
        <div style={{ color: 'var(--status-danger)', fontSize: '0.8125rem', marginTop: '0.5rem', fontFamily: 'var(--font-mono)' }}>
          {jsonError}
        </div>
      )}

      <div className="btn-group" style={{ marginBottom: '1rem' }}>
        <button
          type="button"
          className="btn btn-primary"
          onClick={handleSend}
          disabled={isSending}
          style={{ flex: 1 }}
        >
          {isSending ? 'Sending...' : 'Send Request'}
        </button>
      </div>

      {/* Result section */}
      {result && (
        <div style={{ borderTop: '1px solid var(--border-color)', paddingTop: '1rem', marginBottom: '1rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.5rem' }}>
            <span style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>Status:</span>
            <span className={`badge ${statusCls}`}>{statusLabel}</span>
          </div>

          {result.timedOut ? (
            <div style={{ fontSize: '0.8125rem', color: 'var(--status-danger)', fontFamily: 'var(--font-mono)', padding: '0.5rem', background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.15)', borderRadius: '6px' }}>
              No response received from processor service (timeout: 2s)
            </div>
          ) : (
            <JsonViewer data={result} />
          )}
        </div>
      )}

      {/* Interaction timeline */}
      {timelineEvents.length > 0 && (
        <div>
          <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '0.5rem' }}>
            Interaction Timeline
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.375rem' }}>
            {timelineEvents.map((ev, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.8125rem' }}>
                <span className="mono-cell" style={{ color: 'var(--text-muted)', minWidth: '56px' }}>{ev.timestamp}</span>
                <span className={`badge ${getTimelineBadgeClass(ev.event)}`}>{ev.event}</span>
                {ev.worker && (
                  <span className="mono-cell" style={{ color: 'var(--text-secondary)' }}>{ev.worker}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
