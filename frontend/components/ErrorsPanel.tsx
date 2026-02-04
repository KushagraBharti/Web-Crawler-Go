'use client';

import { Frame } from '@/lib/types';

export function ErrorsPanel({ errors }: { errors: Frame['errors'] }) {
  return (
    <div className="panel">
      <div className="badge">Errors</div>
      <h2 style={{ marginTop: 12 }}>Failure taxonomy</h2>
      <div className="error-list">
        {errors.length === 0 ? (
          <p>No errors reported yet.</p>
        ) : (
          errors.map((err) => (
            <div className="error-item" key={err.class}>
              <span>{err.class}</span>
              <strong>{err.count}</strong>
            </div>
          ))
        )}
      </div>
    </div>
  );
}