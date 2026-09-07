import React, { useEffect, useState } from 'react';
import {
  Activity,
  ConsumerStatus,
  FailureScenario,
  JetStreamInfo,
  ProcessorDirectStatus,
  getProcessorDirectStatus,
  restartProcessorWorker,
  updateProcessorDirectState,
  updateProcessorScenario,
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
  onShowInfo,
  consumerStatus,
  jetstreamInfo,
}) => {
  const [directStatus, setDirectStatus] = useState<ProcessorDirectStatus | null>(null);
  const [isProcessing, setIsProcessing] = useState<boolean>(true);
  const [isToggling, setIsToggling] = useState(false);
  const [isConnected, setIsConnected] = useState(true);
  const [isSettingScenario, setIsSettingScenario] = useState(false);
  const [copiedCli, setCopiedCli] = useState(false);
  const [isFailureLabOpen, setIsFailureLabOpen] = useState<boolean>(true);

  // Poll direct processor HTTP API on port 8082
  const refreshDirectData = async () => {
    try {
      const status = await getProcessorDirectStatus();
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

  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const consumerName = directStatus?.consumer || consumerStatus?.name || 'job-processor';
  const streamName = directStatus?.stream || jetstreamInfo?.stream || 'JOBS';
  const workerCount = directStatus?.workers || consumerStatus?.workers || 1;
  const activeScenario = directStatus?.scenario || 'normal';
  const goroutineStatus = directStatus?.goroutine_status || (isProcessing ? 'RUNNING' : 'PAUSED');
  const activeGoroutines = directStatus?.active_goroutines ?? (isProcessing ? 1 : 0);
  const ackWaitSeconds = directStatus?.ack_wait_seconds || 5;

  const handleSetScenario = async (sc: FailureScenario) => {
    setIsSettingScenario(true);
    try {
      await updateProcessorScenario(sc, true);
      await refreshDirectData();
    } catch (err) {
      console.error('Failed to set processor scenario:', err);
    } finally {
      setIsSettingScenario(false);
    }
  };

  const handleRestart = async () => {
    try {
      await restartProcessorWorker();
      await refreshDirectData();
    } catch (err) {
      console.error('Failed to restart worker:', err);
    }
  };

  const handleCopyCli = () => {
    navigator.clipboard.writeText(`nats consumer info ${streamName} ${consumerName}`);
    setCopiedCli(true);
    setTimeout(() => setCopiedCli(false), 2000);
  };

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
              Consuming From: <strong style={{ color: '#60A5FA' }}>{consumerName} (Pull)</strong>
            </div>
            <div>
              Worker Pool: <strong style={{ color: 'var(--text-secondary)' }}>{workerCount} Worker(s)</strong>
            </div>
            <div>
              HTTP Control Port: <strong style={{ color: '#34D399' }}>:8082 (Direct)</strong>
            </div>
            <div style={{ gridColumn: 'span 2', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderTop: '1px dashed rgba(255, 255, 255, 0.08)', paddingTop: '0.25rem', marginTop: '0.1rem' }}>
              <span>
                Goroutine State: <strong style={{ color: goroutineStatus === 'CRASHED' ? '#EF4444' : (isProcessing ? '#34D399' : '#F59E0B') }}>
                  {goroutineStatus} ({activeGoroutines} active)
                </strong>
              </span>
              {goroutineStatus === 'CRASHED' && (
                <button
                  type="button"
                  onClick={handleRestart}
                  style={{
                    background: 'rgba(139, 92, 246, 0.2)',
                    border: '1px solid #8B5CF6',
                    color: '#C4B5FD',
                    fontSize: '0.62rem',
                    padding: '1px 6px',
                    borderRadius: '3px',
                    cursor: 'pointer'
                  }}
                  title="Manually respawn worker goroutine"
                >
                  Revive Goroutine Now
                </button>
              )}
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
              Ack Policy: <strong style={{ color: '#34D399' }}>Explicit (msg.Ack())</strong>
            </div>
            <div>
              AckWait Threshold: <strong style={{ color: '#FBBF24' }}>{ackWaitSeconds}s (Redelivery Timer)</strong>
            </div>
            <div>
              Bound Worker: <strong style={{ color: '#60A5FA' }}>processor-service (Pull Loop)</strong>
            </div>
            <div>
              Broker Durability: <strong style={{ color: '#34D399' }}>Durable (Survives Restarts)</strong>
            </div>
          </div>
        </div>

        {/* 3. Durable Consumer Failure Lab (Collapsible & Vertically Arranged) */}
        <div style={{
          background: 'rgba(15, 23, 42, 0.85)',
          border: '1px solid rgba(239, 68, 68, 0.25)',
          borderRadius: '6px',
          overflow: 'hidden',
          fontSize: '0.72rem'
        }}>
          {/* Collapsible Header */}
          <div 
            onClick={() => setIsFailureLabOpen(!isFailureLabOpen)}
            style={{ 
              display: 'flex', 
              justifyContent: 'space-between', 
              alignItems: 'center', 
              padding: '0.55rem 0.75rem',
              background: 'rgba(239, 68, 68, 0.08)',
              cursor: 'pointer',
              userSelect: 'none',
              borderBottom: isFailureLabOpen ? '1px solid rgba(239, 68, 68, 0.18)' : 'none',
              transition: 'background 0.15s ease'
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem' }}>
              <span style={{ 
                background: 'rgba(239, 68, 68, 0.2)', 
                color: '#F87171', 
                borderRadius: '3px', 
                padding: '1px 5px', 
                fontSize: '0.65rem', 
                fontWeight: 700 
              }}>
                FAILURE LAB
              </span>
              <strong style={{ color: 'var(--text-bright)', fontSize: '0.76rem' }}>
                Goroutine Failure Scenarios
              </strong>
            </div>

            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <span style={{
                background: activeScenario === 'normal' ? 'rgba(255, 255, 255, 0.05)' : 'rgba(245, 158, 11, 0.15)',
                color: activeScenario === 'normal' ? 'var(--text-dim)' : '#FBBF24',
                padding: '1px 6px',
                borderRadius: '4px',
                fontSize: '0.62rem',
                fontWeight: 600
              }}>
                {activeScenario === 'normal' ? 'MODE: NORMAL' : `ARMED: ${activeScenario.toUpperCase()}`}
              </span>
              <span style={{ 
                color: 'var(--text-dim)', 
                fontSize: '0.72rem', 
                fontFamily: 'monospace',
                fontWeight: 700 
              }}>
                {isFailureLabOpen ? '[-]' : '[+]'}
              </span>
            </div>
          </div>

          {/* Collapsible Body */}
          {isFailureLabOpen && (
            <div style={{ padding: '0.65rem 0.75rem', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {/* Vertical Scenario List */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.45rem' }}>
                {/* Scenario 1: Worker Crash Before ACK */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '0.45rem 0.6rem',
                  background: activeScenario === 'crash_before_ack' ? 'rgba(239, 68, 68, 0.12)' : 'rgba(255, 255, 255, 0.02)',
                  border: `1px solid ${activeScenario === 'crash_before_ack' ? 'rgba(239, 68, 68, 0.4)' : 'rgba(255, 255, 255, 0.06)'}`,
                  borderRadius: '5px',
                  gap: '0.6rem'
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                      <strong style={{ color: activeScenario === 'crash_before_ack' ? '#F87171' : 'var(--text-bright)', fontSize: '0.73rem' }}>
                        1. Worker Crash Before ACK
                      </strong>
                      {activeScenario === 'crash_before_ack' && (
                        <span style={{ background: '#EF4444', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          ARMED
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                      Simulates unhandled goroutine panic before ACK. Broker holds message for 5s AckWait, then redelivers to respawned worker.
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={isSettingScenario}
                    onClick={() => handleSetScenario(activeScenario === 'crash_before_ack' ? 'normal' : 'crash_before_ack')}
                    style={{
                      padding: '4px 9px',
                      fontSize: '0.65rem',
                      fontWeight: 600,
                      borderRadius: '4px',
                      border: activeScenario === 'crash_before_ack' ? '1px solid #EF4444' : '1px solid var(--border-color)',
                      background: activeScenario === 'crash_before_ack' ? 'rgba(239, 68, 68, 0.25)' : 'rgba(255, 255, 255, 0.05)',
                      color: activeScenario === 'crash_before_ack' ? '#FCA5A5' : 'var(--text-secondary)',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap'
                    }}
                  >
                    {activeScenario === 'crash_before_ack' ? 'Disarm' : 'Arm Crash'}
                  </button>
                </div>

                {/* Scenario 2: Exceed AckWait Threshold */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '0.45rem 0.6rem',
                  background: activeScenario === 'exceed_ack_wait' ? 'rgba(245, 158, 11, 0.12)' : 'rgba(255, 255, 255, 0.02)',
                  border: `1px solid ${activeScenario === 'exceed_ack_wait' ? 'rgba(245, 158, 11, 0.4)' : 'rgba(255, 255, 255, 0.06)'}`,
                  borderRadius: '5px',
                  gap: '0.6rem'
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                      <strong style={{ color: activeScenario === 'exceed_ack_wait' ? '#FBBF24' : 'var(--text-bright)', fontSize: '0.73rem' }}>
                        2. Exceed AckWait Threshold (7s)
                      </strong>
                      {activeScenario === 'exceed_ack_wait' && (
                        <span style={{ background: '#F59E0B', color: '#000', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          ARMED
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                      Simulates 7s execution exceeding 5s AckWait. Broker timer expires, followed by late explicit ACK.
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={isSettingScenario}
                    onClick={() => handleSetScenario(activeScenario === 'exceed_ack_wait' ? 'normal' : 'exceed_ack_wait')}
                    style={{
                      padding: '4px 9px',
                      fontSize: '0.65rem',
                      fontWeight: 600,
                      borderRadius: '4px',
                      border: activeScenario === 'exceed_ack_wait' ? '1px solid #F59E0B' : '1px solid var(--border-color)',
                      background: activeScenario === 'exceed_ack_wait' ? 'rgba(245, 158, 11, 0.25)' : 'rgba(255, 255, 255, 0.05)',
                      color: activeScenario === 'exceed_ack_wait' ? '#FDE68A' : 'var(--text-secondary)',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap'
                    }}
                  >
                    {activeScenario === 'exceed_ack_wait' ? 'Disarm' : 'Arm Exceed'}
                  </button>
                </div>

                {/* Scenario 4: Worker NAKs Message */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '0.45rem 0.6rem',
                  background: activeScenario === 'nak_message' ? 'rgba(248, 113, 113, 0.12)' : 'rgba(255, 255, 255, 0.02)',
                  border: `1px solid ${activeScenario === 'nak_message' ? 'rgba(248, 113, 113, 0.4)' : 'rgba(255, 255, 255, 0.06)'}`,
                  borderRadius: '5px',
                  gap: '0.6rem'
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                      <strong style={{ color: activeScenario === 'nak_message' ? '#F87171' : 'var(--text-bright)', fontSize: '0.73rem' }}>
                        4. Worker NAKs Message (msg.Nak())
                      </strong>
                      {activeScenario === 'nak_message' && (
                        <span style={{ background: '#EF4444', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          ARMED
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                      Worker detects transient failure and sends explicit NAK. Broker immediately queues message for redelivery (Attempt #2 succeeds).
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={isSettingScenario}
                    onClick={() => handleSetScenario(activeScenario === 'nak_message' ? 'normal' : 'nak_message')}
                    style={{
                      padding: '4px 9px',
                      fontSize: '0.65rem',
                      fontWeight: 600,
                      borderRadius: '4px',
                      border: activeScenario === 'nak_message' ? '1px solid #EF4444' : '1px solid var(--border-color)',
                      background: activeScenario === 'nak_message' ? 'rgba(239, 68, 68, 0.25)' : 'rgba(255, 255, 255, 0.05)',
                      color: activeScenario === 'nak_message' ? '#FCA5A5' : 'var(--text-secondary)',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap'
                    }}
                  >
                    {activeScenario === 'nak_message' ? 'Disarm' : 'Arm NAK'}
                  </button>
                </div>

                {/* Scenario 5: Worker Terminates Message */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '0.45rem 0.6rem',
                  background: activeScenario === 'term_message' ? 'rgba(220, 38, 38, 0.15)' : 'rgba(255, 255, 255, 0.02)',
                  border: `1px solid ${activeScenario === 'term_message' ? 'rgba(220, 38, 38, 0.4)' : 'rgba(255, 255, 255, 0.06)'}`,
                  borderRadius: '5px',
                  gap: '0.6rem'
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                      <strong style={{ color: activeScenario === 'term_message' ? '#F87171' : 'var(--text-bright)', fontSize: '0.73rem' }}>
                        5. Worker Terminates Message (msg.Term())
                      </strong>
                      {activeScenario === 'term_message' && (
                        <span style={{ background: '#DC2626', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          ARMED
                        </span>
                      )}
                    </div>
                    <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                      Worker detects poison message and calls msg.Term(). Broker marks permanently terminated; no redelivery; ACK floor advances.
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={isSettingScenario}
                    onClick={() => handleSetScenario(activeScenario === 'term_message' ? 'normal' : 'term_message')}
                    style={{
                      padding: '4px 9px',
                      fontSize: '0.65rem',
                      fontWeight: 600,
                      borderRadius: '4px',
                      border: activeScenario === 'term_message' ? '1px solid #DC2626' : '1px solid var(--border-color)',
                      background: activeScenario === 'term_message' ? 'rgba(220, 38, 38, 0.25)' : 'rgba(255, 255, 255, 0.05)',
                      color: activeScenario === 'term_message' ? '#FCA5A5' : 'var(--text-secondary)',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap'
                    }}
                  >
                    {activeScenario === 'term_message' ? 'Disarm' : 'Arm TERM'}
                  </button>
                </div>

                {/* Scenario 6: Consumer State Retention Across Restart */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '0.45rem 0.6rem',
                  background: 'rgba(59, 130, 246, 0.06)',
                  border: '1px solid rgba(59, 130, 246, 0.2)',
                  borderRadius: '5px',
                  gap: '0.6rem'
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                      <strong style={{ color: '#93C5FD', fontSize: '0.73rem' }}>
                        6. Durable State Retention Across Restart
                      </strong>
                      <span style={{ background: 'rgba(59, 130, 246, 0.2)', color: '#60A5FA', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                        STATE
                      </span>
                    </div>
                    <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                      Durable state is preserved in NATS. Previously ACKed jobs are never reprocessed; only pending backlog messages are delivered upon reconnect.
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={handleRestart}
                    style={{
                      padding: '4px 9px',
                      fontSize: '0.65rem',
                      fontWeight: 600,
                      borderRadius: '4px',
                      border: '1px solid rgba(59, 130, 246, 0.4)',
                      background: 'rgba(59, 130, 246, 0.15)',
                      color: '#93C5FD',
                      cursor: 'pointer',
                      whiteSpace: 'nowrap'
                    }}
                    title="Reconnects worker to existing durable consumer object in NATS"
                  >
                    Restart Worker
                  </button>
                </div>
              </div>

              {/* Reset to Normal Mode */}
              {activeScenario !== 'normal' && (
                <div style={{ display: 'flex', justifyContent: 'flex-end', paddingTop: '0.1rem' }}>
                  <button
                    type="button"
                    disabled={isSettingScenario}
                    onClick={() => handleSetScenario('normal')}
                    style={{
                      padding: '3px 8px',
                      fontSize: '0.63rem',
                      fontWeight: 600,
                      borderRadius: '4px',
                      border: '1px solid var(--border-color)',
                      background: 'transparent',
                      color: 'var(--text-muted)',
                      cursor: 'pointer'
                    }}
                  >
                    Reset to Normal Mode
                  </button>
                </div>
              )}

              {/* CLI Command Helper */}
              <div style={{
                fontSize: '0.63rem',
                color: 'var(--text-dim)',
                background: 'rgba(0, 0, 0, 0.3)',
                padding: '0.35rem 0.5rem',
                borderRadius: '4px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center'
              }}>
                <span>
                  CLI Check: <code>nats consumer info {streamName} {consumerName}</code>
                </span>
                <button
                  type="button"
                  onClick={handleCopyCli}
                  style={{
                    background: 'transparent',
                    border: 'none',
                    color: copiedCli ? '#34D399' : '#60A5FA',
                    fontSize: '0.62rem',
                    cursor: 'pointer',
                    padding: '0 4px'
                  }}
                >
                  {copiedCli ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          )}
        </div>
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
        alignItems: 'center',
        marginTop: 'auto'
      }}>
        <span>Worker daemon (:8082). Showing authentic stdout logs in shell console.</span>
        <span style={{ color: 'var(--text-muted)' }}>Consumer: {consumerName}</span>
      </div>
    </div>
  );
};
