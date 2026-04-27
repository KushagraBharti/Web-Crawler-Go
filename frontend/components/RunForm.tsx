'use client';

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import type { FormEvent } from 'react';
import { fetchJSON, startRun } from '@/lib/api';
import type { CreatedRun, SourceMode } from '@/lib/types';

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
  const [createdRun, setCreatedRun] = useState<CreatedRun | null>(null);
  const [selectedSeed, setSelectedSeed] = useState('');

  const set = (key: string, value: string | number | boolean) => {
    setCreatedRun(null);
    setSelectedSeed('');
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const created = await fetchJSON<CreatedRun>('/runs', {
        method: 'POST',
        body: JSON.stringify(form),
      });
      if (form.mode === 'keyword') {
        const topResult = created.seed.results[0] || created.seed.primary_url;
        setCreatedRun(created);
        setSelectedSeed(topResult);
        return;
      }
      await startRun(created.id);
      startTransition(() => router.push(`/runs/${created.id}`));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start crawl');
    }
  };

  const launchSelected = async (seedUrl?: string) => {
    if (!createdRun) {
      return;
    }
    setError('');
    const chosen = seedUrl || selectedSeed || createdRun.seed.primary_url;
    try {
      await startRun(createdRun.id, chosen);
      startTransition(() => router.push(`/runs/${createdRun.id}`));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start crawl');
    }
  };

  const hostFor = (url: string) => {
    try {
      return new URL(url).hostname.replace(/^www\./, '');
    } catch {
      return url;
    }
  };

  const resultLabel = (index: number) => (index === 0 ? 'Top result' : `Result ${index + 1}`);

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

      {createdRun && (
        <div className="search-results" aria-live="polite">
          <div className="search-results__head">
            <div>
              <span className="search-results__eyebrow">Seed selection</span>
              <div className="search-results__title">Choose where this crawl starts</div>
            </div>
            <button
              className="btn-ghost btn-ghost--compact"
              type="button"
              disabled={pending}
              onClick={() => launchSelected(createdRun.seed.results[0] || createdRun.seed.primary_url)}
            >
              Crawl top result
            </button>
          </div>
          <div className="search-results__list">
            {createdRun.seed.results.map((result, index) => (
              <label
                className={`search-result${selectedSeed === result ? ' active' : ''}`}
                key={result}
              >
                <input
                  checked={selectedSeed === result}
                  name="seed-result"
                  type="radio"
                  value={result}
                  onChange={() => setSelectedSeed(result)}
                />
                <span className="search-result__rank">{String(index + 1).padStart(2, '0')}</span>
                <span className="search-result__body">
                  <span className="search-result__label">{resultLabel(index)}</span>
                  <span className="search-result__host">{hostFor(result)}</span>
                  <span className="search-result__url">{result}</span>
                </span>
                <button
                  className="btn-ghost btn-ghost--compact search-result__start"
                  disabled={pending}
                  type="button"
                  onClick={(event) => {
                    event.preventDefault();
                    setSelectedSeed(result);
                    launchSelected(result);
                  }}
                >
                  Start here
                </button>
              </label>
            ))}
          </div>
        </div>
      )}

      {!createdRun && (
        <div className="form-actions">
          <button className="btn-primary" disabled={pending} type="submit">
            {pending ? 'Working...' : form.mode === 'keyword' ? 'Search seeds' : 'Launch crawl'}
          </button>
          <span className="form-note">
            {form.mode === 'keyword'
              ? 'Keyword runs resolve and prefetch the top 10 results first.'
              : 'Results page opens immediately and streams new pages live.'}
          </span>
        </div>
      )}
    </form>
  );
}
