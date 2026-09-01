import React from 'react';

interface ObservabilityPanelProps {
  onShowInfo: (key: string) => void;
}

export const ObservabilityPanel: React.FC<ObservabilityPanelProps> = ({ onShowInfo }) => {
  return (
    <section className="panel observability-panel">
      <div className="panel-header" style={{ marginBottom: '1.25rem' }}>
        <div>
          <div className="panel-title" style={{ gap: '0.625rem' }}>
            <span>OBSERVABILITY SETUP</span>
            <button
              className="info-btn"
              onClick={() => onShowInfo('metrics-observability')}
              title="Learn about NATS & OpenTelemetry Metrics Observability"
            >
              (i)
            </button>
          </div>
          <p className="panel-subtitle">
            Local metrics observability pipeline combining OpenTelemetry application instrumentation, NATS Prometheus exporter, and Grafana OTEL-LGTM.
          </p>
        </div>

        <a
          href="http://localhost:3000"
          target="_blank"
          rel="noopener noreferrer"
          className="btn-open-grafana"
          title="Open Grafana Metrics Dashboard in a new window"
        >
          <span>Open Grafana</span>
          <span style={{ fontSize: '0.875rem' }}>-&gt;</span>
        </a>
      </div>

      {/* Observability Architecture Diagram */}
      <div className="obs-architecture-container">
        {/* Source 1: Application Services (OTel) */}
        <div className="obs-source-group">
          <div className="obs-group-title">APPLICATION METRICS (OTEL)</div>
          
          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">Demo Service</span>
              <span className="badge badge-published" style={{ fontSize: '0.625rem' }}>OTLP</span>
            </div>
            <div className="obs-node-body">
              <div>jobs_submitted_total</div>
              <div>job_submission_duration</div>
              <div>nats_publish_total</div>
              <div>nats_request_total</div>
            </div>
          </div>

          <div className="obs-connector-down">|</div>

          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">Processor Service</span>
              <span className="badge badge-published" style={{ fontSize: '0.625rem' }}>OTLP</span>
            </div>
            <div className="obs-node-body">
              <div>jobs_processed_total</div>
              <div>jobs_failed_total</div>
              <div>job_processing_duration</div>
              <div>nats_messages_acked_total</div>
            </div>
          </div>
        </div>

        {/* Arrow to LGTM */}
        <div className="obs-flow-arrow">
          <div className="obs-arrow-line">-----------------&gt;</div>
          <span className="obs-arrow-label">OTLP gRPC (:4317)</span>
        </div>

        {/* Central: Grafana OTEL-LGTM Stack */}
        <div className="obs-central-group">
          <div className="obs-central-header">
            <span className="obs-central-title">GRAFANA OTEL-LGTM</span>
            <span className="indicator-dot active" />
          </div>
          <div className="obs-central-subtitle">All-in-One Local Observability Stack</div>

          <div className="obs-central-submodules">
            <div className="obs-submodule-item">
              <div className="obs-submodule-name">OpenTelemetry Collector</div>
              <div className="obs-submodule-desc">Receives application OTLP metrics (:4317 / :4318)</div>
            </div>

            <div className="obs-submodule-item">
              <div className="obs-submodule-name">Prometheus Engine</div>
              <div className="obs-submodule-desc">Scrapes NATS Exporter (:7777) & stores timeseries</div>
            </div>

            <div className="obs-submodule-item obs-submodule-highlight">
              <div className="obs-submodule-name">Grafana Dashboard</div>
              <div className="obs-submodule-desc">NATS Platform Demo - Metrics (:3000)</div>
            </div>
          </div>
        </div>

        {/* Arrow from NATS Exporter */}
        <div className="obs-flow-arrow">
          <div className="obs-arrow-line">&lt;-----------------</div>
          <span className="obs-arrow-label">Prometheus Scrape (:7777)</span>
        </div>

        {/* Source 2: NATS Infrastructure */}
        <div className="obs-source-group">
          <div className="obs-group-title">NATS INFRASTRUCTURE METRICS</div>

          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">NATS Server</span>
              <span className="badge badge-received" style={{ fontSize: '0.625rem' }}>:8222</span>
            </div>
            <div className="obs-node-body">
              <div>Core NATS (:4222)</div>
              <div>JetStream Engine</div>
              <div>HTTP Monitoring (:8222)</div>
              <div>/varz, /connz, /jsz</div>
            </div>
          </div>

          <div className="obs-connector-down">|</div>

          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">NATS Prometheus Exporter</span>
              <span className="badge badge-received" style={{ fontSize: '0.625rem' }}>:7777</span>
            </div>
            <div className="obs-node-body">
              <div>gnatsd_varz_connections</div>
              <div>gnatsd_varz_in/out_msgs</div>
              <div>gnatsd_jsz_stream_messages</div>
              <div>gnatsd_jsz_consumer_pending</div>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Access Footnotes / Endpoints */}
      <div className="obs-endpoints-grid">
        <div className="obs-endpoint-card">
          <div className="obs-endpoint-label">Grafana UI</div>
          <div className="obs-endpoint-val">
            <a href="http://localhost:3000" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-cyan)' }}>
              http://localhost:3000
            </a>
          </div>
          <div className="obs-endpoint-hint">User: admin / Pass: admin</div>
        </div>

        <div className="obs-endpoint-card">
          <div className="obs-endpoint-label">OTLP Ingestion Endpoint</div>
          <div className="obs-endpoint-val">localhost:4317 (gRPC) / 4318 (HTTP)</div>
          <div className="obs-endpoint-hint">Periodic push interval: 2s</div>
        </div>

        <div className="obs-endpoint-card">
          <div className="obs-endpoint-label">NATS Exporter Metrics</div>
          <div className="obs-endpoint-val">
            <a href="http://localhost:7777/metrics" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-cyan)' }}>
              http://localhost:7777/metrics
            </a>
          </div>
          <div className="obs-endpoint-hint">Prometheus format, scraped every 2s</div>
        </div>

        <div className="obs-endpoint-card">
          <div className="obs-endpoint-label">Dashboard Name</div>
          <div className="obs-endpoint-val" style={{ color: '#A78BFA' }}>NATS Platform Demo - Metrics</div>
          <div className="obs-endpoint-hint">4 sections: NATS, JetStream, App, Processing</div>
        </div>
      </div>
    </section>
  );
};
