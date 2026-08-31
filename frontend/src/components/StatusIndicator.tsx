import React from 'react';

interface StatusIndicatorProps {
  status: 'active' | 'connected' | 'disconnected' | 'unknown';
  label?: string;
}

export const StatusIndicator: React.FC<StatusIndicatorProps> = ({ status, label }) => {
  const getDisplayLabel = () => {
    if (label) return label;
    switch (status) {
      case 'active':
        return 'Active';
      case 'connected':
        return 'Connected';
      case 'disconnected':
        return 'Disconnected';
      default:
        return 'Unknown';
    }
  };

  return (
    <div className="indicator">
      <span className={`indicator-dot ${status}`} />
      <span>{getDisplayLabel()}</span>
    </div>
  );
};
