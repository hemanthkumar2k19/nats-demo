import React, { useState, useEffect, useCallback } from 'react';
import {
  getQueueGroupStatus,
  updateQueueGroupWorkers,
  sendQueueTestMessages,
  resetQueueGroupDistribution,
  QueueGroupStatus,
} from '../api/demoApi';

interface QueueGroupPanelProps {
  onShowInfo: (key: string) => void;
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
  onMessagesSent?: () => void;
  onActivityUpdated?: () => void;
  onConfigChanged?: (status: QueueGroupStatus) => void;
  isEmbedded?: boolean;
}

const WORKER_COLORS = [
  { fillClass: 'fill-p1', hex: '#3B82F6' },
  { fillClass: 'fill-p2', hex: '#10B981' },
  { fillClass: 'fill-p3', hex: '#F59E0B' },
  { fillClass: 'fill-p4', hex: '#8B5CF6' },
  { fillClass: 'fill-p5', hex: '#06B6D4' },
];

export const QueueGroupPanel: React.FC<QueueGroupPanelProps> = ({
  onShowInfo,
  onAlert,
  onMessagesSent,
  onActivityUpdated,
  onConfigChanged,
  isEmbedded = false,
}) => {
  const [status, setStatus] = useState<QueueGroupStatus | null>(null);
  const [workers, setWorkers] = useState<number>(1);
  const [messageCount, setMessageCount] = useState<number>(10);
  const [isUpdatingWorkers, setIsUpdatingWorkers] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [lastBatch, setLastBatch] = useState<string[]>([]);

  const fetchStatus = useCallback(async () => {
    try {
      const data = await getQueueGroupStatus();
      setStatus(data);
      if (data.workers) {
        setWorkers(data.workers);
      }
    } catch {
      // Quietly ignore polling failures when service is briefly restarting
    }
  }, []);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 2000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  const handleWorkersChange = async (newWorkerCount: number) => {
    if (newWorkerCount === workers || isUpdatingWorkers) return;
    setIsUpdatingWorkers(true);
    try {
      const updated = await updateQueueGroupWorkers(newWorkerCount);
      setStatus(updated);
      setWorkers(updated.workers);
      onConfigChanged?.(updated);
      onAlert?.('success', `Queue group reconfigured with ${updated.workers} active subscriber(s)`);
    } catch (err: any) {
      onAlert?.('error', `Failed to configure workers: ${err.message || 'Unknown error'}`);
    } finally {
      setIsUpdatingWorkers(false);
    }
  };

  const handleResetDistribution = async () => {
    setIsResetting(true);
    try {
      const updated = await resetQueueGroupDistribution();
      setStatus(updated);
      onConfigChanged?.(updated);
      onAlert?.('success', 'Worker distribution counters reset to zero');
    } catch (err: any) {
      onAlert?.('error', `Failed to reset distribution: ${err.message || 'Unknown error'}`);
    } finally {
      setIsResetting(false);
    }
  };

  const handleSendMessages = async () => {
    if (messageCount <= 0 || isSending) return;
    setIsSending(true);
    try {
      const res = await sendQueueTestMessages(messageCount);
      setLastBatch(res.jobs || []);
      onAlert?.('success', `Published ${res.published} test messages to jobs.queue`);
      onMessagesSent?.();
      onActivityUpdated?.();
      await fetchStatus();
    } catch (err: any) {
      onAlert?.('error', `Failed to send queue messages: ${err.message || 'Unknown error'}`);
    } finally {
      setIsSending(false);
    }
  };

  // Calculate total across all workers in distribution
  const distribution = status?.distribution || {};
  const totalDelivered = Object.values(distribution).reduce((acc, count) => acc + count, 0);

  const maxWorkersToDisplay = Math.max(workers, 2);

  return (
    <div className={isEmbedded ? "queue-group-embedded" : "queue-group-panel"}>
      {/* Panel Header */}
      <div className={isEmbedded ? "summary-header" : "panel-header"} style={{ marginBottom: isEmbedded ? '0.75rem' : '1rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 style={{ fontSize: isEmbedded ? '0.8125rem' : '1rem', fontWeight: 700, margin: 0 }}>
            CORE NATS QUEUE GROUP
          </h2>
          <button
            type="button"
            className="node-info-btn"
            onClick={() => onShowInfo('queue-groups')}
            title="Learn about Core NATS Queue Groups"
          >
            (i)
          </button>
        </div>
      </div>

      {/* Meta Chips: Subject, Queue Group, Delivery Semantics */}
      <div className="queue-meta-row" style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Subject</span>
          <span className="queue-meta-value mono">jobs.queue</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Queue Group</span>
          <span className="queue-meta-value mono">job-workers</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Delivery Mode</span>
          <span className="queue-meta-value">Load-Balanced (1 of N)</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Guarantees</span>
          <span className="queue-meta-value">At-Most-Once (Best-Effort)</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">State</span>
          <span className="queue-meta-value">Stateless (No ACK/NAK)</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Delivery Type</span>
          <span className="queue-meta-value">Push (Server-Dispatched)</span>
        </div>
      </div>

      {/* Worker Count Selection (1 to 5) */}
      <div className="form-group" style={{ marginBottom: '1.25rem' }}>
        <label className="form-label" style={{ fontWeight: 600, display: 'block', marginBottom: '0.5rem' }}>
          Active Queue Group Subscribers:
        </label>
        <div className="queue-worker-selector-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '0.375rem' }}>
          {[1, 2, 3, 4, 5].map((w) => (
            <button
              key={w}
              type="button"
              className={`btn-worker-toggle ${workers === w ? 'active' : ''}`}
              style={{ justifyContent: 'center', padding: '0.5rem 0.25rem', flexDirection: 'column', gap: '0.25rem' }}
              onClick={() => handleWorkersChange(w)}
              disabled={isUpdatingWorkers}
            >
              <span className="worker-count-pill">{w}</span>
              <span style={{ fontSize: '0.6875rem' }}>{w === 1 ? '1 Worker' : `${w} Workers`}</span>
            </button>
          ))}
        </div>
        <p className="form-hint" style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.375rem' }}>
          {workers === 1
            ? 'All messages on jobs.queue are delivered exclusively to processor-1.'
            : `Messages on jobs.queue are randomly load-balanced across processor-1 through processor-${workers}.`}
        </p>
      </div>

      {/* Message Count Selection */}
      <div className="form-group" style={{ marginBottom: '1.25rem' }}>
        <label className="form-label" style={{ fontWeight: 600, display: 'block', marginBottom: '0.5rem' }}>
          Messages to Publish:
        </label>
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap' }}>
          {[5, 10, 20].map((count) => (
            <button
              key={count}
              type="button"
              className={`btn btn-secondary ${messageCount === count ? 'btn-selected' : ''}`}
              style={{ padding: '0.25rem 0.75rem', fontSize: '0.8125rem' }}
              onClick={() => setMessageCount(count)}
            >
              {count}
            </button>
          ))}
          <input
            type="number"
            min="1"
            max="100"
            className="form-input"
            style={{ width: '80px', padding: '0.25rem 0.5rem', fontSize: '0.8125rem' }}
            value={messageCount}
            onChange={(e) => setMessageCount(Math.max(1, parseInt(e.target.value, 10) || 1))}
          />
        </div>
      </div>

      {/* Worker Distribution Display with Reset Button */}
      <div className="queue-distribution-box" style={{ marginBottom: '1.25rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.625rem' }}>
          <span style={{ fontSize: '0.8125rem', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Worker Distribution
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.625rem' }}>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
              Total: {totalDelivered}
            </span>
            <button
              type="button"
              className="btn btn-secondary"
              style={{ padding: '0.2rem 0.5rem', fontSize: '0.6875rem' }}
              onClick={handleResetDistribution}
              disabled={isResetting || totalDelivered === 0}
              title="Reset worker distribution counters to zero"
            >
              {isResetting ? 'Resetting...' : 'Reset Counters'}
            </button>
          </div>
        </div>

        {/* Dynamic Worker Bars (1..maxWorkersToDisplay) */}
        {Array.from({ length: maxWorkersToDisplay }, (_, i) => {
          const wName = `processor-${i + 1}`;
          const isOnline = i < workers;
          const count = distribution[wName] || 0;
          const pct = totalDelivered > 0 ? Math.round((count / totalDelivered) * 100) : 0;
          const color = WORKER_COLORS[i % WORKER_COLORS.length];

          return (
            <div key={wName} className="worker-stat-row" style={{ marginBottom: i < maxWorkersToDisplay - 1 ? '0.5rem' : 0 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.8125rem', marginBottom: '0.25rem' }}>
                <span className="mono" style={{ fontWeight: 600, color: isOnline ? undefined : 'var(--text-secondary)' }}>
                  {wName} {!isOnline ? '(Offline)' : ''}
                </span>
                <span className="mono" style={{ color: isOnline ? 'var(--text-primary)' : 'var(--text-secondary)' }}>
                  {count} {totalDelivered > 0 && isOnline ? `(${pct}%)` : ''}
                </span>
              </div>
              <div className="distribution-progress-track">
                <div
                  className={`distribution-progress-fill ${color.fillClass}`}
                  style={{
                    width: `${pct}%`,
                    background: isOnline ? color.hex : 'transparent',
                  }}
                />
              </div>
            </div>
          );
        })}
      </div>

      {/* Action Button */}
      <button
        type="button"
        className="btn btn-primary"
        style={{ width: '100%', padding: '0.625rem', fontWeight: 600 }}
        onClick={handleSendMessages}
        disabled={isSending}
      >
        {isSending ? `Publishing ${messageCount} Messages...` : `Send ${messageCount} Test Messages`}
      </button>

      {/* Last Batch Summary */}
      {lastBatch.length > 0 && (
        <div style={{ marginTop: '0.875rem', fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
          <span>Last batch: </span>
          <span className="mono" style={{ color: 'var(--text-primary)' }}>
            {lastBatch[0]} ... {lastBatch[lastBatch.length - 1]} ({lastBatch.length} messages)
          </span>
        </div>
      )}
    </div>
  );
};
