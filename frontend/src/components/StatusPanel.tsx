import React from 'react';
import { StatusIndicator } from './StatusIndicator';
import { ServiceStatus } from '../api/demoApi';

interface StatusPanelProps {
  services: ServiceStatus[];
  onRefresh: () => void;
  isLoading: boolean;
}

export const StatusPanel: React.FC<StatusPanelProps> = ({ services, onRefresh, isLoading }) => {
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
            <div key={svc.name} className="status-item">
              <span className="status-name">{svc.name}</span>
              <StatusIndicator status={svc.status} label={svc.status === 'active' ? 'Active' : svc.status === 'connected' ? 'Connected' : svc.status === 'disconnected' ? 'Disconnected' : 'Unknown'} />
            </div>
          ))
        )}
      </div>
    </div>
  );
};
