import React from 'react';
import { CoreFlowPublisher } from './CoreFlowPublisher';
import { CoreFlowNatsCli } from './CoreFlowNatsCli';
import { CoreFlowProcessor } from './CoreFlowProcessor';
import { Job, JetStreamInfo, ConsumerStatus, Activity } from '../../api/demoApi';

interface CoreFlowViewProps {
  onPublishJob: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  activities: Activity[];
  isProcessing: boolean;
  onToggleProcessor: (enabled: boolean) => Promise<void>;
  onClearActivity: () => void;
  onSelectJob: (jobId: string) => void;
  onRefreshNats: () => void;
  onShowInfo?: (key: string) => void;
}

export const CoreFlowView: React.FC<CoreFlowViewProps> = ({
  onPublishJob,
  isSubmitting,
  jetstreamInfo,
  consumerStatus,
  activities,
  isProcessing,
  onToggleProcessor,
  onClearActivity,
  onSelectJob,
  onRefreshNats,
  onShowInfo,
}) => {
  return (
    <div className="core-flow-view" style={{ display: 'flex', flexDirection: 'column', gap: '1rem', width: '100%' }}>
      {/* Visual Flow Header / Pipeline Tracker */}
      <div style={{
        background: 'var(--card-bg)',
        border: '1px solid var(--border-color)',
        borderRadius: '6px',
        padding: '0.75rem 1rem',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: '0.5rem',
      }}>
        <div>
          <h3 style={{ margin: 0, fontSize: '0.95rem', fontWeight: 600, color: 'var(--text-bright)' }}>
            NATS Platform Job Lifecycle
          </h3>
          <p style={{ margin: '2px 0 0 0', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
            End-to-end trace: Publish Message to NATS -&gt; Broker persists/routes message -&gt; Processor worker pulls and processes message.
          </p>
        </div>

        {/* Step Progression Indicators */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.75rem' }}>
          <span style={{ 
            background: 'rgba(59, 130, 246, 0.15)', 
            color: '#60A5FA', 
            padding: '2px 8px', 
            borderRadius: '4px',
            border: '1px solid rgba(59, 130, 246, 0.3)'
          }}>
            1. Publish Message
          </span>
          <span style={{ color: 'var(--text-dim)' }}>-&gt;</span>
          <span style={{ 
            background: 'rgba(16, 185, 129, 0.15)', 
            color: '#34D399', 
            padding: '2px 8px', 
            borderRadius: '4px',
            border: '1px solid rgba(16, 185, 129, 0.3)'
          }}>
            2. NATS Broker (Stream/Subject)
          </span>
          <span style={{ color: 'var(--text-dim)' }}>-&gt;</span>
          <span style={{ 
            background: 'rgba(245, 158, 11, 0.15)', 
            color: '#FBBF24', 
            padding: '2px 8px', 
            borderRadius: '4px',
            border: '1px solid rgba(245, 158, 11, 0.3)'
          }}>
            3. Process Job (Worker)
          </span>
        </div>
      </div>

      {/* 3-Column Core Flow Grid */}
      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', 
        gap: '1rem',
        minHeight: '560px',
        alignItems: 'stretch'
      }}>
        {/* Column 1: Publisher View */}
        <CoreFlowPublisher
          onPublish={onPublishJob}
          isSubmitting={isSubmitting}
          onShowInfo={onShowInfo}
        />

        {/* Column 2: NATS View & CLI Guide */}
        <CoreFlowNatsCli
          jetstreamInfo={jetstreamInfo}
          consumerStatus={consumerStatus}
          onRefresh={onRefreshNats}
          onShowInfo={onShowInfo}
        />

        {/* Column 3: Processor View */}
        <CoreFlowProcessor
          activities={activities}
          isProcessing={isProcessing}
          onToggleProcessor={onToggleProcessor}
          onClearActivity={onClearActivity}
          onSelectJob={onSelectJob}
          onShowInfo={onShowInfo}
          consumerStatus={consumerStatus}
          jetstreamInfo={jetstreamInfo}
        />
      </div>
    </div>
  );
};
