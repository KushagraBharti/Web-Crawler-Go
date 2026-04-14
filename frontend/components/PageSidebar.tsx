'use client';

import type { Page } from '@/lib/types';

export function PageSidebar({
  pages,
  selectedId,
  onSelect
}: {
  pages: Page[];
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  return (
    <section className="panel sidebar-panel">
      <div className="panel-header-block">
        <div className="eyebrow">Pages</div>
        <h3>Readable results</h3>
      </div>
      {pages.length === 0 ? (
        <div className="empty-panel">Waiting for the first fetched page.</div>
      ) : (
        <div className="page-list">
          {pages.map((page) => (
            <button
              className={selectedId === page.id ? 'page-list__item active' : 'page-list__item'}
              key={page.id}
              onClick={() => onSelect(page.id)}
              type="button"
            >
              <div className="page-list__title">{page.title || page.url}</div>
              <div className="page-list__meta">
                <span>{page.status_code || 'ERR'}</span>
                <span>Depth {page.depth}</span>
              </div>
              <p>{page.excerpt || page.error_message || 'No readable excerpt yet.'}</p>
            </button>
          ))}
        </div>
      )}
    </section>
  );
}
