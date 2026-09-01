import React from 'react';
import { AddressingSubscription, AddressingEvent } from '../api/demoApi';

interface AddressingPanelProps {
  subscriptions: AddressingSubscription[];
  events: AddressingEvent[];
  onRefresh: () => void;
  isLoading: boolean;
  onShowInfo?: (key: string) => void;
}

export const AddressingPanel: React.FC<AddressingPanelProps> = ({
  subscriptions,
  events,
  onRefresh,
  isLoading,
  onShowInfo,
}) => {
  // Helper to check if a subscription received a subject
  const hasReceived = (event: AddressingEvent, subName: string): boolean => {
    return event.received_by.includes(subName);
  };

  // Helper to format ISO time to simple format hh:mm:ss
  const formatTime = (isoString: string): string => {
    try {
      const d = new Date(isoString);
      return d.toLocaleTimeString();
    } catch {
      return '-';
    }
  };

  return (
    <div className="panel" style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
      {/* 1. Active Subscriptions Section */}
      <div>
        <div className="panel-header" style={{ marginBottom: '0.75rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <h2 className="panel-title">
              <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
              NATS Subject Addressing
            </h2>
            {onShowInfo && (
              <button
                type="button"
                className="node-info-btn"
                onClick={() => onShowInfo('subject-addressing')}
                title="Learn about NATS Subject Addressing"
              >
                (i)
              </button>
            )}
          </div>
        </div>
        <div className="activity-table-wrapper">
          <table className="activity-table">
            <thead>
              <tr>
                <th>Subscription Type</th>
                <th>Subject Pattern</th>
              </tr>
            </thead>
            <tbody>
              {subscriptions.length === 0 ? (
                <tr>
                  <td colSpan={2} style={{ textAlign: 'center', color: 'var(--text-muted)' }}>
                    No active subscriptions loaded
                  </td>
                </tr>
              ) : (
                subscriptions.map((sub) => (
                  <tr key={sub.name}>
                    <td style={{ fontWeight: 600, fontFamily: 'var(--font-sans)' }}>
                      {sub.name === 'exact' ? 'Exact Matching' : sub.name === 'single-level' ? 'Single Level (*)' : 'Multi Level (>)'}
                    </td>
                    <td className="mono-cell" style={{ color: 'var(--accent-cyan)' }}>
                      {sub.subject}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* 2. Observed Message Activity Section */}
      <div>
        <div className="panel-header" style={{ marginBottom: '0.75rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10a2 2 0 01-2 2h-2a2 2 0 01-2-2zm9-4a2 2 0 11-4 0 2 2 0 014 0z" />
            </svg>
            Subject Routing Activity
          </h2>
          <button 
            className="btn btn-secondary" 
            onClick={onRefresh} 
            disabled={isLoading}
            style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
          >
            {isLoading ? 'Loading...' : 'Refresh'}
          </button>
        </div>

        <div className="activity-table-wrapper">
          {events.length === 0 ? (
            <div className="empty-state" style={{ padding: '1.5rem 1rem' }}>
              <p style={{ fontWeight: 500, marginBottom: '0.25rem' }}>No routing activity observed yet</p>
              <p style={{ fontSize: '0.8125rem' }}>Submit jobs or simulate failures to generate lifecycle events on NATS.</p>
            </div>
          ) : (
            <table className="activity-table">
              <thead>
                <tr>
                  <th>Job ID</th>
                  <th>Time</th>
                  <th>Subject</th>
                  <th style={{ textAlign: 'center' }}>Exact</th>
                  <th style={{ textAlign: 'center' }}>Single-Level (*)</th>
                  <th style={{ textAlign: 'center' }}>Multi-Level (&gt;)</th>
                </tr>
              </thead>
              <tbody>
                {events.map((evt, idx) => (
                  <tr key={`${evt.subject}-${idx}`}>
                    <td className="mono-cell" style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                      {evt.job_id || '-'}
                    </td>
                    <td className="mono-cell" style={{ color: 'var(--text-secondary)' }}>
                      {formatTime(evt.timestamp)}
                    </td>
                    <td className="mono-cell" style={{ fontWeight: 600, color: 'var(--accent-cyan)' }}>
                      {evt.subject}
                    </td>
                    <td style={{ textAlign: 'center' }}>
                      <span 
                        style={{ 
                          color: hasReceived(evt, 'exact') ? 'var(--status-success)' : 'var(--text-muted)',
                          fontWeight: hasReceived(evt, 'exact') ? 'bold' : 'normal'
                        }}
                      >
                        {hasReceived(evt, 'exact') ? 'Yes' : 'No'}
                      </span>
                    </td>
                    <td style={{ textAlign: 'center' }}>
                      <span 
                        style={{ 
                          color: hasReceived(evt, 'single-level') ? 'var(--status-success)' : 'var(--text-muted)',
                          fontWeight: hasReceived(evt, 'single-level') ? 'bold' : 'normal'
                        }}
                      >
                        {hasReceived(evt, 'single-level') ? 'Yes' : 'No'}
                      </span>
                    </td>
                    <td style={{ textAlign: 'center' }}>
                      <span 
                        style={{ 
                          color: hasReceived(evt, 'multi-level') ? 'var(--status-success)' : 'var(--text-muted)',
                          fontWeight: hasReceived(evt, 'multi-level') ? 'bold' : 'normal'
                        }}
                      >
                        {hasReceived(evt, 'multi-level') ? 'Yes' : 'No'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
};
