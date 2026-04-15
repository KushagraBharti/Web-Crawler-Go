'use client';

import type { Page } from '@/lib/types';

export function PageDetail({
  page,
  loading,
  error,
}: {
  page?: Page | null;
  loading?: boolean;
  error?: string | null;
}) {
  if (loading) {
    return (
      <div className="article-pane">
        <div className="article-loading">
          <div className="skeleton skeleton--title" />
          <div className="skeleton skeleton--meta" />
          <div className="skeleton skeleton--rule" />
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className={`skeleton ${i % 3 === 2 ? 'skeleton--line-short' : 'skeleton--line'}`}
              style={{ animationDelay: `${i * 0.08}s` }}
            />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="article-pane">
        <div className="article-error-banner">{error}</div>
      </div>
    );
  }

  if (!page) {
    return (
      <div className="article-pane">
        <div className="article-empty">
          <div className="article-empty__icon">↗</div>
          <span className="article-empty__text">Select a page to read</span>
        </div>
      </div>
    );
  }

  return (
    <div className="article-pane">
      {/* Header */}
      <span className="article-section-label">
        {page.source_mode === 'keyword' ? page.source_input : page.host}
      </span>

      <h2 className="article-title">{page.title || page.url}</h2>

      <div className="article-meta-row">
        <span className="article-meta-item">
          <a href={page.url} rel="noreferrer" target="_blank">
            {page.url}
          </a>
        </span>
        <span className="article-meta-item">Status {page.status_code || '—'}</span>
        <span className="article-meta-item">Depth {page.depth}</span>
        {page.fetch_ms > 0 && (
          <span className="article-meta-item">{page.fetch_ms}ms</span>
        )}
        {page.size_bytes > 0 && (
          <span className="article-meta-item">
            {(page.size_bytes / 1024).toFixed(1)}kb
          </span>
        )}
        {page.outgoing_links.length > 0 && (
          <span className="article-meta-item">
            {page.outgoing_links.length} outgoing links
          </span>
        )}
      </div>

      {/* Error state */}
      {page.error_class && (
        <div className="article-error-banner">
          <strong>{page.error_class}</strong>
          {page.error_message ? `: ${page.error_message}` : ''}
        </div>
      )}

      {/* Body */}
      {page.text ? (
        <div className="article-body">{page.text}</div>
      ) : (
        <div className="article-body-empty">
          No readable text was extracted for this page.
        </div>
      )}
    </div>
  );
}
