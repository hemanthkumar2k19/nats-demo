import React, { useState, useEffect } from 'react';

interface HeaderProps {
  systemOk: boolean;
}

export const Header: React.FC<HeaderProps> = ({ systemOk }) => {
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

  return (
    <header className="app-header">
      <div className="app-title-group">
        <span className="app-logo">⚡</span>
        <h1 className="app-title">NATS Platform Demo Console</h1>
      </div>
      
      <div className="app-meta-group">
        <div className="mono-cell" style={{ letterSpacing: '0.05em' }}>
          {time}
        </div>
        {systemOk ? (
          <div className="system-status-indicator">
            <span>●</span>
            <span>SYSTEM OK</span>
          </div>
        ) : (
          <div className="system-status-indicator" style={{ color: 'var(--status-danger)', background: 'rgba(239, 68, 68, 0.08)', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
            <span>●</span>
            <span>SYSTEM ERR</span>
          </div>
        )}
      </div>
    </header>
  );
};
