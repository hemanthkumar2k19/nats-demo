import React from 'react';
import { StatusIndicator } from './StatusIndicator';
import { ServiceStatus, JetStreamInfo } from '../api/demoApi';

interface StatusPanelProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  onRefresh: () => void;
  onToggleProcessor: (enabled: boolean) => Promise<void>;
  isLoading: boolean;
  onShowInfo?: (key: string) => void;
}

export const StatusPanel: React.FC<StatusPanelProps> = ({
  services,
  onRefresh,
  onToggleProcessor,
  isLoading,
  onShowInfo,
}) => {
  // Extract individual service statuses
  const natsService = services.find((s) => s.name.toLowerCase().includes('nats'));
  const demoControlService = services.find((s) => s.name.toLowerCase().includes('control') || s.name.toLowerCase().includes('demo'));
  const jobService = services.find((s) => s.name.toLowerCase().includes('job'));
  const processorService = services.find((s) => s.name.toLowerCase().includes('processor'));

  const natsStatus = natsService?.status || 'unknown';
  const demoControlStatus = demoControlService?.status || 'unknown';
  const jobStatus = jobService?.status || 'unknown';
  const processorStatus = processorService?.status || 'unknown';

  const isProcessing = processorService?.processing ?? false;
  const isNatsConnected = natsStatus === 'connected' || natsStatus === 'active';
  const isProcessorOnline = processorStatus !== 'disconnected' && processorStatus !== 'unknown';

  return (
    <div className="panel platform-status-panel">
      <div className="panel-header" style={{ marginBottom: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            PLATFORM STATUS
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('platform-status')}
              title="Learn about Platform Status"
            >
              (i)
            </button>
          )}
        </div>
        <button
          className="btn btn-secondary"
          onClick={onRefresh}
          disabled={isLoading}
          style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
        >
          {isLoading ? 'Checking...' : 'Refresh'}
        </button>
      </div>

      <div className="platform-status-grid">
        <div className="platform-status-row">
          <div className="status-metric-cell">
            <span className="status-metric-label">NATS Server</span>
            <StatusIndicator
              status={natsStatus as any}
              label={isNatsConnected ? 'Connected' : 'Disconnected'}
            />
          </div>

          <div className="status-metric-cell">
            <span className="status-metric-label">JetStream</span>
            <span
              className="badge"
              style={{
                background: isNatsConnected ? 'rgba(16, 185, 129, 0.12)' : 'rgba(239, 68, 68, 0.12)',
                color: isNatsConnected ? '#34D399' : '#F87171',
                border: isNatsConnected ? '1px solid rgba(16, 185, 129, 0.25)' : '1px solid rgba(239, 68, 68, 0.25)',
              }}
            >
              {isNatsConnected ? 'Available' : 'Unavailable'}
            </span>
          </div>

          <div className="status-metric-cell">
            <span className="status-metric-label">Demo Control</span>
            <StatusIndicator
              status={demoControlStatus as any}
              label={demoControlStatus === 'active' ? 'Active (:8080)' : 'Disconnected'}
            />
          </div>

          <div className="status-metric-cell">
            <span className="status-metric-label">Job Service</span>
            <StatusIndicator
              status={jobStatus as any}
              label={jobStatus === 'active' ? 'Active (:8081)' : 'Disconnected'}
            />
          </div>

          <div className="status-metric-cell">
            <span className="status-metric-label">Processor Service</span>
            <StatusIndicator
              status={isProcessorOnline ? 'active' : 'disconnected'}
              label={isProcessorOnline ? 'Active' : 'Disconnected'}
            />
          </div>

          <div className="status-metric-cell">
            <span className="status-metric-label">Processing</span>
            <button
              className={`status-toggle-btn ${isProcessing ? 'toggle-on' : 'toggle-off'}`}
              onClick={() => onToggleProcessor(!isProcessing)}
              disabled={!isProcessorOnline}
              title={isProcessorOnline ? 'Click to toggle processor state' : 'Processor service is offline'}
            >
              <span className="toggle-dot" />
              <span>{isProcessing ? 'ON' : 'OFF'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
