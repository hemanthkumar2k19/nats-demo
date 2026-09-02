import React from 'react';
import { ServiceStatus, JetStreamInfo, ConsumerStatus, DLQStatus } from '../../api/demoApi';

interface DemoTopologyProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  dlqStatus?: DLQStatus | null;
  onSelectInfo: (componentId: string) => void;
}

export const DemoTopology: React.FC<DemoTopologyProps> = ({
  services,
  jetstreamInfo,
  consumerStatus,
  dlqStatus,
  onSelectInfo,
}) => {
  const natsService = services.find((s) => s.name.toLowerCase().includes('nats'));
  const jobService = services.find((s) => s.name.toLowerCase().includes('job'));
  const demoControlService = services.find((s) => s.name.toLowerCase().includes('control') || s.name.toLowerCase().includes('demo'));
  const processorService = services.find((s) => s.name.toLowerCase().includes('processor'));

  const isNatsConnected = natsService?.status === 'connected' || natsService?.status === 'active';
  const isJobActive = jobService?.status === 'active' || jobService?.status === 'connected';
  const isDemoControlActive = demoControlService?.status === 'active' || demoControlService?.status === 'connected';
  const isProcessorOnline = processorService?.status !== 'disconnected' && processorService?.status !== 'unknown';
  const isProcessing = processorService?.processing ?? false;

  const streamName = jetstreamInfo?.stream || 'JOBS';
  const pendingCount = consumerStatus?.pending ?? jetstreamInfo?.pending ?? 0;
  const ackPendingCount = consumerStatus?.ack_pending ?? 0;
  const redeliveredCount = consumerStatus?.redelivered ?? 0;

  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const consumerName = consumerStatus?.name || 'job-processor';
  const ordering = consumerStatus?.ordering ? consumerStatus.ordering.toUpperCase() : 'NORMAL';
  const workerCount = consumerStatus?.workers ?? processorService?.workers ?? (isProcessorOnline ? 1 : 0);

  return (
    <div className="demo-topology-container">
      <div className="topology-runtime-header">
        <span className="topology-legend-tag">DEPLOYED RUNTIME ARCHITECTURE</span>
        <span className="topology-legend-note">
          UI &amp; Demo Gateway Tier decoupled from Business Messaging Pipeline
        </span>
      </div>

      {/* ========================================================================= */}
      {/* TIER 1: CLIENT & DEMO CONTROL GATEWAY                                     */}
      {/* ========================================================================= */}
      <div className="topology-tier topology-tier-control">
        <div className="topology-tier-header">
          <span className="topology-tier-tag">TIER 1: CLIENT &amp; DEMO CONTROL GATEWAY</span>
        </div>
        <div className="topology-tier-row">
          {/* Component: React UI */}
          <div className="topology-col deployed-col">
            <div className="deployed-boundary-label">DEMO DASHBOARD</div>
            <div className="topology-node deployed-card node-active">
              <div className="node-header">
                <span className="node-title">React UI</span>
                <button
                  type="button"
                  className="node-info-btn"
                  onClick={() => onSelectInfo('react-ui')}
                  title="Learn about React UI"
                >
                  (i)
                </button>
              </div>
              <div className="node-body">
                <span className="node-badge badge-online">Active</span>
                <span className="node-detail">Dashboard (:5173)</span>
                <span className="node-subtle">Developer UI &amp; Controls</span>
              </div>
            </div>
          </div>

          {/* Connector: React UI -> Demo Control Service */}
          <div className="topology-connector horizontal connector-active">
            <div className="connector-flow-group">
              <span className="connector-flow-label">UI Gateway HTTP</span>
              <div className="connector-line-with-arrow">
                <div className="connector-line" />
                <span className="connector-arrow">-&gt;</span>
              </div>
            </div>
          </div>

          {/* Component: Demo Control Service */}
          <div className="topology-col deployed-col">
            <div className="deployed-boundary-label">CONTROL GATEWAY</div>
            <div className={`topology-node deployed-card ${isDemoControlActive ? 'node-active' : 'node-inactive'}`}>
              <div className="node-header">
                <span className="node-title">Demo Control Service</span>
                <button
                  type="button"
                  className="node-info-btn"
                  onClick={() => onSelectInfo('demo-control-service')}
                  title="Learn about Demo Control Service"
                >
                  (i)
                </button>
              </div>
              <div className="node-body">
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
                  <span className={`node-badge ${isDemoControlActive ? 'badge-online' : 'badge-offline'}`}>
                    {isDemoControlActive ? 'Active' : 'Offline'}
                  </span>
                  <span className="node-detail">HTTP API (:8080)</span>
                </div>
                <span className="node-subtle">Activity Taps &amp; Remote Control</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* INTER-TIER VERTICAL CONNECTORS                                            */}
      {/* ========================================================================= */}
      <div className="topology-vertical-bridge">
        <div className="v-bridge-col v-bridge-col-left">
          <div className="v-bridge-flow">
            <div className="v-bridge-line" />
            <span className="v-bridge-arrow">v</span>
          </div>
          <span className="v-bridge-label">POST /jobs, /validate (:8081)</span>
        </div>
        <div className="v-bridge-col v-bridge-col-right">
          <div className="v-bridge-flow">
            <div className="v-bridge-line" />
            <span className="v-bridge-arrow">v</span>
          </div>
          <span className="v-bridge-label">Passive Tap (jobs.&gt;) &amp; Control</span>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* TIER 2: BUSINESS & MESSAGING PIPELINE                                     */}
      {/* ========================================================================= */}
      <div className="topology-tier-header" style={{ marginTop: '0.5rem', marginBottom: '0.5rem' }}>
        <span className="topology-tier-tag">TIER 2: BUSINESS &amp; MESSAGING PIPELINE</span>
      </div>

      <div className="topology-pipeline-layout">
        {/* ========================================================================= */}
        {/* DEPLOYED COMPONENT 1: Job Service                                         */}
        {/* ========================================================================= */}
        <div className="topology-col deployed-col">
          <div className="deployed-boundary-label">DEPLOYED SERVICE</div>
          <div className={`topology-node deployed-card ${isJobActive ? 'node-active' : 'node-inactive'}`}>
            <div className="node-header">
              <span className="node-title">Job Service</span>
              <button
                type="button"
                className="node-info-btn"
                onClick={() => onSelectInfo('job-service')}
                title="Learn about Job Service"
              >
                (i)
              </button>
            </div>
            <div className="node-body">
              <span className={`node-badge ${isJobActive ? 'badge-online' : 'badge-offline'}`}>
                {isJobActive ? 'Active' : 'Offline'}
              </span>
              <span className="node-detail">HTTP API (:8081)</span>
              <span className="node-subtle">Pure Business Service</span>
            </div>
          </div>
        </div>

        {/* Connector 1: Job Service -> NATS Server */}
        <div className={`topology-connector horizontal ${isJobActive && isNatsConnected ? 'connector-active' : 'connector-inactive'}`}>
          <div className="connector-flow-group">
            <span className="connector-flow-label">Publish / Request</span>
            <div className="connector-line-with-arrow">
              <div className="connector-line" />
              <span className="connector-arrow">-&gt;</span>
            </div>
          </div>
        </div>

        {/* ========================================================================= */}
        {/* DEPLOYED COMPONENT 2: NATS Server (Runtime Boundary & Internal Model)    */}
        {/* ========================================================================= */}
        <div className="topology-col deployed-col nats-server-col">
          <div className="deployed-boundary-label">DEPLOYED SERVER BOUNDARY</div>
          <div className={`nats-server-boundary ${isNatsConnected ? 'node-active' : 'node-inactive'}`}>
            {/* NATS Server Header */}
            <div className="nats-server-header">
              <div className="node-header">
                <div className="nats-server-title-group">
                  <span className="node-title">NATS Server</span>
                  <span className="nats-port-tag">Port 4222</span>
                </div>
                <button
                  type="button"
                  className="node-info-btn"
                  onClick={() => onSelectInfo('nats-server')}
                  title="Learn about NATS Server"
                >
                  (i)
                </button>
              </div>
              <div className="node-body" style={{ marginTop: '0.25rem' }}>
                <span className={`node-badge ${isNatsConnected ? 'badge-online' : 'badge-offline'}`}>
                  {isNatsConnected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
            </div>

            {/* Capabilities Box */}
            <div className="nats-capabilities-section">
              <span className="internal-section-title">SERVER CAPABILITIES</span>
              <div className="capabilities-badges-row">
                <div className="capability-pill">
                  <span className="capability-pill-name">Core NATS</span>
                  <span className="capability-pill-desc">Pub/Sub &amp; Req/Reply</span>
                </div>
                <div className="capability-pill capability-pill-js">
                  <span className="capability-pill-name">JetStream</span>
                  <span className="capability-pill-desc">Persistence &amp; Streaming</span>
                </div>
              </div>
            </div>

            {/* Internal JetStream Model Container */}
            <div className="nats-internal-resources-box">
              <div className="internal-resources-header">
                <span className="internal-res-title">JETSTREAM MANAGED RESOURCES</span>
                <span className="internal-res-subtitle">Logical resources inside NATS</span>
              </div>

              {/* Resource A: JOBS Stream */}
              <div className="internal-resource-card stream-res-card">
                <div className="node-header">
                  <div className="res-title-group">
                    <span className="res-type-tag">STREAM</span>
                    <span className="res-name">{streamName}</span>
                  </div>
                  <button
                    type="button"
                    className="node-info-btn"
                    onClick={() => onSelectInfo('jobs-stream')}
                    title="Learn about JOBS Stream"
                  >
                    (i)
                  </button>
                </div>
                <div className="res-body">
                  <div className="node-meta-row">
                    <span className="node-badge badge-stream">
                      {jetstreamInfo?.messages !== undefined ? `${jetstreamInfo.messages} Stored` : 'Persistent Store'}
                    </span>
                    <span className="node-subtle font-mono">jobs.submitted</span>
                  </div>
                </div>
              </div>

              {/* Downward Connector: Stream -> Consumer */}
              <div className="internal-connector-down">
                <div className="internal-line-v" />
                <span className="internal-arrow-v">v</span>
                <span className="internal-connector-label">buffers to</span>
              </div>

              {/* Resource B: job-processor Consumer */}
              <div className="internal-resource-card consumer-res-card">
                <div className="node-header">
                  <div className="res-title-group">
                    <span className="res-type-tag">CONSUMER</span>
                    <span className="res-name">{consumerName}</span>
                  </div>
                  <button
                    type="button"
                    className="node-info-btn"
                    onClick={() => onSelectInfo('consumer')}
                    title="Learn about JetStream Consumers"
                  >
                    (i)
                  </button>
                </div>
                <div className="res-body">
                  <div className="node-meta-row">
                    <span className="node-badge badge-consumer">{consumerType}</span>
                    <span className="node-detail">Ordering: {ordering}</span>
                  </div>
                  <div className="consumer-res-metrics">
                    <span className={`node-metric ${pendingCount > 0 ? 'metric-highlight' : ''}`}>
                      Pending: {pendingCount}
                    </span>
                    <span className="node-metric">
                      Ack Pend: {ackPendingCount}
                    </span>
                    {redeliveredCount > 0 && (
                      <span className="node-metric metric-warning">
                        Redeliv: {redeliveredCount}
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Resource C: JOBS_DLQ Stream */}
              <div className="internal-resource-card stream-res-card" style={{ marginTop: '0.5rem', borderLeftColor: '#EF4444' }}>
                <div className="node-header">
                  <div className="res-title-group">
                    <span className="res-type-tag" style={{ color: '#FCA5A5' }}>DLQ STREAM</span>
                    <span className="res-name">JOBS_DLQ</span>
                  </div>
                  <button
                    type="button"
                    className="node-info-btn"
                    onClick={() => onSelectInfo('dead-letter-queue')}
                    title="Learn about Dead Letter Queue Stream"
                  >
                    (i)
                  </button>
                </div>
                <div className="res-body">
                  <div className="node-meta-row">
                    <span className="node-badge" style={{ background: 'rgba(239, 68, 68, 0.15)', color: '#FCA5A5', border: '1px solid rgba(239, 68, 68, 0.3)' }}>
                      {dlqStatus?.messages !== undefined ? `${dlqStatus.messages} Stored` : '0 Stored'}
                    </span>
                    <span className="node-subtle font-mono">jobs.dlq</span>
                  </div>
                </div>
              </div>

              {/* Downward Connector: DLQ Stream -> DLQ Consumer */}
              <div className="internal-connector-down">
                <div className="internal-line-v" style={{ background: 'rgba(239, 68, 68, 0.4)' }} />
                <span className="internal-arrow-v" style={{ color: '#EF4444' }}>v</span>
                <span className="internal-connector-label">inspects via</span>
              </div>

              {/* Resource D: dlq-inspector Consumer */}
              <div className="internal-resource-card consumer-res-card" style={{ borderLeftColor: '#F59E0B' }}>
                <div className="node-header">
                  <div className="res-title-group">
                    <span className="res-type-tag">DLQ CONSUMER</span>
                    <span className="res-name">dlq-inspector</span>
                  </div>
                  <button
                    type="button"
                    className="node-info-btn"
                    onClick={() => onSelectInfo('dead-letter-queue')}
                    title="Learn about dlq-inspector Consumer"
                  >
                    (i)
                  </button>
                </div>
                <div className="res-body">
                  <div className="node-meta-row">
                    <span className="node-badge badge-consumer">DURABLE</span>
                    <span className="node-detail">Pending: {dlqStatus?.pending ?? 0}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Connector 2: Consumer inside NATS Server -> Processor Service Workers */}
        <div className={`topology-connector horizontal delivery-connector ${isProcessing && isProcessorOnline ? 'connector-active' : 'connector-paused'}`}>
          <div className="connector-flow-group">
            {isProcessing && isProcessorOnline ? (
              <>
                <span className="connector-flow-label">Deliver Messages</span>
                <div className="connector-line-with-arrow">
                  <div className="connector-line" />
                  <span className="connector-arrow">-&gt;</span>
                </div>
              </>
            ) : (
              <div className="connector-paused-group">
                <span className="connector-severed-badge">[ PAUSED ]</span>
                <span className="connector-paused-label">Delivery Suspended</span>
              </div>
            )}
          </div>
        </div>

        {/* ========================================================================= */}
        {/* DEPLOYED COMPONENT 3: Processor Service (Host Application & Worker Pool) */}
        {/* ========================================================================= */}
        <div className="topology-col deployed-col processor-col">
          <div className="deployed-boundary-label">DEPLOYED SERVICE</div>
          <div className={`topology-node deployed-card processor-service-card ${isProcessorOnline ? (isProcessing ? 'node-active' : 'node-paused') : 'node-inactive'}`}>
            {/* Processor Service Header */}
            <div className="node-header">
              <span className="node-title">Processor Service</span>
              <button
                type="button"
                className="node-info-btn"
                onClick={() => onSelectInfo('processor-service')}
                title="Learn about Processor Service"
              >
                (i)
              </button>
            </div>
            <div className="node-body">
              <div className="node-meta-row">
                <span className={`node-badge ${isProcessorOnline ? (isProcessing ? 'badge-online' : 'badge-paused') : 'badge-offline'}`}>
                  {isProcessorOnline ? (isProcessing ? 'Active' : 'Paused') : 'Offline'}
                </span>
                <span className={`node-detail font-bold ${isProcessing ? 'text-success' : 'text-danger'}`}>
                  Processing: {isProcessing ? 'ON' : 'OFF'}
                </span>
              </div>
              <span className="node-subtle">
                Worker Pool: {workerCount} Worker{workerCount > 1 ? 's' : ''}
              </span>
            </div>

            {/* Nested Worker Pool */}
            <div className="processor-workers-section">
              <div className="workers-section-header">
                <span className="workers-section-title">APPLICATION WORKERS</span>
                {workerCount > 1 && (
                  <span className="competing-badge">COMPETING WORKERS</span>
                )}
              </div>

              {workerCount > 1 ? (
                <div className="competing-workers-wrapper">
                  <div className="competing-subtext">
                    Both workers pull from shared consumer '{consumerName}'
                  </div>
                  <div className="topology-workers-row">
                    {Array.from({ length: workerCount }, (_, i) => {
                      const workerName = `processor-${i + 1}`;
                      return (
                        <div
                          key={workerName}
                          className={`topology-worker-card ${isProcessorOnline ? (isProcessing ? 'worker-active' : 'worker-paused') : 'worker-offline'}`}
                        >
                          <div className="worker-header">
                            <span className="font-mono worker-name">{workerName}</span>
                            <span className={`worker-pill ${isProcessorOnline ? (isProcessing ? 'pill-green' : 'pill-amber') : 'pill-red'}`}>
                              {isProcessorOnline ? (isProcessing ? 'Pulling' : 'Idle') : 'Offline'}
                            </span>
                          </div>
                          <div className="worker-detail">Pull Worker {i + 1}</div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              ) : (
                /* Single Worker View */
                <div className={`topology-worker-card single-worker ${isProcessorOnline ? (isProcessing ? 'worker-active' : 'worker-paused') : 'worker-offline'}`}>
                  <div className="worker-header">
                    <span className="font-mono worker-name">processor-1</span>
                    <span className={`worker-pill ${isProcessorOnline ? (isProcessing ? 'pill-green' : 'pill-amber') : 'pill-red'}`}>
                      {isProcessorOnline ? (isProcessing ? 'Pulling' : 'Idle') : 'Offline'}
                    </span>
                  </div>
                  <div className="worker-detail">Pull Worker 1 (Default)</div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
