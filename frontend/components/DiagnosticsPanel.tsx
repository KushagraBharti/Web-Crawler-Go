'use client';

import type { Diagnostics, RunSummary, SearchSeed } from '@/lib/types';

export function DiagnosticsPanel({
  summary,
  seed,
  diagnostics,
}: {
  summary: RunSummary;
  seed: SearchSeed;
  diagnostics: Diagnostics;
}) {
  return (
    <details className="diag-panel">
      <summary className="diag-summary">
        <span className="diag-summary__label">Diagnostics</span>
        <span className="diag-summary__counts">
          {summary.pages_fetched} fetched · {summary.pages_failed} failed ·{' '}
          {diagnostics.errors.length} errors · {diagnostics.skipped_urls.length} skipped
        </span>
      </summary>

      <div className="diag-body">
        {/* Stats column */}
        <div className="diag-col">
          <span className="diag-col__title">Run stats</span>
          <div className="diag-row">
            <span>Fetched</span>
            <strong>{summary.pages_fetched}</strong>
          </div>
          <div className="diag-row">
            <span>Failed</span>
            <strong>{summary.pages_failed}</strong>
          </div>
          <div className="diag-row">
            <span>Queued</span>
            <strong>{summary.pages_queued}</strong>
          </div>
          <div className="diag-row">
            <span>Skipped URLs</span>
            <strong>{diagnostics.skipped_urls.length}</strong>
          </div>
          <div className="diag-row">
            <span>Retries</span>
            <strong>{diagnostics.retry_events.length}</strong>
          </div>
          <div className="diag-row">
            <span>Errors</span>
            <strong>{diagnostics.errors.length}</strong>
          </div>
          {seed.results.length > 0 && (
            <div className="diag-row">
              <span>Search results</span>
              <strong>{seed.results.length}</strong>
            </div>
          )}
        </div>

        {/* Seed column */}
        <div className="diag-col">
          <span className="diag-col__title">Seed</span>
          {seed.primary_url && (
            <div className="diag-row" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: 4 }}>
              <span>Primary URL</span>
              <strong style={{ wordBreak: 'break-all', fontSize: '0.7rem' }}>
                {seed.primary_url}
              </strong>
            </div>
          )}
          {seed.query && (
            <div className="diag-row">
              <span>Query</span>
              <strong>{seed.query}</strong>
            </div>
          )}
        </div>

        {/* Artifacts column */}
        <div className="diag-col">
          <span className="diag-col__title">Artifacts</span>
          {diagnostics.artifact_dir && (
            <div className="diag-path">{diagnostics.artifact_dir}</div>
          )}
          {Object.entries(diagnostics.artifact_files ?? {}).map(([key, value]) => (
            <div key={key} className="diag-row" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: 2 }}>
              <span>{key}</span>
              <span className="diag-path">{value}</span>
            </div>
          ))}
        </div>
      </div>
    </details>
  );
}
