'use client';

import { RunSummary } from '@/lib/types';

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
}

export function RunSummaryPanel({ status, stopReason, summary }: RunSummaryPanelProps) {
  const reasonKey = stopReason || (status === 'running' ? 'running' : 'unknown');
  const reason = REASON_COPY[reasonKey] ?? REASON_COPY.unknown;
  const lastFetched = summary?.last_fetched_at
    ? new Date(summary.last_fetched_at).toLocaleTimeString()
    : '—';

  return (
    <div className="panel span-12">
      <span className="badge badge--accent">Run Summary</span>
      <h3 style={{ marginTop: '1rem' }}>What happened</h3>
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
          <span className="summary-card__value">{status || '—'}</span>
          <span className="summary-card__hint">Current state</span>
        </div>
      </div>
    </div>
  );
}
