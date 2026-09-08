import React, { useState } from 'react';
import { JetStreamInfo, ConsumerStatus } from '../../api/demoApi';

interface CoreFlowNatsCliProps {
  jetstreamInfo?: JetStreamInfo | null;
  consumerStatus?: ConsumerStatus | null;
  onRefresh: () => void;
  onResetStream?: () => Promise<void>;
  isResettingStream?: boolean;
  onShowInfo?: (key: string) => void;
}

interface SetupSnippet {
  id: string;
  title: string;
  scope: string;
  badgeColor: string;
  badgeBg: string;
  cmd: string;
}

interface CliCommandSnippet {
  id: string;
  title: string;
  cmd: string;
}

const SETUP_SNIPPETS: SetupSnippet[] = [
  {
    id: 'setup-local-app',
    title: 'Application Context',
    scope: 'local-app',
    badgeColor: '#60A5FA',
    badgeBg: 'rgba(59, 130, 246, 0.15)',
    cmd: 'export NATS_CONTEXT=local-app',
  },
  {
    id: 'setup-sys-admin',
    title: 'System Admin Context',
    scope: 'sys-admin',
    badgeColor: '#F87171',
    badgeBg: 'rgba(239, 68, 68, 0.15)',
    cmd: 'export NATS_CONTEXT=sys-admin',
  },
];

const OPERATIONAL_COMMANDS: CliCommandSnippet[] = [
  {
    id: 'view-streams',
    title: 'View Streams',
    cmd: 'nats stream ls',
  },
  {
    id: 'view-stream-info',
    title: 'View Stream Info (JOBS)',
    cmd: 'nats stream info JOBS',
  },
  {
    id: 'view-stream-state',
    title: 'Stream State (First & Last Sequence)',
    cmd: 'nats stream state JOBS',
  },
  {
    id: 'view-stream-messages',
    title: 'Stream View - Messages in JOBS',
    cmd: 'nats stream view JOBS 5',
  },
  {
    id: 'view-consumers',
    title: 'View Durable Consumers',
    cmd: 'nats consumer ls JOBS',
  },
  {
    id: 'view-consumer-data',
    title: 'Consumer State, Cursor & ACK',
    cmd: 'nats consumer info JOBS job-processor',
  },
  {
    id: 'view-consumer-report',
    title: 'Consumer Processing Report & Backlog',
    cmd: 'nats consumer report JOBS',
  },
  {
    id: 'reset-stream-cli',
    title: 'Delete Stream & Reset Cursors (CLI)',
    cmd: 'nats stream rm JOBS -f',
  },
];

export const CoreFlowNatsCli: React.FC<CoreFlowNatsCliProps> = ({
  jetstreamInfo,
  consumerStatus,
  onRefresh,
  onResetStream,
  isResettingStream,
  onShowInfo,
}) => {
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const copyToClipboard = (cmd: string, id: string) => {
    navigator.clipboard.writeText(cmd);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <div className="panel" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Panel Header */}
      <div className="panel-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span style={{ 
            background: 'rgba(16, 185, 129, 0.15)', 
            color: '#34D399', 
            borderRadius: '4px', 
            padding: '2px 8px', 
            fontSize: '0.75rem', 
            fontWeight: 700,
            letterSpacing: '0.05em' 
          }}>
            STAGE 2
          </span>
          <h2 className="panel-title" style={{ margin: 0 }}>NATS View &amp; CLI</h2>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
          {onResetStream && (
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onResetStream}
              disabled={isResettingStream}
              style={{
                fontSize: '0.68rem',
                padding: '0.15rem 0.5rem',
                color: '#F87171',
                borderColor: 'rgba(239, 68, 68, 0.25)',
                background: 'rgba(239, 68, 68, 0.08)',
                cursor: isResettingStream ? 'not-allowed' : 'pointer',
              }}
              title="Delete and recreate JOBS stream via job-service (:8081). Resets Seq to 1 & AckFloor to 0."
            >
              {isResettingStream ? 'Recreating...' : 'Recreate Stream'}
            </button>
          )}
          {onShowInfo && (
            <button
              type="button"
              className="node-info-btn"
              onClick={() => onShowInfo('jetstream-engine')}
              title="Learn about NATS Server &amp; JetStream"
            >
              (i)
            </button>
          )}
        </div>
      </div>

      {/* Main Content Area */}
      <div style={{ 
        flex: 1, 
        overflowY: 'auto', 
        padding: '0.75rem 1rem 1rem 1rem', 
        display: 'flex', 
        flexDirection: 'column', 
        gap: '0.85rem' 
      }}>
        {/* Section 1: Setup Phase */}
        <div>
          <div style={{ 
            fontSize: '0.68rem', 
            textTransform: 'uppercase', 
            letterSpacing: '0.05em', 
            color: '#34D399', 
            fontWeight: 700,
            marginBottom: '0.35rem' 
          }}>
            1. Setup Phase (Export Active Context):
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
            {SETUP_SNIPPETS.map((snip) => (
              <div
                key={snip.id}
                style={{
                  background: 'var(--bg-secondary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '4px',
                  padding: '0.45rem 0.6rem',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '0.25rem',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span style={{ fontSize: '0.76rem', fontWeight: 600, color: 'var(--text-bright)' }}>
                    {snip.title}
                  </span>
                  <span style={{ 
                    fontSize: '0.62rem', 
                    padding: '1px 5px', 
                    borderRadius: '3px',
                    background: snip.badgeBg,
                    color: snip.badgeColor,
                    fontFamily: 'monospace'
                  }}>
                    {snip.scope}
                  </span>
                </div>

                <div style={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  background: '#0D1117', 
                  borderRadius: '3px', 
                  border: '1px solid #30363D', 
                  padding: '0.25rem 0.4rem', 
                  gap: '0.5rem', 
                  fontFamily: 'monospace', 
                  fontSize: '0.72rem' 
                }}>
                  <span style={{ flex: 1, color: '#34D399', fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {snip.cmd}
                  </span>
                  <button
                    type="button"
                    className="btn-secondary"
                    onClick={() => copyToClipboard(snip.cmd, snip.id)}
                    style={{ 
                      fontSize: '0.65rem', 
                      padding: '1px 6px', 
                      minWidth: '50px',
                      background: copiedId === snip.id ? 'rgba(16, 185, 129, 0.2)' : undefined,
                      borderColor: copiedId === snip.id ? '#10B981' : undefined,
                      color: copiedId === snip.id ? '#34D399' : undefined
                    }}
                  >
                    {copiedId === snip.id ? 'Copied!' : 'Copy'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Section 2: Handy Commands (Without --context) */}
        <div>
          <div style={{ 
            fontSize: '0.68rem', 
            textTransform: 'uppercase', 
            letterSpacing: '0.05em', 
            color: '#60A5FA', 
            fontWeight: 700,
            marginBottom: '0.35rem' 
          }}>
            2. Operational Commands (No --context Required):
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
            {OPERATIONAL_COMMANDS.map((snip) => (
              <div
                key={snip.id}
                style={{
                  background: 'var(--bg-secondary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '4px',
                  padding: '0.45rem 0.6rem',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '0.25rem',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span style={{ fontSize: '0.76rem', fontWeight: 600, color: 'var(--text-bright)' }}>
                    {snip.title}
                  </span>
                </div>

                <div style={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  background: '#0D1117', 
                  borderRadius: '3px', 
                  border: '1px solid #30363D', 
                  padding: '0.25rem 0.4rem', 
                  gap: '0.5rem', 
                  fontFamily: 'monospace', 
                  fontSize: '0.72rem' 
                }}>
                  <span style={{ flex: 1, color: '#E6EDF3', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {snip.cmd}
                  </span>
                  <button
                    type="button"
                    className="btn-secondary"
                    onClick={() => copyToClipboard(snip.cmd, snip.id)}
                    style={{ 
                      fontSize: '0.65rem', 
                      padding: '1px 6px', 
                      minWidth: '50px',
                      background: copiedId === snip.id ? 'rgba(16, 185, 129, 0.2)' : undefined,
                      borderColor: copiedId === snip.id ? '#10B981' : undefined,
                      color: copiedId === snip.id ? '#34D399' : undefined
                    }}
                  >
                    {copiedId === snip.id ? 'Copied!' : 'Copy'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
