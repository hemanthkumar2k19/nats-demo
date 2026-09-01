import React, { useEffect } from 'react';
import { NatsComponentInfo } from '../../data/natsInfoData';

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
              <p className="info-text" style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                {info.trivia}
              </p>
            </div>
          )}
        </div>

        <div className="info-modal-footer">
          <button className="btn btn-secondary btn-block" onClick={onClose} style={{ fontSize: '0.75rem' }}>
            Close Information
          </button>
        </div>
      </div>
    </div>
  );
};
