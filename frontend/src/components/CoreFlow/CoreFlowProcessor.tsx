import React, { useEffect, useState } from 'react';
import {
  Activity,
  ConsumerStatus,
  JetStreamInfo,
  ProcessorDirectEvent,
  ProcessorDirectStatus,
  clearProcessorDirectEvents,
  getProcessorDirectEvents,
  getProcessorDirectStatus,
  updateProcessorDirectState,
} from '../../api/demoApi';

export interface CoreFlowProcessorProps {
  activities?: Activity[];
  isProcessing?: boolean;
  onToggleProcessor?: (enabled: boolean) => Promise<void>;
  onClearActivity?: () => void;
  onSelectJob?: (jobId: string) => void;
  onShowInfo?: (key: string) => void;
  consumerStatus?: ConsumerStatus | null;
  jetstreamInfo?: JetStreamInfo | null;
}

export const CoreFlowProcessor: React.FC<CoreFlowProcessorProps> = ({
  onSelectJob,
  onShowInfo,
  consumerStatus,
  jetstreamInfo,
}) => {
  const [events, setEvents] = useState<ProcessorDirectEvent[]>([]);
  const [directStatus, setDirectStatus] = useState<ProcessorDirectStatus | null>(null);
  const [isProcessing, setIsProcessing] = useState<boolean>(true);
  const [isToggling, setIsToggling] = useState(false);
  const [isConnected, setIsConnected] = useState(true);

  // Poll direct processor HTTP API on port 8082
  const refreshDirectData = async () => {
    try {
      const [evts, status] = await Promise.all([
        getProcessorDirectEvents(),
        getProcessorDirectStatus(),
      ]);
      setEvents(evts);
      setDirectStatus(status);
      setIsProcessing(status.processing);
      setIsConnected(true);
    } catch {
      setIsConnected(false);
    }
  };

  useEffect(() => {
    refreshDirectData();
    const interval = setInterval(refreshDirectData, 1500);
    return () => clearInterval(interval);
  }, []);

  const handleToggle = async () => {
    setIsToggling(true);
    try {
      const nextState = !isProcessing;
      const res = await updateProcessorDirectState(nextState);
      setIsProcessing(res.processing);
      await refreshDirectData();
    } catch (err) {
      console.error('Failed to toggle processor state:', err);
    } finally {
      setIsToggling(false);
    }
  };

  const handleClear = async () => {
    try {
      await clearProcessorDirectEvents();
      setEvents([]);
    } catch (err) {
      console.error('Failed to clear processor events:', err);
    }
  };

  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const consumerName = directStatus?.consumer || consumerStatus?.name || 'job-processor';
  const streamName = directStatus?.stream || jetstreamInfo?.stream || 'JOBS';
  const workerCount = directStatus?.workers || consumerStatus?.workers || 1;

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
            onClick={handleClear}
            style={{ fontSize: '0.7rem', padding: '2px 8px', borderRadius: '4px' }}
            title="Clear processor execution log"
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

      <div style={{ padding: '0.75rem 1rem 0 1rem', display: 'flex', flexDirection: 'column', gap: '0.65rem' }}>
        {/* 1. Microservice Plane: processor-service (Application Level) */}
        <div style={{
          background: 'rgba(15, 23, 42, 0.75)',
          border: '1px solid var(--border-color)',
          borderRadius: '6px',
          padding: '0.65rem 0.75rem',
          fontSize: '0.72rem'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem' }}>
              <span style={{ 
                width: '7px', 
                height: '7px', 
                borderRadius: '50%', 
                background: isConnected ? (isProcessing ? '#10B981' : '#F59E0B') : '#F87171',
                boxShadow: isConnected && isProcessing ? '0 0 8px rgba(16, 185, 129, 0.8)' : 'none'
              }} />
              <strong style={{ color: 'var(--text-bright)', fontSize: '0.8rem' }}>
                processor-service
              </strong>
              <span style={{ 
                fontSize: '0.62rem', 
                color: 'var(--text-dim)', 
                background: 'rgba(255, 255, 255, 0.05)',
                padding: '1px 5px',
                borderRadius: '3px',
                fontFamily: 'var(--font-mono)'
              }}>
                :8082
              </span>
            </div>
            <span style={{ 
              background: isConnected ? (isProcessing ? 'rgba(16, 185, 129, 0.15)' : 'rgba(245, 158, 11, 0.15)') : 'rgba(239, 68, 68, 0.15)', 
              color: isConnected ? (isProcessing ? '#34D399' : '#FBBF24') : '#F87171', 
              padding: '1px 7px', 
              borderRadius: '4px',
              fontWeight: 600,
              fontSize: '0.65rem',
              letterSpacing: '0.04em'
            }}>
              {isConnected ? (isProcessing ? 'ACTIVE WORKER' : 'PAUSED WORKER') : 'OFFLINE'}
            </span>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.3rem', color: 'var(--text-muted)', fontSize: '0.7rem' }}>
            <div>
              Role: <strong style={{ color: 'var(--text-secondary)' }}>Go Worker Daemon</strong>
            </div>
            <div>
              Consuming From: <strong style={{ color: '#60A5FA' }}>job-processor (Pull)</strong>
            </div>
            <div>
              Worker Pool: <strong style={{ color: 'var(--text-secondary)' }}>{workerCount} Worker(s)</strong>
            </div>
            <div>
              HTTP Control Port: <strong style={{ color: '#34D399' }}>:8082 (Direct)</strong>
            </div>
          </div>

          {/* Modern Tactile Segmented Toggle Switch */}
          <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0.4rem 0.6rem',
            background: isProcessing ? 'rgba(16, 185, 129, 0.05)' : 'rgba(245, 158, 11, 0.05)',
            border: `1px solid ${isProcessing ? 'rgba(16, 185, 129, 0.2)' : 'rgba(245, 158, 11, 0.2)'}`,
            borderRadius: '5px',
            marginTop: '0.5rem',
            transition: 'all 0.2s ease'
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem' }}>
              <span style={{ 
                width: '6px', 
                height: '6px', 
                borderRadius: '50%', 
                background: isProcessing ? '#10B981' : '#F59E0B',
                boxShadow: isProcessing ? '0 0 6px rgba(16, 185, 129, 0.8)' : 'none',
                animation: isProcessing ? 'pulse 1.5s infinite' : 'none'
              }} />
              <div>
                <span style={{ fontSize: '0.74rem', fontWeight: 600, color: isProcessing ? '#34D399' : '#FBBF24' }}>
                  {isProcessing ? 'Worker Pull Loop: ACTIVE' : 'Worker Pull Loop: PAUSED'}
                </span>
                <div style={{ fontSize: '0.64rem', color: 'var(--text-muted)' }}>
                  {isProcessing 
                    ? 'Pulling and executing messages from NATS JetStream' 
                    : 'Paused. Inbound messages buffer safely in NATS stream'}
                </div>
              </div>
            </div>

            {/* Segmented Pill Toggle Buttons */}
            <div style={{
              display: 'inline-flex',
              background: '#0D1117',
              border: '1px solid #30363D',
              borderRadius: '16px',
              padding: '2px',
              gap: '2px'
            }}>
              <button
                type="button"
                disabled={isToggling || !isConnected}
                onClick={() => { if (!isProcessing) handleToggle(); }}
                style={{
                  border: 'none',
                  borderRadius: '12px',
                  padding: '2px 9px',
                  fontSize: '0.65rem',
                  fontWeight: 700,
                  cursor: isToggling || !isConnected ? 'not-allowed' : 'pointer',
                  background: isProcessing ? 'rgba(16, 185, 129, 0.25)' : 'transparent',
                  color: isProcessing ? '#34D399' : 'var(--text-dim)',
                  boxShadow: isProcessing ? '0 0 6px rgba(16, 185, 129, 0.25)' : 'none',
                  transition: 'all 0.15s ease'
                }}
              >
                ON
              </button>
              <button
                type="button"
                disabled={isToggling || !isConnected}
                onClick={() => { if (isProcessing) handleToggle(); }}
                style={{
                  border: 'none',
                  borderRadius: '12px',
                  padding: '2px 9px',
                  fontSize: '0.65rem',
                  fontWeight: 700,
                  cursor: isToggling || !isConnected ? 'not-allowed' : 'pointer',
                  background: !isProcessing ? 'rgba(239, 68, 68, 0.25)' : 'transparent',
                  color: !isProcessing ? '#F87171' : 'var(--text-dim)',
                  boxShadow: !isProcessing ? '0 0 6px rgba(239, 68, 68, 0.25)' : 'none',
                  transition: 'all 0.15s ease'
                }}
              >
                OFF
              </button>
            </div>
          </div>
        </div>

        {/* Visual Link showing how processor-service connects to the NATS Consumer */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '0.4rem',
          fontSize: '0.64rem',
          color: 'var(--text-dim)',
          margin: '-0.3rem 0'
        }}>
          <span>|</span>
          <span>Binds via NATS Go Client: <code>CreateOrUpdateConsumer("{consumerName}")</code></span>
          <span>|</span>
        </div>

        {/* 2. Broker Plane: NATS JetStream Consumer Object */}
        <div style={{
          background: 'rgba(15, 23, 42, 0.75)',
          border: '1px solid rgba(59, 130, 246, 0.2)',
          borderRadius: '6px',
          padding: '0.65rem 0.75rem',
          fontSize: '0.72rem'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.4rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem' }}>
              <span style={{ 
                width: '7px', 
                height: '7px', 
                borderRadius: '50%', 
                background: '#60A5FA'
              }} />
              <strong style={{ color: 'var(--text-bright)', fontSize: '0.8rem' }}>
                {consumerName}
              </strong>
              <span style={{ 
                fontSize: '0.62rem', 
                color: '#60A5FA', 
                background: 'rgba(59, 130, 246, 0.12)',
                padding: '1px 5px',
                borderRadius: '3px',
                fontFamily: 'var(--font-mono)'
              }}>
                NATS Object
              </span>
            </div>
            <span style={{ 
              background: 'rgba(59, 130, 246, 0.15)', 
              color: '#93C5FD', 
              padding: '1px 7px', 
              borderRadius: '4px',
              fontWeight: 600,
              fontSize: '0.65rem',
              letterSpacing: '0.04em'
            }}>
              {consumerType} CONSUMER
            </span>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.3rem', color: 'var(--text-muted)', fontSize: '0.7rem' }}>
            <div>
              Attached Stream: <strong style={{ color: '#34D399' }}>{streamName}</strong>
            </div>
            <div>
              Filter Subject: <strong style={{ color: '#60A5FA' }}>jobs.submitted</strong>
            </div>
            <div>
              Bound Worker: <strong style={{ color: '#60A5FA' }}>processor-service (Pull Loop)</strong>
            </div>
            <div>
              Broker Durability: <strong style={{ color: '#34D399' }}>Durable (Survives Restarts)</strong>
            </div>
          </div>
        </div>
      </div>

      {/* 3. Direct Processor Activity Log */}
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
          marginBottom: '0.3rem',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <span>Processor Activity Log (Direct :8082):</span>
          {events.length > 0 && (
            <span style={{ color: 'var(--text-muted)', fontSize: '0.62rem' }}>
              {events.length} event(s)
            </span>
          )}
        </div>

        {events.length === 0 ? (
          <div style={{ fontSize: '0.7rem', color: 'var(--text-dim)', fontStyle: 'italic', padding: '0.25rem 0' }}>
            {isConnected 
              ? 'No messages processed yet in this session.' 
              : 'Waiting for processor-service to connect on port 8082...'}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            {events.map((item, idx) => (
              <div 
                key={`${item.job_id}-${item.tag}-${item.timestamp}-${idx}`} 
                onClick={() => onSelectJob && onSelectJob(item.job_id)}
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
                  cursor: onSelectJob ? 'pointer' : 'default',
                  transition: 'background 0.15s ease'
                }}
                onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(255, 255, 255, 0.06)')}
                onMouseLeave={(e) => (e.currentTarget.style.background = 'rgba(255, 255, 255, 0.02)')}
                title={item.details || 'Processor event'}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                  <span style={{ color: item.tag_color || '#60A5FA', fontWeight: 700, minWidth: '85px', display: 'inline-block' }}>
                    {item.tag}
                  </span>
                  <span style={{ color: 'var(--text-bright)', fontWeight: 600 }}>{item.job_id}</span>
                  <span style={{ color: 'var(--text-dim)' }}>({item.job_type || 'default'})</span>
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
        <span>Direct telemetry from processor-service (:8082). Zero NATS pollution.</span>
        <span style={{ color: 'var(--text-muted)' }}>Consumer: {consumerName}</span>
      </div>
    </div>
  );
};
