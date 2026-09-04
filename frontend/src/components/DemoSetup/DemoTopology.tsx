import React from 'react';
import { ServiceStatus, JetStreamInfo, ConsumerStatus, DLQStatus, QueueGroupStatus } from '../../api/demoApi';

interface DemoTopologyProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  dlqStatus?: DLQStatus | null;
  queueGroupStatus?: QueueGroupStatus | null;
  onSelectInfo: (componentId: string) => void;
}

export const DemoTopology: React.FC<DemoTopologyProps> = ({
  services,
  jetstreamInfo,
  consumerStatus,
  dlqStatus,
  queueGroupStatus,
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
  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const consumerName = consumerStatus?.name || 'job-processor';
  const ordering = consumerStatus?.ordering ? consumerStatus.ordering.toUpperCase() : 'NORMAL';
  const workerCount = consumerStatus?.workers ?? processorService?.workers ?? (isProcessorOnline ? 1 : 0);
  const queueWorkers = queueGroupStatus?.workers ?? 1;

  return (
    <div className="demo-topology-container">
      <div className="topology-runtime-header">
        <span className="topology-legend-tag">DEPLOYED RUNTIME ARCHITECTURE</span>
        <span className="topology-legend-note">
          Three-Tier decoupled architecture: Client Gateway -&gt; Ingress &amp; NATS Server -&gt; Worker Daemon
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
          <div className="topology-col deployed-col" style={{ width: '220px', minWidth: '220px', maxWidth: '220px' }}>
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
          <div className={`topology-connector-card horizontal ${isDemoControlActive ? 'connector-active' : 'connector-inactive'}`} style={{ width: '180px', minWidth: '160px', maxWidth: '200px' }}>
            <div className="connector-card-header">
              <span className="connector-card-title">UI GATEWAY (:8080)</span>
              <div className="connector-h-arrow-group">
                <div className="connector-h-line" />
                <span className="connector-h-arrow">-&gt;</span>
              </div>
            </div>
            <div className="connector-card-body">
              <div className="connector-card-item">
                <span className="connector-tag tag-rest">REST</span>
                <span className="connector-item-text">GET /activities • /status</span>
              </div>
              <div className="connector-card-item">
                <span className="connector-tag tag-ctrl">CTRL</span>
                <span className="connector-item-text">PUT /consumer • /queue-group</span>
              </div>
              <div className="connector-card-item">
                <span className="connector-tag tag-act">DATA</span>
                <span className="connector-item-text">Activity Polling &amp; DLQ Reprocess</span>
              </div>
            </div>
          </div>

          {/* Component: Demo Control Service */}
          <div className="topology-col deployed-col" style={{ width: '240px', minWidth: '240px', maxWidth: '240px' }}>
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
                <span className="node-subtle">Activity Ring Buffer &amp; Tap</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* INTER-TIER VERTICAL CONNECTORS (Tier 1 -> Tier 2)                         */}
      {/* ========================================================================= */}
      <div className="topology-intertier-bridge-row">
        {/* Connection 1: React UI -> Job Service */}
        <div className={`topology-connector vertical ${isJobActive ? 'connector-active' : 'connector-inactive'}`} style={{ width: '220px', minWidth: '220px', maxWidth: '220px' }}>
          <div className="connector-v-flow">
            <div className="connector-v-arrow-group">
              <div className="connector-v-line" />
              <span className="connector-v-arrow">v</span>
            </div>
            <div className="connector-v-details">
              <span className="connector-v-title">HTTP REST Ingress (:8081)</span>
              <span className="connector-v-sub">POST /jobs, /schedule, /validate</span>
            </div>
          </div>
        </div>

        {/* Center Spacer matching the 180px horizontal connector column */}
        <div style={{ width: '180px', minWidth: '160px', maxWidth: '200px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ fontSize: '0.625rem', color: 'var(--border-color)', fontFamily: 'var(--font-mono)' }}>|</span>
        </div>

        {/* Connection 2: Demo Control Service -> NATS Server */}
        <div className={`topology-connector vertical ${isDemoControlActive && isNatsConnected ? 'connector-active' : 'connector-inactive'}`} style={{ width: '240px', minWidth: '240px', maxWidth: '240px' }}>
          <div className="connector-v-flow">
            <div className="connector-v-arrow-group">
              <div className="connector-v-line" />
              <span className="connector-v-arrow">v</span>
            </div>
            <div className="connector-v-details">
              <span className="connector-v-title">NATS TCP Client (:4222)</span>
              <span className="connector-v-sub">jobs.&gt; Tap • Replay • Control RPC</span>
            </div>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* TIER 2: BUSINESS INGRESS & NATS SERVER ENGINE                             */}
      {/* ========================================================================= */}
      <div className="topology-tier-header" style={{ marginTop: '0.25rem', marginBottom: '0.35rem' }}>
        <span className="topology-tier-tag">TIER 2: BUSINESS INGRESS &amp; NATS SERVER ENGINE</span>
      </div>

      <div className="topology-ingress-broker-layout">
        {/* Component 1: Job Service */}
        <div className="topology-col deployed-col" style={{ width: '220px', minWidth: '220px', maxWidth: '220px' }}>
          <div className="deployed-boundary-label">BUSINESS INGRESS</div>
          <div className={`topology-node deployed-card job-service-card ${isJobActive ? 'node-active' : 'node-inactive'}`}>
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
              <div style={{ marginTop: '0.35rem', paddingTop: '0.35rem', borderTop: '1px solid var(--border-color)', fontSize: '0.625rem', color: 'var(--text-muted)' }}>
                W3C <span className="font-mono text-cyan">traceparent</span> context injection
              </div>
            </div>
          </div>
        </div>

        {/* Connector: Job Service -> NATS Server */}
        <div className={`topology-connector-card horizontal ${isJobActive && isNatsConnected ? 'connector-active' : 'connector-inactive'}`} style={{ width: '180px', minWidth: '160px', maxWidth: '200px' }}>
          <div className="connector-card-header">
            <span className="connector-card-title">NATS TCP INGRESS (:4222)</span>
            <div className="connector-h-arrow-group">
              <div className="connector-h-line" />
              <span className="connector-h-arrow">-&gt;</span>
            </div>
          </div>
          <div className="connector-card-body">
            <div className="connector-card-item">
              <span className="connector-tag tag-pub">PUB</span>
              <span className="connector-item-text">jobs.submitted • jobs.queue</span>
            </div>
            <div className="connector-card-item">
              <span className="connector-tag tag-rpc">RPC</span>
              <span className="connector-item-text">jobs.validate (Request)</span>
            </div>
            <div className="connector-card-item">
              <span className="connector-tag tag-hdr">HDR</span>
              <span className="connector-item-text">W3C traceparent • Msg-Id</span>
            </div>
          </div>
        </div>

        {/* Component 2: WIDENED NATS Server Block (Dual Engine: Core NATS & JetStream) */}
        <div className="topology-col deployed-col nats-server-col-wide">
          <div className="deployed-boundary-label">MESSAGE BROKER &amp; PERSISTENCE ENGINE</div>
          <div className={`nats-server-boundary ${isNatsConnected ? 'node-active' : 'node-inactive'}`}>
            {/* NATS Server Header */}
            <div className="nats-server-header">
              <div className="node-header">
                <div className="nats-server-title-group">
                  <span className="node-title">NATS Server</span>
                  <span className="nats-port-tag">Port 4222 / 8222</span>
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
              <div className="node-body" style={{ marginTop: '0.2rem' }}>
                <span className={`node-badge ${isNatsConnected ? 'badge-online' : 'badge-offline'}`}>
                  {isNatsConnected ? 'Connected & Operational' : 'Disconnected'}
                </span>
              </div>
            </div>

            {/* Dual Engine Side-by-Side Grid */}
            <div className="nats-dual-engine-grid">
              {/* Compartment A: Core NATS Engine (In-Memory / Transient) */}
              <div className="engine-compartment core-engine">
                <div className="engine-header">
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                    <span className="engine-title" style={{ color: '#60A5FA' }}>Core NATS</span>
                    <button
                      type="button"
                      className="node-info-btn"
                      onClick={() => onSelectInfo('queue-groups')}
                      title="Learn about Core NATS"
                    >
                      (i)
                    </button>
                  </div>
                  <span className="engine-tag tag-core">IN-MEMORY • TRANSIENT</span>
                </div>

                {/* Core NATS Item 1: Transient Pub/Sub */}
                <div className="internal-item-box">
                  <div className="internal-item-title-row">
                    <span className="internal-item-name">Transient Pub / Sub</span>
                    <span className="internal-item-meta">jobs.*, jobs.&gt;</span>
                  </div>
                  <span className="internal-item-desc">At-Most-Once fanout • Discarded if offline</span>
                </div>

                {/* Core NATS Item 2: Queue Groups */}
                <div className="internal-item-box item-highlight">
                  <div className="internal-item-title-row">
                    <span className="internal-item-name">Queue Group</span>
                    <span className="internal-item-meta">jobs.queue</span>
                  </div>
                  <span className="internal-item-desc">
                    Group: <span className="font-mono text-cyan">job-workers</span> ({queueWorkers} Sub{queueWorkers > 1 ? 's' : ''} • 1-of-N balance)
                  </span>
                </div>

                {/* Core NATS Item 3: Request / Reply */}
                <div className="internal-item-box">
                  <div className="internal-item-title-row">
                    <span className="internal-item-name">Request / Reply RPC</span>
                    <span className="internal-item-meta">jobs.validate</span>
                  </div>
                  <span className="internal-item-desc">Synchronous reply inbox • 2s Timeout</span>
                </div>
              </div>

              {/* Compartment B: JetStream Engine (Persistent Storage & Stateful Cursors) */}
              <div className="engine-compartment js-engine">
                <div className="engine-header">
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                    <span className="engine-title" style={{ color: '#C4B5FD' }}>JetStream Engine</span>
                    <button
                      type="button"
                      className="node-info-btn"
                      onClick={() => onSelectInfo('jobs-stream')}
                      title="Learn about JetStream"
                    >
                      (i)
                    </button>
                  </div>
                  <span className="engine-tag tag-js">PERSISTENT STORAGE</span>
                </div>

                {/* JetStream Pipeline 1: JOBS Stream & Pull Consumer */}
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
                      <span className="node-badge badge-stream">PERSISTENT LOG</span>
                      <span className="node-subtle font-mono">jobs.submitted</span>
                    </div>
                  </div>
                </div>

                <div className="internal-connector-down">
                  <div className="internal-line-v" />
                  <span className="internal-arrow-v">v</span>
                  <span className="internal-connector-label">buffers to pull cursor</span>
                </div>

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
                      <span className="node-detail">Pull Mode</span>
                      <span className="node-detail">•</span>
                      <span className="node-detail">{consumerType === 'DURABLE' ? 'DeliverAll' : 'DeliverNew'}</span>
                      <span className="node-detail">•</span>
                      <span className="node-detail">Ordering: {ordering}</span>
                    </div>
                  </div>
                </div>

                {/* JetStream Pipeline 2: JOBS_DLQ Stream & Inspector */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', gap: '0.35rem', alignItems: 'center', marginTop: '0.2rem' }}>
                  <div className="internal-resource-card stream-res-card" style={{ borderLeftColor: '#EF4444' }}>
                    <div className="node-header">
                      <span className="res-name" style={{ fontSize: '0.6875rem' }}>JOBS_DLQ</span>
                      <button
                        type="button"
                        className="node-info-btn"
                        onClick={() => onSelectInfo('dead-letter-queue')}
                        title="Learn about DLQ Stream"
                      >
                        (i)
                      </button>
                    </div>
                    <div className="node-meta-row">
                      <span className="node-badge" style={{ fontSize: '0.5625rem', background: 'rgba(239, 68, 68, 0.15)', color: '#FCA5A5', border: '1px solid rgba(239, 68, 68, 0.3)' }}>
                        POISON STORE
                      </span>
                      <span className="node-subtle font-mono" style={{ fontSize: '0.5625rem' }}>jobs.dlq</span>
                    </div>
                  </div>

                  <span className="font-mono text-muted" style={{ fontSize: '0.625rem' }}>-&gt;</span>

                  <div className="internal-resource-card consumer-res-card" style={{ borderLeftColor: '#F59E0B' }}>
                    <div className="node-header">
                      <span className="res-name" style={{ fontSize: '0.6875rem' }}>dlq-inspector</span>
                      <button
                        type="button"
                        className="node-info-btn"
                        onClick={() => onSelectInfo('dead-letter-queue')}
                        title="Learn about dlq-inspector"
                      >
                        (i)
                      </button>
                    </div>
                    <div className="node-meta-row">
                      <span className="node-badge badge-consumer" style={{ fontSize: '0.5625rem' }}>
                        DURABLE CURSOR
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* INTER-TIER DELIVERY & FEEDBACK BRIDGE (Tier 2 NATS -> Tier 3 Processor)  */}
      {/* ========================================================================= */}
      <div className="topology-delivery-bridge-grid">
        {/* Card 1: Downstream Message Delivery & Pull Dispatch */}
        <div className={`delivery-bridge-card downstream ${isProcessing && isProcessorOnline ? '' : 'paused'}`}>
          <div className="delivery-card-header">
            <div className="delivery-card-title-group">
              <span className={`delivery-card-arrow down ${isProcessing && isProcessorOnline ? '' : 'paused'}`}>v</span>
              <span className="delivery-card-title">Downstream Message Delivery (NATS -&gt; Processor)</span>
            </div>
            <span className={`delivery-card-badge ${isProcessing && isProcessorOnline ? 'badge-active' : 'badge-paused'}`}>
              {isProcessing && isProcessorOnline ? 'DELIVERY ACTIVE' : 'DELIVERY PAUSED'}
            </span>
          </div>
          <div className="delivery-card-body">
            <div className="delivery-card-item">
              <span className="delivery-card-label label-pull">PULL</span>
              <span className="delivery-card-text">Stream JOBS -&gt; Consumer '{consumerName}' ({consumerType})</span>
            </div>
            <div className="delivery-card-item">
              <span className="delivery-card-label label-queue">QUEUE</span>
              <span className="delivery-card-text">Core NATS 1-of-N -&gt; Group 'job-workers' on jobs.queue</span>
            </div>
            <div className="delivery-card-item">
              <span className="delivery-card-label label-rpc">RPC</span>
              <span className="delivery-card-text">Sync Request Dispatch -&gt; Responder jobs.validate</span>
            </div>
            <div className="delivery-card-item">
              <span className="delivery-card-label label-timing">POLICY</span>
              <span className="delivery-card-text">AckWait: 5s • NakWithDelay Backoff • Ordering: {ordering}</span>
            </div>
          </div>
        </div>

        {/* Card 2: Upstream Acknowledgment & Lifecycle Feedback */}
        <div className="delivery-bridge-card upstream">
          <div className="delivery-card-header">
            <div className="delivery-card-title-group">
              <span className="delivery-card-arrow up">^</span>
              <span className="delivery-card-title">Upstream Protocol Acks &amp; Lifecycle Feedback</span>
            </div>
            <span className="delivery-card-badge badge-feedback">BIDIRECTIONAL FEEDBACK</span>
          </div>
          <div className="delivery-card-body">
            <div className="delivery-card-item">
              <span className="delivery-card-label label-ack">ACKS</span>
              <span className="delivery-card-text">Explicit msg.Ack() • msg.NakWithDelay(d) • msg.Term()</span>
            </div>
            <div className="delivery-card-item">
              <span className="delivery-card-label label-evt">EVENTS</span>
              <span className="delivery-card-text">jobs.received • jobs.processing • jobs.completed • jobs.failed</span>
            </div>
            <div className="delivery-card-item">
              <span className="delivery-card-label label-dlq">POISON</span>
              <span className="delivery-card-text">Max Deliveries (3) Routing -&gt; Stream JOBS_DLQ (jobs.dlq)</span>
            </div>
            <div className="delivery-card-item">
              <span className="delivery-card-label label-meta">METRICS</span>
              <span className="delivery-card-text">Delivery Counts • Stream Sequence • Worker Attribution</span>
            </div>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* TIER 3: WORKER DAEMON & CONSUMER BINDINGS                                 */}
      {/* ========================================================================= */}
      <div className="topology-tier-header" style={{ marginBottom: '0.35rem' }}>
        <span className="topology-tier-tag">TIER 3: WORKER DAEMON &amp; CONSUMER BINDINGS</span>
      </div>

      <div className="processor-service-wide-card">
        {/* Processor Service Header */}
        <div className="node-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <span className="node-title" style={{ fontSize: '0.875rem' }}>Processor Service</span>
            <span className={`node-badge ${isProcessorOnline ? (isProcessing ? 'badge-online' : 'badge-paused') : 'badge-offline'}`}>
              {isProcessorOnline ? (isProcessing ? 'Active' : 'Paused') : 'Offline'}
            </span>
            <span className={`node-detail font-bold ${isProcessing ? 'text-success' : 'text-danger'}`}>
              Processing: {isProcessing ? 'ON' : 'OFF'}
            </span>
            <span className="node-subtle">
              Total Worker Pool: {workerCount} Worker{workerCount > 1 ? 's' : ''}
            </span>
          </div>
          <button
            type="button"
            className="node-info-btn"
            onClick={() => onSelectInfo('processor-service')}
            title="Learn about Processor Service"
          >
            (i)
          </button>
        </div>

        {/* Dual Compartment Worker Grid */}
        <div className="processor-dual-worker-grid">
          {/* Compartment 1: JetStream Competing Pull Workers */}
          <div className="worker-compartment">
            <div className="worker-compartment-header">
              <span className="worker-compartment-title">JetStream Pull Workers</span>
              <span className="competing-badge">
                {workerCount === 1 ? '1 PULL WORKER' : `${workerCount} COMPETING WORKERS`}
              </span>
            </div>
            <div className="worker-subtext">
              Workers pull from durable consumer '{consumerName}' on stream '{streamName}'
            </div>
            <div className="topology-workers-row" style={{ display: 'grid', gridTemplateColumns: `repeat(${Math.min(workerCount, 3)}, 1fr)`, gap: '0.35rem' }}>
              {Array.from({ length: workerCount }, (_, i) => {
                const workerName = `processor-${i + 1}`;
                return (
                  <div
                    key={workerName}
                    className={`topology-worker-card ${isProcessorOnline ? (isProcessing ? 'worker-active' : 'worker-paused') : 'worker-offline'}`}
                  >
                    <div className="worker-header">
                      <span className="font-mono worker-name" style={{ fontSize: '0.75rem' }}>{workerName}</span>
                      <span className={`worker-pill ${isProcessorOnline ? (isProcessing ? 'pill-green' : 'pill-amber') : 'pill-red'}`}>
                        {isProcessorOnline ? (isProcessing ? 'Pull' : 'Idle') : 'Off'}
                      </span>
                    </div>
                    <div className="worker-detail" style={{ fontSize: '0.625rem' }}>JS Worker {i + 1}</div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Compartment 2: Core NATS Queue Group Subscribers */}
          <div className="worker-compartment">
            <div className="worker-compartment-header">
              <span className="worker-compartment-title">Core NATS Queue Group</span>
              <span className="competing-badge" style={{ background: 'rgba(59, 130, 246, 0.15)', color: '#60A5FA', borderColor: 'rgba(59, 130, 246, 0.3)' }}>
                {isProcessorOnline
                  ? (queueWorkers === 1 ? '1 SUBSCRIBER' : `${queueWorkers} SUBSCRIBERS (1-OF-N)`)
                  : 'OFFLINE'}
              </span>
            </div>
            <div className="worker-subtext">
              Subscribers bound to queue group 'job-workers' on subject 'jobs.queue'
            </div>
            <div className="topology-workers-row" style={{ display: 'grid', gridTemplateColumns: `repeat(${Math.min(queueWorkers, 3)}, 1fr)`, gap: '0.35rem' }}>
              {Array.from({ length: queueWorkers }, (_, i) => {
                const wName = `processor-${i + 1}`;
                return (
                  <div
                    key={wName}
                    className={`topology-worker-card ${isProcessorOnline ? (isProcessing ? 'worker-active' : 'worker-paused') : 'worker-offline'}`}
                  >
                    <div className="worker-header">
                      <span className="font-mono worker-name" style={{ fontSize: '0.75rem' }}>{wName}</span>
                      <span className={`worker-pill ${isProcessorOnline ? (isProcessing ? 'pill-green' : 'pill-amber') : 'pill-red'}`}>
                        {isProcessorOnline ? (isProcessing ? 'Sub' : 'Idle') : 'Off'}
                      </span>
                    </div>
                    <div className="worker-detail font-mono" style={{ fontSize: '0.625rem' }}>jobs.queue</div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Footer: Lifecycle Event Emission */}
        <div className="processor-lifecycle-footer">
          <span>Processor emits lifecycle events on completion/failure:</span>
          <div className="lifecycle-chips-group">
            <span className="lifecycle-chip">jobs.received</span>
            <span className="lifecycle-chip">jobs.completed</span>
            <span className="lifecycle-chip">jobs.failed</span>
            <span className="lifecycle-chip">jobs.dlq.published</span>
          </div>
        </div>
      </div>
    </div>
  );
};
