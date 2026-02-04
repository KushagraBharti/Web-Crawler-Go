'use client';

import { useMemo } from 'react';
import { Frame } from '@/lib/types';

const labels = [
  { key: 'flaky', title: 'Flaky', hint: 'high error rates', color: 'var(--accent)' },
  { key: 'slow', title: 'Slow Responder', hint: 'p95 latency spikes', color: 'var(--accent-3)' },
  { key: 'busy', title: 'Busy Host', hint: 'high inflight load', color: 'var(--accent-2)' },
  { key: 'steady', title: 'Steady', hint: 'stable performance', color: 'var(--ink)' }
];

export function PersonalityCards({ hosts }: { hosts: Frame['hosts'] }) {
  const picks = useMemo(
    () =>
      hosts
        .map((host) => {
          let label = labels[3];
          if (host.error_rate > 0.2) label = labels[0];
          else if (host.p95_ms > 1500) label = labels[1];
          else if (host.inflight >= 3) label = labels[2];
          return { host, label, score: host.error_rate * 3 + host.p95_ms / 2000 + host.inflight / 5 };
        })
        .sort((a, b) => b.score - a.score)
        .slice(0, 3),
    [hosts]
  );

  return (
    <section className="panel">
      <div className="badge">Personality Cards</div>
      <h2 style={{ marginTop: 12 }}>Host personalities</h2>
      <div className="personality-grid">
        {picks.length === 0 ? (
          <p>No host telemetry yet.</p>
        ) : (
          picks.map(({ host, label }) => (
            <div className="personality-card" key={host.host}>
              <div className="personality-label" style={{ borderColor: label.color, color: label.color }}>
                {label.title}
              </div>
              <h3>{host.host}</h3>
              <p>{label.hint}</p>
              <div className="personality-metrics">
                <span>P95: {host.p95_ms} ms</span>
                <span>Errors: {Math.round(host.error_rate * 100)}%</span>
                <span>Reuse: {Math.round(host.reuse_rate * 100)}%</span>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
