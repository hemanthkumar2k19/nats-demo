import React, { useState, useEffect, useCallback } from 'react';
import {
  getConsumerStatus,
  updateConsumerConfig,
  resetConsumerDistribution,
  sendJetStreamTestMessages,
  ConsumerStatus,
} from '../api/demoApi';

interface ConsumerLabPanelProps {
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
  onConfigChanged?: (status: ConsumerStatus) => void;
  onShowInfo?: (key: string) => void;
  onActivityUpdated?: () => void;
  isEmbedded?: boolean;
  isProcessing?: boolean;
}

const WORKER_COLORS = [
  { fillClass: 'fill-p1', hex: '#3B82F6' },
  { fillClass: 'fill-p2', hex: '#10B981' },
  { fillClass: 'fill-p3', hex: '#F59E0B' },
  { fillClass: 'fill-p4', hex: '#8B5CF6' },
  { fillClass: 'fill-p5', hex: '#06B6D4' },
];

export const ConsumerLabPanel: React.FC<ConsumerLabPanelProps> = ({
  onAlert,
  onConfigChanged,
  onShowInfo,
  onActivityUpdated,
  isEmbedded = false,
  isProcessing,
}) => {
  const [consumerType, setConsumerType] = useState<'durable' | 'ephemeral'>('durable');
  const [workers, setWorkers] = useState<number>(1);
  const [ordering, setOrdering] = useState<'normal' | 'ordered'>('normal');
  const [deliverPolicy, setDeliverPolicy] = useState<'all' | 'new' | 'last' | 'last_per_subject'>('all');
  const [ackPolicy, setAckPolicy] = useState<'explicit' | 'none' | 'all'>('explicit');
  const [status, setStatus] = useState<ConsumerStatus | null>(null);
  const [isApplying, setIsApplying] = useState(false);
  const [messageCount, setMessageCount] = useState<number>(10);
  const [isSending, setIsSending] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [lastBatch, setLastBatch] = useState<string[]>([]);

  const fetchStatus = useCallback(async () => {
    try {
      const data = await getConsumerStatus();
      setStatus(data);
    } catch {
      // Quietly ignore polling errors if server temporarily unavailable
    }
  }, []);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 2000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  // Sync state from status
  useEffect(() => {
    if (status) {
      if (status.type === 'durable' || status.type === 'ephemeral') {
        setConsumerType(status.type);
      }
      if (status.workers) {
        setWorkers(status.workers);
      }
      if (status.ordering === 'normal' || status.ordering === 'ordered') {
        setOrdering(status.ordering);
      }
      if (status.deliver_policy) {
        setDeliverPolicy(status.deliver_policy as any);
      }
      if (status.ack_policy) {
        setAckPolicy(status.ack_policy as any);
      }
    }
  }, [status?.name, status?.type, status?.workers, status?.ordering, status?.deliver_policy, status?.ack_policy]);

  const handleReconfigure = async (
    newType: 'durable' | 'ephemeral',
    newOrdering: 'normal' | 'ordered',
    newWorkers: number,
    newDeliverPolicy: 'all' | 'new' | 'last' | 'last_per_subject' = deliverPolicy,
    newAckPolicy: 'explicit' | 'none' | 'all' = ackPolicy,
  ) => {
    if (isApplying) return;
    setIsApplying(true);
    try {
      const updated = await updateConsumerConfig({
        type: newType,
        workers: newOrdering === 'ordered' ? 1 : newWorkers,
        ordering: newOrdering,
        deliver_policy: newDeliverPolicy,
        ack_policy: newAckPolicy,
      });
      setStatus(updated);
      setConsumerType(updated.type as any);
      setOrdering(updated.ordering as any);
      setWorkers(updated.workers);
      if (updated.deliver_policy) setDeliverPolicy(updated.deliver_policy as any);
      if (updated.ack_policy) setAckPolicy(updated.ack_policy as any);
      onConfigChanged?.(updated);
      onAlert?.('success', `Consumer reconfigured: ${updated.type.toUpperCase()} [Deliver: ${newDeliverPolicy.toUpperCase()}, Ack: ${newAckPolicy.toUpperCase()}] with ${updated.workers} worker(s)`);
    } catch (err: any) {
      onAlert?.('error', `Failed to update consumer: ${err.message || 'Unknown error'}`);
    } finally {
      setIsApplying(false);
    }
  };

  const handleSendMessages = async () => {
    if (messageCount <= 0 || isSending) return;
    setIsSending(true);
    try {
      const res = await sendJetStreamTestMessages(messageCount);
      setLastBatch(res.jobs || []);
      onAlert?.('success', `Published ${res.published} test messages to JOBS stream`);
      onActivityUpdated?.();
      await fetchStatus();
    } catch (err: any) {
      onAlert?.('error', `Failed to send stream messages: ${err.message || 'Unknown error'}`);
    } finally {
      setIsSending(false);
    }
  };

  const handleResetDistribution = async () => {
    setIsResetting(true);
    try {
      const updated = await resetConsumerDistribution();
      setStatus(updated);
      onConfigChanged?.(updated);
      onAlert?.('success', 'Consumer distribution counters reset to zero');
    } catch (err: any) {
      onAlert?.('error', `Failed to reset distribution: ${err.message || 'Unknown error'}`);
    } finally {
      setIsResetting(false);
    }
  };

  const distribution = status?.distribution || {};
  const totalDelivered = Object.values(distribution).reduce((acc, count) => acc + count, 0);
  const maxWorkersToDisplay = Math.max(workers, 2);

  return (
    <div className={isEmbedded ? "queue-group-embedded" : "panel consumer-lab-panel"}>
      {/* Panel Header */}
      <div className={isEmbedded ? "summary-header" : "panel-header"} style={{ marginBottom: isEmbedded ? '0.75rem' : '1rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 style={{ fontSize: isEmbedded ? '0.8125rem' : '1rem', fontWeight: 700, margin: 0 }}>
            JETSTREAM CONSUMER LAB
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('consumer-lab')}
              title="Learn about Consumer Lab"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      {/* Meta Chips: Stream, Consumer, Delivery Mode, Guarantees */}
      <div className="queue-meta-row" style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Stream</span>
          <span className="queue-meta-value mono">JOBS</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Consumer</span>
          <span className="queue-meta-value mono">{status?.name || 'job-processor'}</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Delivery Mode</span>
          <span className="queue-meta-value">Pull (Batch Fetch)</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Guarantees</span>
          <span className="queue-meta-value">At-Least-Once (ACK)</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">State</span>
          <span className="queue-meta-value">Stateful Cursors</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Delivery Type</span>
          <span className="queue-meta-value">Pull (Client-Fetched)</span>
        </div>
      </div>

      {/* Config Selectors: Durability & Ordering */}
      <div className="form-row-2col" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.5rem', marginBottom: '0.75rem' }}>
        <div className="form-group">
          <label className="form-label" style={{ fontWeight: 600, display: 'block', marginBottom: '0.35rem', fontSize: '0.75rem' }}>
            Consumer Durability:
          </label>
          <div style={{ display: 'flex', gap: '0.25rem' }}>
            <button
              type="button"
              className={`btn-worker-toggle ${consumerType === 'durable' ? 'active' : ''}`}
              style={{ flex: 1, justifyContent: 'center', padding: '0.35rem 0.25rem' }}
              onClick={() => handleReconfigure('durable', ordering, workers)}
              disabled={isApplying}
            >
              <span>Durable</span>
            </button>
            <button
              type="button"
              className={`btn-worker-toggle ${consumerType === 'ephemeral' ? 'active' : ''}`}
              style={{ flex: 1, justifyContent: 'center', padding: '0.35rem 0.25rem' }}
              onClick={() => handleReconfigure('ephemeral', ordering, workers)}
              disabled={isApplying}
            >
              <span>Ephemeral</span>
            </button>
          </div>
        </div>

        <div className="form-group">
          <label className="form-label" style={{ fontWeight: 600, display: 'block', marginBottom: '0.35rem', fontSize: '0.75rem' }}>
            Message Ordering:
          </label>
          <div style={{ display: 'flex', gap: '0.25rem' }}>
            <button
              type="button"
              className={`btn-worker-toggle ${ordering === 'normal' ? 'active' : ''}`}
              style={{ flex: 1, justifyContent: 'center', padding: '0.35rem 0.25rem' }}
              onClick={() => handleReconfigure(consumerType, 'normal', workers)}
              disabled={isApplying}
            >
              <span>Normal</span>
            </button>
            <button
              type="button"
              className={`btn-worker-toggle ${ordering === 'ordered' ? 'active' : ''}`}
              style={{ flex: 1, justifyContent: 'center', padding: '0.35rem 0.25rem' }}
              onClick={() => handleReconfigure(consumerType, 'ordered', 1)}
              disabled={isApplying}
            >
              <span>Ordered</span>
            </button>
          </div>
        </div>
      </div>

      {/* Deliver Policy Selector */}
      <div className="form-group" style={{ marginBottom: '0.75rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
          <label className="form-label" style={{ fontWeight: 600, fontSize: '0.75rem', margin: 0 }}>
            Deliver Policy:
          </label>
          <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)' }}>Where consumer begins reading</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '0.25rem' }}>
          <button
            type="button"
            className={`btn-worker-toggle ${deliverPolicy === 'all' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, 'all', ackPolicy)}
            disabled={isApplying}
            title="DeliverAll: Replays all historical stream messages from sequence 1"
          >
            <span>DeliverAll</span>
          </button>
          <button
            type="button"
            className={`btn-worker-toggle ${deliverPolicy === 'new' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, 'new', ackPolicy)}
            disabled={isApplying}
            title="DeliverNew: Ignores history; only delivers messages published after consumer creation"
          >
            <span>DeliverNew</span>
          </button>
          <button
            type="button"
            className={`btn-worker-toggle ${deliverPolicy === 'last' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, 'last', ackPolicy)}
            disabled={isApplying}
            title="DeliverLast: Delivers only the single most recent message in the stream"
          >
            <span>DeliverLast</span>
          </button>
          <button
            type="button"
            className={`btn-worker-toggle ${deliverPolicy === 'last_per_subject' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, 'last_per_subject', ackPolicy)}
            disabled={isApplying}
            title="DeliverLastPerSubject: Delivers the latest message for each distinct subject"
          >
            <span>LastPerSubject</span>
          </button>
        </div>
        <p className="form-hint" style={{ fontSize: '0.6875rem', color: 'var(--text-secondary)', marginTop: '0.25rem', marginBottom: 0 }}>
          {deliverPolicy === 'all' && 'DeliverAll: Delivers entire stream history from sequence 1 to all competing workers.'}
          {deliverPolicy === 'new' && 'DeliverNew: Skips historical stream messages; delivers only newly arriving messages.'}
          {deliverPolicy === 'last' && 'DeliverLast: Starts from the single latest message published in the stream.'}
          {deliverPolicy === 'last_per_subject' && 'DeliverLastPerSubject: Starts with the latest message for each individual subject.'}
        </p>
      </div>

      {/* Ack Policy Selector */}
      <div className="form-group" style={{ marginBottom: '0.75rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
          <label className="form-label" style={{ fontWeight: 600, fontSize: '0.75rem', margin: 0 }}>
            Acknowledgment (Ack) Policy:
          </label>
          <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)' }}>Worker confirmation contract</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '0.25rem' }}>
          <button
            type="button"
            className={`btn-worker-toggle ${ackPolicy === 'explicit' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, deliverPolicy, 'explicit')}
            disabled={isApplying}
            title="Explicit: Each message must be individually confirmed with msg.Ack()"
          >
            <span>Explicit (1:1)</span>
          </button>
          <button
            type="button"
            className={`btn-worker-toggle ${ackPolicy === 'none' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, deliverPolicy, 'none')}
            disabled={isApplying}
            title="None: Fire-and-forget; server considers messages ACKed immediately upon dispatch"
          >
            <span>None (Fire & Forget)</span>
          </button>
          <button
            type="button"
            className={`btn-worker-toggle ${ackPolicy === 'all' ? 'active' : ''}`}
            style={{ padding: '0.35rem 0.2rem', justifyContent: 'center', fontSize: '0.6875rem' }}
            onClick={() => handleReconfigure(consumerType, ordering, workers, deliverPolicy, 'all')}
            disabled={isApplying}
            title="All: Cumulative ACK; confirming message N automatically acknowledges 1..N"
          >
            <span>Cumulative (All)</span>
          </button>
        </div>
        <p className="form-hint" style={{ fontSize: '0.6875rem', color: 'var(--text-secondary)', marginTop: '0.25rem', marginBottom: 0 }}>
          {ackPolicy === 'explicit' && 'Explicit ACK: Standard at-least-once. Workers must call msg.Ack() per message.'}
          {ackPolicy === 'none' && 'AckNone: Fire-and-forget. Server marks messages delivered immediately without worker ACKs.'}
          {ackPolicy === 'all' && 'Cumulative AckAll: Acknowledging message sequence N acknowledges all messages up to N.'}
        </p>
      </div>

      {/* Live NATS Consumer State Badge Bar */}
      <div style={{ background: 'rgba(0, 0, 0, 0.2)', border: '1px solid var(--border-color)', borderRadius: '6px', padding: '0.5rem 0.75rem', marginBottom: '1rem', fontSize: '0.6875rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
          <span style={{ fontWeight: 700, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Active NATS Consumer: <span style={{ color: 'var(--accent-cyan)' }}>{status?.name || (consumerType === 'durable' ? 'job-processor' : 'ephemeral')}</span>
          </span>
          <span style={{ fontSize: '0.625rem', color: 'var(--text-muted)' }}>Configured via NATS Go SDK</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '0.5rem', fontFamily: 'var(--font-mono)' }}>
          <div>
            <span style={{ color: 'var(--text-secondary)' }}>Type: </span>
            <span style={{ color: consumerType === 'durable' ? '#A78BFA' : '#38BDF8', fontWeight: 600 }}>
              {consumerType.toUpperCase()}
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--text-secondary)' }}>DeliverPolicy: </span>
            <span style={{ color: '#FCD34D', fontWeight: 600 }}>
              {(status?.deliver_policy || deliverPolicy).toUpperCase()}
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--text-secondary)' }}>AckPolicy: </span>
            <span style={{ color: '#34D399', fontWeight: 600 }}>
              {(status?.ack_policy || ackPolicy).toUpperCase()}
            </span>
          </div>
          <div>
            <span style={{ color: 'var(--text-secondary)' }}>FilterSubject: </span>
            <span style={{ color: '#F472B6', fontWeight: 600 }}>jobs.submitted</span>
          </div>
        </div>
      </div>

      {/* Worker Count Selection (1 to 5) */}
      <div className="form-group" style={{ marginBottom: '1.25rem' }}>
        <label className="form-label" style={{ fontWeight: 600, display: 'block', marginBottom: '0.5rem' }}>
          Active Pull Workers (Competing Pool):
        </label>
        <div className="queue-worker-toggle-group queue-worker-selector-grid" style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '0.375rem' }}>
          {[1, 2, 3, 4, 5].map((w) => (
            <button
              key={w}
              type="button"
              className={`btn-worker-toggle ${workers === w ? 'active' : ''}`}
              style={{ padding: '0.4rem 0.25rem', flexDirection: 'column', gap: '0.2rem', alignItems: 'center' }}
              onClick={() => handleReconfigure(consumerType, ordering, w)}
              disabled={isApplying || (ordering === 'ordered' && w > 1)}
              title={ordering === 'ordered' && w > 1 ? 'Ordered consumers are locked to 1 worker' : undefined}
            >
              <span className="worker-count-pill" style={{ margin: 0 }}>{w}</span>
              <span style={{ fontSize: '0.6875rem' }}>{w === 1 ? '1 Worker' : `${w} Workers`}</span>
            </button>
          ))}
        </div>
        <p className="form-hint" style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.375rem' }}>
          {ordering === 'ordered'
            ? 'Ordered consumption is strictly locked to 1 worker to preserve exact stream sequence.'
            : workers === 1
            ? 'All messages pulled from the consumer stream are processed by processor-1.'
            : `${workers} competing pull workers concurrently fetch and acknowledge messages from the shared stream consumer.`}
        </p>
      </div>

      {/* Messages to Publish */}
      <div className="form-group" style={{ marginBottom: '1.25rem' }}>
        <label className="form-label" style={{ fontWeight: 600, display: 'block', marginBottom: '0.5rem' }}>
          Messages to Publish to Stream:
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

      {/* Worker Distribution Display & Stream Metrics */}
      <div className="queue-distribution-box" style={{ marginBottom: '1.25rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
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

        {/* Stream Metrics Sub-Bar */}
        <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '0.75rem', fontSize: '0.6875rem', color: 'var(--text-secondary)' }}>
          <span className={status?.pending && status.pending > 0 ? 'text-warning font-bold' : ''}>
            Pending: {status?.pending ?? 0}
          </span>
          <span>•</span>
          <span>Ack Pending: {status?.ack_pending ?? 0}</span>
          <span>•</span>
          <span className={status?.redelivered && status.redelivered > 0 ? 'text-orange font-bold' : ''}>
            Redelivered: {status?.redelivered ?? 0}
          </span>
        </div>

        {/* Dynamic Worker Bars */}
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
                  {wName} {!isOnline ? '(Idle)' : ''}
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
        {isSending ? `Publishing ${messageCount} Messages...` : `Send ${messageCount} Test Messages to Stream`}
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
