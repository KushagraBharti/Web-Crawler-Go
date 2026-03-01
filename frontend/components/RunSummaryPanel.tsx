'use client';

import { Frame, RunSummary } from '@/lib/types';

type ReasonCopy = { title: string; detail: string };

const REASON_COPY: Record<string, ReasonCopy> = {
  running: {
    title: 'Still running',
    detail: 'The crawler is active and still discovering pages.',
  },
  manual: {
    title: 'Stopped manually',
    detail: 'A user stopped the run from the dashboard or API.',
  },
  max_pages: {
    title: 'Page limit reached',
    detail: 'The run stopped after hitting the max pages limit.',
  },
  time_budget: {
    title: 'Time budget reached',
    detail: 'The run stopped after the time budget elapsed.',
  },
  unknown: {
    title: 'Stopped (unknown reason)',
    detail: 'The stop cause was not recorded.',
  },
};

const formatNumber = (value: number | undefined) =>
  (value ?? 0).toLocaleString();

const formatBytes = (bytes: number | undefined) => {
  const total = bytes ?? 0;
  if (total < 1024) return `${total} B`;
  const kb = total / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  if (mb < 1024) return `${mb.toFixed(1)} MB`;
  const gb = mb / 1024;
  return `${gb.toFixed(1)} GB`;
};

interface RunSummaryPanelProps {
  status: string;
  stopReason: string;
  summary: RunSummary | null;
  errors?: Frame['errors'];
}

export function RunSummaryPanel({ status, stopReason, summary, errors }: RunSummaryPanelProps) {
  const reasonKey = stopReason || (status === 'running' ? 'running' : 'unknown');
  const reason = REASON_COPY[reasonKey] ?? REASON_COPY.unknown;
  const lastFetched = summary?.last_fetched_at
    ? new Date(summary.last_fetched_at).toLocaleTimeString()
    : '\u2014';

  const isLoading = summary === null && !status;
  const totalErrors = errors?.reduce((sum, err) => sum + err.count, 0) ?? 0;

  return (
    <div className="panel span-12" role="region" aria-labelledby="summary-heading" aria-live="polite">
      <span className="badge badge--accent">Run Summary</span>
      <h3 id="summary-heading" style={{ marginTop: '1rem' }}>What happened</h3>

      {isLoading ? (
        <div role="status" aria-label="Loading run summary">
          <div className="skeleton skeleton--text" style={{ marginTop: '1rem' }} />
          <div className="summary-grid" style={{ marginTop: '1.5rem' }}>
            {Array.from({ length: 6 }).map((_, i) => (
              <div className="skeleton skeleton--card" key={i} />
            ))}
          </div>
        </div>
      ) : (
        <>
          <div className="summary-callout">
            <div className="summary-callout__title">{reason.title}</div>
            <p className="summary-callout__detail">{reason.detail}</p>
          </div>

          <div className="summary-grid">
            <div className="summary-card">
              <span className="summary-card__label">Pages fetched</span>
              <span className="summary-card__value">{formatNumber(summary?.pages_fetched)}</span>
              <span className="summary-card__hint">Successful responses</span>
            </div>
            <div className="summary-card">
              <span className="summary-card__label">Pages failed</span>
              <span className="summary-card__value">{formatNumber(summary?.pages_failed)}</span>
              <span className="summary-card__hint">Errors or blocked pages</span>
            </div>
            <div className="summary-card">
              <span className="summary-card__label">Unique hosts</span>
              <span className="summary-card__value">{formatNumber(summary?.unique_hosts)}</span>
              <span className="summary-card__hint">Distinct domains reached</span>
            </div>
            <div className="summary-card">
              <span className="summary-card__label">Data downloaded</span>
              <span className="summary-card__value">{formatBytes(summary?.total_bytes)}</span>
              <span className="summary-card__hint">Total response bytes</span>
            </div>
            <div className="summary-card">
              <span className="summary-card__label">Last page fetched</span>
              <span className="summary-card__value">{lastFetched}</span>
              <span className="summary-card__hint">Most recent fetch time</span>
            </div>
            <div className="summary-card">
              <span className="summary-card__label">Run status</span>
              <span className="summary-card__value">{status || '\u2014'}</span>
              <span className="summary-card__hint">Current state</span>
            </div>
          </div>

          {errors && errors.length > 0 && (
            <div className={`health-summary health-summary--${totalErrors > 10 ? 'error' : 'warn'}`} aria-live="polite">
              <span className="badge badge--error" style={{ fontSize: '0.6875rem' }}>
                {errors.length} error {errors.length === 1 ? 'type' : 'types'}
              </span>
              <span>{totalErrors} total errors</span>
              <span style={{ marginLeft: 'auto', fontSize: '0.75rem', color: 'var(--text-tertiary)' }}>
                Top: {errors[0]?.class} ({errors[0]?.count})
              </span>
            </div>
          )}
        </>
      )}
    </div>
  );
}
