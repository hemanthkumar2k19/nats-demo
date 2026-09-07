import React, { useState } from 'react';
import { ServiceStatus, JetStreamInfo, ConsumerStatus, DLQStatus, QueueGroupStatus } from '../../api/demoApi';
import { DemoTopology } from './DemoTopology';

interface DemoSetupPanelProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  dlqStatus?: DLQStatus | null;
  queueGroupStatus?: QueueGroupStatus | null;
  onShowInfo: (key: string) => void;
}

export const DemoSetupPanel: React.FC<DemoSetupPanelProps> = ({
  services,
  jetstreamInfo,
  consumerStatus,
  dlqStatus,
  queueGroupStatus,
  onShowInfo,
}) => {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);

  return (
    <section className={`panel demo-setup-panel panel-collapsible ${isExpanded ? 'expanded' : 'collapsed'}`}>
      <div
        className="panel-header demo-setup-header panel-header-interactive"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
            </svg>
            CURRENT DEMO SETUP
          </h2>
          <span className="panel-subtitle">
            Current runtime topology and contextual NATS information - click (i) on any component to explore concepts
          </span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <button
            type="button"
            className="collapse-toggle-btn"
            onClick={(e) => {
              e.stopPropagation();
              setIsExpanded(!isExpanded);
            }}
            aria-expanded={isExpanded}
            title={isExpanded ? 'Collapse Current Demo Setup' : 'Expand Current Demo Setup'}
          >
            <span>{isExpanded ? 'Collapse' : 'Expand'}</span>
            <svg
              className={`collapse-chevron ${isExpanded ? 'open' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>
      </div>

      <div className={`panel-collapsible-body ${isExpanded ? 'open' : 'collapsed'}`}>
        <DemoTopology
          services={services}
          jetstreamInfo={jetstreamInfo}
          consumerStatus={consumerStatus}
          dlqStatus={dlqStatus}
          queueGroupStatus={queueGroupStatus}
          onSelectInfo={onShowInfo}
        />
      </div>
    </section>
  );
};

