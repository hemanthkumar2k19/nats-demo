import React, { useState } from 'react';
import { Job, scheduleJob, ScheduleJobRequest } from '../api/demoApi';

interface DelayedRetryPanelProps {
  onSubmitJob: (job: Job) => Promise<void>;
  isSubmitting: boolean;
  onShowInfo?: (key: string) => void;
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
}

export const DelayedRetryPanel: React.FC<DelayedRetryPanelProps> = ({
  onSubmitJob,
  isSubmitting,
  onShowInfo,
  onAlert,
}) => {
  // Demo 1: NAK with Delay state
  const [nakJobId, setNakJobId] = useState<string>(() => `nak-job-${Math.floor(100 + Math.random() * 900)}`);
  const [nakDelaySec, setNakDelaySec] = useState<number>(5);
  const [isSubmittingNak, setIsSubmittingNak] = useState<boolean>(false);
  const [nakStatus, setNakStatus] = useState<string | null>(null);

  // Demo 2: AckWait state
  const [ackWaitJobId, setAckWaitJobId] = useState<string>(() => `ackwait-job-${Math.floor(100 + Math.random() * 900)}`);
  const [isSubmittingAckWait, setIsSubmittingAckWait] = useState<boolean>(false);
  const [ackWaitStatus, setAckWaitStatus] = useState<string | null>(null);

  // Demo 3: Scheduled Delivery state
  const [schedJobId, setSchedJobId] = useState<string>(() => `sched-job-${Math.floor(100 + Math.random() * 900)}`);
  const [schedDelaySec, setSchedDelaySec] = useState<number>(5);
  const [isScheduling, setIsScheduling] = useState<boolean>(false);
  const [schedStatus, setSchedStatus] = useState<string | null>(null);

  // Handle Demo 1: NAK with Delay
  const handleTriggerNak = async () => {
    if (!nakJobId.trim()) return;
    setIsSubmittingNak(true);
    setNakStatus(null);

    const job: Job = {
      job_id: nakJobId.trim(),
      type: 'image-processing',
      payload: {
        file: 'image-nak-test.jpg',
        simulate_failure: true,
        simulate_failure_count: 1,
        nak_delay_seconds: nakDelaySec,
      },
      delivery_mode: 'JETSTREAM',
    };

    try {
      await onSubmitJob(job);
      const msg = `Job ${nakJobId.trim()} submitted with ${nakDelaySec}s NAK delay. Worker will NAK on attempt #1, waiting ${nakDelaySec}s before redelivery.`;
      setNakStatus(msg);
      if (onAlert) onAlert('success', msg);
      setNakJobId(`nak-job-${Math.floor(100 + Math.random() * 900)}`);
    } catch (err: any) {
      const errMsg = err.message || 'Failed to submit NAK demo job';
      setNakStatus(`Error: ${errMsg}`);
      if (onAlert) onAlert('error', errMsg);
    } finally {
      setIsSubmittingNak(false);
    }
  };

  // Handle Demo 2: AckWait Timeout
  const handleTriggerAckWait = async () => {
    if (!ackWaitJobId.trim()) return;
    setIsSubmittingAckWait(true);
    setAckWaitStatus(null);

    const job: Job = {
      job_id: ackWaitJobId.trim(),
      type: 'data-sync',
      payload: {
        file: 'ackwait-test.dat',
        simulate_no_ack: true,
      },
      delivery_mode: 'JETSTREAM',
    };

    try {
      await onSubmitJob(job);
      const msg = `Job ${ackWaitJobId.trim()} submitted. Worker will receive and intentionally skip ACK. JetStream AckWait (5s) will trigger redelivery to another worker.`;
      setAckWaitStatus(msg);
      if (onAlert) onAlert('success', msg);
      setAckWaitJobId(`ackwait-job-${Math.floor(100 + Math.random() * 900)}`);
    } catch (err: any) {
      const errMsg = err.message || 'Failed to submit AckWait demo job';
      setAckWaitStatus(`Error: ${errMsg}`);
      if (onAlert) onAlert('error', errMsg);
    } finally {
      setIsSubmittingAckWait(false);
    }
  };

  // Handle Demo 3: Scheduled / Delayed Delivery
  const handleTriggerSchedule = async () => {
    if (!schedJobId.trim()) return;
    setIsScheduling(true);
    setSchedStatus(null);

    const req: ScheduleJobRequest = {
      job_id: schedJobId.trim(),
      type: 'email-alert',
      payload: {
        recipient: 'user@example.com',
        scheduled_delay: schedDelaySec,
      },
      delivery_mode: 'JETSTREAM',
      deliver_after_seconds: schedDelaySec,
    };

    try {
      const resp = await scheduleJob(req);
      const msg = `Job ${resp.job_id} scheduled! Application timer will wait ${resp.delay_seconds}s before publishing to NATS at ${new Date(resp.scheduled_for).toLocaleTimeString()}.`;
      setSchedStatus(msg);
      if (onAlert) onAlert('success', msg);
      setSchedJobId(`sched-job-${Math.floor(100 + Math.random() * 900)}`);
    } catch (err: any) {
      const errMsg = err.message || 'Failed to schedule job';
      setSchedStatus(`Error: ${errMsg}`);
      if (onAlert) onAlert('error', errMsg);
    } finally {
      setIsScheduling(false);
    }
  };

  return (
    <div className="panel">
      {/* Header */}
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Delayed & Retry Delivery Lab
          </h2>
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('delayed-retry-delivery')}
              title="Learn about Delayed and Retry Delivery in NATS"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      <div style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginBottom: '1.25rem' }}>
        Compare explicit retry delays (<code>msg.NakWithDelay</code>), missing ACK timeouts (<code>AckWait</code>), and application-level scheduled releases.
      </div>

      {/* 3 Interactive Scenario Cards Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '1rem' }}>
        
        {/* Card 1: NAK with Delay */}
        <div style={{
          background: 'rgba(255, 255, 255, 0.02)',
          border: '1px solid var(--border-color)',
          borderRadius: '8px',
          padding: '1rem',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
        }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--accent-cyan)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                Scenario 1: Explicit Retry
              </span>
              <span className="badge badge-info" style={{ fontSize: '0.6875rem' }}>JetStream</span>
            </div>
            <h3 style={{ fontSize: '0.9375rem', fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 0.5rem 0' }}>
              NAK with Delay
            </h3>
            <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', margin: '0 0 0.875rem 0', lineHeight: 1.4 }}>
              Worker handles transient failure and explicitly requests a redelivery backoff window via <code>msg.NakWithDelay(5s)</code>.
            </p>

            <div style={{ background: 'rgba(0,0,0,0.2)', padding: '0.625rem', borderRadius: '6px', marginBottom: '0.875rem', fontSize: '0.6875rem', color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
              <div>t=0s: Delivered -&gt; Worker NAKs</div>
              <div style={{ color: 'var(--accent-amber)' }}>t=0s-5s: Quiet backoff window</div>
              <div style={{ color: 'var(--accent-green)' }}>t=5s: Redelivered &amp; Completed</div>
            </div>

            <div className="form-group" style={{ marginBottom: '0.75rem' }}>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>Job ID</label>
              <input
                type="text"
                className="form-input"
                style={{ fontSize: '0.8125rem' }}
                value={nakJobId}
                onChange={(e) => setNakJobId(e.target.value)}
                placeholder="nak-job-101"
              />
            </div>

            <div className="form-group" style={{ marginBottom: '0.875rem' }}>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>Retry Delay (Seconds)</label>
              <select
                className="form-select"
                style={{ fontSize: '0.8125rem' }}
                value={nakDelaySec}
                onChange={(e) => setNakDelaySec(Number(e.target.value))}
              >
                <option value={3}>3 Seconds</option>
                <option value={5}>5 Seconds (Recommended)</option>
                <option value={10}>10 Seconds</option>
              </select>
            </div>
          </div>

          <div>
            <button
              type="button"
              className="btn btn-primary"
              style={{ width: '100%', fontSize: '0.8125rem' }}
              onClick={handleTriggerNak}
              disabled={isSubmittingNak || isSubmitting}
            >
              {isSubmittingNak ? 'Submitting...' : `Trigger NAK with ${nakDelaySec}s Delay`}
            </button>
            {nakStatus && (
              <div style={{ marginTop: '0.5rem', fontSize: '0.6875rem', color: 'var(--accent-cyan)' }}>
                {nakStatus}
              </div>
            )}
          </div>
        </div>

        {/* Card 2: AckWait Timeout */}
        <div style={{
          background: 'rgba(255, 255, 255, 0.02)',
          border: '1px solid var(--border-color)',
          borderRadius: '8px',
          padding: '1rem',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
        }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--accent-amber)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                Scenario 2: Missing ACK
              </span>
              <span className="badge badge-warning" style={{ fontSize: '0.6875rem' }}>Consumer</span>
            </div>
            <h3 style={{ fontSize: '0.9375rem', fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 0.5rem 0' }}>
              AckWait Timeout
            </h3>
            <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', margin: '0 0 0.875rem 0', lineHeight: 1.4 }}>
              Worker receives the message but crashes/hangs without calling ACK. JetStream <code>AckWait (5s)</code> timer fires and redelivers.
            </p>

            <div style={{ background: 'rgba(0,0,0,0.2)', padding: '0.625rem', borderRadius: '6px', marginBottom: '0.875rem', fontSize: '0.6875rem', color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
              <div>t=0s: Delivered to Worker-1</div>
              <div style={{ color: 'var(--accent-amber)' }}>t=0s-5s: Worker hangs (No ACK)</div>
              <div style={{ color: 'var(--accent-green)' }}>t=5s: Redelivered to Worker-2</div>
            </div>

            <div className="form-group" style={{ marginBottom: '0.75rem' }}>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>Job ID</label>
              <input
                type="text"
                className="form-input"
                style={{ fontSize: '0.8125rem' }}
                value={ackWaitJobId}
                onChange={(e) => setAckWaitJobId(e.target.value)}
                placeholder="ackwait-job-101"
              />
            </div>

            <div className="form-group" style={{ marginBottom: '0.875rem' }}>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>Consumer AckWait Timer</label>
              <div className="mono-cell" style={{ fontSize: '0.8125rem', padding: '0.375rem 0.5rem', background: 'rgba(0,0,0,0.2)' }}>
                AckWait = 5s (Fixed Consumer Setting)
              </div>
            </div>
          </div>

          <div>
            <button
              type="button"
              className="btn btn-primary"
              style={{ width: '100%', fontSize: '0.8125rem' }}
              onClick={handleTriggerAckWait}
              disabled={isSubmittingAckWait || isSubmitting}
            >
              {isSubmittingAckWait ? 'Simulating...' : 'Trigger Missing ACK (5s AckWait)'}
            </button>
            {ackWaitStatus && (
              <div style={{ marginTop: '0.5rem', fontSize: '0.6875rem', color: 'var(--accent-amber)' }}>
                {ackWaitStatus}
              </div>
            )}
          </div>
        </div>

        {/* Card 3: Scheduled / Delayed Delivery */}
        <div style={{
          background: 'rgba(255, 255, 255, 0.02)',
          border: '1px solid var(--border-color)',
          borderRadius: '8px',
          padding: '1rem',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
        }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <span style={{ fontSize: '0.75rem', fontWeight: 700, color: 'var(--accent-purple)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                Scenario 3: Deferred Publish
              </span>
              <span className="badge badge-purple" style={{ fontSize: '0.6875rem' }}>App Scheduler</span>
            </div>
            <h3 style={{ fontSize: '0.9375rem', fontWeight: 600, color: 'var(--text-primary)', margin: '0 0 0.5rem 0' }}>
              Scheduled Delivery
            </h3>
            <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', margin: '0 0 0.875rem 0', lineHeight: 1.4 }}>
              Demonstrates application-level scheduled delivery. Message is held by an application timer and published only when due.
            </p>

            <div style={{ background: 'rgba(0,0,0,0.2)', padding: '0.625rem', borderRadius: '6px', marginBottom: '0.875rem', fontSize: '0.6875rem', color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
              <div style={{ color: 'var(--accent-purple)' }}>t=0s: Registered as SCHEDULED</div>
              <div>t=0s-5s: App Timer waits</div>
              <div style={{ color: 'var(--accent-green)' }}>t=5s: Published &amp; Delivered</div>
            </div>

            <div className="form-group" style={{ marginBottom: '0.75rem' }}>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>Job ID</label>
              <input
                type="text"
                className="form-input"
                style={{ fontSize: '0.8125rem' }}
                value={schedJobId}
                onChange={(e) => setSchedJobId(e.target.value)}
                placeholder="sched-job-101"
              />
            </div>

            <div className="form-group" style={{ marginBottom: '0.875rem' }}>
              <label className="form-label" style={{ fontSize: '0.75rem' }}>Schedule Delay (Seconds)</label>
              <select
                className="form-select"
                style={{ fontSize: '0.8125rem' }}
                value={schedDelaySec}
                onChange={(e) => setSchedDelaySec(Number(e.target.value))}
              >
                <option value={3}>3 Seconds</option>
                <option value={5}>5 Seconds (Recommended)</option>
                <option value={10}>10 Seconds</option>
              </select>
            </div>
          </div>

          <div>
            <button
              type="button"
              className="btn btn-primary"
              style={{ width: '100%', fontSize: '0.8125rem' }}
              onClick={handleTriggerSchedule}
              disabled={isScheduling || isSubmitting}
            >
              {isScheduling ? 'Scheduling...' : `Schedule Job (${schedDelaySec}s Delay)`}
            </button>
            {schedStatus && (
              <div style={{ marginTop: '0.5rem', fontSize: '0.6875rem', color: 'var(--accent-purple)' }}>
                {schedStatus}
              </div>
            )}
          </div>
        </div>

      </div>

      {/* Educational Footer Callout */}
      <div style={{
        marginTop: '1.25rem',
        padding: '0.75rem 1rem',
        background: 'rgba(0, 0, 0, 0.25)',
        border: '1px solid var(--border-color)',
        borderRadius: '6px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        fontSize: '0.75rem',
      }}>
        <div style={{ display: 'flex', gap: '1.5rem', flexWrap: 'wrap' }}>
          <div>
            <strong style={{ color: 'var(--accent-cyan)' }}>NAK + Delay:</strong> "Worker says: retry me after X seconds."
          </div>
          <div>
            <strong style={{ color: 'var(--accent-amber)' }}>AckWait:</strong> "Broker says: no ACK received in 5s; redelivering."
          </div>
          <div>
            <strong style={{ color: 'var(--accent-purple)' }}>Scheduler:</strong> "App says: do not publish to NATS until scheduled time."
          </div>
        </div>
      </div>
    </div>
  );
};
