import React from 'react';
import { Activity } from '../api/demoApi';

interface ActivityPanelProps {
  activities: Activity[];
  onRefresh: () => void;
  isLoading: boolean;
  onSelectJob: (jobId: string) => void;
}

export const ActivityPanel: React.FC<ActivityPanelProps> = ({ activities, onRefresh, isLoading, onSelectJob }) => {
  const getBadgeClass = (event: string) => {
    switch (event.toUpperCase()) {
      case 'SUBMITTED':
        return 'badge-submitted';
      case 'PROCESSING':
        return 'badge-processing';
      case 'COMPLETED':
        return 'badge-completed';
      case 'FAILED':
        return 'badge-failed';
      default:
        return '';
    }
  };

  return (
    <div className="panel" style={{ flex: 1 }}>
      <div className="panel-header">
        <h2 className="panel-title">
          <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
          </svg>
          Activity Log
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
        {activities.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📥</div>
            <p style={{ fontWeight: 500, marginBottom: '0.25rem' }}>No activity logged yet</p>
            <p style={{ fontSize: '0.8125rem' }}>Submit a job using the panel on the left to see NATS activity.</p>
          </div>
        ) : (
          <table className="activity-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Job ID</th>
                <th>Event</th>
                <th>Subject</th>
                <th>Worker</th>
                <th style={{ textAlign: 'center' }}>Dlv. Count</th>
              </tr>
            </thead>
            <tbody>
              {activities.map((act, index) => (
                <tr key={`${act.job_id}-${act.event}-${index}`}>
                  <td className="mono-cell" style={{ color: 'var(--text-secondary)' }}>
                    {act.timestamp}
                  </td>
                  <td className="mono-cell" style={{ fontWeight: 600 }}>
                    <button
                      onClick={() => onSelectJob(act.job_id)}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: 'var(--accent-cyan)',
                        fontFamily: 'inherit',
                        fontSize: 'inherit',
                        fontWeight: 'inherit',
                        cursor: 'pointer',
                        padding: 0,
                        textDecoration: 'underline',
                      }}
                    >
                      {act.job_id}
                    </button>
                  </td>
                  <td>
                    <span className={`badge ${getBadgeClass(act.event)}`}>
                      {act.event}
                    </span>
                  </td>
                  <td className="mono-cell" style={{ color: 'var(--accent-cyan)' }}>
                    {act.subject}
                  </td>
                  <td className="mono-cell" style={{ color: 'var(--text-secondary)' }}>
                    {act.worker || '-'}
                  </td>
                  <td className="mono-cell" style={{ textAlign: 'center', color: act.delivery_count > 1 ? 'var(--status-warning)' : 'var(--text-muted)' }}>
                    {act.delivery_count}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};
