import React, { useState, useEffect, useRef } from 'react';
import {
  SagaInstance,
  StartSagaRequest,
  startSaga,
  advanceSagaStep,
  getSagaStatus,
  listSagas,
} from '../api/demoApi';

interface SagaPanelProps {
  onShowInfo: (key: string) => void;
  onAlert?: (type: 'success' | 'error' | 'warning', message: string) => void;
}

export const SagaPanel: React.FC<SagaPanelProps> = ({ onShowInfo, onAlert }) => {
  const [jobId, setJobId] = useState<string>('');
  const [item, setItem] = useState<string>('MacBook Pro M3');
  const [amount, setAmount] = useState<number>(1999);
  const [mode, setMode] = useState<'interactive' | 'auto'>('interactive');
  const [activeSaga, setActiveSaga] = useState<SagaInstance | null>(null);
  const [recentSagas, setRecentSagas] = useState<SagaInstance[]>([]);
  const [isTriggering, setIsTriggering] = useState<boolean>(false);
  const [isStepping, setIsStepping] = useState<boolean>(false);

  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Load recent sagas on mount
  useEffect(() => {
    loadRecentSagas();
    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
      }
    };
  }, []);

  // Poll status while active saga is running (for auto mode or external updates)
  useEffect(() => {
    if (!activeSaga) return;

    const isRunning =
      activeSaga.state === 'STARTED' ||
      activeSaga.state === 'OP1_PENDING' ||
      activeSaga.state === 'OP2_PENDING' ||
      activeSaga.state === 'COMPENSATING';

    if (isRunning && mode === 'auto') {
      if (!pollIntervalRef.current) {
        pollIntervalRef.current = setInterval(async () => {
          try {
            const updated = await getSagaStatus(activeSaga.job_id);
            setActiveSaga(updated);
            if (
              updated.state === 'COMPLETED' ||
              updated.state === 'FAILED' ||
              updated.state === 'COMPENSATION_FAILED'
            ) {
              if (pollIntervalRef.current) {
                clearInterval(pollIntervalRef.current);
                pollIntervalRef.current = null;
              }
              loadRecentSagas();
            }
          } catch (e) {
            // Ignore temporary polling errors
          }
        }, 400);
      }
    } else {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
        pollIntervalRef.current = null;
      }
    }
  }, [activeSaga?.state, activeSaga?.job_id, mode]);

  const loadRecentSagas = async () => {
    try {
      const list = await listSagas();
      setRecentSagas(list);
    } catch (e) {
      // Ignore
    }
  };

  const handleTriggerSaga = async () => {
    setIsTriggering(true);
    const targetJobId = jobId.trim() || `order-${Date.now().toString().slice(-4)}`;

    const req: StartSagaRequest = {
      job_id: targetJobId,
      type: 'order-fulfillment',
      interactive: mode === 'interactive',
      step_delay_ms: 750,
      payload: {
        item: item,
        amount: amount,
      },
    };

    try {
      const inst = await startSaga(req);
      setActiveSaga(inst);
      setJobId('');
      if (onAlert) {
        onAlert('success', `Triggered 2-Op Saga [${inst.saga_id}] over NATS (Subject: saga.start)`);
      }
      loadRecentSagas();
    } catch (err: any) {
      if (onAlert) {
        onAlert('error', `Failed to trigger Saga: ${err.message}`);
      }
    } finally {
      setIsTriggering(false);
    }
  };

  const handleAdvanceStep = async (step: 'op1' | 'op2', action: 'SUCCESS' | 'FAIL', errorMsg?: string) => {
    if (!activeSaga) return;
    setIsStepping(true);

    try {
      const updated = await advanceSagaStep(activeSaga.job_id, {
        step: step,
        action: action,
        error: errorMsg,
      });
      setActiveSaga(updated);
      if (onAlert) {
        if (action === 'SUCCESS') {
          onAlert('success', `${step.toUpperCase()} completed successfully via NATS`);
        } else {
          onAlert('warning', `${step.toUpperCase()} failed: ${errorMsg || 'Simulated failure'}`);
        }
      }
      loadRecentSagas();
    } catch (err: any) {
      if (onAlert) {
        onAlert('error', `Step advance error: ${err.message}`);
      }
    } finally {
      setIsStepping(false);
    }
  };

  const handleReset = () => {
    setActiveSaga(null);
  };

  const findStep = (name: string, type: 'FORWARD' | 'COMPENSATION') => {
    if (!activeSaga) return undefined;
    return activeSaga.steps.find((s) => s.name === name && s.type === type);
  };

  const step1 = findStep('reserve', 'FORWARD');
  const step2 = findStep('payment', 'FORWARD');
  const stepComp = findStep('release', 'COMPENSATION');

  const isOp1Active = Boolean(activeSaga && (activeSaga.state === 'OP1_PENDING' || activeSaga.state === 'STARTED'));
  const isOp2Active = Boolean(activeSaga && activeSaga.state === 'OP2_PENDING');
  const isTerminal = Boolean(
    activeSaga && (
      activeSaga.state === 'COMPLETED' ||
      activeSaga.state === 'FAILED' ||
      activeSaga.state === 'COMPENSATION_FAILED'
    )
  );

  return (
    <div className="saga-panel">
      {/* Panel Header */}
      <div className="panel-header" style={{ marginBottom: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <h2 className="panel-title">
            <svg style={{ width: '14px', height: '14px' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            EVENT-DRIVEN SAGA LAB (2-OP PATTERN)
          </h2>
          <button
            type="button"
            className="node-info-btn"
            onClick={() => onShowInfo('saga-orchestration')}
            title="Learn about the Event-Driven Saga Pattern"
          >
            (i)
          </button>
        </div>
        <span className="badge badge-published">EVENT-DRIVEN ORCHESTRATION</span>
      </div>

      <p className="panel-subtitle">
        Coordinates multi-service distributed transactions over NATS with forward step execution and automatic compensating rollback on failure.
      </p>

      {/* Meta Chips: Pattern, Forward Path, Compensation, Transport */}
      <div className="queue-meta-row" style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Pattern</span>
          <span className="queue-meta-value mono">2-Op Event-Driven Saga</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Forward Path</span>
          <span className="queue-meta-value mono">saga.op1.reserve -&gt; saga.op2.payment</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Rollback Compensation</span>
          <span className="queue-meta-value mono">saga.op1.compensate (release)</span>
        </div>
        <div className="queue-meta-chip">
          <span className="queue-meta-label">Transport</span>
          <span className="queue-meta-value mono">Core NATS Events</span>
        </div>
      </div>

      {/* Trigger & Controls Card */}
      <div className="saga-launcher-card">
        <div className="saga-form-row">
          <div className="form-group" style={{ flex: 2, minWidth: '180px' }}>
            <label className="form-label">Order Item</label>
            <input
              type="text"
              value={item}
              onChange={(e) => setItem(e.target.value)}
              placeholder="e.g. MacBook Pro M3"
              className="form-input"
              disabled={isOp1Active || isOp2Active}
            />
          </div>

          <div className="form-group" style={{ width: '120px' }}>
            <label className="form-label">Amount ($)</label>
            <input
              type="number"
              value={amount}
              onChange={(e) => setAmount(Number(e.target.value))}
              className="form-input"
              disabled={isOp1Active || isOp2Active}
            />
          </div>

          <div className="form-group" style={{ width: '150px' }}>
            <label className="form-label">Order ID (Optional)</label>
            <input
              type="text"
              value={jobId}
              onChange={(e) => setJobId(e.target.value)}
              placeholder="e.g. order-881"
              className="form-input code"
              disabled={isOp1Active || isOp2Active}
            />
          </div>

          <div className="form-group" style={{ minWidth: '200px' }}>
            <label className="form-label">Execution Mode</label>
            <div style={{ display: 'flex', gap: '0.25rem' }}>
              <button
                type="button"
                className={`btn-worker-toggle ${mode === 'interactive' ? 'active' : ''}`}
                style={{ flex: 1, padding: '0.35rem 0.5rem', fontSize: '0.75rem', justifyContent: 'center' }}
                onClick={() => setMode('interactive')}
                disabled={isOp1Active || isOp2Active}
              >
                <span>Interactive</span>
              </button>
              <button
                type="button"
                className={`btn-worker-toggle ${mode === 'auto' ? 'active' : ''}`}
                style={{ flex: 1, padding: '0.35rem 0.5rem', fontSize: '0.75rem', justifyContent: 'center' }}
                onClick={() => setMode('auto')}
                disabled={isOp1Active || isOp2Active}
              >
                <span>Auto Run</span>
              </button>
            </div>
          </div>

          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'flex-end', marginLeft: 'auto' }}>
            <button
              type="button"
              className="btn btn-primary"
              onClick={handleTriggerSaga}
              disabled={isTriggering || isOp1Active || isOp2Active}
              style={{ minWidth: '150px' }}
            >
              {isTriggering ? 'Triggering...' : 'Trigger via NATS'}
            </button>
            {activeSaga && (
              <button
                type="button"
                className="btn btn-secondary"
                onClick={handleReset}
                title="Reset active workbench for next test"
              >
                Reset Lab
              </button>
            )}
          </div>
        </div>

        {/* Quick Demo Presets */}
        <div className="saga-preset-bar">
          <span className="saga-preset-label">Quick Scenarios:</span>
          <button
            type="button"
            className="btn-saga-preset"
            onClick={() => {
              setItem('MacBook Pro M3');
              setAmount(1999);
              setMode('interactive');
            }}
            disabled={isOp1Active || isOp2Active}
          >
            Normal Success Flow
          </button>
          <button
            type="button"
            className="btn-saga-preset"
            onClick={() => {
              setItem('Sony A7 IV Camera');
              setAmount(2498);
              setMode('interactive');
            }}
            disabled={isOp1Active || isOp2Active}
          >
            Payment Declined (Triggers Rollback)
          </button>
          <button
            type="button"
            className="btn-saga-preset"
            onClick={() => {
              setItem('Vintage Rolex Watch');
              setAmount(8500);
              setMode('interactive');
            }}
            disabled={isOp1Active || isOp2Active}
          >
            Inventory Out of Stock (Op 1 Fail)
          </button>
        </div>

        <div className="saga-scenario-hint">
          <span>
            <strong>Interactive Mode:</strong> Click <strong>Trigger via NATS</strong>, then click <strong>Complete</strong> or <strong>Fail</strong> on the active step card to observe forward progression or compensating rollback across NATS subjects.
          </span>
        </div>
      </div>

      {/* 2-Operation Balanced Visual Pipeline */}
      {activeSaga ? (
        <div className="saga-visual-container">
          <div className="saga-instance-header">
            <div className="saga-meta-left">
              <span className="saga-title-label">ACTIVE SAGA</span>
              <span className="saga-mono-id">{activeSaga.saga_id}</span>
              <span className="saga-job-tag">Order: {activeSaga.job_id}</span>
              <span className="saga-job-tag">Item: {activeSaga.payload?.item || 'Item'} (${activeSaga.payload?.amount || 0})</span>
            </div>
            <div className="saga-meta-right">
              <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>STATE:</span>
              <span className={`badge ${activeSaga.state === 'COMPLETED' ? 'badge-published' : activeSaga.state === 'FAILED' ? 'badge-received' : 'badge-processing'}`}>
                {activeSaga.state}
              </span>
            </div>
          </div>

          {/* 3-Card Balanced Pipeline Grid */}
          <div className="saga-2op-pipeline">
            {/* Op 1 Card: Reserve Inventory */}
            <div className={`op-card ${isOp1Active ? 'active-step' : ''} ${step1?.status ? step1.status.toLowerCase() : 'pending'}`}>
              <div className="op-card-header">
                <div className="op-card-header-top">
                  <span className="op-card-num">OP 1</span>
                  <span className={`status-indicator-tag ${step1?.status ? step1.status.toLowerCase() : isOp1Active ? 'running' : 'pending'}`}>
                    {step1?.status || (isOp1Active ? 'AWAITING ACTION' : 'PENDING')}
                  </span>
                </div>
                <h3 className="op-card-title">Reserve Inventory</h3>
                <span className="op-card-subject">saga.op1.reserve</span>
              </div>

              <div className="op-card-body">
                <div className="op-card-meta">
                  <span>Target Item:</span>
                  <strong style={{ color: 'var(--text-primary)' }}>{activeSaga.payload?.item || 'Stock Item'}</strong>
                </div>
                {step1?.duration_ms !== undefined && (
                  <div className="op-card-meta">
                    <span>Duration:</span>
                    <span style={{ fontFamily: 'var(--font-mono)' }}>{step1.duration_ms}ms</span>
                  </div>
                )}
                {step1?.details && (
                  <div style={{ color: 'var(--text-muted)', fontSize: '0.6875rem' }}>
                    {step1.details}
                  </div>
                )}
                {step1?.error && <div className="step-error-box">{step1.error}</div>}
              </div>

              {/* Interactive buttons for Op 1 */}
              {isOp1Active && mode === 'interactive' && (
                <div className="op-interactive-buttons">
                  <button
                    type="button"
                    className="btn-step-action success"
                    onClick={() => handleAdvanceStep('op1', 'SUCCESS')}
                    disabled={isStepping}
                  >
                    <svg style={{ width: '12px', height: '12px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Complete Op 1 (Stock Reserved)</span>
                  </button>
                  <button
                    type="button"
                    className="btn-step-action fail"
                    onClick={() => handleAdvanceStep('op1', 'FAIL', 'Out of stock: inventory allocation rejected')}
                    disabled={isStepping}
                  >
                    <svg style={{ width: '12px', height: '12px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                    <span>Fail Op 1 (Out of Stock)</span>
                  </button>
                </div>
              )}
            </div>

            {/* Directional Connector */}
            <div className="pipeline-bridge">
              <div className="bridge-arrow">-&gt;</div>
              <div className="bridge-label">saga.op1.completed</div>
            </div>

            {/* Op 2 Card: Process Payment */}
            <div className={`op-card ${isOp2Active ? 'active-step' : ''} ${step2?.status ? step2.status.toLowerCase() : 'pending'}`}>
              <div className="op-card-header">
                <div className="op-card-header-top">
                  <span className="op-card-num">OP 2</span>
                  <span className={`status-indicator-tag ${step2?.status ? step2.status.toLowerCase() : isOp2Active ? 'running' : 'pending'}`}>
                    {step2?.status || (isOp2Active ? 'AWAITING ACTION' : 'PENDING')}
                  </span>
                </div>
                <h3 className="op-card-title">Process Payment</h3>
                <span className="op-card-subject">saga.op2.payment</span>
              </div>

              <div className="op-card-body">
                <div className="op-card-meta">
                  <span>Charge Amount:</span>
                  <strong style={{ color: 'var(--text-primary)' }}>${activeSaga.payload?.amount || 0}</strong>
                </div>
                {step2?.duration_ms !== undefined && (
                  <div className="op-card-meta">
                    <span>Duration:</span>
                    <span style={{ fontFamily: 'var(--font-mono)' }}>{step2.duration_ms}ms</span>
                  </div>
                )}
                {step2?.details && (
                  <div style={{ color: 'var(--text-muted)', fontSize: '0.6875rem' }}>
                    {step2.details}
                  </div>
                )}
                {step2?.error && <div className="step-error-box">{step2.error}</div>}
              </div>

              {/* Interactive buttons for Op 2 */}
              {isOp2Active && mode === 'interactive' && (
                <div className="op-interactive-buttons">
                  <button
                    type="button"
                    className="btn-step-action success"
                    onClick={() => handleAdvanceStep('op2', 'SUCCESS')}
                    disabled={isStepping}
                  >
                    <svg style={{ width: '12px', height: '12px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Complete Op 2 (Payment Charged)</span>
                  </button>
                  <button
                    type="button"
                    className="btn-step-action fail"
                    onClick={() => handleAdvanceStep('op2', 'FAIL', 'Card declined: Insufficient funds')}
                    disabled={isStepping}
                  >
                    <svg style={{ width: '12px', height: '12px', flexShrink: 0 }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M3 10h10a5 5 0 015 5v2m-15-7l4-4m-4 4l4 4" />
                    </svg>
                    <span>Fail Op 2 (Triggers Rollback!)</span>
                  </button>
                </div>
              )}
            </div>

            {/* Directional Connector */}
            <div className="pipeline-bridge">
              <div className="bridge-arrow">-&gt;</div>
              <div className="bridge-label">saga.completed / failed</div>
            </div>

            {/* Outcome Card */}
            <div className={`op-card outcome ${activeSaga.state === 'COMPLETED' ? 'success' : activeSaga.state === 'FAILED' ? 'failed' : 'pending'}`}>
              <div className="op-card-header">
                <div className="op-card-header-top">
                  <span className="op-card-num">RESULT</span>
                  <span className="status-indicator-tag pending">OUTCOME</span>
                </div>
                <h3 className="op-card-title">Saga Outcome</h3>
                <span className="op-card-subject">saga.completed / failed</span>
              </div>

              <div className="op-card-body">
                <div className="op-card-meta">
                  <span>Final State:</span>
                  <span className={`badge ${activeSaga.state === 'COMPLETED' ? 'badge-published' : activeSaga.state === 'FAILED' ? 'badge-received' : 'badge-processing'}`}>
                    {activeSaga.state}
                  </span>
                </div>
                {activeSaga.state === 'COMPLETED' && (
                  <div style={{ color: '#34D399', fontSize: '0.75rem', marginTop: '0.5rem', lineHeight: 1.4 }}>
                    Order fulfilled. Both Reserve Inventory and Process Payment completed successfully.
                  </div>
                )}
                {activeSaga.state === 'FAILED' && activeSaga.compensated_steps.length > 0 && (
                  <div style={{ color: '#FBBF24', fontSize: '0.75rem', marginTop: '0.5rem', lineHeight: 1.4 }}>
                    Rollback Demonstrated. Op 2 failed, triggering compensation: inventory reservation released.
                  </div>
                )}
                {activeSaga.state === 'FAILED' && activeSaga.compensated_steps.length === 0 && (
                  <div style={{ color: '#F87171', fontSize: '0.75rem', marginTop: '0.5rem', lineHeight: 1.4 }}>
                    Saga Aborted. Failed at Op 1 before forward dependencies were established.
                  </div>
                )}
              </div>

              {isTerminal && (
                <div style={{ marginTop: 'auto', paddingTop: '0.5rem' }}>
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={handleReset}
                    style={{ width: '100%', fontSize: '0.75rem', padding: '0.4rem' }}
                  >
                    Start Another Run
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Dedicated Compensating Rollback Track */}
          <div className={`saga-compensation-track ${stepComp?.status ? (stepComp.status === 'SUCCESS' ? 'success' : 'active') : ''}`}>
            <div className="comp-track-left">
              <span className="comp-track-badge">ROLLBACK ACTION</span>
              <div>
                <div className="comp-track-title">Compensating Action: Release Inventory</div>
                <div className="comp-track-sub">Subject: saga.op1.compensate -&gt; saga.op1.compensated</div>
              </div>
            </div>
            <div className="comp-track-right">
              <span className="comp-track-status">
                {stepComp?.status ? `STATUS: ${stepComp.status}` : 'STATUS: IDLE (ON STANDBY)'}
              </span>
              {stepComp?.details && (
                <span style={{ color: '#34D399', fontSize: '0.6875rem' }}>({stepComp.details})</span>
              )}
            </div>
          </div>

          {/* Event Stream Log */}
          {activeSaga.steps.length > 0 && (
            <div className="saga-timeline-card">
              <div className="saga-timeline-title">
                <span>NATS Event Transmission Ledger</span>
                <span style={{ fontSize: '0.625rem', fontFamily: 'var(--font-mono)' }}>{activeSaga.steps.length} transitions</span>
              </div>
              <div className="saga-table-wrap">
                <table className="saga-table">
                  <thead>
                    <tr>
                      <th>STEP</th>
                      <th>TYPE</th>
                      <th>STATUS</th>
                      <th>DETAILS</th>
                      <th>LATENCY</th>
                    </tr>
                  </thead>
                  <tbody>
                    {activeSaga.steps.map((st, idx) => (
                      <tr key={idx}>
                        <td><code>{st.name === 'reserve' ? 'OP 1 (Reserve)' : st.name === 'payment' ? 'OP 2 (Payment)' : 'Release (Compensate)'}</code></td>
                        <td>
                          <span className={`badge ${st.type === 'FORWARD' ? 'badge-primary' : 'badge-received'}`} style={{ fontSize: '0.625rem' }}>
                            {st.type}
                          </span>
                        </td>
                        <td>
                          <span className={`status-indicator-tag ${st.status.toLowerCase()}`}>
                            {st.status}
                          </span>
                        </td>
                        <td>
                          {st.details && <span>{st.details}</span>}
                          {st.error && <span style={{ color: '#F87171' }}>{st.error}</span>}
                        </td>
                        <td>
                          <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>
                            {st.duration_ms > 0 ? `${st.duration_ms}ms` : '-'}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      ) : (
        <div className="saga-empty-workbench">
          <div className="empty-title">Ready to Demonstrate 2-Op Event-Driven Saga</div>
          <p className="empty-desc">
            Click <strong>Trigger via NATS</strong> above or select a <strong>Quick Scenario</strong> preset to launch an order transaction. You can then use the step action buttons to observe forward completion or compensating rollback over NATS events.
          </p>
        </div>
      )}

      {/* Recent Sagas Table */}
      {recentSagas.length > 0 && (
        <div className="saga-recent-card">
          <div className="saga-timeline-title">
            <span>Past Saga Runs</span>
            <span style={{ fontSize: '0.625rem', fontFamily: 'var(--font-mono)' }}>{recentSagas.length} recorded</span>
          </div>
          <div className="saga-table-wrap">
            <table className="saga-table">
              <thead>
                <tr>
                  <th>SAGA ID</th>
                  <th>ORDER ID</th>
                  <th>STATE</th>
                  <th>COMPLETED</th>
                  <th>COMPENSATED</th>
                  <th>ACTION</th>
                </tr>
              </thead>
              <tbody>
                {recentSagas.slice(0, 5).map((s) => (
                  <tr key={s.saga_id}>
                    <td><code>{s.saga_id}</code></td>
                    <td><code>{s.job_id}</code></td>
                    <td>
                      <span className={`badge ${s.state === 'COMPLETED' ? 'badge-published' : s.state === 'FAILED' ? 'badge-received' : 'badge-processing'}`}>
                        {s.state}
                      </span>
                    </td>
                    <td>{s.completed_steps.join(', ') || 'none'}</td>
                    <td>{s.compensated_steps.join(', ') || 'none'}</td>
                    <td>
                      <button
                        type="button"
                        className="btn btn-secondary"
                        style={{ padding: '0.15rem 0.45rem', fontSize: '0.6875rem' }}
                        onClick={() => setActiveSaga(s)}
                      >
                        Inspect
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
