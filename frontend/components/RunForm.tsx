'use client';

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import type { FormEvent } from 'react';
import { fetchJSON } from '@/lib/api';
import type { SourceMode } from '@/lib/types';

const defaults = {
  mode: 'url' as SourceMode,
  input: '',
  max_depth: 2,
  max_pages: 50,
  time_budget_seconds: 180,
  max_links_per_page: 25,
  global_concurrency: 16,
  per_host_concurrency: 4,
  respect_robots: true,
};

export function RunForm() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [error, setError] = useState('');
  const [form, setForm] = useState({ ...defaults });

  const set = (key: string, value: string | number | boolean) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const created = await fetchJSON<{ id: string }>('/runs', {
        method: 'POST',
        body: JSON.stringify(form),
      });
      await fetchJSON(`/runs/${created.id}/start`, { method: 'POST' });
      startTransition(() => router.push(`/runs/${created.id}`));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start crawl');
    }
  };

  return (
    <form onSubmit={onSubmit}>
      <div className="form-title">Start a crawl</div>
      <div className="form-sub">Resolves a seed then crawls outward.</div>

      <div className="mode-tabs">
        <button
          type="button"
          className={`mode-tab${form.mode === 'url' ? ' active' : ''}`}
          onClick={() => set('mode', 'url')}
        >
          URL
        </button>
        <button
          type="button"
          className={`mode-tab${form.mode === 'keyword' ? ' active' : ''}`}
          onClick={() => set('mode', 'keyword')}
        >
          Keyword
        </button>
      </div>

      <div className="field-group">
        <label className="field-label" htmlFor="crawl-input">
          {form.mode === 'url' ? 'Seed URL' : 'Search query'}
        </label>
        <input
          id="crawl-input"
          className="field-input"
          required
          placeholder={
            form.mode === 'url'
              ? 'https://en.wikipedia.org/wiki/Web_crawler'
              : 'Alan Turing'
          }
          value={form.input}
          onChange={(e) => set('input', e.target.value)}
        />
      </div>

      <div className="num-grid">
        <div className="field-group">
          <label className="field-label" htmlFor="max-depth">Max depth</label>
          <input
            id="max-depth"
            className="field-input"
            type="number"
            min={1}
            max={10}
            value={form.max_depth}
            onChange={(e) => set('max_depth', Number(e.target.value))}
          />
        </div>
        <div className="field-group">
          <label className="field-label" htmlFor="max-pages">Max pages</label>
          <input
            id="max-pages"
            className="field-input"
            type="number"
            min={1}
            value={form.max_pages}
            onChange={(e) => set('max_pages', Number(e.target.value))}
          />
        </div>
        <div className="field-group">
          <label className="field-label" htmlFor="time-budget">Time budget (sec)</label>
          <input
            id="time-budget"
            className="field-input"
            type="number"
            min={10}
            value={form.time_budget_seconds}
            onChange={(e) => set('time_budget_seconds', Number(e.target.value))}
          />
        </div>
        <div className="field-group">
          <label className="field-label" htmlFor="links-per-page">Links per page</label>
          <input
            id="links-per-page"
            className="field-input"
            type="number"
            min={1}
            value={form.max_links_per_page}
            onChange={(e) => set('max_links_per_page', Number(e.target.value))}
          />
        </div>
      </div>

      <details className="advanced-section">
        <summary>
          <span className="adv-arrow">▸</span>
          Advanced crawl controls
        </summary>
        <div className="advanced-inner">
          <div className="num-grid">
            <div className="field-group">
              <label className="field-label" htmlFor="global-conc">Global concurrency</label>
              <input
                id="global-conc"
                className="field-input"
                type="number"
                min={1}
                value={form.global_concurrency}
                onChange={(e) => set('global_concurrency', Number(e.target.value))}
              />
            </div>
            <div className="field-group">
              <label className="field-label" htmlFor="host-conc">Per-host concurrency</label>
              <input
                id="host-conc"
                className="field-input"
                type="number"
                min={1}
                value={form.per_host_concurrency}
                onChange={(e) => set('per_host_concurrency', Number(e.target.value))}
              />
            </div>
          </div>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={form.respect_robots}
              onChange={(e) => set('respect_robots', e.target.checked)}
            />
            Respect robots.txt
          </label>
        </div>
      </details>

      {error && <div className="form-error">{error}</div>}

      <div className="form-actions">
        <button className="btn-primary" disabled={pending} type="submit">
          {pending ? 'Launching…' : 'Launch crawl →'}
        </button>
        <span className="form-note">
          Results page opens immediately and streams new pages live.
        </span>
      </div>
    </form>
  );
}
