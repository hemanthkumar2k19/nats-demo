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
  updateProcessorWorkers,
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
  const [isFailureLabOpen, setIsFailureLabOpen] = useState<boolean>(false);
  const [isScaling, setIsScaling] = useState(false);
  const [failureLabTab, setFailureLabTab] = useState<'partA' | 'partB'>('partA');

  const handleScaleWorkers = async (newCount: number) => {
    if (newCount === workerCount || isScaling) return;
    setIsScaling(true);
    try {
      await updateProcessorWorkers(newCount);
      await refreshDirectData();
    } catch (err) {
      console.error('Failed to scale workers:', err);
    } finally {
      setIsScaling(false);
    }
  };

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
  const activeGoroutines = directStatus?.active_goroutines ?? (isProcessing ? workerCount : 0);
  const crashedWorker = directStatus?.crashed_worker || '';
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
        {/* 1. Broker Plane: NATS JetStream Consumer Object */}
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
                JetStream Object
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

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.45rem', marginTop: '0.2rem' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Attached Stream
              </label>
              <input
                type="text"
                readOnly
                value={streamName}
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#34D399',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Filter Subject
              </label>
              <input
                type="text"
                readOnly
                value="jobs.submitted"
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#60A5FA',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Ack Policy
              </label>
              <input
                type="text"
                readOnly
                value="Explicit (msg.Ack())"
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#34D399',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                AckWait Threshold
              </label>
              <input
                type="text"
                readOnly
                value={`${ackWaitSeconds}s (Redelivery Timer)`}
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#FBBF24',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Ack Floor (Watermark)
              </label>
              <input
                type="text"
                readOnly
                value={`Stream Seq ${directStatus?.ack_floor ?? consumerStatus?.ack_floor ?? 0}`}
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#60A5FA',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Delivered Sequence
              </label>
              <input
                type="text"
                readOnly
                value={`Stream Seq ${directStatus?.delivered_seq ?? consumerStatus?.delivered_seq ?? 0}`}
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#A78BFA',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
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

        {/* 2. Microservice Plane: processor-service (Application Level) */}
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
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.45rem', marginTop: '0.2rem' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Role
              </label>
              <input
                type="text"
                readOnly
                value="Go Worker Daemon"
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                Consuming From
              </label>
              <input
                type="text"
                readOnly
                value={`${consumerName} (Pull)`}
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: '1px solid rgba(255, 255, 255, 0.08)',
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color: '#60A5FA',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
            </div>

            {/* Dynamic Worker Pool Scaling Stepper */}
            <div style={{ gridColumn: 'span 2', display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'rgba(255, 255, 255, 0.03)', padding: '0.35rem 0.5rem', borderRadius: '4px', marginTop: '0.1rem', border: '1px solid rgba(255, 255, 255, 0.05)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                <span style={{ fontWeight: 600, color: 'var(--text-bright)', fontSize: '0.69rem' }}>
                  Worker Pool Size:
                </span>
                <span style={{ color: '#60A5FA', fontWeight: 700, fontSize: '0.7rem' }}>
                  {workerCount} Concurrent Worker(s)
                </span>
              </div>
              <div style={{ display: 'flex', gap: '3px', alignItems: 'center' }}>
                {[1, 2, 3, 5].map((count) => (
                  <button
                    key={count}
                    type="button"
                    disabled={isScaling}
                    onClick={() => handleScaleWorkers(count)}
                    style={{
                      border: count === workerCount ? '1px solid #60A5FA' : '1px solid var(--border-color)',
                      background: count === workerCount ? 'rgba(59, 130, 246, 0.25)' : 'rgba(255, 255, 255, 0.04)',
                      color: count === workerCount ? '#93C5FD' : 'var(--text-dim)',
                      borderRadius: '3px',
                      padding: '1px 6px',
                      fontSize: '0.65rem',
                      fontWeight: 700,
                      cursor: isScaling ? 'not-allowed' : 'pointer',
                      transition: 'all 0.15s ease'
                    }}
                    title={`Scale JetStream worker pool to ${count} worker goroutines`}
                  >
                    {count}W
                  </button>
                ))}
              </div>
            </div>

            <div style={{ gridColumn: 'span 2', display: 'flex', flexDirection: 'column', gap: '0.15rem' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <label style={{ fontSize: '0.61rem', color: 'var(--text-dim)', textTransform: 'uppercase', letterSpacing: '0.04em', fontWeight: 600 }}>
                  Goroutine State
                </label>
                {(goroutineStatus === 'CRASHED' || goroutineStatus === 'DEGRADED') && (
                  <button
                    type="button"
                    onClick={handleRestart}
                    style={{
                      background: 'rgba(139, 92, 246, 0.2)',
                      border: '1px solid #8B5CF6',
                      color: '#C4B5FD',
                      fontSize: '0.6rem',
                      padding: '1px 6px',
                      borderRadius: '3px',
                      cursor: 'pointer'
                    }}
                    title="Manually respawn crashed worker goroutine"
                  >
                    Revive Goroutine Now
                  </button>
                )}
              </div>
              <input
                type="text"
                readOnly
                value={
                  crashedWorker
                    ? `${goroutineStatus} (${activeGoroutines}/${workerCount} active - ${crashedWorker} crashed)`
                    : `${goroutineStatus} (${activeGoroutines}/${workerCount} active)`
                }
                style={{
                  background: 'rgba(0, 0, 0, 0.35)',
                  border: `1px solid ${
                    goroutineStatus === 'CRASHED'
                      ? 'rgba(239, 68, 68, 0.5)'
                      : goroutineStatus === 'DEGRADED'
                      ? 'rgba(245, 158, 11, 0.5)'
                      : 'rgba(255, 255, 255, 0.08)'
                  }`,
                  borderRadius: '4px',
                  padding: '0.3rem 0.5rem',
                  fontSize: '0.68rem',
                  color:
                    goroutineStatus === 'CRASHED'
                      ? '#EF4444'
                      : goroutineStatus === 'DEGRADED'
                      ? '#FBBF24'
                      : isProcessing
                      ? '#34D399'
                      : '#F59E0B',
                  fontFamily: 'var(--font-mono)',
                  outline: 'none',
                  cursor: 'default',
                  width: '100%',
                  boxSizing: 'border-box'
                }}
              />
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
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                style={{
                  color: 'var(--text-dim)',
                  transform: isFailureLabOpen ? 'rotate(180deg)' : 'rotate(0deg)',
                  transition: 'transform 0.2s ease'
                }}
              >
                <path d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>

          {/* Collapsible Body */}
          {isFailureLabOpen && (
            <div style={{ padding: '0.65rem 0.75rem', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {/* Tabs Switcher: Part A (Single-Worker) vs Part B (Multi-Worker) */}
              <div style={{ display: 'flex', gap: '0.4rem', borderBottom: '1px solid rgba(255, 255, 255, 0.08)', paddingBottom: '0.45rem' }}>
                <button
                  type="button"
                  onClick={() => setFailureLabTab('partA')}
                  style={{
                    flex: 1,
                    padding: '5px 8px',
                    fontSize: '0.68rem',
                    fontWeight: failureLabTab === 'partA' ? 700 : 500,
                    background: failureLabTab === 'partA' ? 'rgba(59, 130, 246, 0.18)' : 'rgba(255, 255, 255, 0.02)',
                    border: `1px solid ${failureLabTab === 'partA' ? 'rgba(59, 130, 246, 0.5)' : 'rgba(255, 255, 255, 0.06)'}`,
                    color: failureLabTab === 'partA' ? '#93C5FD' : 'var(--text-dim)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: '0.35rem',
                    transition: 'all 0.15s ease'
                  }}
                >
                  <span>Part A: Single-Worker (Steps 1-5)</span>
                  {activeScenario === 'out_of_order_ack' && (
                    <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: '#3B82F6' }} />
                  )}
                </button>
                <button
                  type="button"
                  onClick={() => setFailureLabTab('partB')}
                  style={{
                    flex: 1,
                    padding: '5px 8px',
                    fontSize: '0.68rem',
                    fontWeight: failureLabTab === 'partB' ? 700 : 500,
                    background: failureLabTab === 'partB' ? 'rgba(239, 68, 68, 0.18)' : 'rgba(255, 255, 255, 0.02)',
                    border: `1px solid ${failureLabTab === 'partB' ? 'rgba(239, 68, 68, 0.5)' : 'rgba(255, 255, 255, 0.06)'}`,
                    color: failureLabTab === 'partB' ? '#FCA5A5' : 'var(--text-dim)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: '0.35rem',
                    transition: 'all 0.15s ease'
                  }}
                >
                  <span>Part B: Multi-Worker (Steps 6-12)</span>
                  {['crash_before_ack', 'exceed_ack_wait', 'nak_message', 'term_message'].includes(activeScenario) && (
                    <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: '#EF4444' }} />
                  )}
                </button>
              </div>

              {/* Tab 1: Part A (Single-Worker Scenarios: Steps 1-5) */}
              {failureLabTab === 'partA' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.45rem' }}>
                  {/* Step 1: Baseline Ingestion & Storage */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          1. Baseline Ingestion &amp; Storage
                        </strong>
                        <span style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60A5FA', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Click <strong>Publish Message</strong> in Stage 1. Commits record to disk under <code>JOBS</code> stream; routes to <code>job-processor</code> consumer.
                      </div>
                    </div>
                  </div>

                  {/* Step 2: Server Deduplication */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          2. Server Deduplication (Nats-Msg-Id)
                        </strong>
                        <span style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60A5FA', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Keep same <code>Nats-Msg-Id</code> and click <strong>Publish Message</strong> again. Checks 2-min cache; broker suppresses duplicate disk write.
                      </div>
                    </div>
                  </div>

                  {/* Step 3: Decoupled Ingestion & Buffering */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          3. Decoupled Ingestion &amp; Buffering
                        </strong>
                        <span style={{ background: 'rgba(245, 158, 11, 0.15)', color: '#FBBF24', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Toggle Worker Pull Loop to <strong>OFF</strong> above, then click <strong>Publish Message</strong> 3 times. Messages buffer safely on disk in broker.
                      </div>
                    </div>
                  </div>

                  {/* Step 4: Sequential Catch-up Drain */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          4. Sequential Catch-up Drain
                        </strong>
                        <span style={{ background: 'rgba(16, 185, 129, 0.15)', color: '#34D399', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Toggle Worker Pull Loop back to <strong>ON</strong> above. Fetches unacknowledged messages sequentially starting from <code>AckFloor + 1</code>.
                      </div>
                    </div>
                  </div>

                  {/* Step 5: Out-of-Order ACK & Hole Safety */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: activeScenario === 'out_of_order_ack' ? 'rgba(59, 130, 246, 0.12)' : 'rgba(255, 255, 255, 0.02)',
                    border: `1px solid ${activeScenario === 'out_of_order_ack' ? 'rgba(59, 130, 246, 0.4)' : 'rgba(255, 255, 255, 0.06)'}`,
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: activeScenario === 'out_of_order_ack' ? '#60A5FA' : 'var(--text-bright)', fontSize: '0.73rem' }}>
                          5. Out-of-Order ACK &amp; Hole Safety
                        </strong>
                        {activeScenario === 'out_of_order_ack' && (
                          <span style={{ background: '#3B82F6', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                            ARMED
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Pulls 3 messages. Msg 1 &amp; 3 ACKed immediately while Msg 2 delays 10s. AckFloor stays pinned at 1 until Msg 2 resolves.
                      </div>
                    </div>
                    <button
                      type="button"
                      disabled={isSettingScenario}
                      onClick={() => handleSetScenario(activeScenario === 'out_of_order_ack' ? 'normal' : 'out_of_order_ack')}
                      style={{
                        padding: '4px 9px',
                        fontSize: '0.65rem',
                        fontWeight: 600,
                        borderRadius: '4px',
                        border: activeScenario === 'out_of_order_ack' ? '1px solid #3B82F6' : '1px solid var(--border-color)',
                        background: activeScenario === 'out_of_order_ack' ? 'rgba(59, 130, 246, 0.25)' : 'rgba(255, 255, 255, 0.05)',
                        color: activeScenario === 'out_of_order_ack' ? '#93C5FD' : 'var(--text-secondary)',
                        cursor: 'pointer',
                        whiteSpace: 'nowrap'
                      }}
                    >
                      {activeScenario === 'out_of_order_ack' ? 'Disarm' : 'Arm Out-of-Order'}
                    </button>
                  </div>
                </div>
              )}

              {/* Tab 2: Part B (Multi-Worker Scenarios: Steps 6-12) */}
              {failureLabTab === 'partB' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.45rem' }}>
                  {/* Step 6: Demand Work Distribution */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          6. Demand Work Distribution (Competing Pull)
                        </strong>
                        <span style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60A5FA', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Set workers to <strong>3W</strong>, click <strong>Batch (6 Jobs)</strong>. Delivers reactively based on worker pull requests.
                      </div>
                    </div>
                  </div>

                  {/* Step 7: Dynamic Scale-Out */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          7. Dynamic Scale-Out (Scenario 10)
                        </strong>
                        <span style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60A5FA', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Set <strong>1W</strong>, toggle <strong>OFF</strong>, click <strong>Batch (6 Jobs)</strong>, switch to <strong>3W</strong>, toggle <strong>ON</strong>. Spawns workers without repartitioning pauses.
                      </div>
                    </div>
                  </div>

                  {/* Step 8: Dynamic Scale-In */}
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0.45rem 0.6rem',
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: '5px',
                    gap: '0.6rem'
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.15rem' }}>
                        <strong style={{ color: 'var(--text-bright)', fontSize: '0.73rem' }}>
                          8. Dynamic Scale-In (Scenario 9)
                        </strong>
                        <span style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60A5FA', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                          OPERATOR ACTION
                        </span>
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Set <strong>3W</strong>, click <strong>Batch (6 Jobs)</strong>, then immediately click <strong>1W</strong>. Retiring workers finish in-flight jobs gracefully.
                      </div>
                    </div>
                  </div>

                  {/* Step 9: Worker Crash Failover */}
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
                          9. Worker Crash Failover (Scenario 7)
                        </strong>
                        {activeScenario === 'crash_before_ack' && (
                          <span style={{ background: '#EF4444', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                            ARMED
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Simulates panic in 1 worker before ACK. Surviving workers continue; broker redelivers to healthy peer after 5s AckWait.
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

                  {/* Step 10: Slow Worker Timeout */}
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
                          10. Slow Worker Timeout (Scenario 8)
                        </strong>
                        {activeScenario === 'exceed_ack_wait' && (
                          <span style={{ background: '#F59E0B', color: '#000', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                            ARMED
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        One worker delays 7s (&gt;5s AckWait). Broker redelivers to healthy peer at 5s; original worker sends late ACK at 7s.
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

                  {/* Step 11: Fast Retry via NAK */}
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
                          11. Fast Retry via NAK (msg.Nak())
                        </strong>
                        {activeScenario === 'nak_message' && (
                          <span style={{ background: '#EF4444', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                            ARMED
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Worker sends explicit NAK. Broker immediately requeues message for redelivery without waiting 5s.
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

                  {/* Step 12: Poison Pill Termination */}
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
                          12. Poison Pill Termination (msg.Term())
                        </strong>
                        {activeScenario === 'term_message' && (
                          <span style={{ background: '#DC2626', color: '#FFF', fontSize: '0.58rem', fontWeight: 700, padding: '0 4px', borderRadius: '2px' }}>
                            ARMED
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: '0.63rem', color: 'var(--text-muted)', lineHeight: '1.3' }}>
                        Worker sends msg.Term(). Broker marks permanently terminated; no redelivery; ACK floor advances.
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
                </div>
              )}

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
