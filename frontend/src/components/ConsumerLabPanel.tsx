import React, { useState, useEffect, useCallback } from 'react';
import { getConsumerStatus, updateConsumerConfig, ConsumerStatus } from '../api/demoApi';

interface ConsumerLabPanelProps {
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
}

export const ConsumerLabPanel: React.FC<ConsumerLabPanelProps> = ({ onAlert }) => {
  const [consumerType, setConsumerType] = useState<'durable' | 'ephemeral'>('durable');
  const [workers, setWorkers] = useState<number>(1);
  const [ordering, setOrdering] = useState<'normal' | 'ordered'>('normal');
  const [status, setStatus] = useState<ConsumerStatus | null>(null);
  const [isApplying, setIsApplying] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const fetchStatus = useCallback(async () => {
    try {
      const data = await getConsumerStatus();
      setStatus(data);
      setIsLoading(false);
    } catch {
      // Quietly ignore polling errors if server temporarily unavailable
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 2000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  // Sync form inputs when status is first loaded or updated from outside
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
    }
  }, [status?.name, status?.type, status?.workers, status?.ordering]);

  const handleApply = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsApplying(true);
    try {
      const updated = await updateConsumerConfig({
        type: consumerType,
        workers: ordering === 'ordered' ? 1 : workers,
        ordering,
      });
      setStatus(updated);
      onAlert?.('success', `Consumer reconfigured: ${updated.type.toUpperCase()} (${updated.name}) with ${updated.workers} worker(s) [${updated.ordering}]`);
    } catch (err: any) {
      onAlert?.('error', `Failed to update consumer: ${err.message || 'Unknown error'}`);
    } finally {
      setIsApplying(false);
    }
  };

  const handleOrderingChange = (val: 'normal' | 'ordered') => {
    setOrdering(val);
    if (val === 'ordered') {
      setWorkers(1);
    }
  };

  return (
    <div className="panel consumer-lab-panel">
      <div className="panel-header">
        <h2>CONSUMER LAB</h2>
        <span className="badge badge-nats">JetStream Consumer</span>
      </div>

      <form onSubmit={handleApply} className="consumer-lab-form">
        <div className="form-group">
          <label className="form-label">Consumer Type</label>
          <div className="radio-group-horizontal">
            <label className="radio-label">
              <input
                type="radio"
                name="consumerType"
                value="durable"
                checked={consumerType === 'durable'}
                onChange={() => setConsumerType('durable')}
              />
              <span>Durable (Persistent State)</span>
            </label>
            <label className="radio-label">
              <input
                type="radio"
                name="consumerType"
                value="ephemeral"
                checked={consumerType === 'ephemeral'}
                onChange={() => setConsumerType('ephemeral')}
              />
              <span>Ephemeral (Temporary)</span>
            </label>
          </div>
        </div>

        <div className="form-group">
          <label className="form-label">Workers (Competing Consumers)</label>
          <div className="select-container">
            <select
              value={workers}
              disabled={ordering === 'ordered'}
              onChange={(e) => setWorkers(parseInt(e.target.value, 10))}
              className="form-select"
            >
              <option value={1}>1 Worker (processor-1)</option>
              <option value={2}>2 Workers (processor-1, processor-2)</option>
            </select>
          </div>
          {ordering === 'ordered' && (
            <span className="field-hint">* Ordered consumer requires single worker (locked to 1)</span>
          )}
        </div>

        <div className="form-group">
          <label className="form-label">Ordering</label>
          <div className="radio-group-horizontal">
            <label className="radio-label">
              <input
                type="radio"
                name="ordering"
                value="normal"
                checked={ordering === 'normal'}
                onChange={() => handleOrderingChange('normal')}
              />
              <span>Normal (Competing)</span>
            </label>
            <label className="radio-label">
              <input
                type="radio"
                name="ordering"
                value="ordered"
                checked={ordering === 'ordered'}
                onChange={() => handleOrderingChange('ordered')}
              />
              <span>Ordered (Strict Sequence)</span>
            </label>
          </div>
        </div>

        <div className="form-group">
          <label className="form-label">Delivery Semantics</label>
          <div className="readonly-box">
            <span>At Least Once (Explicit ACK + Redelivery)</span>
          </div>
        </div>

        <button
          type="submit"
          className="btn btn-primary btn-block"
          disabled={isApplying}
        >
          {isApplying ? 'Applying Configuration...' : 'Apply Configuration'}
        </button>
      </form>

      <div className="consumer-status-card">
        <div className="consumer-status-card-header">
          <h3>CONSUMER STATUS</h3>
          {status && (
            <span className={`status-pill ${status.status === 'ACTIVE' ? 'pill-active' : 'pill-stopped'}`}>
              {status.status}
            </span>
          )}
        </div>

        {isLoading ? (
          <div className="card-loading">Loading consumer status...</div>
        ) : status ? (
          <div className="consumer-metrics-grid">
            <div className="consumer-metric-row">
              <span className="metric-label">Name:</span>
              <span className="metric-value font-mono">{status.name}</span>
            </div>
            <div className="consumer-metric-row">
              <span className="metric-label">Type:</span>
              <span className="metric-value uppercase">{status.type}</span>
            </div>
            <div className="consumer-metric-row">
              <span className="metric-label">Workers:</span>
              <span className="metric-value">{status.workers} Active</span>
            </div>
            <div className="consumer-metric-row">
              <span className="metric-label">Ordering:</span>
              <span className="metric-value capitalize">{status.ordering}</span>
            </div>
            <div className="consumer-metric-row">
              <span className="metric-label">Pending Messages:</span>
              <span className={`metric-value ${status.pending > 0 ? 'text-warning font-bold' : ''}`}>
                {status.pending}
              </span>
            </div>
            <div className="consumer-metric-row">
              <span className="metric-label">Ack Pending:</span>
              <span className="metric-value">{status.ack_pending}</span>
            </div>
            <div className="consumer-metric-row">
              <span className="metric-label">Redelivered:</span>
              <span className={`metric-value ${status.redelivered > 0 ? 'text-orange font-bold' : ''}`}>
                {status.redelivered}
              </span>
            </div>
          </div>
        ) : (
          <div className="card-empty">No consumer information available</div>
        )}
      </div>
    </div>
  );
};
