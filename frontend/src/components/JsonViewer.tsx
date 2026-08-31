import React from 'react';

interface JsonViewerProps {
  data: Record<string, any> | null;
  title?: string;
}

export const JsonViewer: React.FC<JsonViewerProps> = ({ data, title }) => {
  if (!data) return null;

  return (
    <div style={{ marginTop: '0.75rem' }}>
      {title && (
        <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '0.25rem' }}>
          {title}
        </div>
      )}
      <div className="json-viewer-container">
        <pre className="json-code">
          {JSON.stringify(data, null, 2)}
        </pre>
      </div>
    </div>
  );
};
