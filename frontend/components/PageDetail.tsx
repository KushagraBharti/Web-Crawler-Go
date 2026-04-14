'use client';

import type { Page } from '@/lib/types';

export function PageDetail({ page }: { page?: Page | null }) {
  if (!page) {
    return (
      <section className="panel detail-panel">
        <div className="panel-header-block">
          <div className="eyebrow">Page detail</div>
          <h3>Select a crawled page</h3>
        </div>
        <div className="empty-panel">Choose a page from the left to read the extracted content.</div>
      </section>
    );
  }

  return (
    <section className="panel detail-panel">
      <div className="panel-header-block">
        <div className="eyebrow">Page detail</div>
        <h3>{page.title || page.url}</h3>
        <a className="detail-link" href={page.url} rel="noreferrer" target="_blank">
          {page.url}
        </a>
      </div>

      <div className="detail-stats">
        <div><span>Status</span><strong>{page.status_code || 'error'}</strong></div>
        <div><span>Depth</span><strong>{page.depth}</strong></div>
        <div><span>Latency</span><strong>{page.fetch_ms} ms</strong></div>
        <div><span>Links</span><strong>{page.outgoing_links.length}</strong></div>
      </div>

      {page.error_class ? (
        <div className="error-banner">
          {page.error_class}: {page.error_message || 'crawl failed'}
        </div>
      ) : null}

      <article className="detail-copy">
        {page.text ? page.text : 'No readable text was extracted for this page.'}
      </article>
    </section>
  );
}
