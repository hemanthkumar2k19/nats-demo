import React, { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { StatusPanel } from './components/StatusPanel';
import { JobPanel } from './components/JobPanel';
import { ActivityPanel } from './components/ActivityPanel';
import { ReplayPanel } from './components/ReplayPanel';
import { JobInspectorPanel } from './components/JobInspectorPanel';
import { AddressingPanel } from './components/AddressingPanel';
import { RequestReplyPanel } from './components/RequestReplyPanel';
import { DemoSetupPanel } from './components/DemoSetup/DemoSetupPanel';
import { ObservabilityPanel } from './components/ObservabilityPanel';
import { InfoPopover } from './components/DemoSetup/InfoPopover';
import { NATS_COMPONENTS_INFO } from './content/natsInfo';
import { 
  Job, 
  Activity, 
  ServiceStatus, 
  ValidationResult,
  submitJob, 
  validateJob, 
  getServiceStatus, 
  getActivity,
  getJobDetail,
  replayJobs,
  JobDetailResponse,
  ReplayRequest,
  AddressingSubscription,
  AddressingEvent,
  getAddressingSubscriptions,
  getAddressingActivity,
  JetStreamInfo,
  updateProcessorState,
  ConsumerStatus,
  getConsumerStatus
} from './api/demoApi';

const DEFAULT_SERVICES: ServiceStatus[] = [
  { name: 'NATS Server (4222)', status: 'unknown', details: 'Status API not implemented' },
  { name: 'Demo Service (8080)', status: 'unknown', details: 'Status API not implemented' },
  { name: 'Processor Service', status: 'unknown', details: 'Status API not implemented' },
];

export const App: React.FC = () => {
  const [services, setServices] = useState<ServiceStatus[]>(DEFAULT_SERVICES);
  const [jetstreamInfo, setJetstreamInfo] = useState<JetStreamInfo | null>(null);
  const [consumerStatus, setConsumerStatus] = useState<ConsumerStatus | null>(null);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [isValidating, setIsValidating] = useState<boolean>(false);
  const [isRefreshingStatus, setIsRefreshingStatus] = useState<boolean>(false);
  const [isRefreshingActivity, setIsRefreshingActivity] = useState<boolean>(false);

  // Inspector state
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [selectedJobDetail, setSelectedJobDetail] = useState<JobDetailResponse | null>(null);
  const [isLoadingInspector, setIsLoadingInspector] = useState<boolean>(false);
  const [inspectorError, setInspectorError] = useState<string | null>(null);

  // Addressing state
  const [subscriptions, setSubscriptions] = useState<AddressingSubscription[]>([]);
  const [addressingEvents, setAddressingEvents] = useState<AddressingEvent[]>([]);
  const [isRefreshingAddressing, setIsRefreshingAddressing] = useState<boolean>(false);

  // Global Contextual NATS Info modal state
  const [activeInfoKey, setActiveInfoKey] = useState<string | null>(null);

  // Load initial state and set up status polling
  useEffect(() => {
    refreshStatus(true);
    refreshActivity(true);
    loadSubscriptions();
    refreshAddressing(true);

    const interval = setInterval(() => {
      refreshStatus(true);
      refreshAddressing(true);
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const refreshStatus = async (silent = false) => {
    if (!silent) setIsRefreshingStatus(true);
    if (!silent) setError(null);
    try {
      const res = await getServiceStatus();
      setServices(res.services);
      setJetstreamInfo(res.jetstream || null);

      try {
        const cStatus = await getConsumerStatus();
        setConsumerStatus(cStatus);
      } catch {
        // Consumer API may be quiet if processor is offline
      }
    } catch (err: any) {
      if (!silent) {
        setError(err.message || 'Failed to refresh service status');
      }
      setServices([
        { name: 'NATS Server (4222)', status: 'disconnected', details: 'Demo backend is unreachable' },
        { name: 'demo-service (8080)', status: 'disconnected', details: 'Demo backend is unreachable' },
        { name: 'processor-service', status: 'disconnected', details: 'Demo backend is unreachable' },
      ]);
      setJetstreamInfo(null);
    } finally {
      if (!silent) {
        setIsRefreshingStatus(false);
      }
    }
  };

  const refreshActivity = async (silent = false) => {
    if (!silent) setIsRefreshingActivity(true);
    setError(null);
    try {
      const res = await getActivity();
      setActivities(res);
      await refreshAddressing(silent);
    } catch (err: any) {
      if (!silent) {
        setError(err.message || 'Failed to fetch activity logs');
      }
    } finally {
      if (!silent) setIsRefreshingActivity(false);
    }
  };

  const loadSubscriptions = async () => {
    try {
      const res = await getAddressingSubscriptions();
      setSubscriptions(res);
    } catch (err: any) {
      console.error('Failed to load subscriptions:', err);
    }
  };

  const refreshAddressing = async (silent = false) => {
    if (!silent) setIsRefreshingAddressing(true);
    try {
      const res = await getAddressingActivity();
      setAddressingEvents(res);
    } catch (err: any) {
      if (!silent) {
        setError(err.message || 'Failed to refresh addressing activity');
      }
    } finally {
      if (!silent) {
        setIsRefreshingAddressing(false);
      }
    }
  };

  const handleJobSubmit = async (job: Job) => {
    setIsSubmitting(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await submitJob(job);
      setSuccess(`Job "${job.job_id}" submitted successfully.`);
      
      const localTime = new Date().toLocaleTimeString();
      const newActivity: Activity = {
        timestamp: localTime,
        job_id: res.job_id,
        event: res.status || 'SUBMITTED',
        subject: 'jobs.submitted',
        worker: '',
        delivery_count: 1,
        delivery_mode: job.delivery_mode,
      };
      setActivities((prev) => [newActivity, ...prev]);

      // Refresh addressing and status activity after short delay to let events propagate
      setTimeout(() => {
        refreshActivity(true);
        refreshStatus(true);
      }, 500);
    } catch (err: any) {
      setError(err.message || 'Job submission failed');
      throw err;
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleToggleProcessor = async (enabled: boolean) => {
    setError(null);
    setSuccess(null);
    try {
      const res = await updateProcessorState(enabled);
      setSuccess(`Processor state changed to ${res.status}`);
      await refreshStatus(true);
    } catch (err: any) {
      setError(err.message || 'Failed to toggle processor state');
    }
  };

  const handleJobValidate = async (job: Job) => {
    setIsValidating(true);
    setError(null);
    setSuccess(null);
    try {
      const res: ValidationResult = await validateJob(job);
      if (res.timedOut) {
        setError('Request timeout: No response received from processor service (is it ON?)');
      } else if (res.valid === false) {
        setError(`Validation Failed: ${res.message}`);
      } else {
        setSuccess(`Validation Success: ${res.message}`);
      }
    } catch (err: any) {
      setError(err.message || 'Validation failed');
      throw err;
    } finally {
      setIsValidating(false);
    }
  };

  const handleSelectJob = async (jobId: string) => {
    setSelectedJobId(jobId);
    setIsLoadingInspector(true);
    setInspectorError(null);
    setSelectedJobDetail(null);
    try {
      const detail = await getJobDetail(jobId);
      setSelectedJobDetail(detail);
    } catch (err: any) {
      setInspectorError(err.message || 'Failed to load job details');
    } finally {
      setIsLoadingInspector(false);
    }
  };

  const handleTriggerReplay = async (req: ReplayRequest) => {
    return replayJobs(req);
  };

  const natsConnected = !services.some(
    (s) => s.name.toLowerCase().includes('nats') && s.status === 'disconnected'
  );
  const systemOk = !error && !services.some((s) => s.status === 'disconnected');

  return (
    <div className="app-container">
      <Header natsConnected={natsConnected} systemOk={systemOk} />

      {/* Global Alerts Banner */}
      {error && (
        <div className="alert-banner">
          <svg style={{ width: '20px', height: '20px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div style={{ flex: 1 }}>{error}</div>
          <span className="alert-close" onClick={() => setError(null)}>x</span>
        </div>
      )}

      {success && (
        <div className="alert-banner alert-banner-success">
          <svg style={{ width: '20px', height: '20px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div style={{ flex: 1 }}>{success}</div>
          <span className="alert-close" onClick={() => setSuccess(null)}>x</span>
        </div>
      )}

      {/* Full-Width Platform Status Bar */}
      <StatusPanel 
        services={services} 
        jetstreamInfo={jetstreamInfo}
        onRefresh={() => refreshStatus(false)} 
        onToggleProcessor={handleToggleProcessor}
        isLoading={isRefreshingStatus}
        onShowInfo={setActiveInfoKey}
      />

      {/* Current Demo Setup & NATS Information Topology & Consumer Lab */}
      <DemoSetupPanel
        services={services}
        jetstreamInfo={jetstreamInfo}
        consumerStatus={consumerStatus}
        onShowInfo={setActiveInfoKey}
        onAlert={(type, msg) => {
          if (type === 'success') {
            setSuccess(msg);
            refreshStatus(true);
            refreshActivity(true);
          } else if (type === 'error') {
            setError(msg);
          }
        }}
        onConfigChanged={setConsumerStatus}
      />

      <main className="dashboard-grid">
        {/* Left Column: DEMO ACTIONS */}
        <div className="left-column">
          <div className="column-section-header">
            <span>DEMO ACTIONS</span>
          </div>

          <JobPanel 
            onSubmitJob={handleJobSubmit} 
            onValidateJob={handleJobValidate} 
            isSubmitting={isSubmitting}
            isValidating={isValidating}
            onShowInfo={setActiveInfoKey}
          />

          <RequestReplyPanel
            activities={activities}
            onValidated={() => refreshActivity(true)}
            onShowInfo={setActiveInfoKey}
          />
          
          <ReplayPanel 
            onTriggerReplay={handleTriggerReplay}
            onShowInfo={setActiveInfoKey}
          />
        </div>

        {/* Right Column: LIVE OBSERVABILITY */}
        <div className="right-column">
          <div className="column-section-header">
            <span>LIVE OBSERVABILITY</span>
          </div>

          <ActivityPanel 
            activities={activities} 
            onRefresh={() => refreshActivity(false)} 
            isLoading={isRefreshingActivity}
            onSelectJob={handleSelectJob}
            onShowInfo={setActiveInfoKey}
          />

          {(selectedJobId || isLoadingInspector || inspectorError) && (
            <JobInspectorPanel 
              job={selectedJobDetail}
              isLoading={isLoadingInspector}
              error={inspectorError}
              onClose={() => setSelectedJobId(null)}
              onShowInfo={setActiveInfoKey}
            />
          )}

          <AddressingPanel 
            subscriptions={subscriptions}
            events={addressingEvents}
            onRefresh={() => refreshAddressing(false)}
            isLoading={isRefreshingAddressing}
            onShowInfo={setActiveInfoKey}
          />
        </div>
      </main>

      {/* Dedicated Observability Setup Section */}
      <ObservabilityPanel onShowInfo={setActiveInfoKey} />

      {/* Global Contextual NATS Information Modal */}
      <InfoPopover
        info={activeInfoKey ? NATS_COMPONENTS_INFO[activeInfoKey] : null}
        onClose={() => setActiveInfoKey(null)}
      />
    </div>
  );
};

