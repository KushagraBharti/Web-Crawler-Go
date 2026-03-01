'use client';

import { Frame } from '@/lib/types';

interface ErrorsPanelProps {
  errors: Frame['errors'];
}

export function ErrorsPanel({ errors }: ErrorsPanelProps) {
  const total = errors.reduce((sum, err) => sum + err.count, 0);

  return (
    <div className="panel" role="region" aria-labelledby="errors-heading">
      <span className="badge badge--error">Errors</span>
      <h3 id="errors-heading" style={{ marginTop: '1rem' }}>Failure Types</h3>
      {total > 0 && (
        <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }} aria-live="polite">
          {total} total errors
        </p>
      )}

      <div className="error-list" aria-live="polite" aria-relevant="additions text">
        {errors.length === 0 ? (
          <div className="empty-state" role="status">No errors</div>
        ) : (
          errors.map((err) => (
            <div className="error-item" key={err.class}>
              <span className="error-item__class">{err.class}</span>
              <span className="error-item__count" aria-label={`${err.count} occurrences`}>{err.count}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
