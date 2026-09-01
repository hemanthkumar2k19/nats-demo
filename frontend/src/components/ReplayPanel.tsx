import React, { useState } from 'react';
import { ReplayRequest, ReplayResponse } from '../api/demoApi';

interface ReplayPanelProps {
  onTriggerReplay: (req: ReplayRequest) => Promise<ReplayResponse>;
  onShowInfo?: (key: string) => void;
}

export const ReplayPanel: React.FC<ReplayPanelProps> = ({ onTriggerReplay, onShowInfo }) => {
  const [fromSeq, setFromSeq] = useState<string>('100');
  const [toSeq, setToSeq] = useState<string>('120');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [result, setResult] = useState<ReplayResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);
    setResult(null);

    const fromNum = parseInt(fromSeq, 10);
    const toNum = parseInt(toSeq, 10);

    if (isNaN(fromNum) || isNaN(toNum)) {
      setError('Sequence numbers must be valid integers');
      setIsLoading(false);
      return;
    }

    try {
      const res = await onTriggerReplay({
        from_sequence: fromNum,
        to_sequence: toNum,
      });
      setResult(res);
    } catch (err: any) {
      setError(err.message || 'Failed to trigger replay');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="panel">
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 7.89H18" />
            </svg>
            JetStream Replay
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('jetstream-replay')}
              title="Learn about JetStream Replay"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label className="form-label" htmlFor="fromSeq">From Sequence</label>
          <input
            id="fromSeq"
            type="number"
            className="form-input code"
            value={fromSeq}
            onChange={(e) => setFromSeq(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="toSeq">To Sequence</label>
          <input
            id="toSeq"
            type="number"
            className="form-input code"
            value={toSeq}
            onChange={(e) => setToSeq(e.target.value)}
            disabled={isLoading}
          />
        </div>

        {error && (
          <div style={{ color: 'var(--status-danger)', fontSize: '0.8125rem', marginTop: '0.5rem', marginBottom: '0.5rem', fontFamily: 'var(--font-mono)' }}>
            * {error}
          </div>
        )}

        {result && (
          <div style={{ color: 'var(--status-success)', fontSize: '0.8125rem', marginTop: '0.5rem', marginBottom: '0.5rem', fontFamily: 'var(--font-mono)' }}>
            * Replay: {result.status} (Consumer: {result.consumer})
          </div>
        )}

        <button
          type="submit"
          className="btn btn-primary"
          disabled={isLoading}
          style={{ width: '100%', marginTop: '0.5rem' }}
        >
          {isLoading ? 'Triggering...' : 'Start Replay'}
        </button>
      </form>
    </div>
  );
};
