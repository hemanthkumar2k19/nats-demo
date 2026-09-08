import React, { useEffect } from 'react';
import { NatsComponentInfo } from '../content/natsInfo';

interface InfoPopoverProps {
  info: NatsComponentInfo | null;
  onClose: () => void;
}

export const InfoPopover: React.FC<InfoPopoverProps> = ({ info, onClose }) => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  if (!info) return null;

  return (
    <div className="info-modal-backdrop" onClick={onClose}>
      <div className="info-modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="info-modal-header">
          <div>
            <span className="badge badge-nats" style={{ marginBottom: '0.25rem', display: 'inline-block' }}>
              NATS ARCHITECTURE
            </span>
            <h3 className="info-modal-title">{info.title}</h3>
            <div className="info-modal-role">{info.role}</div>
          </div>
          <button className="info-modal-close-btn" onClick={onClose} title="Close (Esc)">
            [ x ]
          </button>
        </div>

        <div className="info-modal-body">
          {info.diagram && (
            <div className="info-section">
              <h4 className="info-section-heading" style={{ color: '#34D399' }}>
                Delivery Sequence &amp; Cursor State Model
              </h4>
              <pre style={{
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                borderRadius: '6px',
                padding: '0.75rem 1rem',
                fontFamily: 'monospace',
                fontSize: '0.78rem',
                color: '#34D399',
                overflowX: 'auto',
                lineHeight: 1.45,
                margin: 0,
              }}>
                {info.diagram}
              </pre>
            </div>
          )}

          {info.qna && info.qna.length > 0 && (
            <div className="info-section">
              <h4 className="info-section-heading" style={{ color: '#60A5FA' }}>
                Core Evaluation Questions &amp; Answers
              </h4>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.65rem' }}>
                {info.qna.map((item, idx) => (
                  <div key={idx} style={{
                    background: 'var(--bg-secondary)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '6px',
                    padding: '0.65rem 0.85rem',
                  }}>
                    <div style={{ fontWeight: 600, fontSize: '0.82rem', color: 'var(--text-bright)', marginBottom: '0.25rem' }}>
                      {item.question}
                    </div>
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)', lineHeight: 1.4 }}>
                      {item.answer}
                    </div>
                    {item.formula && (
                      <div style={{ marginTop: '0.35rem', fontFamily: 'monospace', fontSize: '0.74rem', color: '#60A5FA', background: 'rgba(59, 130, 246, 0.08)', padding: '2px 6px', borderRadius: '4px', display: 'inline-block' }}>
                        Formula: {item.formula}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="info-section">
            <h4 className="info-section-heading">NATS Concepts</h4>
            <ul className="info-concepts-list">
              {info.concepts.map((concept, idx) => (
                <li key={idx} className="info-concept-item">
                  <span className="info-bullet">+</span>
                  <span>{concept}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="info-section">
            <h4 className="info-section-heading">How This Demo Uses It</h4>
            <p className="info-text">{info.demoUsage}</p>
          </div>

          {info.trivia && (
            <div className="info-section info-trivia-box">
              <h4 className="info-section-heading" style={{ color: 'var(--accent-cyan)' }}>
                NATS Insight / Trivia
              </h4>
              <p className="info-text" style={{ fontStyle: 'italic', margin: 0 }}>
                {info.trivia}
              </p>
            </div>
          )}
        </div>

        <div className="info-modal-footer">
          <span style={{ fontSize: '0.75rem', color: 'var(--text-dim)' }}>
            Press Esc or click outside to dismiss
          </span>
          <button className="btn btn-secondary" style={{ padding: '0.35rem 0.75rem', fontSize: '0.8rem' }} onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
