'use client';

import { Frame } from '@/lib/types';

interface ErrorsPanelProps {
  errors: Frame['errors'];
}

export function ErrorsPanel({ errors }: ErrorsPanelProps) {
  const total = errors.reduce((sum, err) => sum + err.count, 0);

  return (
    <div className="panel">
      <span className="badge badge--error">Errors</span>
      <h3 style={{ marginTop: '1rem' }}>Failure Types</h3>
      {total > 0 && (
        <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
          {total} total errors
        </p>
      )}

      <div className="error-list">
        {errors.length === 0 ? (
          <div className="empty-state">No errors</div>
        ) : (
          errors.map((err) => (
            <div className="error-item" key={err.class}>
              <span className="error-item__class">{err.class}</span>
              <span className="error-item__count">{err.count}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
