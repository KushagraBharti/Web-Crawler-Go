'use client';

import { useMemo } from 'react';
import { Frame } from '@/lib/types';

const personalities = {
  flaky: { title: 'Flaky', hint: 'High error rate', badge: 'personality-card__badge--flaky' },
  slow: { title: 'Slow', hint: 'High latency', badge: 'personality-card__badge--slow' },
  busy: { title: 'Busy', hint: 'High load', badge: 'personality-card__badge--busy' },
  steady: { title: 'Steady', hint: 'Performing well', badge: 'personality-card__badge--steady' }
} as const;

type PersonalityKey = keyof typeof personalities;

interface PersonalityCardsProps {
  hosts: Frame['hosts'];
}

export function PersonalityCards({ hosts }: PersonalityCardsProps) {
  const picks = useMemo(() => {
    return hosts
      .map((host) => {
        let key: PersonalityKey = 'steady';
        if (host.error_rate > 0.2) key = 'flaky';
        else if (host.p95_ms > 1500) key = 'slow';
        else if (host.inflight >= 3) key = 'busy';

        const score = host.error_rate * 3 + host.p95_ms / 2000 + host.inflight / 5;
        return { host, personality: personalities[key], score };
      })
      .sort((a, b) => b.score - a.score)
      .slice(0, 3);
  }, [hosts]);

  return (
    <section className="panel">
      <span className="badge">Analysis</span>
      <h3 style={{ marginTop: '1rem' }}>Notable Hosts</h3>
      <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
        Top 3 hosts by behavior score
      </p>

      <div className="personality-grid">
        {picks.length === 0 ? (
          <div className="empty-state" style={{ gridColumn: 'span 3' }}>
            No host data yet
          </div>
        ) : (
          picks.map(({ host, personality }) => (
            <div className="personality-card" key={host.host}>
              <span className={`personality-card__badge ${personality.badge}`}>
                {personality.title}
              </span>
              <h4 className="personality-card__host">{host.host}</h4>
              <p className="personality-card__hint">{personality.hint}</p>
              <div className="personality-card__metrics">
                <span>P95: {host.p95_ms} ms</span>
                <span>Errors: {(host.error_rate * 100).toFixed(1)}%</span>
                <span>Reuse: {(host.reuse_rate * 100).toFixed(0)}%</span>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
