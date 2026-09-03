import React from 'react';

interface ObservabilityPanelProps {
  onShowInfo: (key: string) => void;
}

export const ObservabilityPanel: React.FC<ObservabilityPanelProps> = ({ onShowInfo }) => {
  return (
    <section className="panel observability-panel">
      <div className="panel-header" style={{ marginBottom: '1rem' }}>
        <div>
          <div className="panel-title" style={{ gap: '0.625rem' }}>
            <span>OBSERVABILITY SETUP (LGTM ARCHITECTURE)</span>
            <button
              className="info-btn"
              onClick={() => onShowInfo('metrics-observability')}
              title="Learn about NATS, OpenTelemetry Tracing &amp; Metrics Observability"
            >
              (i)
            </button>
          </div>
          <p className="panel-subtitle">
            Source-to-destination telemetry flows, transport conduits, and Grafana OTEL-LGTM container.
          </p>
        </div>

        <div style={{ display: 'flex', gap: '0.4rem', flexWrap: 'wrap' }}>
          <a
            href="http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22loki%22,%7B%22expr%22:%22%7Bservice%3D%5C%22nats%5C%22%7D%22%7D%5D"
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-secondary"
            style={{ display: 'inline-flex', alignItems: 'center', gap: '0.3rem', textDecoration: 'none', fontSize: '0.75rem', borderColor: 'rgba(245, 158, 11, 0.4)', color: '#FBBF24' }}
            title="Explore NATS Server Logs in Loki"
          >
            <span>NATS Logs</span>
            <span style={{ fontSize: '0.75rem' }}>-&gt;</span>
          </a>
          <a
            href="http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22loki%22,%7B%22expr%22:%22%7Bservice%3D%5C%22nats-events%5C%22%7D%22%7D%5D"
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-secondary"
            style={{ display: 'inline-flex', alignItems: 'center', gap: '0.3rem', textDecoration: 'none', fontSize: '0.75rem', borderColor: 'rgba(217, 70, 239, 0.4)', color: '#F0ABFC' }}
            title="Explore NATS Operational Events & Advisories in Loki"
          >
            <span>NATS Events</span>
            <span style={{ fontSize: '0.75rem' }}>-&gt;</span>
          </a>
          <a
            href="http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22prometheus%22,%7B%22expr%22:%22%7Bjob%3D%5C%22nats-exporter%5C%22%7D%22%7D%5D"
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-secondary"
            style={{ display: 'inline-flex', alignItems: 'center', gap: '0.3rem', textDecoration: 'none', fontSize: '0.75rem', borderColor: 'rgba(6, 182, 212, 0.4)', color: '#38BDF8' }}
            title="Explore NATS Exporter Infrastructure Metrics in Prometheus"
          >
            <span>NATS Metrics</span>
            <span style={{ fontSize: '0.75rem' }}>-&gt;</span>
          </a>
          <a
            href="http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22tempo%22,%7B%22queryType%22:%22traceql%22%7D%5D"
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-secondary"
            style={{ display: 'inline-flex', alignItems: 'center', gap: '0.3rem', textDecoration: 'none', fontSize: '0.75rem', borderColor: 'rgba(129, 140, 248, 0.4)', color: '#818cf8' }}
            title="Explore Application Distributed Traces in Tempo"
          >
            <span>Application Traces</span>
            <span style={{ fontSize: '0.75rem' }}>-&gt;</span>
          </a>
          <a
            href="http://localhost:3000"
            target="_blank"
            rel="noopener noreferrer"
            className="btn-open-grafana"
            style={{ fontSize: '0.75rem', padding: '0.4rem 0.75rem' }}
            title="Open Grafana Dashboard in a new window"
          >
            <span>Open Grafana</span>
            <span style={{ fontSize: '0.75rem' }}>-&gt;</span>
          </a>
        </div>
      </div>

      {/* 3-Column T-Shape Architecture with Inner LGTM Boxes */}
      <div className="obs-architecture-container">
        {/* Column 1: Source - Application Layer */}
        <div className="obs-source-group">
          <div className="obs-group-title">SOURCE: APP SERVICES</div>
          
          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">Job Service</span>
              <span className="badge badge-published" style={{ fontSize: '0.625rem' }}>:8081</span>
            </div>
            <div className="obs-node-body">
              <div className="obs-node-item">
                <span className="obs-tag tag-trace">TRACES</span>
                <span>W3C traceparent &bull; POST /jobs</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-metric">METRICS</span>
                <span>jobs_submitted_total &bull; latency</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-proto">RPC SPAN</span>
                <span>jobs.validate (2s timeout)</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-app">FORMAT</span>
                <span>OTel Go SDK (OTLP / gRPC)</span>
              </div>
            </div>
          </div>

          <div className="obs-connector-down">|</div>

          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">Processor Service</span>
              <span className="badge badge-received" style={{ fontSize: '0.625rem' }}>WORKER</span>
            </div>
            <div className="obs-node-body">
              <div className="obs-node-item">
                <span className="obs-tag tag-trace">TRACES</span>
                <span>NATS Header Extract &bull; Spans</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-metric">METRICS</span>
                <span>jobs_processed &bull; jobs_failed</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-proto">LIFECYCLE</span>
                <span>jobs.received &bull; jobs.completed</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-app">FORMAT</span>
                <span>OTel Go SDK (OTLP / gRPC)</span>
              </div>
            </div>
          </div>
        </div>

        {/* Connector 1: Coloured OTLP Transport Bridge */}
        <div className="obs-conduit-card">
          <div className="obs-conduit-header">
            <span className="obs-conduit-title">OTLP TRANSPORT</span>
            <span className="badge badge-published" style={{ fontSize: '0.5625rem' }}>:4317</span>
          </div>

          {/* Metric Track (Green) */}
          <div className="obs-flow-track">
            <div className="obs-flow-meta">
              <span style={{ color: '#34D399', fontWeight: 700 }}>M &bull; METRICS</span>
              <span>2s Periodic Flush</span>
            </div>
            <div className="obs-flow-line-wrap">
              <div className="obs-flow-line flow-line-m" />
              <span className="flow-arrow flow-m">-&gt;</span>
            </div>
          </div>

          {/* Trace Track (Indigo) */}
          <div className="obs-flow-track">
            <div className="obs-flow-meta">
              <span style={{ color: '#818CF8', fontWeight: 700 }}>T &bull; TRACES</span>
              <span>W3C traceparent</span>
            </div>
            <div className="obs-flow-line-wrap">
              <div className="obs-flow-line flow-line-t" />
              <span className="flow-arrow flow-t">-&gt;</span>
            </div>
          </div>
        </div>

        {/* Column 2: Central LGTM Box with Inner Boxes */}
        <div className="obs-central-group">
          <div className="obs-central-header">
            <div className="obs-central-title-group">
              <span className="obs-central-title">GRAFANA OTEL-LGTM</span>
              <span className="indicator-dot active" />
            </div>
            <span className="badge badge-published" style={{ fontSize: '0.625rem' }}>:3000</span>
          </div>
          <div className="obs-central-subtitle">All-in-One Local Container (Collector, Tempo, Loki, Prometheus, Grafana)</div>

          {/* Inner Box 1: Gateway Card */}
          <div className="obs-lgtm-gateway-card">
            <div className="obs-gateway-header">
              <span className="obs-gateway-title">OpenTelemetry Collector Gateway</span>
              <span className="badge badge-received" style={{ fontSize: '0.5625rem' }}>:4317 / :4318</span>
            </div>
            <span className="obs-gateway-desc">Ingests OTLP Traces &amp; Metrics -&gt; Pipelines -&gt; Dispatches to Tempo (:3200) &amp; Prometheus (:9090)</span>
          </div>

          {/* Inner Boxes: 2x2 LGTM Engine Grid */}
          <div className="obs-lgtm-grid-2x2">
            {/* Inner Box L: Loki */}
            <div className="obs-engine-card engine-loki">
              <div className="obs-engine-header">
                <span className="obs-engine-title">L &bull; Loki Engine</span>
                <span className="obs-engine-pill">:3100</span>
              </div>
              <div className="obs-engine-body">
                <div className="obs-engine-item"><span style={{ color: '#FBBF24', fontWeight: 700 }}>LOGS:</span> {'{service="nats"}'}</div>
                <div className="obs-engine-item"><span style={{ color: '#FBBF24', fontWeight: 700 }}>EVENTS:</span> {'{service="nats-events"}'}</div>
                <div className="obs-engine-item">LogQL stream filtering &amp; parsing</div>
              </div>
            </div>

            {/* Inner Box G: Grafana */}
            <div className="obs-engine-card engine-grafana">
              <div className="obs-engine-header">
                <span className="obs-engine-title">G &bull; Grafana UI</span>
                <span className="obs-engine-pill">:3000</span>
              </div>
              <div className="obs-engine-body">
                <div className="obs-engine-item"><span style={{ color: '#C084FC', fontWeight: 700 }}>DASHBOARD:</span> LGTM Top 20</div>
                <div className="obs-engine-item"><span style={{ color: '#C084FC', fontWeight: 700 }}>EXPLORE:</span> Trace, Log, Metric</div>
                <div className="obs-engine-item">Trace-to-Log drilldown</div>
              </div>
            </div>

            {/* Inner Box T: Tempo */}
            <div className="obs-engine-card engine-tempo">
              <div className="obs-engine-header">
                <span className="obs-engine-title">T &bull; Tempo Traces</span>
                <span className="obs-engine-pill">:3200</span>
              </div>
              <div className="obs-engine-body">
                <div className="obs-engine-item"><span style={{ color: '#818cf8', fontWeight: 700 }}>SPANS:</span> W3C Waterfall Trees</div>
                <div className="obs-engine-item"><span style={{ color: '#818cf8', fontWeight: 700 }}>QUERY:</span> TraceQL by Trace ID</div>
                <div className="obs-engine-item">End-to-End HTTP -&gt; NATS -&gt; Worker</div>
              </div>
            </div>

            {/* Inner Box M: Prometheus */}
            <div className="obs-engine-card engine-prom">
              <div className="obs-engine-header">
                <span className="obs-engine-title">M &bull; Prometheus</span>
                <span className="obs-engine-pill">:9090</span>
              </div>
              <div className="obs-engine-body">
                <div className="obs-engine-item"><span style={{ color: '#34D399', fontWeight: 700 }}>INFRA:</span> Scrapes Exporter (:7777)</div>
                <div className="obs-engine-item"><span style={{ color: '#34D399', fontWeight: 700 }}>APP:</span> Ingests OTLP Metrics</div>
                <div className="obs-engine-item">PromQL rate &amp; capacity eval</div>
              </div>
            </div>
          </div>
        </div>

        {/* Connector 2: Coloured Infra Pipeline Bridge */}
        <div className="obs-conduit-card">
          <div className="obs-conduit-header">
            <span className="obs-conduit-title">INFRA PIPELINE</span>
            <span className="badge badge-published" style={{ fontSize: '0.5625rem' }}>LOKI/PROM</span>
          </div>

          {/* Metric Scrape Track (Green) */}
          <div className="obs-flow-track">
            <div className="obs-flow-meta">
              <span style={{ color: '#34D399', fontWeight: 700 }}>M &bull; SCRAPE</span>
              <span>GET :7777 (2s)</span>
            </div>
            <div className="obs-flow-line-wrap">
              <span className="flow-arrow flow-m">&lt;-</span>
              <div className="obs-flow-line flow-line-m" />
            </div>
          </div>

          {/* Log Track (Amber) */}
          <div className="obs-flow-track">
            <div className="obs-flow-meta">
              <span style={{ color: '#FBBF24', fontWeight: 700 }}>L &bull; LOGS</span>
              <span>Fluent Bit Tail</span>
            </div>
            <div className="obs-flow-line-wrap">
              <span className="flow-arrow flow-l">&lt;-</span>
              <div className="obs-flow-line flow-line-l" />
            </div>
          </div>

          {/* Event Track (Magenta) */}
          <div className="obs-flow-track">
            <div className="obs-flow-meta">
              <span style={{ color: '#F0ABFC', fontWeight: 700 }}>E &bull; EVENTS</span>
              <span>AdvisoryListener</span>
            </div>
            <div className="obs-flow-line-wrap">
              <span className="flow-arrow flow-e">&lt;-</span>
              <div className="obs-flow-line flow-line-e" />
            </div>
          </div>
        </div>

        {/* Column 3: Source - NATS Infrastructure */}
        <div className="obs-source-group">
          <div className="obs-group-title">SOURCE: NATS INFRA</div>

          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">NATS Server</span>
              <span className="badge badge-received" style={{ fontSize: '0.625rem' }}>:8222</span>
            </div>
            <div className="obs-node-body">
              <div className="obs-node-item">
                <span className="obs-tag tag-proto">ENGINE</span>
                <span>Core NATS &bull; JetStream</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-log">LOGS</span>
                <span>Writes /data/nats.log</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-event">EVENTS</span>
                <span>$SYS &bull; $JS.EVENT.ADVISORY</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-metric">MONITOR</span>
                <span>/varz, /connz, /subz, /jsz</span>
              </div>
            </div>
          </div>

          <div className="obs-connector-down">|</div>

          <div className="obs-node-card">
            <div className="obs-node-header">
              <span className="obs-node-name">Exporter &amp; Daemons</span>
              <span className="badge badge-published" style={{ fontSize: '0.625rem' }}>AGENTS</span>
            </div>
            <div className="obs-node-body">
              <div className="obs-node-item">
                <span className="obs-tag tag-metric">EXPORTER</span>
                <span>nats-prometheus-exporter (:7777)</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-log">FORWARD</span>
                <span>Fluent Bit tailing log</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-event">LISTENER</span>
                <span>AdvisoryListener to Loki</span>
              </div>
              <div className="obs-node-item">
                <span className="obs-tag tag-app">FLAGS</span>
                <span>-varz, -connz, -jsz=all</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
