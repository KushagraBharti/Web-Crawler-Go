'use client';

import { API_BASE } from '@/lib/api';
import { PageRow } from '@/lib/types';

const formatBytes = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  if (mb < 1024) return `${mb.toFixed(1)} MB`;
  const gb = mb / 1024;
  return `${gb.toFixed(1)} GB`;
};

interface PagesTableProps {
  pages: PageRow[];
  runId: string;
}

export function PagesTable({ pages, runId }: PagesTableProps) {
  const jsonUrl = `${API_BASE}/runs/${runId}/pages?limit=200`;
  return (
    <div className="panel span-12">
      <span className="badge">Data Collected</span>
      <h3 style={{ marginTop: '1rem' }}>Latest pages</h3>
      <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
        This crawler stores metadata only (URL, status, timings, size). HTML bodies are not saved in v1.
      </p>
      <div className="pages-table__actions">
        <a className="pages-table__link" href={jsonUrl} target="_blank" rel="noreferrer">
          Open JSON feed
        </a>
        <span>Showing the latest 50 pages</span>
      </div>

      {pages.length === 0 ? (
        <div className="empty-state">No pages collected yet.</div>
      ) : (
        <div className="pages-table">
          <div className="pages-table__header">
            <span>URL</span>
            <span>Status</span>
            <span>Type</span>
            <span>Size</span>
            <span>Depth</span>
            <span>Latency</span>
            <span>Error</span>
          </div>
          {pages.map((page, index) => {
            const statusLabel = page.status_code ? page.status_code.toString() : '—';
            const latency = page.fetch_ms ? `${page.fetch_ms} ms` : '—';
            const size = page.size_bytes ? formatBytes(page.size_bytes) : '—';
            const error = page.error_class ? page.error_class : '—';
            return (
              <div className="pages-table__row" key={`${page.url}-${index}`}>
                <a className="pages-table__url" href={page.url} target="_blank" rel="noreferrer">
                  {page.url}
                </a>
                <span>{statusLabel}</span>
                <span>{page.content_type || '—'}</span>
                <span>{size}</span>
                <span>{page.depth}</span>
                <span>{latency}</span>
                <span className={page.error_class ? 'pages-table__error' : ''}>{error}</span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
