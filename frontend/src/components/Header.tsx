import React, { useState, useEffect } from 'react';

interface HeaderProps {
  systemOk?: boolean;
  natsConnected?: boolean;
}

export const Header: React.FC<HeaderProps> = ({ systemOk = true, natsConnected = true }) => {
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
      <div className="app-title-group">
        <span className="app-badge-logo">NATS</span>
        <h1 className="app-title">NATS Inspector</h1>
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
