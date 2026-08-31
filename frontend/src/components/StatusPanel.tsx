import React from 'react';
import { StatusIndicator } from './StatusIndicator';
import { ServiceStatus, JetStreamInfo } from '../api/demoApi';

interface StatusPanelProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  onRefresh: () => void;
  onToggleProcessor: (enabled: boolean) => Promise<void>;
  isLoading: boolean;
}

export const StatusPanel: React.FC<StatusPanelProps> = ({
  services,
  jetstreamInfo,
  onRefresh,
  onToggleProcessor,
  isLoading,
}) => {
  return (
    <div className="panel">
      <div className="panel-header">
        <h2 className="panel-title">
          <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          Services Status
        </h2>
        <button 
          className="btn btn-secondary" 
          onClick={onRefresh} 
          disabled={isLoading}
          style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
        >
          {isLoading ? 'Checking...' : 'Refresh'}
        </button>
      </div>

      <div className="status-list">
        {services.length === 0 ? (
          <div style={{ color: 'var(--text-muted)', fontSize: '0.875rem', textAlign: 'center', padding: '1rem 0' }}>
            No service statuses loaded. Click Refresh.
          </div>
        ) : (
          services.map((svc) => (
            <div key={svc.name} className="status-item" style={{ flexWrap: 'wrap', gap: '0.25rem' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                <span className="status-name">{svc.name}</span>
                <StatusIndicator 
                  status={svc.status} 
                  label={
                    svc.status === 'active' ? 'Active' : 
                    svc.status === 'connected' ? 'Connected' : 
                    svc.status === 'disconnected' ? 'Disconnected' : 
                    svc.status === 'stopped' ? 'Stopped' : svc.status
                  } 
                />
              </div>
              {svc.name === 'processor-service' && svc.status !== 'disconnected' && (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%', marginTop: '0.25rem', paddingLeft: '0.5rem', borderLeft: '2px solid var(--border-color)' }}>
                  <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                    Processing: <strong style={{ color: svc.processing ? '#34D399' : '#F87171' }}>{svc.processing ? 'ON' : 'OFF'}</strong>
                  </span>
                  <button
                    className={`btn ${svc.processing ? 'btn-secondary' : 'btn-primary'}`}
                    style={{ padding: '0.125rem 0.375rem', fontSize: '0.75rem', height: 'auto', minHeight: 'unset' }}
                    onClick={() => onToggleProcessor(!svc.processing)}
                  >
                    {svc.processing ? 'Turn OFF' : 'Turn ON'}
                  </button>
                </div>
              )}
            </div>
          ))
        )}
      </div>

      {jetstreamInfo && (
        <div style={{ marginTop: '1rem', borderTop: '1px solid var(--border-color)', paddingTop: '0.75rem' }}>
          <div style={{ fontSize: '0.75rem', fontWeight: 600, textTransform: 'uppercase', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
            JetStream Info
          </div>
          <div className="status-item">
            <span className="status-name">Stream</span>
            <span style={{ fontSize: '0.8125rem', fontWeight: 500, fontFamily: 'var(--font-mono)' }}>{jetstreamInfo.stream}</span>
          </div>
          <div className="status-item">
            <span className="status-name">Pending Messages</span>
            <span className="badge" style={{ 
              fontSize: '0.8125rem', 
              fontFamily: 'var(--font-mono)',
              background: jetstreamInfo.pending > 0 ? 'rgba(245, 158, 11, 0.15)' : 'rgba(16, 185, 129, 0.15)',
              color: jetstreamInfo.pending > 0 ? '#FBBF24' : '#34D399',
              border: jetstreamInfo.pending > 0 ? '1px solid rgba(245, 158, 11, 0.3)' : '1px solid rgba(16, 185, 129, 0.3)',
              padding: '0.125rem 0.375rem',
              borderRadius: '4px',
              fontWeight: 600
            }}>
              {jetstreamInfo.pending}
            </span>
          </div>
        </div>
      )}
    </div>
  );
};
