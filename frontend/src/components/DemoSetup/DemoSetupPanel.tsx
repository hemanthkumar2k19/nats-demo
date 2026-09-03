import React, { useState } from 'react';
import { ServiceStatus, JetStreamInfo, ConsumerStatus, DLQStatus, QueueGroupStatus } from '../../api/demoApi';
import { DemoTopology } from './DemoTopology';
import { ConsumerLabPanel } from '../ConsumerLabPanel';
import { QueueGroupPanel } from '../QueueGroupPanel';

interface DemoSetupPanelProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  dlqStatus?: DLQStatus | null;
  queueGroupStatus?: QueueGroupStatus | null;
  onShowInfo: (key: string) => void;
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
  onActivityUpdated?: () => void;
  onConfigChanged?: (status: ConsumerStatus) => void;
  onQueueGroupConfigChanged?: (status: QueueGroupStatus) => void;
}

export const DemoSetupPanel: React.FC<DemoSetupPanelProps> = ({
  services,
  jetstreamInfo,
  consumerStatus,
  dlqStatus,
  queueGroupStatus,
  onShowInfo,
  onAlert,
  onActivityUpdated,
  onConfigChanged,
  onQueueGroupConfigChanged,
}) => {
  const [rightPanelMode, setRightPanelMode] = useState<'queue-group' | 'consumer'>('queue-group');
  const processorService = services.find((s) => s.name.toLowerCase().includes('processor'));
  const isProcessing = processorService?.processing ?? false;

  return (
    <section className="panel demo-setup-panel">
      <div className="panel-header demo-setup-header">
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
      </div>

      <div className="demo-setup-layout">
        {/* Left Side: Topology Visualizer */}
        <div className="demo-topology-wrapper">
          <DemoTopology
            services={services}
            jetstreamInfo={jetstreamInfo}
            consumerStatus={consumerStatus}
            dlqStatus={dlqStatus}
            queueGroupStatus={queueGroupStatus}
            onSelectInfo={onShowInfo}
          />
        </div>

        {/* Right Side: Consumer Lab & Controller */}
        <div className="demo-consumer-lab-wrapper">
          {/* Segmented Switcher: Core NATS Queue Group on LEFT, JetStream Consumer Lab on RIGHT */}
          <div className="lab-switcher-bar" style={{ display: 'flex', gap: '0.25rem', marginBottom: '0.75rem', background: 'rgba(0, 0, 0, 0.25)', padding: '0.25rem', borderRadius: '6px' }}>
            <button
              type="button"
              className={`studio-sub-btn ${rightPanelMode === 'queue-group' ? 'active' : ''}`}
              style={{ flex: 1, padding: '0.3rem 0.5rem', fontSize: '0.75rem', textAlign: 'center' }}
              onClick={() => setRightPanelMode('queue-group')}
            >
              Core NATS Queue Group
            </button>
            <button
              type="button"
              className={`studio-sub-btn ${rightPanelMode === 'consumer' ? 'active' : ''}`}
              style={{ flex: 1, padding: '0.3rem 0.5rem', fontSize: '0.75rem', textAlign: 'center' }}
              onClick={() => setRightPanelMode('consumer')}
            >
              JetStream Consumer Lab
            </button>
          </div>

          {rightPanelMode === 'consumer' ? (
            <ConsumerLabPanel
              onAlert={onAlert}
              onConfigChanged={onConfigChanged}
              onShowInfo={onShowInfo}
              onActivityUpdated={onActivityUpdated}
              isEmbedded={true}
              isProcessing={isProcessing}
            />
          ) : (
            <QueueGroupPanel
              onShowInfo={onShowInfo}
              onAlert={onAlert}
              onActivityUpdated={onActivityUpdated}
              onConfigChanged={onQueueGroupConfigChanged}
              isEmbedded={true}
            />
          )}
        </div>
      </div>
    </section>
  );
};
