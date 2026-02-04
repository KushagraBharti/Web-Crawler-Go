'use client';

import type { FormEvent } from 'react';
import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { fetchJSON } from '@/lib/api';

const defaults = {
  max_depth: 3,
  max_pages: 5000,
  time_budget_seconds: 600,
  max_links_per_page: 200,
  global_concurrency: 64,
  per_host_concurrency: 4,
  respect_robots: true
};

export function RunForm() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [error, setError] = useState('');
  const [form, setForm] = useState({
    seed_url: '',
    ...defaults
  });

  const update = (key: string, value: string | number | boolean) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const run = await fetchJSON<{ id: string }>('/runs', {
        method: 'POST',
        body: JSON.stringify(form)
      });
      await fetchJSON(`/runs/${run.id}/start`, { method: 'POST' });
      startTransition(() => {
        router.push(`/runs/${run.id}`);
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start run');
    }
  };

  return (
    <form className="panel" onSubmit={onSubmit}>
      <div className="badge">Create Run</div>
      <h2 style={{ marginTop: 16 }}>Launch a crawl run</h2>
      <p style={{ marginTop: 8 }}>
        Set safety limits, concurrency, and a seed URL. The crawler will self-throttle to keep things stable.
      </p>
      <div className="form-grid" style={{ marginTop: 24 }}>
        <label>
          Seed URL
          <input
            type="url"
            required
            placeholder="https://example.com"
            value={form.seed_url}
            onChange={(e) => update('seed_url', e.target.value)}
          />
        </label>
        <label>
          Max Depth
          <input
            type="number"
            value={form.max_depth}
            onChange={(e) => update('max_depth', Number(e.target.value))}
          />
        </label>
        <label>
          Max Pages
          <input
            type="number"
            value={form.max_pages}
            onChange={(e) => update('max_pages', Number(e.target.value))}
          />
        </label>
        <label>
          Time Budget (sec)
          <input
            type="number"
            value={form.time_budget_seconds}
            onChange={(e) => update('time_budget_seconds', Number(e.target.value))}
          />
        </label>
        <label>
          Max Links / Page
          <input
            type="number"
            value={form.max_links_per_page}
            onChange={(e) => update('max_links_per_page', Number(e.target.value))}
          />
        </label>
        <label>
          Global Concurrency
          <input
            type="number"
            value={form.global_concurrency}
            onChange={(e) => update('global_concurrency', Number(e.target.value))}
          />
        </label>
        <label>
          Per-host Concurrency
          <input
            type="number"
            value={form.per_host_concurrency}
            onChange={(e) => update('per_host_concurrency', Number(e.target.value))}
          />
        </label>
        <label className="toggle">
          Respect Robots
          <input
            type="checkbox"
            checked={form.respect_robots}
            onChange={(e) => update('respect_robots', e.target.checked)}
          />
        </label>
      </div>
      {error ? <p style={{ color: '#b3311a', marginTop: 12 }}>{error}</p> : null}
      <div style={{ marginTop: 24, display: 'flex', gap: 12, alignItems: 'center' }}>
        <button className="button" disabled={pending} type="submit">
          {pending ? 'Starting…' : 'Start Crawl'}
        </button>
        <span style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: '0.16em', color: 'var(--muted)' }}>
          Live dashboard opens automatically
        </span>
      </div>
    </form>
  );
}
