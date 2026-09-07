import React, { useState } from 'react';
import { Job } from '../../api/demoApi';

interface CoreFlowPublisherProps {
  onPublish: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  onShowInfo?: (key: string) => void;
}

const DEFAULT_PAYLOAD = {
  file: 'image-101.jpg',
};

interface PublishedReceipt {
  jobId: string;
  type: string;
  subject: string;
  msgId: string;
  timestamp: string;
}

export const CoreFlowPublisher: React.FC<CoreFlowPublisherProps> = ({
  onPublish,
  isSubmitting,
  onShowInfo,
}) => {
  const [jobId, setJobId] = useState<string>(`job-${Math.floor(100 + Math.random() * 900)}`);
  const [jobType, setJobType] = useState<string>('image-processing');
  const [payloadStr, setPayloadStr] = useState<string>(JSON.stringify(DEFAULT_PAYLOAD, null, 2));
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [publishedHistory, setPublishedHistory] = useState<PublishedReceipt[]>([]);

  const regenerateJobId = () => {
    setJobId(`job-${Math.floor(100 + Math.random() * 900)}`);
  };

  const formatPayloadJson = () => {
    try {
      const parsed = JSON.parse(payloadStr);
      setPayloadStr(JSON.stringify(parsed, null, 2));
      setJsonError(null);
    } catch (err: any) {
      setJsonError(`Invalid JSON: ${err.message}`);
    }
  };

  const resetToDefaultPayload = () => {
    setPayloadStr(JSON.stringify(DEFAULT_PAYLOAD, null, 2));
    setJobType('image-processing');
    setJsonError(null);
  };

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    setJsonError(null);

    if (!jobId.trim()) {
      setJsonError('Job ID is required');
      return;
    }

    try {
      const parsed = JSON.parse(payloadStr);
      const job: Job = {
        job_id: jobId,
        type: jobType,
        payload: parsed,
        delivery_mode: 'JETSTREAM',
      };

      await onPublish(job);

      const receipt: PublishedReceipt = {
        jobId,
        type: jobType,
        subject: 'jobs.submitted',
        msgId: `msg-${jobId}`,
        timestamp: new Date().toLocaleTimeString(),
      };

      setPublishedHistory((prev) => [receipt, ...prev.slice(0, 7)]);

      // Auto-increment numeric suffix for consecutive test runs
      const match = jobId.match(/^(.*?)-(\d+)$/);
      if (match) {
        setJobId(`${match[1]}-${parseInt(match[2], 10) + 1}`);
      } else {
        setJobId(`job-${Math.floor(100 + Math.random() * 900)}`);
      }
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        setJsonError(`Invalid JSON: ${err.message}`);
      } else {
        setJsonError(err.message || 'Publish failed');
      }
    }
  };

  return (
    <div className="panel" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Panel Header */}
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span style={{ 
            background: 'rgba(59, 130, 246, 0.15)', 
            color: '#60A5FA', 
            borderRadius: '4px', 
            padding: '2px 8px', 
            fontSize: '0.75rem', 
            fontWeight: 700,
            letterSpacing: '0.05em' 
          }}>
            STAGE 1
          </span>
          <h2 className="panel-title" style={{ margin: 0 }}>Submit Job</h2>
        </div>
        {onShowInfo && (
          <button
            type="button"
            className="node-info-btn"
            onClick={() => onShowInfo('submit-job')}
            title="Learn about Publishers in NATS JetStream"
          >
            (i)
          </button>
        )}
      </div>

      <div style={{ padding: '0.75rem 1rem 0 1rem' }}>
        <p style={{ margin: '0 0 0.75rem 0', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          Dispatches jobs to NATS JetStream. Messages are durably persisted to the <strong>JOBS</strong> stream on subject <code>jobs.submitted</code> for processing.
        </p>

        {/* JetStream Transport Badge */}
        <div style={{
          background: 'rgba(16, 185, 129, 0.12)',
          border: '1px solid rgba(16, 185, 129, 0.3)',
          borderRadius: '6px',
          padding: '0.4rem 0.65rem',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          fontSize: '0.72rem',
          marginBottom: '0.75rem'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
            <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: '#10B981' }} />
            <strong style={{ color: '#34D399' }}>Transport: NATS JetStream</strong>
          </div>
          <span style={{ color: 'var(--text-dim)' }}>Stream: JOBS | At-Least-Once</span>
        </div>
      </div>

      {/* Form Section */}
      <form onSubmit={handleSend} style={{ display: 'flex', flexDirection: 'column', flex: 1, padding: '0 1rem 0.5rem 1rem' }}>
        
        {/* Job ID & Job Type Grid */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.65rem', marginBottom: '0.65rem' }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.2rem' }}>
              <label className="form-label" style={{ fontSize: '0.72rem', margin: 0 }}>Job ID</label>
              <button
                type="button"
                onClick={regenerateJobId}
                style={{
                  background: 'none',
                  border: 'none',
                  padding: 0,
                  fontSize: '0.65rem',
                  color: '#60A5FA',
                  cursor: 'pointer',
                  textDecoration: 'underline'
                }}
                title="Generate new ID"
              >
                Regen
              </button>
            </div>
            <input
              type="text"
              className="form-input code"
              value={jobId}
              onChange={(e) => setJobId(e.target.value)}
              placeholder="e.g. job-101"
              required
              style={{ padding: '0.4rem 0.6rem' }}
            />
          </div>

          <div>
            <label className="form-label" style={{ fontSize: '0.72rem', marginBottom: '0.2rem' }}>Job Type</label>
            <input
              type="text"
              className="form-input"
              value={jobType}
              onChange={(e) => setJobType(e.target.value)}
              placeholder="e.g. image-processing"
              required
              style={{ padding: '0.4rem 0.6rem' }}
            />
          </div>
        </div>

        {/* Outgoing Message Envelope Inspector */}
        <div style={{ 
          background: 'rgba(15, 23, 42, 0.85)', 
          border: '1px solid rgba(59, 130, 246, 0.25)', 
          borderRadius: '6px', 
          padding: '0.5rem 0.7rem', 
          marginBottom: '0.65rem',
          fontSize: '0.7rem',
          fontFamily: 'var(--font-mono)'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.3rem' }}>
            <span style={{ color: 'var(--text-dim)', fontWeight: 600, letterSpacing: '0.04em' }}>
              OUTGOING NATS ENVELOPE
            </span>
            <span style={{ 
              fontSize: '0.6rem', 
              color: '#93C5FD', 
              background: 'rgba(59, 130, 246, 0.15)', 
              padding: '1px 5px', 
              borderRadius: '3px' 
            }}>
              Wire Format
            </span>
          </div>

          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '2px' }}>
            <span style={{ color: 'var(--text-muted)' }}>Target Subject:</span>
            <span style={{ color: '#60A5FA', fontWeight: 600 }}>jobs.submitted</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '2px' }}>
            <span style={{ color: 'var(--text-muted)' }}>Storage Destination:</span>
            <span style={{ color: '#34D399', fontWeight: 500 }}>JOBS Stream (JetStream)</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-muted)' }}>Header [Nats-Msg-Id]:</span>
            <span style={{ color: '#E2E8F0' }}>msg-{jobId}</span>
          </div>
        </div>

        {/* Payload JSON Editor */}
        <div style={{ marginBottom: '0.65rem', flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.2rem' }}>
            <label className="form-label" style={{ fontSize: '0.72rem', margin: 0 }}>Payload (JSON)</label>
            <div style={{ display: 'flex', gap: '0.5rem' }}>
              <button
                type="button"
                onClick={formatPayloadJson}
                style={{
                  background: 'none',
                  border: 'none',
                  padding: 0,
                  fontSize: '0.65rem',
                  color: 'var(--text-muted)',
                  cursor: 'pointer',
                  textDecoration: 'underline'
                }}
              >
                Format JSON
              </button>
              <button
                type="button"
                onClick={resetToDefaultPayload}
                style={{
                  background: 'none',
                  border: 'none',
                  padding: 0,
                  fontSize: '0.65rem',
                  color: 'var(--text-muted)',
                  cursor: 'pointer',
                  textDecoration: 'underline'
                }}
              >
                Reset Default
              </button>
            </div>
          </div>
          <textarea
            className="form-input code"
            rows={3}
            value={payloadStr}
            onChange={(e) => setPayloadStr(e.target.value)}
            style={{ 
              resize: 'vertical', 
              width: '100%', 
              flex: 1,
              lineHeight: 1.4,
              fontSize: '0.72rem',
              padding: '0.45rem 0.6rem'
            }}
          />
        </div>

        {jsonError && (
          <div style={{ 
            color: '#F87171', 
            background: 'rgba(239, 68, 68, 0.1)', 
            border: '1px solid rgba(239, 68, 68, 0.25)', 
            borderRadius: '4px', 
            padding: '0.3rem 0.5rem', 
            fontSize: '0.7rem', 
            marginBottom: '0.65rem' 
          }}>
            * {jsonError}
          </div>
        )}

        {/* Publish Action Button */}
        <button
          type="submit"
          className="btn btn-primary"
          disabled={isSubmitting}
          style={{ 
            width: '100%', 
            padding: '0.55rem 1rem', 
            fontSize: '0.8rem',
            fontWeight: 600,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '0.4rem',
            boxShadow: '0 0 16px rgba(37, 99, 235, 0.25)'
          }}
        >
          {isSubmitting ? (
            <span>Submitting Job to JetStream...</span>
          ) : (
            <span>Submit Job to NATS JetStream -&gt;</span>
          )}
        </button>
      </form>

      {/* Publisher Activity Log (Bottom of Panel) */}
      <div style={{ 
        borderTop: '1px solid var(--border-color)', 
        padding: '0.5rem 1rem', 
        background: 'rgba(0, 0, 0, 0.25)',
        maxHeight: '130px',
        overflowY: 'auto'
      }}>
        <div style={{ 
          fontSize: '0.68rem', 
          fontWeight: 600, 
          color: 'var(--text-dim)', 
          textTransform: 'uppercase', 
          letterSpacing: '0.04em',
          marginBottom: '0.3rem' 
        }}>
          Job Submission Log:
        </div>

        {publishedHistory.length === 0 ? (
          <div style={{ fontSize: '0.7rem', color: 'var(--text-dim)', fontStyle: 'italic', padding: '0.25rem 0' }}>
            No messages published yet in this session.
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            {publishedHistory.map((item, idx) => (
              <div 
                key={`${item.jobId}-${idx}`} 
                style={{ 
                  display: 'flex', 
                  justifyContent: 'space-between', 
                  alignItems: 'center',
                  fontSize: '0.68rem',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-secondary)',
                  background: 'rgba(255, 255, 255, 0.02)',
                  padding: '2px 6px',
                  borderRadius: '3px'
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                  <span style={{ color: '#10B981', fontWeight: 700 }}>[OK]</span>
                  <span style={{ color: 'var(--text-bright)', fontWeight: 600 }}>{item.jobId}</span>
                  <span style={{ color: 'var(--text-dim)' }}>({item.type})</span>
                </div>
                <span style={{ color: 'var(--text-dim)' }}>{item.timestamp}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
