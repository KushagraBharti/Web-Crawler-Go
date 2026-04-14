'use client';

import type { FormEvent } from 'react';
import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { fetchJSON } from '@/lib/api';
import type { SourceMode } from '@/lib/types';

const defaults: {
  mode: SourceMode;
  input: string;
  max_depth: number;
  max_pages: number;
  time_budget_seconds: number;
  max_links_per_page: number;
  global_concurrency: number;
  per_host_concurrency: number;
  respect_robots: boolean;
} = {
  mode: 'url',
  input: '',
  max_depth: 2,
  max_pages: 50,
  time_budget_seconds: 180,
  max_links_per_page: 25,
  global_concurrency: 16,
  per_host_concurrency: 4,
  respect_robots: true
} as const;

export function RunForm() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [error, setError] = useState('');
  const [form, setForm] = useState({ ...defaults });

  const update = (key: string, value: string | number | boolean) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    setError('');
    try {
      const created = await fetchJSON<{ id: string }>('/runs', {
        method: 'POST',
        body: JSON.stringify(form)
      });
      await fetchJSON(`/runs/${created.id}/start`, { method: 'POST' });
      startTransition(() => router.push(`/runs/${created.id}`));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start crawl');
    }
  };

  return (
    <form className="launch-card" onSubmit={onSubmit}>
      <div className="eyebrow">Result-first crawler</div>
      <h2>Start from a URL or a search phrase.</h2>
      <p>
        The crawler resolves a seed, captures readable page content, and writes JSON artifacts you
        can inspect from the terminal.
      </p>

      <div className="mode-switch">
        <button
          className={form.mode === 'url' ? 'mode-switch__item active' : 'mode-switch__item'}
          onClick={() => update('mode', 'url')}
          type="button"
        >
          URL
        </button>
        <button
          className={form.mode === 'keyword' ? 'mode-switch__item active' : 'mode-switch__item'}
          onClick={() => update('mode', 'keyword')}
          type="button"
        >
          Keyword
        </button>
      </div>

      <label className="field">
        <span>{form.mode === 'url' ? 'Seed URL' : 'Search query'}</span>
        <input
          className="input"
          placeholder={form.mode === 'url' ? 'https://en.wikipedia.org/wiki/Web_crawler' : 'Alan Turing'}
          required
          value={form.input}
          onChange={(e) => update('input', e.target.value)}
        />
      </label>

      <div className="grid-two">
        <label className="field">
          <span>Max depth</span>
          <input className="input" min={1} type="number" value={form.max_depth} onChange={(e) => update('max_depth', Number(e.target.value))} />
        </label>
        <label className="field">
          <span>Max pages</span>
          <input className="input" min={1} type="number" value={form.max_pages} onChange={(e) => update('max_pages', Number(e.target.value))} />
        </label>
        <label className="field">
          <span>Time budget (sec)</span>
          <input className="input" min={10} type="number" value={form.time_budget_seconds} onChange={(e) => update('time_budget_seconds', Number(e.target.value))} />
        </label>
        <label className="field">
          <span>Links per page</span>
          <input className="input" min={1} type="number" value={form.max_links_per_page} onChange={(e) => update('max_links_per_page', Number(e.target.value))} />
        </label>
      </div>

      <details className="advanced">
        <summary>Advanced crawl controls</summary>
        <div className="grid-two">
          <label className="field">
            <span>Global concurrency</span>
            <input className="input" min={1} type="number" value={form.global_concurrency} onChange={(e) => update('global_concurrency', Number(e.target.value))} />
          </label>
          <label className="field">
            <span>Per-host concurrency</span>
            <input className="input" min={1} type="number" value={form.per_host_concurrency} onChange={(e) => update('per_host_concurrency', Number(e.target.value))} />
          </label>
        </div>
        <label className="checkbox">
          <input
            checked={form.respect_robots}
            type="checkbox"
            onChange={(e) => update('respect_robots', e.target.checked)}
          />
          <span>Respect robots.txt</span>
        </label>
      </details>

      {error ? <div className="error-banner">{error}</div> : null}

      <div className="launch-actions">
        <button className="primary-button" disabled={pending} type="submit">
          {pending ? 'Launching crawl...' : 'Launch crawl'}
        </button>
        <div className="microcopy">Results page opens immediately and streams new pages as they land.</div>
      </div>
    </form>
  );
}
