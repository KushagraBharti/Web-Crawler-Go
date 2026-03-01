'use client';

import { useEffect, useMemo, useRef, useState, useTransition, useId } from 'react';
import { useRunStore } from '@/lib/store';
import { API_BASE, APIError, fetchJSON } from '@/lib/api';
import { LineChart } from './LineChart';
import { GraphCanvas } from './GraphCanvas';
import { HostsTable } from './HostsTable';
import { PersonalityCards } from './PersonalityCards';
import { RunSummaryPanel } from './RunSummaryPanel';
import { PagesTable } from './PagesTable';
import { Frame, PageRow, RunSummary } from '@/lib/types';

export function DashboardClient({ runId }: { runId: string }) {
  const applyFrame = useRunStore((s) => s.applyFrame);
  const reset = useRunStore((s) => s.reset);
  const throughput = useRunStore((s) => s.throughput);
  const frontier = useRunStore((s) => s.frontier);
  const fetchQueue = useRunStore((s) => s.fetch);
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
  const [stopFeedback, setStopFeedback] = useState<string>('');
  const [isStopping, startStopTransition] = useTransition();
  const [, startFrameTransition] = useTransition();
  const [, startStorageTransition] = useTransition();
  const [, startSummaryTransition] = useTransition();

  // Collapsible section state
  const [graphOpen, setGraphOpen] = useState(false);
  const [personalityOpen, setPersonalityOpen] = useState(false);
  const graphId = useId();
  const personalityId = useId();

  // Reset store on runId change
  useEffect(() => {
    reset();
  }, [runId, reset]);

  // SSE connection
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
    source.onerror = () => {
      setStatus((prev) => (prev === 'stopped' ? prev : 'reconnecting'));
    };

    return () => {
      source.removeEventListener('frame', onFrame as EventListener);
      source.close();
      sourceRef.current = null;
    };
  }, [applyFrame, runId, startFrameTransition]);

  // Polling for run status and pages
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
          setStatus((prev) => (prev === 'stopped' ? prev : 'error'));
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
    }, 2500);

    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [runId, startStorageTransition, startSummaryTransition]);

  const stopRun = async () => {
    setStopFeedback('');
    startStopTransition(() => {
      void (async () => {
        try {
          await fetchJSON(`/runs/${runId}/stop`, { method: 'POST' });
          setStatus('stopped');
          setStopFeedback('Run stopped successfully.');
          sourceRef.current?.close();
        } catch (err) {
          if (err instanceof APIError) {
            setStopFeedback(err.message);
          } else if (err instanceof Error) {
            setStopFeedback(err.message);
          } else {
            setStopFeedback('Failed to stop run');
          }
        }
      })();
    });
  };

  useEffect(() => {
    const terminal = runStatus === 'stopped' || runStatus === 'failed' || runStatus === 'finished';
    if (terminal) {
      sourceRef.current?.close();
    }
  }, [runStatus]);

  const queueSeries = useMemo(() => [frontier, fetchQueue, parse], [frontier, fetchQueue, parse]);
  const currentThroughput = throughput[throughput.length - 1] || 0;
  const displayStatus = runStatus === 'stopped' || runStatus === 'failed' || runStatus === 'finished'
    ? 'stopped'
    : status;

  return (
    <main className="stagger">
      {/* Skip link */}
      <a href="#main-content" className="skip-link">Skip to main content</a>

      {/* Storage banner */}
      {storageMode === 'memory' && (
        <section className="storage-banner" role="status" aria-live="polite">
          <span className="badge badge--warning">Memory mode</span>
          <p>
            Data will not persist after the server stops. Start Postgres to enable
            durable run history.
          </p>
        </section>
      )}

      {/* Dashboard header */}
      <header className="dashboard-header">
        <div className="dashboard-header__info">
          <h1>Dashboard</h1>
          <div className="dashboard-header__status" role="status" aria-live="polite">
            {displayStatus === 'live' && <span className="live-indicator" aria-label="Live connection active">Live</span>}
            {displayStatus !== 'live' && (
              <span className="badge badge--warning" aria-label={`Connection status: ${displayStatus}`}>{displayStatus}</span>
            )}
            <span>Run: <code className="dashboard-header__run-id">{runId}</code></span>
            <span>Updated: {lastUpdated ? new Date(lastUpdated).toLocaleTimeString() : '\u2014'}</span>
          </div>
        </div>
        <button
          className="button button--secondary"
          onClick={stopRun}
          disabled={isStopping || displayStatus === 'stopped'}
          aria-label={isStopping ? 'Stopping crawl...' : 'Stop crawl'}
        >
          {isStopping ? 'Stopping...' : 'Stop'}
        </button>
      </header>
      {stopFeedback && (
        <p
          style={{
            marginTop: '-1.75rem',
            marginBottom: '-0.5rem',
            color: stopFeedback.includes('successfully') ? 'var(--success)' : 'var(--error)',
            fontSize: '0.9rem'
          }}
        >
          {stopFeedback}
        </p>
      )}

      <div id="main-content">
        {/* === PRIMARY ZONE === */}

        {/* 1. Run Summary + inline error health */}
        <section className="grid" style={{ marginBottom: '1.5rem' }}>
          <RunSummaryPanel status={runStatus} stopReason={stopReason} summary={summary} errors={errors} />
        </section>

        {/* 2. Throughput + Backpressure */}
        <section className="grid" style={{ marginBottom: '1.5rem' }}>
          <div className="panel span-4">
            <span className="badge badge--success">Throughput</span>
            <div style={{ marginTop: '1rem' }}>
              <span className="metric-large" aria-live="polite">{currentThroughput.toFixed(1)}</span>
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

        {/* 3. Host Health */}
        <HostsTable hosts={hosts} />

        {/* 4. Recent Pages */}
        <section className="grid" style={{ marginTop: '1.5rem' }}>
          <PagesTable pages={pages} runId={runId} />
        </section>

        {/* === SECONDARY ZONE (collapsible) === */}

        {/* 5. Domain Graph */}
        <section style={{ marginTop: '1.5rem' }}>
          <div className="accordion">
            <button
              type="button"
              className="accordion__trigger"
              aria-expanded={graphOpen}
              aria-controls={graphId}
              onClick={() => setGraphOpen((prev) => !prev)}
            >
              <span>Domain Graph ({nodes.length} hosts, {edges.length} connections)</span>
              <svg className="accordion__icon" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
              </svg>
            </button>
            <div id={graphId} className="accordion__content" data-open={graphOpen} role="region" aria-label="Domain graph visualization">
              <div className="accordion__inner">
                <div>
                  <GraphCanvas nodes={nodes} edges={edges} />
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* 6. Personality Analysis */}
        <section style={{ marginTop: '1.5rem' }}>
          <div className="accordion">
            <button
              type="button"
              className="accordion__trigger"
              aria-expanded={personalityOpen}
              aria-controls={personalityId}
              onClick={() => setPersonalityOpen((prev) => !prev)}
            >
              <span>Host Personality Analysis</span>
              <svg className="accordion__icon" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
              </svg>
            </button>
            <div id={personalityId} className="accordion__content" data-open={personalityOpen} role="region" aria-label="Host personality analysis">
              <div className="accordion__inner">
                <PersonalityCards hosts={hosts} />
              </div>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
