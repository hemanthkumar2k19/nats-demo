import React, { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { StatusPanel } from './components/StatusPanel';
import { JobPanel } from './components/JobPanel';
import { ActivityPanel } from './components/ActivityPanel';
import { ReplayPanel } from './components/ReplayPanel';
import { JobInspectorPanel } from './components/JobInspectorPanel';
import { 
  Job, 
  Activity, 
  ServiceStatus, 
  submitJob, 
  validateJob, 
  getServiceStatus, 
  getActivity,
  getJobDetail,
  replayJobs,
  JobDetailResponse,
  ReplayRequest
} from './api/demoApi';

const DEFAULT_SERVICES: ServiceStatus[] = [
  { name: 'NATS Server (4222)', status: 'unknown', details: 'Status API not implemented' },
  { name: 'Demo Service (8080)', status: 'unknown', details: 'Status API not implemented' },
  { name: 'Processor Service', status: 'unknown', details: 'Status API not implemented' },
];

export const App: React.FC = () => {
  const [services, setServices] = useState<ServiceStatus[]>(DEFAULT_SERVICES);
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

  // Load initial state and set up status polling
  useEffect(() => {
    refreshStatus(true);
    refreshActivity(true);

    const interval = setInterval(() => {
      refreshStatus(true);
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const refreshStatus = async (silent = false) => {
    if (!silent) setIsRefreshingStatus(true);
    if (!silent) setError(null);
    try {
      const res = await getServiceStatus();
      setServices(res);
    } catch (err: any) {
      if (!silent) {
        setError(err.message || 'Failed to refresh service status');
      }
      setServices([
        { name: 'NATS Server (4222)', status: 'disconnected', details: 'Demo backend is unreachable' },
        { name: 'demo-service (8080)', status: 'disconnected', details: 'Demo backend is unreachable' },
        { name: 'processor-service', status: 'disconnected', details: 'Demo backend is unreachable' },
      ]);
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
    } catch (err: any) {
      if (!silent) {
        setError(err.message || 'Failed to fetch activity logs');
      }
    } finally {
      if (!silent) setIsRefreshingActivity(false);
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
      };
      setActivities((prev) => [newActivity, ...prev]);
    } catch (err: any) {
      setError(err.message || 'Job submission failed');
      throw err;
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleJobValidate = async (job: Job) => {
    setIsValidating(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await validateJob(job);
      if (res.valid) {
        setSuccess(`Validation Success: ${res.message}`);
      } else {
        setError(`Validation Failed: ${res.message}`);
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

  return (
    <div className="app-container">
      <Header systemOk={!error && !services.some(s => s.status === 'disconnected')} />

      {/* Global Alerts Banner */}
      {error && (
        <div className="alert-banner">
          <svg style={{ width: '20px', height: '20px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div style={{ flex: 1 }}>{error}</div>
          <span className="alert-close" onClick={() => setError(null)}>✕</span>
        </div>
      )}

      {success && (
        <div className="alert-banner alert-banner-success">
          <svg style={{ width: '20px', height: '20px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div style={{ flex: 1 }}>{success}</div>
          <span className="alert-close" onClick={() => setSuccess(null)}>✕</span>
        </div>
      )}

      <main className="dashboard-grid">
        {/* Left Column: Forms & Services */}
        <div className="left-column">
          <JobPanel 
            onSubmitJob={handleJobSubmit} 
            onValidateJob={handleJobValidate} 
            isSubmitting={isSubmitting}
            isValidating={isValidating}
          />
          
          <ReplayPanel 
            onTriggerReplay={handleTriggerReplay}
          />

          <StatusPanel 
            services={services} 
            onRefresh={() => refreshStatus(false)} 
            isLoading={isRefreshingStatus}
          />
        </div>

        {/* Right Column: Activity Logs & Inspector */}
        <div className="right-column">
          <ActivityPanel 
            activities={activities} 
            onRefresh={() => refreshActivity(false)} 
            isLoading={isRefreshingActivity}
            onSelectJob={handleSelectJob}
          />

          {(selectedJobId || isLoadingInspector || inspectorError) && (
            <JobInspectorPanel 
              job={selectedJobDetail}
              isLoading={isLoadingInspector}
              error={inspectorError}
              onClose={() => setSelectedJobId(null)}
            />
          )}
        </div>
      </main>
    </div>
  );
};

