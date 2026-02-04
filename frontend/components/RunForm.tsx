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
      <span className="badge">New Crawl</span>
      <h2 style={{ marginTop: '1rem', marginBottom: '0.5rem' }}>Start a crawl</h2>
      <p style={{ marginBottom: '1.5rem' }}>
        Configure limits and concurrency. The crawler will self-throttle.
      </p>

      <div className="form-grid">
        <div className="form-group form-group--full">
          <label className="form-label">Target URL</label>
          <input
            className="form-input"
            type="url"
            required
            placeholder="https://example.com"
            value={form.seed_url}
            onChange={(e) => update('seed_url', e.target.value)}
          />
        </div>

        <div className="form-group">
          <label className="form-label">Max Depth</label>
          <input
            className="form-input"
            type="number"
            min={1}
            max={10}
            value={form.max_depth}
            onChange={(e) => update('max_depth', Number(e.target.value))}
          />
        </div>
        <div className="form-group">
          <label className="form-label">Max Pages</label>
          <input
            className="form-input"
            type="number"
            min={1}
            value={form.max_pages}
            onChange={(e) => update('max_pages', Number(e.target.value))}
          />
        </div>

        <div className="form-group">
          <label className="form-label">Time Budget (sec)</label>
          <input
            className="form-input"
            type="number"
            min={10}
            value={form.time_budget_seconds}
            onChange={(e) => update('time_budget_seconds', Number(e.target.value))}
          />
        </div>
        <div className="form-group">
          <label className="form-label">Links per Page</label>
          <input
            className="form-input"
            type="number"
            min={1}
            value={form.max_links_per_page}
            onChange={(e) => update('max_links_per_page', Number(e.target.value))}
          />
        </div>

        <div className="form-group">
          <label className="form-label">Global Concurrency</label>
          <input
            className="form-input"
            type="number"
            min={1}
            max={256}
            value={form.global_concurrency}
            onChange={(e) => update('global_concurrency', Number(e.target.value))}
          />
        </div>
        <div className="form-group">
          <label className="form-label">Per-Host Concurrency</label>
          <input
            className="form-input"
            type="number"
            min={1}
            max={16}
            value={form.per_host_concurrency}
            onChange={(e) => update('per_host_concurrency', Number(e.target.value))}
          />
        </div>

        <div className="form-group form-group--full">
          <div
            className={`toggle-wrapper ${form.respect_robots ? 'active' : ''}`}
            onClick={() => update('respect_robots', !form.respect_robots)}
          >
            <div>
              <span className="form-label">Respect robots.txt</span>
            </div>
            <div className="toggle-track" />
          </div>
        </div>
      </div>

      {error && (
        <p style={{ color: 'var(--error)', marginTop: '1rem', fontSize: '0.875rem' }}>
          {error}
        </p>
      )}

      <div style={{ marginTop: '1.5rem', display: 'flex', alignItems: 'center', gap: '1rem' }}>
        <button className="button" disabled={pending} type="submit">
          {pending ? 'Starting...' : 'Start Crawl'}
        </button>
        <span style={{ fontSize: '0.8125rem', color: 'var(--text-tertiary)' }}>
          Dashboard opens automatically
        </span>
      </div>
    </form>
  );
}
