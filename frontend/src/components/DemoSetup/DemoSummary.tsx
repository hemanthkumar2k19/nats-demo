import React from 'react';
import { ServiceStatus, ConsumerStatus } from '../../api/demoApi';

interface DemoSummaryProps {
  services: ServiceStatus[];
  consumerStatus?: ConsumerStatus | null;
}

export const DemoSummary: React.FC<DemoSummaryProps> = ({
  services,
  consumerStatus,
}) => {
  const processorService = services.find((s) => s.name.toLowerCase().includes('processor'));
  const isProcessorOnline = processorService?.status !== 'disconnected' && processorService?.status !== 'unknown';
  const isProcessing = processorService?.processing ?? false;

  const consumerType = consumerStatus?.type ? consumerStatus.type.toUpperCase() : 'DURABLE';
  const workers = consumerStatus?.workers ?? processorService?.workers ?? 1;
  const ordering = consumerStatus?.ordering ? consumerStatus.ordering.toUpperCase() : 'NORMAL';

  return (
    <div className="demo-summary-card">
      <div className="summary-header">
        <span className="summary-title">CURRENT DEMO</span>
        <span className="badge badge-nats">Active Config</span>
      </div>

      <div className="summary-list">
        <div className="summary-item">
          <span className="summary-label">Delivery:</span>
          <span className="summary-value font-mono">JetStream / Core</span>
        </div>

        <div className="summary-item">
          <span className="summary-label">Consumer:</span>
          <span className="summary-value uppercase">{consumerType}</span>
        </div>

        <div className="summary-item">
          <span className="summary-label">Workers:</span>
          <span className="summary-value">
            {workers} {workers === 1 ? 'Worker' : 'Workers (Competing)'}
          </span>
        </div>

        <div className="summary-item">
          <span className="summary-label">Semantics:</span>
          <span className="summary-value">At Least Once</span>
        </div>

        <div className="summary-item">
          <span className="summary-label">Ordering:</span>
          <span className="summary-value">{ordering}</span>
        </div>

        <div className="summary-item">
          <span className="summary-label">Processor:</span>
          <span className={`summary-value font-bold ${isProcessorOnline ? (isProcessing ? 'text-success' : 'text-danger') : 'text-muted'}`}>
            {isProcessorOnline ? (isProcessing ? 'ON' : 'OFF') : 'OFFLINE'}
          </span>
        </div>
      </div>
    </div>
  );
};
