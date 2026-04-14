'use client';

import type { Diagnostics, RunSummary, SearchSeed } from '@/lib/types';

export function DiagnosticsPanel({
  summary,
  seed,
  diagnostics
}: {
  summary: RunSummary;
  seed: SearchSeed;
  diagnostics: Diagnostics;
}) {
  return (
    <details className="panel diagnostics-panel">
      <summary className="diagnostics-summary">
        <span className="eyebrow">Diagnostics</span>
        <span className="diagnostics-counts">
          {summary.pages_fetched} fetched &middot; {summary.pages_failed} failed &middot;{' '}
          {diagnostics.errors.length} errors &middot; {diagnostics.skipped_urls.length} skipped
        </span>
      </summary>

      <div className="summary-band">
        <div><span>Queued</span><strong>{summary.pages_queued}</strong></div>
        <div><span>Fetched</span><strong>{summary.pages_fetched}</strong></div>
        <div><span>Failed</span><strong>{summary.pages_failed}</strong></div>
        <div><span>Seed</span><strong>{seed.primary_url ? 'resolved' : 'pending'}</strong></div>
      </div>

      <div className="diagnostics-grid">
        <div>
          <h4>Artifacts</h4>
          <p className="diagnostics-copy">{diagnostics.artifact_dir}</p>
          <div className="artifact-list">
            {Object.entries(diagnostics.artifact_files).map(([key, value]) => (
              <div key={key}>
                <span>{key}</span>
                <code>{value}</code>
              </div>
            ))}
          </div>
        </div>
        <div>
          <h4>Event counts</h4>
          <div className="artifact-list">
            <div><span>Search results</span><strong>{seed.results.length}</strong></div>
            <div><span>Skipped URLs</span><strong>{diagnostics.skipped_urls.length}</strong></div>
            <div><span>Retries</span><strong>{diagnostics.retry_events.length}</strong></div>
            <div><span>Errors</span><strong>{diagnostics.errors.length}</strong></div>
          </div>
        </div>
      </div>
    </details>
  );
}
