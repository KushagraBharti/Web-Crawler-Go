'use client';

import type { Page } from '@/lib/types';

export function PageSidebar({
  pages,
  selectedId,
  newPageIds,
  onSelect,
}: {
  pages: Page[];
  selectedId?: string;
  newPageIds: Set<string>;
  onSelect: (id: string) => void;
}) {
  return (
    <aside className="page-index">
      <div className="page-index__head">
        <div className="page-index__heading">Index</div>
        <div className="page-index__count">
          {pages.length} {pages.length === 1 ? 'page' : 'pages'}
        </div>
      </div>

      <div className="page-index__list">
        {pages.length === 0 ? (
          <div className="page-index__empty">Waiting for first page…</div>
        ) : (
          pages.map((page, i) => {
            const isActive = selectedId === page.id;
            const isNew = newPageIds.has(page.id);
            const hasError = Boolean(page.error_class);
            return (
              <button
                key={page.id}
                type="button"
                className={[
                  'page-entry',
                  isActive ? 'active' : '',
                  isNew ? 'page-entry--new' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
                onClick={() => onSelect(page.id)}
              >
                <div className="page-entry__row">
                  <span className="page-entry__num">
                    {String(i + 1).padStart(2, '0')}
                  </span>
                  <span className="page-entry__title">
                    {page.title || page.url}
                  </span>
                </div>
                {(page.excerpt || page.error_message) && (
                  <div className="page-entry__excerpt">
                    {page.excerpt || page.error_message}
                  </div>
                )}
                <div className="page-entry__meta">
                  <span className={`page-entry__tag${hasError ? ' page-entry__tag--err' : ''}`}>
                    {hasError ? page.error_class : `${page.status_code || '—'}`}
                  </span>
                  <span className="page-entry__tag">D{page.depth}</span>
                  {page.fetch_ms > 0 && (
                    <span className="page-entry__tag">{page.fetch_ms}ms</span>
                  )}
                </div>
              </button>
            );
          })
        )}
      </div>
    </aside>
  );
}
