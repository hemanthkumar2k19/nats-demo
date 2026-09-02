import React, { useState } from 'react';
import { Activity, AddressingSubscription, AddressingEvent } from '../api/demoApi';
import { ActivityPanel } from './ActivityPanel';
import { AddressingPanel } from './AddressingPanel';

export type ObservabilityView = 'activity' | 'addressing';

interface ObservabilityPanelContainerProps {
  // Activity Log props
  activities: Activity[];
  onRefreshActivity: () => void;
  isLoadingActivity: boolean;
  onSelectJob: (jobId: string) => void;
  // Addressing props
  subscriptions: AddressingSubscription[];
  addressingEvents: AddressingEvent[];
  onRefreshAddressing: () => void;
  isLoadingAddressing: boolean;
  // Common
  onShowInfo: (key: string) => void;
}

export const ObservabilityPanelContainer: React.FC<ObservabilityPanelContainerProps> = ({
  activities,
  onRefreshActivity,
  isLoadingActivity,
  onSelectJob,
  subscriptions,
  addressingEvents,
  onRefreshAddressing,
  isLoadingAddressing,
  onShowInfo,
}) => {
  const [obsView, setObsView] = useState<ObservabilityView>('activity');

  return (
    <div className="observability-panel-container">
      {/* Top Segmented View Switcher */}
      <div className="obs-switcher-bar">
        <button
          type="button"
          className={`obs-switcher-btn ${obsView === 'activity' ? 'active' : ''}`}
          onClick={() => setObsView('activity')}
        >
          <svg style={{ width: '13px', height: '13px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
          </svg>
          <span>Live Activity Log</span>
          <span className="obs-switcher-badge font-mono">{activities.length}</span>
        </button>

        <button
          type="button"
          className={`obs-switcher-btn ${obsView === 'addressing' ? 'active' : ''}`}
          onClick={() => setObsView('addressing')}
        >
          <svg style={{ width: '13px', height: '13px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <span>Subject Addressing & Wildcards</span>
          <span className="obs-switcher-badge font-mono">{subscriptions.length} Subs</span>
        </button>
      </div>

      {/* Render Active Observability View */}
      <div className="obs-content-area">
        {obsView === 'activity' ? (
          <ActivityPanel
            activities={activities}
            onRefresh={onRefreshActivity}
            isLoading={isLoadingActivity}
            onSelectJob={onSelectJob}
            onShowInfo={onShowInfo}
          />
        ) : (
          <AddressingPanel
            subscriptions={subscriptions}
            events={addressingEvents}
            onRefresh={onRefreshAddressing}
            isLoading={isLoadingAddressing}
            onShowInfo={onShowInfo}
          />
        )}
      </div>
    </div>
  );
};
