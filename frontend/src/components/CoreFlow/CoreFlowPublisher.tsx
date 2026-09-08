import React, { useState, useMemo } from 'react';
import { Job } from '../../api/demoApi';

interface CoreFlowPublisherProps {
  onPublish: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  onShowInfo?: (key: string) => void;
}

interface CustomHeader {
  id: string;
  key: string;
  value: string;
}

const DEFAULT_PAYLOAD = {
  file: 'image-101.jpg',
  priority: 'normal',
};

interface PublishedReceipt {
  jobId: string;
  type: string;
  subject: string;
  deliveryMode: string;
  msgId: string;
  timestamp: string;
}

export const CoreFlowPublisher: React.FC<CoreFlowPublisherProps> = ({
  onPublish,
  isSubmitting,
  onShowInfo,
}) => {
  const [jobId, setJobId] = useState<string>(`job-${Math.floor(100 + Math.random() * 900)}`);
  const [subject, setSubject] = useState<string>('jobs.submitted');
  const deliveryMode = 'JETSTREAM';
  const [jobType, setJobType] = useState<string>('image-processing');
  const [natsMsgId, setNatsMsgId] = useState<string>(`msg-${Math.floor(100 + Math.random() * 900)}`);
  const [source, setSource] = useState<string>('job-service');
  const [contentType, setContentType] = useState<string>('application/json');
  const [customHeaders, setCustomHeaders] = useState<CustomHeader[]>([]);
  const [payloadStr, setPayloadStr] = useState<string>(JSON.stringify(DEFAULT_PAYLOAD, null, 2));
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [publishedHistory, setPublishedHistory] = useState<PublishedReceipt[]>([]);
  const [showAdvancedHeaders, setShowAdvancedHeaders] = useState<boolean>(false);
  const [isBatchPublishing, setIsBatchPublishing] = useState<boolean>(false);

  const handleBatchPublish = async (count = 6) => {
    setIsBatchPublishing(true);
    setJsonError(null);
    try {
      const parsed = JSON.parse(payloadStr);
      const baseNum = Math.floor(1000 + Math.random() * 9000);
      for (let i = 1; i <= count; i++) {
        const batchJobId = `job-${baseNum}-${i}`;
        const job: Job = {
          job_id: batchJobId,
          type: jobType.trim() || 'generic-message',
          subject: subject.trim() || 'jobs.submitted',
          delivery_mode: 'JETSTREAM',
          msg_id: `msg-${baseNum}-${i}`,
          source: source.trim() || 'job-service',
          content_type: contentType.trim() || 'application/json',
          headers: Object.keys(headersMap).length > 0 ? headersMap : undefined,
          payload: parsed,
        };
        await onPublish(job);
        const receipt: PublishedReceipt = {
          jobId: job.job_id,
          type: job.type || 'generic-message',
          subject: subject.trim() || 'jobs.submitted',
          deliveryMode: 'JETSTREAM',
          msgId: job.msg_id || job.job_id,
          timestamp: new Date().toLocaleTimeString(),
        };
        setPublishedHistory((prev) => [receipt, ...prev.slice(0, 7)]);
      }
    } catch (err: any) {
      setJsonError(err.message || 'Batch publish failed');
    } finally {
      setIsBatchPublishing(false);
    }
  };

  const regenerateIds = () => {
    const num = Math.floor(100 + Math.random() * 900);
    setJobId(`job-${num}`);
    setNatsMsgId(`msg-${num}`);
  };

  const addCustomHeader = () => {
    setCustomHeaders((prev) => [
      ...prev,
      { id: `hdr-${Date.now()}-${Math.random()}`, key: '', value: '' },
    ]);
  };

  const updateCustomHeader = (id: string, field: 'key' | 'value', val: string) => {
    setCustomHeaders((prev) =>
      prev.map((h) => (h.id === id ? { ...h, [field]: val } : h))
    );
  };

  const removeCustomHeader = (id: string) => {
    setCustomHeaders((prev) => prev.filter((h) => h.id !== id));
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
    setSubject('jobs.submitted');
    setCustomHeaders([]);
    setJsonError(null);
  };

  // Convert custom headers list into map
  const headersMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const h of customHeaders) {
      const k = h.key.trim();
      if (k) {
        map[k] = h.value;
      }
    }
    return map;
  }, [customHeaders]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    setJsonError(null);

    const trimmedSubject = subject.trim();
    if (!trimmedSubject) {
      setJsonError('Target Subject is required');
      return;
    }

    if (!jobId.trim()) {
      setJsonError('Message/Job ID is required');
      return;
    }

    try {
      const parsed = JSON.parse(payloadStr);
      const job: Job = {
        job_id: jobId.trim(),
        type: jobType.trim() || 'generic-message',
        subject: trimmedSubject,
        delivery_mode: deliveryMode,
        msg_id: natsMsgId.trim() || jobId.trim(),
        source: source.trim() || 'job-service',
        content_type: contentType.trim() || 'application/json',
        headers: Object.keys(headersMap).length > 0 ? headersMap : undefined,
        payload: parsed,
      };

      await onPublish(job);

      const receipt: PublishedReceipt = {
        jobId: job.job_id,
        type: job.type || 'generic-message',
        subject: trimmedSubject,
        deliveryMode,
        msgId: job.msg_id || job.job_id,
        timestamp: new Date().toLocaleTimeString(),
      };

      setPublishedHistory((prev) => [receipt, ...prev.slice(0, 7)]);

      // Auto-increment numeric suffix for consecutive test runs
      const match = jobId.match(/^(.*?)-(\d+)$/);
      if (match) {
        const nextNum = parseInt(match[2], 10) + 1;
        setJobId(`${match[1]}-${nextNum}`);
        setNatsMsgId(`msg-${nextNum}`);
      } else {
        const num = Math.floor(100 + Math.random() * 900);
        setJobId(`job-${num}`);
        setNatsMsgId(`msg-${num}`);
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
          <h2 className="panel-title" style={{ margin: 0 }}>Publish Message</h2>
        </div>
        {onShowInfo && (
          <button
            type="button"
            className="node-info-btn"
            onClick={() => onShowInfo('submit-job')}
            title="Learn about NATS Message Publishing &amp; Envelope Headers"
          >
            (i)
          </button>
        )}
      </div>

      <div style={{ padding: '0.75rem 1rem 0 1rem' }}>
        <p style={{ margin: '0 0 0.65rem 0', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          Dispatches messages to NATS JetStream. Messages are durably persisted to the <strong>JOBS</strong> stream on subject <code>jobs.submitted</code> for at-least-once processing.
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
          marginBottom: '0.65rem'
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
        
        {/* Target Subject & Presets */}
        <div style={{ marginBottom: '0.6rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.2rem' }}>
            <label className="form-label" style={{ fontSize: '0.72rem', margin: 0 }}>
              Target Subject
            </label>
            <div style={{ display: 'flex', gap: '0.3rem' }}>
              <button
                type="button"
                onClick={() => setSubject('jobs.submitted')}
                style={{
                  background: subject === 'jobs.submitted' ? 'rgba(59, 130, 246, 0.2)' : 'none',
                  border: '1px solid rgba(59, 130, 246, 0.3)',
                  borderRadius: '3px',
                  padding: '1px 5px',
                  fontSize: '0.62rem',
                  color: '#60A5FA',
                  cursor: 'pointer',
                }}
                title="Default stream subject"
              >
                Reset: jobs.submitted
              </button>
            </div>
          </div>
          <input
            type="text"
            className="form-input code"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="e.g. jobs.submitted"
            required
            style={{ padding: '0.38rem 0.6rem', fontSize: '0.75rem' }}
          />
        </div>

        {/* Message ID & Type Grid */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.65rem', marginBottom: '0.6rem' }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.2rem' }}>
              <label className="form-label" style={{ fontSize: '0.72rem', margin: 0 }}>Message / Job ID</label>
              <button
                type="button"
                onClick={regenerateIds}
                style={{
                  background: 'none',
                  border: 'none',
                  padding: 0,
                  fontSize: '0.65rem',
                  color: '#60A5FA',
                  cursor: 'pointer',
                  textDecoration: 'underline'
                }}
                title="Regenerate IDs"
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
              style={{ padding: '0.38rem 0.6rem', fontSize: '0.75rem' }}
            />
          </div>

          <div>
            <label className="form-label" style={{ fontSize: '0.72rem', marginBottom: '0.2rem' }}>Message Type</label>
            <input
              type="text"
              className="form-input"
              value={jobType}
              onChange={(e) => setJobType(e.target.value)}
              placeholder="e.g. image-processing"
              style={{ padding: '0.38rem 0.6rem', fontSize: '0.75rem' }}
            />
          </div>
        </div>

        {/* NATS Envelope Headers (Deduplication ID, Source, Content-Type) */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.65rem', marginBottom: '0.6rem' }}>
          <div>
            <label className="form-label" style={{ fontSize: '0.72rem', marginBottom: '0.2rem' }}>
              Header [Nats-Msg-Id]
            </label>
            <input
              type="text"
              className="form-input code"
              value={natsMsgId}
              onChange={(e) => setNatsMsgId(e.target.value)}
              placeholder="e.g. msg-101"
              title="Used by JetStream for deduplication within 2-min window"
              style={{ padding: '0.38rem 0.6rem', fontSize: '0.75rem' }}
            />
          </div>

          <div>
            <label className="form-label" style={{ fontSize: '0.72rem', marginBottom: '0.2rem' }}>
              Header [X-Source]
            </label>
            <input
              type="text"
              className="form-input code"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder="e.g. job-service"
              style={{ padding: '0.38rem 0.6rem', fontSize: '0.75rem' }}
            />
          </div>
        </div>

        {/* Custom Headers Extensibility Accordion */}
        <div style={{ marginBottom: '0.6rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <button
              type="button"
              onClick={() => setShowAdvancedHeaders(!showAdvancedHeaders)}
              style={{
                background: 'none',
                border: 'none',
                padding: '2px 0',
                fontSize: '0.7rem',
                color: '#60A5FA',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '0.3rem'
              }}
            >
              <span>{showAdvancedHeaders ? '[-]' : '[+]'}</span>
              <span>Custom Headers {customHeaders.length > 0 ? `(${customHeaders.length})` : ''}</span>
            </button>
            {showAdvancedHeaders && (
              <button
                type="button"
                onClick={addCustomHeader}
                style={{
                  background: 'rgba(59, 130, 246, 0.15)',
                  border: '1px solid rgba(59, 130, 246, 0.3)',
                  borderRadius: '3px',
                  padding: '1px 6px',
                  fontSize: '0.65rem',
                  color: '#93C5FD',
                  cursor: 'pointer'
                }}
              >
                + Add Header
              </button>
            )}
          </div>

          {showAdvancedHeaders && (
            <div style={{ 
              marginTop: '0.35rem', 
              padding: '0.45rem', 
              background: 'rgba(15, 23, 42, 0.5)', 
              borderRadius: '5px',
              border: '1px solid var(--border-color)'
            }}>
              {customHeaders.length === 0 ? (
                <div style={{ fontSize: '0.68rem', color: 'var(--text-dim)', fontStyle: 'italic' }}>
                  No custom headers added. Click "+ Add Header" to append key-value pairs to nats.Msg.Header.
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.3rem' }}>
                  {customHeaders.map((hdr) => (
                    <div key={hdr.id} style={{ display: 'flex', gap: '0.3rem', alignItems: 'center' }}>
                      <input
                        type="text"
                        placeholder="Header Key (e.g. X-Priority)"
                        className="form-input code"
                        value={hdr.key}
                        onChange={(e) => updateCustomHeader(hdr.id, 'key', e.target.value)}
                        style={{ flex: 1, padding: '0.25rem 0.45rem', fontSize: '0.68rem' }}
                      />
                      <input
                        type="text"
                        placeholder="Value (e.g. high)"
                        className="form-input code"
                        value={hdr.value}
                        onChange={(e) => updateCustomHeader(hdr.id, 'value', e.target.value)}
                        style={{ flex: 1, padding: '0.25rem 0.45rem', fontSize: '0.68rem' }}
                      />
                      <button
                        type="button"
                        onClick={() => removeCustomHeader(hdr.id)}
                        style={{
                          background: 'rgba(239, 68, 68, 0.15)',
                          border: '1px solid rgba(239, 68, 68, 0.3)',
                          color: '#F87171',
                          borderRadius: '3px',
                          padding: '2px 6px',
                          cursor: 'pointer',
                          fontSize: '0.68rem'
                        }}
                        title="Remove header"
                      >
                        x
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Payload JSON Editor */}
        <div style={{ marginBottom: '0.6rem', flex: 1, display: 'flex', flexDirection: 'column' }}>
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

        {/* Publish Action Buttons (Single Message & Concurrent Batch) */}
        <div style={{ display: 'flex', gap: '0.45rem' }}>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={isSubmitting || isBatchPublishing}
            style={{ 
              flex: 1,
              padding: '0.55rem 0.8rem', 
              fontSize: '0.78rem',
              fontWeight: 600,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '0.4rem',
              boxShadow: '0 0 16px rgba(37, 99, 235, 0.25)'
            }}
          >
            {isSubmitting ? (
              <span>Publishing...</span>
            ) : (
              <span>Publish Message -&gt;</span>
            )}
          </button>
          <button
            type="button"
            onClick={() => handleBatchPublish(6)}
            disabled={isSubmitting || isBatchPublishing}
            style={{
              padding: '0.55rem 0.85rem',
              fontSize: '0.76rem',
              fontWeight: 700,
              borderRadius: '6px',
              border: '1px solid #10B981',
              background: 'rgba(16, 185, 129, 0.15)',
              color: '#34D399',
              cursor: isSubmitting || isBatchPublishing ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '0.35rem',
              transition: 'all 0.15s ease',
              whiteSpace: 'nowrap'
            }}
            title="Publish a burst of 6 concurrent messages to demonstrate worker pool throughput and horizontal scaling"
          >
            {isBatchPublishing ? 'Publishing 6x...' : 'Batch (6 Jobs)'}
          </button>
        </div>
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
          Message Publication Log:
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
                  <span style={{ color: '#60A5FA' }}>[{item.subject}]</span>
                  <span style={{ color: 'var(--text-dim)' }}>({item.deliveryMode})</span>
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
