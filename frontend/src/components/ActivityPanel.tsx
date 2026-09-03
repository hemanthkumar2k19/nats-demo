import React, { useState, useMemo } from 'react';
import { Activity } from '../api/demoApi';

export type ActivityCategory = 'ALL' | 'BUSINESS' | 'LIFECYCLE';

export const isLifecycleActivity = (act: Activity): boolean => {
  // Pure worker processing, acknowledgment, transport, and demo simulation events
  const lifecycleEvents = [
    'RECEIVED',
    'DELIVERED',
    'REQUEST_RECEIVED',
    'PROCESSING',
    'ACKED',
    'STORED',
    'DEDUPLICATED',
    'NAK_WITH_DELAY',
    'ACK_TIMEOUT_SIMULATED',
    'DLQ_PUBLISHED',
    'REPLAYED',
    'NO CONSUMER',
    'NO_ACTIVE_CONSUMER',
    'REQUEST_TIMEOUT',
  ];

  const lifecycleSubjects = [
    'jobs.received',
    'jobs.queue.received',
    'jobs.delivered',
    'jobs.processing',
    'jobs.processing.started',
    'jobs.processing.completed',
    'jobs.processing.failed',
    'jobs.acked',
    'jobs.stored',
    'jobs.deduplicated',
    'jobs.nak.delayed',
    'jobs.ack.timeout',
    'jobs.dlq',
    'jobs.dlq.published',
    'jobs.replayed',
    'jobs.noconsumer',
    'jobs.request.received',
  ];

  const ev = (act.event || '').toUpperCase();
  const sub = (act.subject || '').toLowerCase();

  return lifecycleEvents.includes(ev) || lifecycleSubjects.includes(sub);
};

export const isBusinessActivity = (act: Activity): boolean => {
  // All Saga transactions and domain job submissions are business messages
  return !isLifecycleActivity(act);
};

interface ActivityPanelProps {
  activities: Activity[];
  onRefresh: () => void;
  onClearActivity: () => void;
  isLoading: boolean;
  onSelectJob: (jobId: string) => void;
  onShowInfo?: (key: string) => void;
}

export const ActivityPanel: React.FC<ActivityPanelProps> = ({
  activities,
  onRefresh,
  onClearActivity,
  isLoading,
  onSelectJob,
  onShowInfo,
}) => {
  const [messageCategory, setMessageCategory] = useState<ActivityCategory>('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedEvent, setSelectedEvent] = useState('ALL');
  const [selectedMode, setSelectedMode] = useState('ALL');
  const [selectedWorker, setSelectedWorker] = useState('ALL');
  const [eventLimit, setEventLimit] = useState<number>(15);

  const businessCount = useMemo(() => activities.filter(isBusinessActivity).length, [activities]);
  const lifecycleCount = useMemo(() => activities.length - businessCount, [activities, businessCount]);

  const getBadgeClass = (event: string) => {
    switch (event.toUpperCase()) {
      case 'PUBLISHED':
      case 'SUBMITTED':
      case 'REQUEST_SENT':
        return 'badge-submitted';
      case 'STORED':
        return 'badge-stored';
      case 'DEDUPLICATED':
        return 'badge-deduplicated';
      case 'REDELIVERED':
        return 'badge-redelivered';
      case 'RECEIVED':
      case 'DELIVERED':
      case 'REQUEST_RECEIVED':
        return 'badge-delivered';
      case 'PROCESSING':
      case 'REPLY_SENT':
      case 'SAGA_STARTED':
      case 'SAGA_STEP':
        return 'badge-processing';
      case 'COMPENSATING':
      case 'SAGA_COMPENSATING':
        return 'badge-retrying';
      case 'COMPLETED':
      case 'ACKED':
      case 'REPLY_RECEIVED':
      case 'SAGA_COMPLETED':
      case 'SAGA_COMPENSATED':
        return 'badge-completed';
      case 'FAILED':
      case 'NO CONSUMER':
      case 'NO_ACTIVE_CONSUMER':
      case 'REQUEST_TIMEOUT':
      case 'SAGA_FAILED':
        return 'badge-failed';
      case 'REPLAYED':
        return 'badge-replayed';
      case 'REPROCESSED':
        return 'badge-reprocessed';
      case 'DLQ_PUBLISHED':
      case 'DLQ':
        return 'badge-dlq';
      default:
        return '';
    }
  };

  // Dynamic filter options derived from activities
  const eventOptions = useMemo(() => {
    const events = Array.from(new Set(activities.map((a) => a.event).filter(Boolean)));
    return ['ALL', ...events.sort()];
  }, [activities]);

  const workerOptions = useMemo(() => {
    const workers = Array.from(new Set(activities.map((a) => a.worker).filter(Boolean)));
    return ['ALL', ...workers.sort()];
  }, [activities]);

  // Filtered activities
  const filteredActivities = useMemo(() => {
    return activities.filter((act) => {
      if (messageCategory === 'BUSINESS' && !isBusinessActivity(act)) {
        return false;
      }
      if (messageCategory === 'LIFECYCLE' && isBusinessActivity(act)) {
        return false;
      }
      if (selectedEvent !== 'ALL' && act.event !== selectedEvent) {
        return false;
      }
      if (selectedMode !== 'ALL' && act.delivery_mode !== selectedMode) {
        return false;
      }
      if (selectedWorker !== 'ALL' && act.worker !== selectedWorker) {
        return false;
      }
      if (searchQuery.trim() !== '') {
        const q = searchQuery.toLowerCase().trim();
        const matchesJobId = act.job_id.toLowerCase().includes(q);
        const matchesMsgId = (act.msg_id || '').toLowerCase().includes(q);
        const matchesJobType = (act.job_type || '').toLowerCase().includes(q);
        const matchesSubject = act.subject.toLowerCase().includes(q);
        const matchesWorker = (act.worker || '').toLowerCase().includes(q);
        const matchesEvent = act.event.toLowerCase().includes(q);
        return (
          matchesJobId ||
          matchesMsgId ||
          matchesJobType ||
          matchesSubject ||
          matchesWorker ||
          matchesEvent
        );
      }
      return true;
    });
  }, [activities, messageCategory, selectedEvent, selectedMode, selectedWorker, searchQuery]);

  // Capped visible activities
  const displayedActivities = useMemo(() => {
    if (eventLimit === 0) return filteredActivities;
    return filteredActivities.slice(0, eventLimit);
  }, [filteredActivities, eventLimit]);

  const hasActiveFilters =
    searchQuery !== '' ||
    selectedEvent !== 'ALL' ||
    selectedMode !== 'ALL' ||
    selectedWorker !== 'ALL' ||
    messageCategory !== 'ALL';

  const handleClearFilters = () => {
    setSearchQuery('');
    setSelectedEvent('ALL');
    setSelectedMode('ALL');
    setSelectedWorker('ALL');
    setMessageCategory('ALL');
  };

  const formatCompactId = (id?: string) => {
    if (!id) return '-';
    if (id.length <= 13) return id;
    return `${id.slice(0, 7)}...${id.slice(-4)}`;
  };

  return (
    <div className="panel" style={{ flex: 1 }}>
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
            </svg>
            Activity Log
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('activity-log')}
              title="Learn about Activity Log"
            >
              (i)
            </button>
          )}
        </div>
        <div style={{ display: 'flex', gap: '0.35rem' }}>
          {activities.length > 0 && (
            <button
              className="btn btn-secondary"
              onClick={onClearActivity}
              style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
              title="Clear all activity logs to start a fresh demo"
            >
              Clear Log
            </button>
          )}
          <button 
            className="btn btn-secondary" 
            onClick={onRefresh} 
            disabled={isLoading}
            style={{ padding: '0.125rem 0.5rem', fontSize: '0.75rem' }}
          >
            {isLoading ? 'Loading...' : 'Refresh'}
          </button>
        </div>
      </div>

      {/* Search and Filter Toolbar */}
      <div className="activity-toolbar">
        {/* Category Segmented Switcher (Both / Business A / Lifecycle B) */}
        <div className="activity-category-bar">
          <button
            type="button"
            className={`activity-cat-btn ${messageCategory === 'ALL' ? 'active' : ''}`}
            onClick={() => setMessageCategory('ALL')}
            title="Show all logged events (Both business messages and internal lifecycle events)"
          >
            <span>All Messages</span>
            <span className="cat-count-pill">{activities.length}</span>
          </button>
          <button
            type="button"
            className={`activity-cat-btn business ${messageCategory === 'BUSINESS' ? 'active' : ''}`}
            onClick={() => setMessageCategory('BUSINESS')}
            title="Filter to show only domain business payload messages (A)"
          >
            <span className="cat-dot business"></span>
            <span>Business Messages (A)</span>
            <span className="cat-count-pill">{businessCount}</span>
          </button>
          <button
            type="button"
            className={`activity-cat-btn lifecycle ${messageCategory === 'LIFECYCLE' ? 'active' : ''}`}
            onClick={() => setMessageCategory('LIFECYCLE')}
            title="Filter to show only internal worker and flow lifecycle events (B)"
          >
            <span className="cat-dot lifecycle"></span>
            <span>Flow / Lifecycle Events (B)</span>
            <span className="cat-count-pill">{lifecycleCount}</span>
          </button>
        </div>

        <div className="activity-search-box">
          <svg className="search-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            className="activity-search-input"
            placeholder="Search Job ID, Corr ID, Msg ID, Type, Subject..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button
              className="search-clear-btn"
              onClick={() => setSearchQuery('')}
              title="Clear search query"
            >
              x
            </button>
          )}
        </div>

        <div className="activity-filters-row">
          <div className="filter-group">
            <label className="filter-label">Event:</label>
            <select
              className="filter-select"
              value={selectedEvent}
              onChange={(e) => setSelectedEvent(e.target.value)}
            >
              {eventOptions.map((ev) => (
                <option key={ev} value={ev}>
                  {ev === 'ALL' ? 'All Events' : ev}
                </option>
              ))}
            </select>
          </div>

          <div className="filter-group">
            <label className="filter-label">Mode:</label>
            <select
              className="filter-select"
              value={selectedMode}
              onChange={(e) => setSelectedMode(e.target.value)}
            >
              <option value="ALL">All Modes</option>
              <option value="CORE">CORE</option>
              <option value="JETSTREAM">JETSTREAM</option>
            </select>
          </div>

          <div className="filter-group">
            <label className="filter-label">Worker:</label>
            <select
              className="filter-select"
              value={selectedWorker}
              onChange={(e) => setSelectedWorker(e.target.value)}
            >
              {workerOptions.map((w) => (
                <option key={w} value={w}>
                  {w === 'ALL' ? 'All Workers' : w}
                </option>
              ))}
            </select>
          </div>

          {hasActiveFilters && (
            <button className="btn-clear-filters" onClick={handleClearFilters}>
              Clear
            </button>
          )}

          <div className="filter-group" style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
            <label className="filter-label">Cap:</label>
            <select
              className="filter-select font-mono"
              value={eventLimit}
              onChange={(e) => setEventLimit(Number(e.target.value))}
              title="Cap number of events shown in view"
            >
              <option value={15}>15 Events</option>
              <option value={30}>30 Events</option>
              <option value={50}>50 Events</option>
              <option value={0}>All Events</option>
            </select>
          </div>

          <span className="filter-result-count">
            {eventLimit > 0 && filteredActivities.length > eventLimit
              ? `Showing latest ${displayedActivities.length} of ${filteredActivities.length}`
              : `${filteredActivities.length} events`}
          </span>
        </div>
      </div>

      <div className="activity-table-wrapper" style={{ maxHeight: '420px', overflowY: 'auto' }}>
        {activities.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">[ ]</div>
            <p style={{ fontWeight: 500, marginBottom: '0.25rem' }}>No activity logged yet</p>
            <p style={{ fontSize: '0.8125rem' }}>Submit a job using the panel on the left to see NATS activity.</p>
          </div>
        ) : filteredActivities.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">[ ? ]</div>
            <p style={{ fontWeight: 500, marginBottom: '0.25rem' }}>No matching events found</p>
            <p style={{ fontSize: '0.8125rem' }}>Try adjusting or clearing your search and filter criteria.</p>
            <button className="btn btn-secondary" style={{ marginTop: '0.5rem' }} onClick={handleClearFilters}>
              Reset Filters
            </button>
          </div>
        ) : (
          <table className="activity-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Job ID</th>
                <th>NATS Msg ID</th>
                <th>Type</th>
                <th>Mode</th>
                <th>Event</th>
                <th>Seq</th>
                <th>Subject</th>
                <th>Worker</th>
                <th style={{ textAlign: 'center' }}>Dlv</th>
              </tr>
            </thead>
            <tbody>
              {displayedActivities.map((act, index) => (
                <tr key={`${act.job_id}-${act.event}-${index}`}>
                  <td className="mono-cell" style={{ color: 'var(--text-secondary)' }}>
                    {act.timestamp}
                  </td>
                  <td className="mono-cell" style={{ fontWeight: 600 }}>
                    <button
                      onClick={() => onSelectJob(act.job_id)}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: 'var(--accent-cyan)',
                        fontFamily: 'inherit',
                        fontSize: 'inherit',
                        fontWeight: 'inherit',
                        cursor: 'pointer',
                        padding: 0,
                        textDecoration: 'underline',
                      }}
                      title="Inspect Job Details"
                    >
                      {act.job_id}
                    </button>
                  </td>
                  <td className="mono-cell">
                    {act.msg_id || act.job_id ? (
                      <div className="hover-legend-wrapper">
                        <span className="hover-legend-badge">
                          {formatCompactId(act.msg_id || act.job_id)}
                        </span>
                        <div className="hover-legend-tooltip">
                          <span className="hover-legend-title">NATS Msg ID</span>
                          <span>{act.msg_id || act.job_id}</span>
                        </div>
                      </div>
                    ) : (
                      <span style={{ color: 'var(--text-muted)' }}>-</span>
                    )}
                  </td>
                  <td className="mono-cell">
                    <span className="badge-type">{act.job_type || 'default'}</span>
                  </td>
                  <td className="mono-cell" style={{ fontSize: '0.8125rem', color: act.delivery_mode === 'JETSTREAM' ? '#A78BFA' : 'var(--text-secondary)' }}>
                    {act.delivery_mode || '-'}
                  </td>
                  <td>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem', alignItems: 'flex-start' }}>
                      <span className={`badge ${getBadgeClass(act.event)}`}>
                        {act.event}
                      </span>
                      <span className={`event-category-tag ${isBusinessActivity(act) ? 'business' : 'lifecycle'}`}>
                        {isBusinessActivity(act) ? 'BUSINESS' : 'LIFECYCLE'}
                      </span>
                    </div>
                  </td>
                  <td className="mono-cell" style={{ color: 'var(--text-muted)' }}>
                    {act.sequence ? `#${act.sequence}` : '-'}
                  </td>
                  <td className="mono-cell" style={{ color: 'var(--accent-cyan)' }}>
                    {act.subject}
                  </td>
                  <td className="mono-cell" style={{ color: 'var(--text-secondary)' }}>
                    {act.worker || '-'}
                  </td>
                  <td className="mono-cell" style={{ textAlign: 'center', color: act.delivery_count > 1 ? 'var(--status-warning)' : 'var(--text-muted)' }}>
                    {act.delivery_count}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};
