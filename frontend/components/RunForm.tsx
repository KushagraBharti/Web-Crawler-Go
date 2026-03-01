'use client';

import type { FormEvent } from 'react';
import { useState, useTransition, useCallback, useId } from 'react';
import { useRouter } from 'next/navigation';
import { APIError, fetchJSON } from '@/lib/api';
import { Switch } from './Switch';

const PRESETS = {
  safe: {
    max_depth: 2,
    max_pages: 300,
    time_budget_seconds: 120,
    max_links_per_page: 50,
    global_concurrency: 8,
    per_host_concurrency: 2,
    respect_robots: true,
  },
  balanced: {
    max_depth: 3,
    max_pages: 1000,
    time_budget_seconds: 300,
    max_links_per_page: 100,
    global_concurrency: 16,
    per_host_concurrency: 3,
    respect_robots: true,
  },
} as const;

type PresetKey = keyof typeof PRESETS;

const defaultForm = {
  max_depth: 3,
  max_pages: 1000,
  time_budget_seconds: 300,
  max_links_per_page: 100,
  global_concurrency: 16,
  per_host_concurrency: 3,
  respect_robots: true,
};

const HARD_LIMITS = {
  max_pages: 2000,
  time_budget_seconds: 300,
  max_links_per_page: 100,
  global_concurrency: 32,
  per_host_concurrency: 4,
};

export function RunForm() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [error, setError] = useState('');
  const [preset, setPreset] = useState<PresetKey | 'custom'>('balanced');
  const [form, setForm] = useState({ seed_url: '', ...defaultForm });
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const advancedId = useId();

  const update = useCallback((key: string, value: string | number | boolean) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    if (key !== 'seed_url') {
      setPreset('custom');
    }
  }, []);

  const applyPreset = useCallback((nextPreset: PresetKey | 'custom') => {
    setPreset(nextPreset);
    if (nextPreset === 'custom') {
      return;
    }
    setForm((prev) => ({
      ...prev,
      ...PRESETS[nextPreset],
    }));
  }, []);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const run = await fetchJSON<{ id: string }>('/runs', {
        method: 'POST',
        body: JSON.stringify(form),
      });
      await fetchJSON(`/runs/${run.id}/start`, { method: 'POST' });
      startTransition(() => {
        router.push(`/runs/${run.id}`);
      });
    } catch (err) {
      if (err instanceof APIError && err.details && typeof err.details === 'object') {
        const fieldErrors = Object.entries(err.details as Record<string, unknown>)
          .map(([field, message]) => `${field}: ${String(message)}`)
          .join(' | ');
        setError(fieldErrors || err.message);
        return;
      }
      setError(err instanceof Error ? err.message : 'Failed to start run');
    }
  };

  return (
    <form className="panel" onSubmit={onSubmit}>
      <span className="badge">New Crawl</span>
      <h2 style={{ marginTop: '1rem', marginBottom: '0.5rem' }}>Start a crawl</h2>
      <p style={{ marginBottom: '1.5rem' }}>
        Enter a URL and hit Start. Tune advanced settings if you need to.
      </p>

      {/* Basic fields */}
      <div className="form-grid">
        <div className="form-group form-group--full">
          <label className="form-label" htmlFor="seed-url">Target URL</label>
          <input
            id="seed-url"
            className="form-input"
            type="url"
            required
            placeholder="https://example.com"
            value={form.seed_url}
            onChange={(e) => update('seed_url', e.target.value)}
            autoFocus
          />
        </div>
        <div className="form-group form-group--full">
          <label className="form-label" htmlFor="crawl-preset">Preset</label>
          <select
            id="crawl-preset"
            className="form-input"
            value={preset}
            onChange={(e) => applyPreset(e.target.value as PresetKey | 'custom')}
          >
            <option value="safe">Safe</option>
            <option value="balanced">Balanced</option>
            <option value="custom">Custom</option>
          </select>
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="max-depth">Max Depth</label>
          <input
            id="max-depth"
            className="form-input"
            type="number"
            min={1}
            max={10}
            value={form.max_depth}
            onChange={(e) => update('max_depth', Number(e.target.value))}
          />
        </div>
        <div className="form-group">
          <label className="form-label" htmlFor="max-pages">Max Pages</label>
          <input
            id="max-pages"
            className="form-input"
            type="number"
            min={1}
            max={HARD_LIMITS.max_pages}
            value={form.max_pages}
            onChange={(e) => update('max_pages', Number(e.target.value))}
          />
        </div>
      </div>

      {/* Advanced accordion */}
      <div className="accordion" style={{ marginTop: '1.25rem' }}>
        <button
          type="button"
          className="accordion__trigger"
          aria-expanded={advancedOpen}
          aria-controls={advancedId}
          onClick={() => setAdvancedOpen((prev) => !prev)}
        >
          <span>Advanced Settings</span>
          <svg className="accordion__icon" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
          </svg>
        </button>
        <div
          id={advancedId}
          className="accordion__content"
          data-open={advancedOpen}
          role="region"
          aria-label="Advanced crawl settings"
        >
          <div className="accordion__inner">
            <div className="form-grid">
              <div className="form-group">
                <label className="form-label" htmlFor="time-budget">Time Budget (sec)</label>
                <input
                  id="time-budget"
                  className="form-input"
                  type="number"
                  min={1}
                  max={HARD_LIMITS.time_budget_seconds}
                  value={form.time_budget_seconds}
                  onChange={(e) => update('time_budget_seconds', Number(e.target.value))}
                />
              </div>
              <div className="form-group">
                <label className="form-label" htmlFor="links-per-page">Links per Page</label>
                <input
                  id="links-per-page"
                  className="form-input"
                  type="number"
                  min={1}
                  max={HARD_LIMITS.max_links_per_page}
                  value={form.max_links_per_page}
                  onChange={(e) => update('max_links_per_page', Number(e.target.value))}
                />
              </div>
              <div className="form-group">
                <label className="form-label" htmlFor="global-concurrency">Global Concurrency</label>
                <input
                  id="global-concurrency"
                  className="form-input"
                  type="number"
                  min={1}
                  max={HARD_LIMITS.global_concurrency}
                  value={form.global_concurrency}
                  onChange={(e) => update('global_concurrency', Number(e.target.value))}
                />
              </div>
              <div className="form-group">
                <label className="form-label" htmlFor="per-host-concurrency">Per-Host Concurrency</label>
                <input
                  id="per-host-concurrency"
                  className="form-input"
                  type="number"
                  min={1}
                  max={HARD_LIMITS.per_host_concurrency}
                  value={form.per_host_concurrency}
                  onChange={(e) => update('per_host_concurrency', Number(e.target.value))}
                />
              </div>
              <div className="form-group form-group--full">
                <Switch
                  label="Respect robots.txt"
                  checked={form.respect_robots}
                  onChange={(checked) => update('respect_robots', checked)}
                  description="When enabled, the crawler obeys robots.txt directives"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <p role="alert" style={{ color: 'var(--error)', marginTop: '1rem', fontSize: '0.875rem' }}>
          {error}
        </p>
      )}

      {/* Submit */}
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
