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
    <div className="panel span-12" aria-labelledby="pages-table-heading">
      <span className="badge">Data Collected</span>
      <h3 id="pages-table-heading" style={{ marginTop: '1rem' }}>Latest pages</h3>
      <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
        This crawler stores metadata only (URL, status, timings, size). HTML bodies are not saved in v1.
      </p>
      <div className="pages-table__actions">
        <a
          className="pages-table__link"
          href={jsonUrl}
          target="_blank"
          rel="noreferrer"
          aria-label="Open JSON feed of crawled pages in new tab"
        >
          Open JSON feed
        </a>
        <span>Showing the latest 50 pages</span>
      </div>

      {pages.length === 0 ? (
        <div className="empty-state" role="status">
          <div className="skeleton skeleton--table-row" />
          <div className="skeleton skeleton--table-row" />
          <div className="skeleton skeleton--table-row" />
          <p style={{ marginTop: '1rem' }}>No pages collected yet.</p>
        </div>
      ) : (
        <div className="table-scroll" role="region" aria-label="Crawled pages table" tabIndex={0}>
          <table className="semantic-table" aria-labelledby="pages-table-heading">
            <thead>
              <tr>
                <th scope="col">URL</th>
                <th scope="col">Status</th>
                <th scope="col">Type</th>
                <th scope="col">Size</th>
                <th scope="col">Depth</th>
                <th scope="col">Latency</th>
                <th scope="col">Error</th>
              </tr>
            </thead>
            <tbody>
              {pages.map((page, index) => (
                <tr key={`${page.url}-${index}`}>
                  <td>
                    <a
                      href={page.url}
                      target="_blank"
                      rel="noreferrer"
                      aria-label={`Open page in new tab`}
                      style={{ color: 'var(--text)', fontWeight: 500, wordBreak: 'break-all' }}
                    >
                      {page.url}
                    </a>
                  </td>
                  <td>{page.status_code ? page.status_code.toString() : '\u2014'}</td>
                  <td>{page.content_type || '\u2014'}</td>
                  <td>{page.size_bytes ? formatBytes(page.size_bytes) : '\u2014'}</td>
                  <td>{page.depth}</td>
                  <td>{page.fetch_ms ? `${page.fetch_ms} ms` : '\u2014'}</td>
                  <td className={page.error_class ? 'cell--danger' : ''}>
                    {page.error_class || '\u2014'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
