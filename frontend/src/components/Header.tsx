import React, { useState, useEffect } from 'react';

export type DashboardView = 'nats-demo' | 'studio';

interface HeaderProps {
  systemOk?: boolean;
  natsConnected?: boolean;
  activeView?: DashboardView;
  onViewChange?: (view: DashboardView) => void;
}

export const Header: React.FC<HeaderProps> = ({ 
  systemOk = true, 
  natsConnected = true,
  activeView = 'nats-demo',
  onViewChange,
}) => {
  const [time, setTime] = useState<string>('');

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setTime(now.toTimeString().split(' ')[0]);
    };
    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, []);

  const isConnected = natsConnected && systemOk;

  return (
    <header className="app-header">
      <div className="app-title-group" style={{ display: 'flex', alignItems: 'center', gap: '1.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span className="app-badge-logo">NATS</span>
          <h1 className="app-title">NATS Inspector</h1>
        </div>

        {/* View Switcher Pills */}
        {onViewChange && (
          <div style={{ 
            display: 'inline-flex', 
            background: 'var(--bg-secondary)', 
            border: '1px solid var(--border-color)', 
            borderRadius: '6px', 
            padding: '2px',
            gap: '2px'
          }}>
            <button
              type="button"
              style={{
                background: activeView === 'nats-demo' ? '#10B981' : 'transparent',
                color: activeView === 'nats-demo' ? '#FFFFFF' : 'var(--text-muted)',
                border: 'none',
                borderRadius: '4px',
                padding: '4px 12px',
                fontSize: '0.78rem',
                fontWeight: activeView === 'nats-demo' ? 600 : 500,
                cursor: 'pointer',
                transition: 'all 0.15s ease',
              }}
              onClick={() => onViewChange('nats-demo')}
            >
              NATS Demo
            </button>
            <button
              type="button"
              style={{
                background: activeView === 'studio' ? '#3B82F6' : 'transparent',
                color: activeView === 'studio' ? '#FFFFFF' : 'var(--text-muted)',
                border: 'none',
                borderRadius: '4px',
                padding: '4px 12px',
                fontSize: '0.78rem',
                fontWeight: activeView === 'studio' ? 600 : 500,
                cursor: 'pointer',
                transition: 'all 0.15s ease',
              }}
              onClick={() => onViewChange('studio')}
            >
              Capability Studio
            </button>
          </div>
        )}
      </div>

      <div className="app-meta-group">
        <div className="mono-cell" style={{ letterSpacing: '0.05em' }}>
          {time}
        </div>
        <a
          href="http://localhost:3000"
          target="_blank"
          rel="noopener noreferrer"
          className="mono-cell"
          style={{ textDecoration: 'none', color: '#C084FC', background: 'rgba(192, 132, 252, 0.08)', border: '1px solid rgba(192, 132, 252, 0.25)', borderRadius: '4px', padding: '0.2rem 0.5rem', display: 'inline-flex', alignItems: 'center', gap: '0.25rem' }}
          title="Open Grafana Dashboard"
        >
          <span>Grafana (:3000)</span>
          <span style={{ fontSize: '0.75rem' }}>-&gt;</span>
        </a>
        {isConnected ? (
          <div className="system-status-indicator">
            <span style={{ fontWeight: 'bold' }}>*</span>
            <span>NATS CONNECTED</span>
          </div>
        ) : (
          <div className="system-status-indicator" style={{ color: 'var(--status-danger)', background: 'rgba(239, 68, 68, 0.08)', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
            <span style={{ fontWeight: 'bold' }}>*</span>
            <span>NATS DISCONNECTED</span>
          </div>
        )}
      </div>
    </header>
  );
};
