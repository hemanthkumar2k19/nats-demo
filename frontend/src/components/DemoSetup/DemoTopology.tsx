import React from 'react';
import { ServiceStatus, JetStreamInfo, ConsumerStatus } from '../../api/demoApi';

interface DemoTopologyProps {
  services: ServiceStatus[];
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  onSelectInfo: (componentId: string) => void;
}

export const DemoTopology: React.FC<DemoTopologyProps> = ({
  services,
  jetstreamInfo,
  consumerStatus,
  onSelectInfo,
}) => {
  const natsService = services.find((s) => s.name.toLowerCase().includes('nats'));
  const demoService = services.find((s) => s.name.toLowerCase().includes('demo'));
  const processorService = services.find((s) => s.name.toLowerCase().includes('processor'));

  const isNatsConnected = natsService?.status === 'connected' || natsService?.status === 'active';
  const isDemoActive = demoService?.status === 'active' || demoService?.status === 'connected';
  const isProcessorOnline = processorService?.status !== 'disconnected' && processorService?.status !== 'unknown';
  const isProcessing = processorService?.processing ?? false;

  const streamName = jetstreamInfo?.stream || 'JOBS';
  const pendingCount = consumerStatus?.pending ?? jetstreamInfo?.pending ?? 0;

  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const consumerName = consumerStatus?.name || 'job-processor';
  const ordering = consumerStatus?.ordering ? consumerStatus.ordering.toUpperCase() : 'NORMAL';
  const workerCount = consumerStatus?.workers ?? processorService?.workers ?? (isProcessorOnline ? 1 : 0);

  return (
    <div className="demo-topology-container">
      <div className="topology-pipeline-layout">
        {/* Column 1: Demo Service */}
        <div className="topology-col node-col">
          <div className={`topology-node ${isDemoActive ? 'node-active' : 'node-inactive'}`}>
            <div className="node-header">
              <span className="node-title">Demo Service</span>
              <button
                className="node-info-btn"
                onClick={() => onSelectInfo('demo-service')}
                title="Learn about Demo Service"
              >
                (i)
              </button>
            </div>
            <div className="node-body">
              <span className={`node-badge ${isDemoActive ? 'badge-online' : 'badge-offline'}`}>
                {isDemoActive ? 'Active' : 'Offline'}
              </span>
              <span className="node-detail">HTTP / Publisher</span>
            </div>
          </div>
        </div>

        {/* Connector 1: Demo Service -> NATS Server */}
        <div className={`topology-connector horizontal ${isDemoActive && isNatsConnected ? 'connector-active' : 'connector-inactive'}`}>
          <div className="connector-line" />
          <span className="connector-arrow">-&gt;</span>
        </div>

        {/* Column 2: NATS Server */}
        <div className="topology-col node-col">
          <div className={`topology-node ${isNatsConnected ? 'node-active' : 'node-inactive'}`}>
            <div className="node-header">
              <span className="node-title">NATS Server</span>
              <button
                className="node-info-btn"
                onClick={() => onSelectInfo('nats-server')}
                title="Learn about NATS Server"
              >
                (i)
              </button>
            </div>
            <div className="node-body">
              <span className={`node-badge ${isNatsConnected ? 'badge-online' : 'badge-offline'}`}>
                {isNatsConnected ? 'Connected' : 'Disconnected'}
              </span>
              <span className="node-detail">Port 4222</span>
            </div>
          </div>
        </div>

        {/* Connector 2: NATS Server -> JOBS Stream */}
        <div className={`topology-connector horizontal ${isNatsConnected ? 'connector-active' : 'connector-inactive'}`}>
          <div className="connector-line" />
          <span className="connector-arrow">-&gt;</span>
        </div>

        {/* Column 3: JetStream Pipeline (JOBS Stream -> Consumer -> Processor) */}
        <div className="topology-col stream-pipeline-col">
          {/* Node 3: JOBS Stream */}
          <div className={`topology-node stream-node ${isNatsConnected ? 'node-active' : 'node-inactive'}`}>
            <div className="node-header">
              <span className="node-title">JOBS Stream</span>
              <button
                className="node-info-btn"
                onClick={() => onSelectInfo('jobs-stream')}
                title="Learn about JOBS Stream"
              >
                (i)
              </button>
            </div>
            <div className="node-body">
              <span className="node-badge badge-stream">{streamName}</span>
              <span className="node-detail">Persistent Store</span>
            </div>
          </div>

          {/* Vertical Connector: JOBS Stream -> Consumer */}
          <div className={`topology-connector vertical ${isNatsConnected ? 'connector-active' : 'connector-inactive'}`}>
            <div className="connector-line-v" />
            <span className="connector-arrow-v">v</span>
          </div>

          {/* Node 4: Consumer */}
          <div className={`topology-node consumer-node ${isNatsConnected ? 'node-active' : 'node-inactive'}`}>
            <div className="node-header">
              <span className="node-title">Consumer</span>
              <button
                className="node-info-btn"
                onClick={() => onSelectInfo('consumer')}
                title="Learn about JetStream Consumers"
              >
                (i)
              </button>
            </div>
            <div className="node-body">
              <div className="node-meta-row">
                <span className="node-badge badge-consumer">{consumerType}</span>
                <span className="node-subtle font-mono">{consumerName}</span>
              </div>
              <div className="node-meta-row" style={{ marginTop: '0.2rem' }}>
                <span className="node-detail">Ordering: {ordering}</span>
                <span className={`node-metric ${pendingCount > 0 ? 'metric-highlight' : ''}`}>
                  Pending: {pendingCount}
                </span>
              </div>
            </div>
          </div>

          {/* Connector: Consumer -> Worker Pool */}
          <div className={`topology-connector vertical ${isProcessing && isProcessorOnline ? 'connector-active' : 'connector-paused'}`}>
            <div className="connector-line-v" />
            {!(isProcessing && isProcessorOnline) ? (
              <span className="connector-severed-badge">[ PAUSED ]</span>
            ) : workerCount === 1 ? (
              <span className="connector-arrow-v">v</span>
            ) : null}
          </div>

          {workerCount > 1 ? (
            <div className="topology-competing-pool">
              {/* Branch Header & Split Lines */}
              <div className={`topology-branch-indicator ${isProcessing && isProcessorOnline ? 'connector-active' : 'connector-paused'}`}>
                <div className="branch-split-line" />
                <div className="branch-arrow-row">
                  <span className="connector-arrow-v">v</span>
                  <span className="competing-badge">COMPETING</span>
                  <span className="connector-arrow-v">v</span>
                </div>
              </div>

              {/* Multiple Worker Cards */}
              <div className="topology-workers-row">
                {Array.from({ length: workerCount }, (_, i) => {
                  const workerName = `processor-${i + 1}`;
                  return (
                    <div
                      key={workerName}
                      className={`topology-node worker-subnode ${isProcessorOnline ? (isProcessing ? 'node-active' : 'node-paused') : 'node-inactive'}`}
                    >
                      <div className="node-header">
                        <span className="node-title font-mono">{workerName}</span>
                        <button
                          className="node-info-btn"
                          onClick={() => onSelectInfo('processor-service')}
                          title="Learn about Processor Worker"
                        >
                          (i)
                        </button>
                      </div>
                      <div className="node-body">
                        <span className={`node-badge ${isProcessorOnline ? (isProcessing ? 'badge-online' : 'badge-paused') : 'badge-offline'}`}>
                          {isProcessorOnline ? (isProcessing ? 'Active' : 'Paused') : 'Offline'}
                        </span>
                        <span className="node-detail">Pull Worker {i + 1}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : (
            /* Single Worker Node (processor-1) */
            <div className={`topology-node processor-node ${isProcessorOnline ? (isProcessing ? 'node-active' : 'node-paused') : 'node-inactive'}`}>
              <div className="node-header">
                <span className="node-title font-mono">processor-1</span>
                <button
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
                  <span className="node-subtle">1 Worker Active</span>
                </div>
                <span className="node-detail">
                  Processing: {isProcessing ? 'ON' : 'OFF'}
                </span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
