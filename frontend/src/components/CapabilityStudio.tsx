import React, { useState } from 'react';
import { Job, JetStreamInfo, ReplayRequest, ReplayResponse, Activity } from '../api/demoApi';
import { JobPanel } from './JobPanel';
import { DeduplicationPanel } from './DeduplicationPanel';
import { RequestReplyPanel } from './RequestReplyPanel';
import { DLQPanel } from './DLQPanel';
import { ReplayPanel } from './ReplayPanel';
import { QueueGroupPanel } from './QueueGroupPanel';

export type StudioTab = 'pubsub' | 'queue-group' | 'request-reply' | 'dlq' | 'replay';

interface CapabilityStudioProps {
  // Job Actions
  onSubmitJob: (job: Job) => Promise<void>;
  onValidateJob: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  isValidating: boolean;
  // Replay Actions
  jetstreamInfo?: JetStreamInfo | null;
  onTriggerReplay: (req: ReplayRequest) => Promise<ReplayResponse>;
  onRefreshStatus: () => void;
  isRefreshingStatus: boolean;
  // Request / Reply
  activities: Activity[];
  onRefreshActivity: () => void;
  // Common
  onShowInfo: (key: string) => void;
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
  onRefreshAll?: () => void;
}

export const CapabilityStudio: React.FC<CapabilityStudioProps> = ({
  onSubmitJob,
  onValidateJob,
  isSubmitting,
  isValidating,
  jetstreamInfo,
  onTriggerReplay,
  onRefreshStatus,
  isRefreshingStatus,
  activities,
  onRefreshActivity,
  onShowInfo,
  onAlert,
  onRefreshAll,
}) => {
  const [activeTab, setActiveTab] = useState<StudioTab>('pubsub');
  const [pubsubSubMode, setPubsubSubMode] = useState<'standard' | 'dedup'>('standard');

  return (
    <div className="capability-studio">
      {/* Studio Header & Segmented Pill Navigation Bar */}
      <div className="studio-tabs-bar">
        <button
          type="button"
          className={`studio-tab-btn ${activeTab === 'pubsub' ? 'active' : ''}`}
          onClick={() => setActiveTab('pubsub')}
        >
          <span className="tab-label">Pub/Sub & Stream</span>
        </button>

        <button
          type="button"
          className={`studio-tab-btn ${activeTab === 'queue-group' ? 'active' : ''}`}
          onClick={() => setActiveTab('queue-group')}
        >
          <span className="tab-label">Queue Groups</span>
        </button>

        <button
          type="button"
          className={`studio-tab-btn ${activeTab === 'request-reply' ? 'active' : ''}`}
          onClick={() => setActiveTab('request-reply')}
        >
          <span className="tab-label">Request / Reply</span>
        </button>

        <button
          type="button"
          className={`studio-tab-btn ${activeTab === 'dlq' ? 'active' : ''}`}
          onClick={() => setActiveTab('dlq')}
        >
          <span className="tab-label">Dead Letter Queue</span>
        </button>

        <button
          type="button"
          className={`studio-tab-btn ${activeTab === 'replay' ? 'active' : ''}`}
          onClick={() => setActiveTab('replay')}
        >
          <span className="tab-label">Stream Replay</span>
        </button>
      </div>

      {/* Active Tab Panel Content */}
      <div className="studio-content-area">
        {activeTab === 'pubsub' && (
          <div>
            <div className="studio-sub-toggle-bar">
              <button
                type="button"
                className={`studio-sub-btn ${pubsubSubMode === 'standard' ? 'active' : ''}`}
                onClick={() => setPubsubSubMode('standard')}
              >
                Standard Submission
              </button>
              <button
                type="button"
                className={`studio-sub-btn ${pubsubSubMode === 'dedup' ? 'active' : ''}`}
                onClick={() => setPubsubSubMode('dedup')}
              >
                Deduplication Test Bench
              </button>
            </div>

            {pubsubSubMode === 'standard' ? (
              <JobPanel
                onSubmitJob={onSubmitJob}
                onValidateJob={onValidateJob}
                isSubmitting={isSubmitting}
                isValidating={isValidating}
                onShowInfo={onShowInfo}
              />
            ) : (
              <DeduplicationPanel
                onSubmitJob={onSubmitJob}
                isSubmitting={isSubmitting}
                onShowInfo={onShowInfo}
              />
            )}
          </div>
        )}

        {activeTab === 'queue-group' && (
          <QueueGroupPanel
            onShowInfo={onShowInfo}
            onAlert={onAlert}
            onMessagesSent={onRefreshActivity}
          />
        )}

        {activeTab === 'request-reply' && (
          <RequestReplyPanel
            activities={activities}
            onValidated={onRefreshActivity}
            onShowInfo={onShowInfo}
          />
        )}

        {activeTab === 'dlq' && (
          <DLQPanel
            onShowInfo={onShowInfo}
            onAlert={onAlert}
            onRefreshAll={onRefreshAll}
            onActivityUpdated={onRefreshActivity}
          />
        )}

        {activeTab === 'replay' && (
          <ReplayPanel
            jetstreamInfo={jetstreamInfo}
            onTriggerReplay={onTriggerReplay}
            onShowInfo={onShowInfo}
            onRefresh={onRefreshStatus}
            isRefreshing={isRefreshingStatus}
          />
        )}
      </div>
    </div>
  );
};
