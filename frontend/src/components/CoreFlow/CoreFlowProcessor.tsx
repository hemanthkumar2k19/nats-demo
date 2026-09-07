import React, { useMemo, useState } from 'react';
import { Activity, ConsumerStatus, JetStreamInfo } from '../../api/demoApi';

export interface CoreFlowProcessorProps {
  activities: Activity[];
  isProcessing: boolean;
  onToggleProcessor: (enabled: boolean) => Promise<void>;
  onClearActivity: () => void;
  onSelectJob: (jobId: string) => void;
  onShowInfo?: (key: string) => void;
  consumerStatus?: ConsumerStatus | null;
  jetstreamInfo?: JetStreamInfo | null;
}

interface ProcessorLogEvent {
  jobId: string;
  jobType: string;
  tag: string;
  tagColor: string;
  timestamp: string;
}

export const CoreFlowProcessor: React.FC<CoreFlowProcessorProps> = ({
  activities,
  isProcessing,
  onToggleProcessor,
  onClearActivity,
  onSelectJob,
  onShowInfo,
  consumerStatus,
  jetstreamInfo,
}) => {
  const [isToggling, setIsToggling] = useState(false);

  // Extract the 4 processor lifecycle events per message: Pulled -> Processing -> Completed -> ACK Sent
  const processorEvents = useMemo(() => {
    const events: ProcessorLogEvent[] = [];

    for (const act of activities) {
      if (!act.job_id) continue;

      const rawEvent = (act.event || '').toUpperCase();
      let tag = '';
      let tagColor = '';

      if (rawEvent.includes('ACK') && !rawEvent.includes('TIMEOUT')) {
        // Step 4: ACK Sent
        tag = '[ACK SENT]';
        tagColor = '#10B981';
      } else if (rawEvent.includes('COMPLETED')) {
        // Step 3: Completed
        tag = '[COMPLETED]';
        tagColor = '#34D399';
      } else if (rawEvent.includes('PROCESSING')) {
        // Step 2: Processing
        tag = '[PROCESSING]';
        tagColor = '#FBBF24';
      } else if (rawEvent.includes('DELIVER') || rawEvent.includes('RECEIVE')) {
        // Step 1: Pulled
        tag = '[PULLED]';
        tagColor = '#60A5FA';
      } else if (rawEvent.includes('NAK')) {
        tag = '[NAK SENT]';
        tagColor = '#F87171';
      } else if (rawEvent.includes('FAIL')) {
        tag = '[FAILED]';
        tagColor = '#EF4444';
      } else if (rawEvent.includes('TIMEOUT')) {
        tag = '[NO ACK / TIMEOUT]';
        tagColor = '#FB923C';
      } else {
        // Skip publisher/broker-only events (e.g. PUBLISHED, STORED, DEDUPLICATED)
        continue;
      }

      events.push({
        jobId: act.job_id,
        jobType: act.job_type || 'default',
        tag,
        tagColor,
        timestamp: act.timestamp,
      });

      if (events.length >= 40) break;
    }

    return events;
  }, [activities]);


  const handleToggle = async () => {
    setIsToggling(true);
    try {
      await onToggleProcessor(!isProcessing);
    } finally {
      setIsToggling(false);
    }
  };

  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const consumerName = consumerStatus?.name || 'job-processor';
  const streamName = jetstreamInfo?.stream || 'JOBS';
  const workerCount = consumerStatus?.workers || 1;

  return (
    <div className="panel" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Panel Header */}
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span style={{ 
            background: 'rgba(245, 158, 11, 0.15)', 
            color: '#FBBF24', 
            borderRadius: '4px', 
            padding: '2px 8px', 
            fontSize: '0.75rem', 
            fontWeight: 700,
            letterSpacing: '0.05em' 
          }}>
            STAGE 3
          </span>
          <h2 className="panel-title" style={{ margin: 0 }}>Processor View</h2>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <button
            type="button"
            className="btn-secondary"
            onClick={onClearActivity}
            style={{ fontSize: '0.7rem', padding: '2px 8px', borderRadius: '4px' }}
            title="Clear list"
          >
            Clear
          </button>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('processor-service')}
              title="Learn about Processor Service"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      <div style={{ padding: '0.75rem 1rem 0 1rem' }}>
        <p style={{ margin: '0 0 0.75rem 0', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          Business worker microservice. Consumes domain messages from NATS JetStream, executes business logic, and acknowledges messages.
        </p>

        {/* 1. Microservice Identity & NATS Consumer Card */}
        <div style={{
          background: 'rgba(15, 23, 42, 0.75)',
          border: '1px solid var(--border-color)',
          borderRadius: '6px',
          padding: '0.6rem 0.75rem',
          marginBottom: '0.75rem',
          fontSize: '0.72rem'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
              <span style={{ 
                width: '7px', 
                height: '7px', 
                borderRadius: '50%', 
                background: isProcessing ? '#10B981' : '#F87171' 
              }} />
              <strong style={{ color: 'var(--text-bright)', fontSize: '0.78rem' }}>
                processor-service
              </strong>
              <span style={{ color: 'var(--text-dim)', fontSize: '0.68rem' }}>
                (Go Microservice)
              </span>
            </div>
            <span style={{ 
              background: 'rgba(59, 130, 246, 0.15)', 
              color: '#93C5FD', 
              padding: '1px 6px', 
              borderRadius: '3px',
              fontWeight: 600,
              fontSize: '0.65rem'
            }}>
              {consumerType} CONSUMER
            </span>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.25rem', color: 'var(--text-muted)', fontSize: '0.7rem' }}>
            <div>
              Consumer Name: <strong style={{ color: 'var(--text-secondary)' }}>{consumerName}</strong>
            </div>
            <div>
              Attached Stream: <strong style={{ color: '#34D399' }}>{streamName}</strong>
            </div>
            <div>
              Durable on Broker: <strong style={{ color: consumerType === 'DURABLE' ? '#34D399' : '#FBBF24' }}>
                {consumerType === 'DURABLE' ? 'Yes (Survives Restarts)' : 'No (Ephemeral)'}
              </strong>
            </div>
            <div>
              Active Workers: <strong style={{ color: 'var(--text-secondary)' }}>{workerCount} Worker(s)</strong>
            </div>
          </div>
        </div>

        {/* 2. Active Status, ON/OFF Toggle & Processing Sign */}
        <div style={{
          background: isProcessing ? 'rgba(16, 185, 129, 0.08)' : 'rgba(239, 68, 68, 0.08)',
          border: `1px solid ${isProcessing ? 'rgba(16, 185, 129, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
          borderRadius: '6px',
          padding: '0.6rem 0.75rem',
          marginBottom: '0.75rem'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <span style={{ 
                  width: '8px', 
                  height: '8px', 
                  borderRadius: '50%', 
                  background: isProcessing ? '#10B981' : '#F87171',
                  boxShadow: isProcessing ? '0 0 10px rgba(16, 185, 129, 0.8)' : 'none'
                }} />
                <span style={{ fontSize: '0.78rem', fontWeight: 700, color: isProcessing ? '#34D399' : '#F87171' }}>
                  {isProcessing ? 'STATUS: ACTIVE (ON)' : 'STATUS: PAUSED (OFF)'}
                </span>
              </div>
              <div style={{ fontSize: '0.68rem', color: 'var(--text-muted)', marginTop: '2px' }}>
                {isProcessing 
                  ? 'Pull loop actively consuming & executing messages from JOBS stream' 
                  : 'Pull loop stopped. Messages buffer safely in NATS stream'}
              </div>
            </div>

            <button
              type="button"
              className={isProcessing ? 'btn-danger' : 'btn-success'}
              style={{ fontSize: '0.75rem', padding: '4px 12px', borderRadius: '4px', minWidth: '95px' }}
              onClick={handleToggle}
              disabled={isToggling}
            >
              {isToggling ? 'Updating...' : isProcessing ? 'Pause (OFF)' : 'Resume (ON)'}
            </button>
          </div>

          {/* Animated Processing Sign */}
          <div style={{ marginTop: '0.5rem', paddingTop: '0.4rem', borderTop: '1px solid rgba(255, 255, 255, 0.06)' }}>
            {isProcessing ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', fontSize: '0.7rem' }}>
                <span style={{ 
                  display: 'inline-block',
                  width: '6px',
                  height: '6px',
                  borderRadius: '50%',
                  background: '#10B981',
                  animation: 'pulse 1.5s infinite'
                }} />
                <span style={{ color: '#34D399', fontWeight: 600 }}>
                  PROCESSING ENGINE: LISTENING
                </span>
                <span style={{ color: 'var(--text-dim)' }}>
                  - Ready to pull &amp; process incoming messages
                </span>
              </div>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', fontSize: '0.7rem' }}>
                <span style={{ color: '#F87171', fontWeight: 600 }}>
                  [PAUSED]
                </span>
                <span style={{ color: 'var(--text-dim)' }}>
                  Processing halted. Messages will accumulate in stream.
                </span>
              </div>
            )}
          </div>
        </div>


      </div>

      {/* 3. Simple Processor Activity Log (Clean one-line rows matching Publisher format) */}
      <div style={{ 
        flex: 1, 
        borderTop: '1px solid var(--border-color)', 
        padding: '0.5rem 1rem', 
        background: 'rgba(0, 0, 0, 0.25)',
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
          Processor Activity Log:
        </div>

        {processorEvents.length === 0 ? (
          <div style={{ fontSize: '0.7rem', color: 'var(--text-dim)', fontStyle: 'italic', padding: '0.25rem 0' }}>
            No messages processed yet in this session.
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            {processorEvents.map((item, idx) => (
              <div 
                key={`${item.jobId}-${item.tag}-${item.timestamp}-${idx}`} 
                onClick={() => onSelectJob(item.jobId)}
                style={{ 
                  display: 'flex', 
                  justifyContent: 'space-between', 
                  alignItems: 'center',
                  fontSize: '0.68rem',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-secondary)',
                  background: 'rgba(255, 255, 255, 0.02)',
                  padding: '2px 6px',
                  borderRadius: '3px',
                  cursor: 'pointer',
                  transition: 'background 0.15s ease'
                }}
                onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(255, 255, 255, 0.06)')}
                onMouseLeave={(e) => (e.currentTarget.style.background = 'rgba(255, 255, 255, 0.02)')}
                title="Click to inspect full job details in Inspector"
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                  <span style={{ color: item.tagColor, fontWeight: 700, minWidth: '85px', display: 'inline-block' }}>
                    {item.tag}
                  </span>
                  <span style={{ color: 'var(--text-bright)', fontWeight: 600 }}>{item.jobId}</span>
                  <span style={{ color: 'var(--text-dim)' }}>({item.jobType})</span>
                </div>
                <span style={{ color: 'var(--text-dim)' }}>{item.timestamp}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Footer Info */}
      <div style={{ 
        borderTop: '1px solid var(--border-color)', 
        padding: '0.45rem 1rem', 
        background: 'rgba(0, 0, 0, 0.2)',
        fontSize: '0.65rem',
        color: 'var(--text-dim)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <span>Processor Service consumes from JOBS stream via pull consumer.</span>
        <span style={{ color: 'var(--text-muted)' }}>Consumer: {consumerName}</span>
      </div>
    </div>
  );
};
