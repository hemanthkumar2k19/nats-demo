import React, { useState, useEffect } from 'react';
import { AddressingSubscription, AddressingEvent, JobDetailResponse } from '../api/demoApi';
import { AddressingPanel } from './AddressingPanel';
import { JobInspectorPanel } from './JobInspectorPanel';

export type DockTab = 'addressing' | 'inspector';

interface ObservabilityDockProps {
  // Addressing props
  subscriptions: AddressingSubscription[];
  events: AddressingEvent[];
  onRefreshAddressing: () => void;
  isLoadingAddressing: boolean;
  // Inspector props
  selectedJobId: string | null;
  selectedJobDetail: JobDetailResponse | null;
  isLoadingInspector: boolean;
  inspectorError: string | null;
  onCloseInspector: () => void;
  // Common
  onShowInfo: (key: string) => void;
}

export const ObservabilityDock: React.FC<ObservabilityDockProps> = ({
  subscriptions,
  events,
  onRefreshAddressing,
  isLoadingAddressing,
  selectedJobId,
  selectedJobDetail,
  isLoadingInspector,
  inspectorError,
  onCloseInspector,
  onShowInfo,
}) => {
  const [activeTab, setActiveTab] = useState<DockTab>('addressing');

  // Automatically switch to Job Inspector when a job is clicked in the Activity Log
  useEffect(() => {
    if (selectedJobId) {
      setActiveTab('inspector');
    }
  }, [selectedJobId]);

  const handleCloseInspector = () => {
    onCloseInspector();
    setActiveTab('addressing');
  };

  return (
    <div className="observability-dock">
      {/* Dock Navigation Bar */}
      <div className="dock-tabs-bar">
        <button
          type="button"
          className={`dock-tab-btn ${activeTab === 'addressing' ? 'active' : ''}`}
          onClick={() => setActiveTab('addressing')}
        >
          <svg style={{ width: '13px', height: '13px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <span className="dock-tab-title">Subject Addressing & Routing</span>
          <span className="dock-tab-badge font-mono">{subscriptions.length} Subs</span>
        </button>

        <button
          type="button"
          className={`dock-tab-btn ${activeTab === 'inspector' ? 'active' : ''}`}
          onClick={() => setActiveTab('inspector')}
        >
          <svg style={{ width: '13px', height: '13px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <span className="dock-tab-title">Job Inspector</span>
          {selectedJobId ? (
            <span className="dock-tab-badge badge-active font-mono">{selectedJobId}</span>
          ) : (
            <span className="dock-tab-badge font-mono text-muted">None Selected</span>
          )}
        </button>
      </div>

      {/* Dock Content */}
      <div className="dock-content-area">
        {activeTab === 'addressing' && (
          <AddressingPanel
            subscriptions={subscriptions}
            events={events}
            onRefresh={onRefreshAddressing}
            isLoading={isLoadingAddressing}
            onShowInfo={onShowInfo}
          />
        )}

        {activeTab === 'inspector' && (
          <div>
            {!selectedJobId && !isLoadingInspector && !inspectorError ? (
              <div className="dock-empty-inspector">
                <p>No job selected for inspection.</p>
                <span className="text-muted" style={{ fontSize: '0.75rem' }}>
                  Click on any row in the Activity Log above to inspect its headers, payload, delivery sequence, and OpenTelemetry trace.
                </span>
              </div>
            ) : (
              <JobInspectorPanel
                job={selectedJobDetail}
                isLoading={isLoadingInspector}
                error={inspectorError}
                onClose={handleCloseInspector}
                onShowInfo={onShowInfo}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
};
