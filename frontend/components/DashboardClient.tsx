'use client';

import { useEffect, useMemo, useRef, useState, useTransition } from 'react';
import { useRunStore } from '@/lib/store';
import { API_BASE, fetchJSON } from '@/lib/api';
import { LineChart } from './LineChart';
import { GraphCanvas } from './GraphCanvas';
import { HostsTable } from './HostsTable';
import { ErrorsPanel } from './ErrorsPanel';
import { PersonalityCards } from './PersonalityCards';
import { RunSummaryPanel } from './RunSummaryPanel';
import { PagesTable } from './PagesTable';
import { Frame, PageRow, RunSummary } from '@/lib/types';

export function DashboardClient({ runId }: { runId: string }) {
  const applyFrame = useRunStore((s) => s.applyFrame);
  const throughput = useRunStore((s) => s.throughput);
  const frontier = useRunStore((s) => s.frontier);
  const fetch = useRunStore((s) => s.fetch);
  const parse = useRunStore((s) => s.parse);
  const errors = useRunStore((s) => s.errors);
  const hosts = useRunStore((s) => s.hosts);
  const nodes = useRunStore((s) => s.nodes);
  const edges = useRunStore((s) => s.edges);
  const lastUpdated = useRunStore((s) => s.lastUpdated);

  const sourceRef = useRef<EventSource | null>(null);
  const [status, setStatus] = useState<'connecting' | 'live' | 'reconnecting' | 'stopped' | 'error'>('connecting');
  const [storageMode, setStorageMode] = useState<'memory' | 'postgres' | null>(null);
  const [runStatus, setRunStatus] = useState<string>('');
  const [stopReason, setStopReason] = useState<string>('');
  const [summary, setSummary] = useState<RunSummary | null>(null);
  const [pages, setPages] = useState<PageRow[]>([]);
  const [isStopping, startStopTransition] = useTransition();
  const [, startFrameTransition] = useTransition();
  const [, startStorageTransition] = useTransition();
  const [, startSummaryTransition] = useTransition();

  useEffect(() => {
    const source = new EventSource(`${API_BASE}/runs/${runId}/events`);
    sourceRef.current = source;

    const onFrame = (event: MessageEvent<string>) => {
      try {
        const frame = JSON.parse(event.data) as Frame;
        startFrameTransition(() => {
          applyFrame(frame);
        });
        setStatus('live');
      } catch {
        setStatus('error');
      }
    };

    source.addEventListener('frame', onFrame as EventListener);
    source.onerror = () => setStatus('reconnecting');

    return () => {
      source.removeEventListener('frame', onFrame as EventListener);
      source.close();
    };
  }, [applyFrame, runId, startFrameTransition]);

  useEffect(() => {
    let cancelled = false;

    const loadRun = async () => {
      try {
        const data = await fetchJSON<{
          storage_mode?: string;
          status?: string;
          stop_reason?: string;
          summary?: RunSummary;
        }>(`/runs/${runId}`);
        if (cancelled) return;
        const mode = data.storage_mode === 'postgres' ? 'postgres' : 'memory';
        startStorageTransition(() => setStorageMode(mode));
        startSummaryTransition(() => {
          setRunStatus(data.status || '');
          setStopReason(data.stop_reason || '');
          setSummary(data.summary ?? null);
        });
      } catch {
        if (!cancelled) {
          startStorageTransition(() => setStorageMode('memory'));
        }
      }
    };

    const loadPages = async () => {
      try {
        const data = await fetchJSON<{ items: PageRow[] }>(`/runs/${runId}/pages?limit=50`);
        if (cancelled) return;
        startSummaryTransition(() => setPages(data.items || []));
      } catch {
        if (!cancelled) {
          startSummaryTransition(() => setPages([]));
        }
      }
    };

    void loadRun();
    void loadPages();
    const interval = setInterval(() => {
      void loadRun();
      void loadPages();
    }, 4000);

    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [runId, startStorageTransition, startSummaryTransition]);

  const stopRun = async () => {
    startStopTransition(() => {
      void fetchJSON(`/runs/${runId}/stop`, { method: 'POST' }).then(() => {
        setStatus('stopped');
      });
    });
  };

  const queueSeries = useMemo(() => [frontier, fetch, parse], [frontier, fetch, parse]);
  const currentThroughput = throughput[throughput.length - 1] || 0;

  return (
    <main className="stagger">
      {storageMode === 'memory' && (
        <section className="storage-banner" role="status" aria-live="polite">
          <span className="badge badge--warning">Memory mode</span>
          <p>
            Data will not persist after the server stops. Start Postgres to enable
            durable run history.
          </p>
        </section>
      )}
      <header className="dashboard-header">
        <div className="dashboard-header__info">
          <h1>Dashboard</h1>
          <div className="dashboard-header__status">
            {status === 'live' && <span className="live-indicator">Live</span>}
            {status !== 'live' && (
              <span className="badge badge--warning">{status}</span>
            )}
            <span>Run: <code className="dashboard-header__run-id">{runId}</code></span>
            <span>Updated: {lastUpdated ? new Date(lastUpdated).toLocaleTimeString() : '—'}</span>
          </div>
        </div>
        <button
          className="button button--secondary"
          onClick={stopRun}
          disabled={isStopping || status === 'stopped'}
        >
          {isStopping ? 'Stopping...' : 'Stop'}
        </button>
      </header>

      <section className="grid">
        <RunSummaryPanel status={runStatus} stopReason={stopReason} summary={summary} />
      </section>

      <section className="grid">
        <div className="panel span-4">
          <span className="badge badge--success">Throughput</span>
          <div style={{ marginTop: '1rem' }}>
            <span className="metric-large">{currentThroughput.toFixed(1)}</span>
            <span style={{ marginLeft: '0.5rem', color: 'var(--text-tertiary)' }}>pages/sec</span>
          </div>
          <LineChart data={throughput} label="Throughput" color="var(--success)" />
        </div>

        <div className="panel span-8">
          <span className="badge">Queue Depths</span>
          <h3 style={{ marginTop: '1rem' }}>Backpressure</h3>
          <div className="queue-grid">
            <LineChart data={queueSeries[0]} label="Frontier" color="var(--accent)" />
            <LineChart data={queueSeries[1]} label="Fetch" color="var(--info)" />
            <LineChart data={queueSeries[2]} label="Parse" color="var(--success)" />
          </div>
        </div>
      </section>

      <section className="grid">
        <div className="panel span-7">
          <span className="badge">Network</span>
          <h3 style={{ marginTop: '1rem' }}>Domain Graph</h3>
          <p style={{ fontSize: '0.875rem', marginTop: '0.25rem' }}>
            {nodes.length} hosts, {edges.length} connections
          </p>
          <GraphCanvas nodes={nodes} edges={edges} />
        </div>

        <div className="span-5">
          <ErrorsPanel errors={errors} />
        </div>
      </section>

      <PersonalityCards hosts={hosts} />

      <HostsTable hosts={hosts} />

      <section className="grid">
        <PagesTable pages={pages} runId={runId} />
      </section>
    </main>
  );
}
