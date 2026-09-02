import React, { useState } from 'react';
import { JetStreamInfo, ReplayRequest, ReplayResponse } from '../api/demoApi';

interface ReplayPanelProps {
  jetstreamInfo?: JetStreamInfo | null;
  onTriggerReplay: (req: ReplayRequest) => Promise<ReplayResponse>;
  onShowInfo?: (key: string) => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

const getInitialStartTime = (): string => {
  const d = new Date(Date.now() - 3600000);
  return d.toISOString().slice(0, 16);
};

const getInitialEndTime = (): string => {
  const d = new Date();
  return d.toISOString().slice(0, 16);
};

export const ReplayPanel: React.FC<ReplayPanelProps> = ({
  jetstreamInfo,
  onTriggerReplay,
  onShowInfo,
  onRefresh,
  isRefreshing = false,
}) => {
  const streamName = jetstreamInfo?.stream || 'JOBS';
  const totalMsgs = jetstreamInfo?.messages ?? 0;
  const firstSeq = jetstreamInfo?.first_seq ?? (totalMsgs > 0 ? 1 : 0);
  const lastSeq = jetstreamInfo?.last_seq ?? totalMsgs;
  const [replayFrom, setReplayFrom] = useState<'sequence' | 'time'>('sequence');
  const [startSeq, setStartSeq] = useState<string>('1');
  const [endSeq, setEndSeq] = useState<string>('100');
  const [startTime, setStartTime] = useState<string>(getInitialStartTime());
  const [endTime, setEndTime] = useState<string>(getInitialEndTime());
  const [replayMode, setReplayMode] = useState<'instant' | 'original'>('instant');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [result, setResult] = useState<ReplayResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);
    setResult(null);

    if (replayFrom === 'sequence') {
      const startNum = parseInt(startSeq, 10);
      const endNum = parseInt(endSeq, 10);

      if (isNaN(startNum) || isNaN(endNum)) {
        setError('Sequence numbers must be valid integers');
        setIsLoading(false);
        return;
      }

      if (startNum < 1) {
        setError('Start Sequence must be greater than or equal to 1');
        setIsLoading(false);
        return;
      }

      if (endNum < startNum) {
        setError('End Sequence must be greater than or equal to Start Sequence');
        setIsLoading(false);
        return;
      }

      try {
        const res = await onTriggerReplay({
          replay_from: 'sequence',
          start_sequence: startNum,
          end_sequence: endNum,
          from_sequence: startNum,
          to_sequence: endNum,
          replay_mode: replayMode,
        });
        setResult(res);
      } catch (err: any) {
        setError(err.message || 'Failed to trigger replay');
      } finally {
        setIsLoading(false);
      }
    } else {
      if (!startTime || !endTime) {
        setError('Start Time and End Time must both be valid values');
        setIsLoading(false);
        return;
      }

      const startMs = new Date(startTime).getTime();
      const endMs = new Date(endTime).getTime();

      if (isNaN(startMs) || isNaN(endMs)) {
        setError('Start Time and End Time must be valid dates');
        setIsLoading(false);
        return;
      }

      if (startMs >= endMs) {
        setError('Start Time must be earlier than End Time');
        setIsLoading(false);
        return;
      }

      try {
        const res = await onTriggerReplay({
          replay_from: 'time',
          start_time: startTime,
          end_time: endTime,
          from_time: startTime,
          to_time: endTime,
          replay_mode: replayMode,
        });
        setResult(res);
      } catch (err: any) {
        setError(err.message || 'Failed to trigger replay');
      } finally {
        setIsLoading(false);
      }
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
        {onRefresh && (
          <button
            type="button"
            className="btn btn-secondary"
            onClick={onRefresh}
            disabled={isRefreshing}
            style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
          >
            {isRefreshing ? 'Checking...' : 'Refresh'}
          </button>
        )}
      </div>

      <form onSubmit={handleSubmit}>
        {/* Stream Source & Stored Messages Indicator */}
        <div className="form-group">
          <label className="form-label">Stream</label>
          <div className="readonly-box" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.625rem' }}>
              <span className="mono-cell" style={{ color: 'var(--accent-cyan)', fontWeight: 600 }}>{streamName}</span>
              <span
                className="node-badge"
                style={{
                  background: totalMsgs > 0 ? 'rgba(16, 185, 129, 0.15)' : 'rgba(107, 114, 128, 0.15)',
                  color: totalMsgs > 0 ? '#34D399' : 'var(--text-muted)',
                  border: totalMsgs > 0 ? '1px solid rgba(16, 185, 129, 0.3)' : '1px solid rgba(107, 114, 128, 0.25)',
                  fontSize: '0.6875rem',
                  padding: '0.125rem 0.5rem',
                  borderRadius: '4px',
                  fontWeight: 600,
                }}
              >
                {totalMsgs} stored {totalMsgs === 1 ? 'msg' : 'msgs'}
              </span>
            </div>
            <span style={{ fontSize: '0.6875rem', color: 'var(--text-muted)' }}>
              {lastSeq > 0 ? `Seq #${firstSeq} - #${lastSeq}` : 'Stream Empty'}
            </span>
          </div>
        </div>

        {totalMsgs === 0 && (
          <div style={{
            fontSize: '0.75rem',
            color: 'var(--text-muted)',
            background: 'rgba(255, 255, 255, 0.03)',
            border: '1px dashed var(--border-subtle)',
            borderRadius: '4px',
            padding: '0.5rem 0.75rem',
            marginBottom: '0.75rem',
          }}>
            No historical messages currently stored in <strong>{streamName}</strong>. Submit jobs first via the Pub/Sub panel to populate the stream for replay.
          </div>
        )}

        {/* Replay Position Selector */}
        <div className="form-group">
          <label className="form-label">Replay From</label>
          <div style={{ display: 'flex', gap: '1.25rem', marginTop: '0.25rem' }}>
            <label className="radio-label">
              <input
                type="radio"
                name="replayFrom"
                value="sequence"
                checked={replayFrom === 'sequence'}
                onChange={() => {
                  setReplayFrom('sequence');
                  setError(null);
                }}
                disabled={isLoading}
              />
              <span>Sequence</span>
            </label>
            <label className="radio-label">
              <input
                type="radio"
                name="replayFrom"
                value="time"
                checked={replayFrom === 'time'}
                onChange={() => {
                  setReplayFrom('time');
                  setError(null);
                }}
                disabled={isLoading}
              />
              <span>Time</span>
            </label>
          </div>
        </div>

        {/* Sequence Mode Inputs */}
        {replayFrom === 'sequence' && (
          <div className="form-row-2col">
            <div className="form-group">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <label className="form-label" htmlFor="startSeq">Start Sequence</label>
                {totalMsgs > 0 && (
                  <span style={{ fontSize: '0.6875rem', color: 'var(--text-muted)' }}>
                    First: #{firstSeq}
                  </span>
                )}
              </div>
              <input
                id="startSeq"
                type="number"
                min="1"
                className="form-input code"
                value={startSeq}
                onChange={(e) => setStartSeq(e.target.value)}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <label className="form-label" htmlFor="endSeq">End Sequence</label>
                {totalMsgs > 0 && (
                  <span style={{ fontSize: '0.6875rem', color: 'var(--accent-cyan)' }}>
                    Last: #{lastSeq}
                  </span>
                )}
              </div>
              <input
                id="endSeq"
                type="number"
                min="1"
                className="form-input code"
                value={endSeq}
                onChange={(e) => setEndSeq(e.target.value)}
                disabled={isLoading}
              />
            </div>
          </div>
        )}

        {/* Time Mode Inputs */}
        {replayFrom === 'time' && (
          <div className="form-row-2col">
            <div className="form-group">
              <label className="form-label" htmlFor="startTime">Start Time</label>
              <input
                id="startTime"
                type="datetime-local"
                className="form-input code"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="endTime">End Time</label>
              <input
                id="endTime"
                type="datetime-local"
                className="form-input code"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
                disabled={isLoading}
              />
            </div>
          </div>
        )}

        {/* Replay Mode Selector */}
        <div className="form-group">
          <label className="form-label">Replay Mode</label>
          <div style={{ display: 'flex', gap: '1.25rem', marginTop: '0.25rem' }}>
            <label className="radio-label">
              <input
                type="radio"
                name="replayMode"
                value="instant"
                checked={replayMode === 'instant'}
                onChange={() => setReplayMode('instant')}
                disabled={isLoading}
              />
              <span>Instant</span>
            </label>
            <label className="radio-label">
              <input
                type="radio"
                name="replayMode"
                value="original"
                checked={replayMode === 'original'}
                onChange={() => setReplayMode('original')}
                disabled={isLoading}
              />
              <span>Original Timing</span>
            </label>
          </div>
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
