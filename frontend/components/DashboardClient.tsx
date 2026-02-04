'use client';

import { useEffect, useMemo, useRef, useState, useTransition } from 'react';
import { useRunStore } from '@/lib/store';
import { API_BASE, fetchJSON } from '@/lib/api';
import { LineChart } from './LineChart';
import { GraphCanvas } from './GraphCanvas';
import { HostsTable } from './HostsTable';
import { ErrorsPanel } from './ErrorsPanel';
import { PersonalityCards } from './PersonalityCards';
import { Frame } from '@/lib/types';

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
  const [status, setStatus] = useState('connecting');
  const [isStopping, startStopTransition] = useTransition();
  const [, startFrameTransition] = useTransition();

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

  const stopRun = async () => {
    startStopTransition(() => {
      void fetchJSON(`/runs/${runId}/stop`, { method: 'POST' }).then(() => {
        setStatus('stopped');
      });
    });
  };

  const queueSeries = useMemo(() => [frontier, fetch, parse], [frontier, fetch, parse]);

  return (
    <div className="stagger">
      <div className="hero">
        <div>
          <div className="badge">Live Run</div>
          <h1>Crawler Control Panel</h1>
          <p>
            Watching <strong>{runId}</strong>. The system is {status}. Last update{' '}
            {lastUpdated ? new Date(lastUpdated).toLocaleTimeString() : '—'}.
          </p>
        </div>
        <button className="button secondary" onClick={stopRun} disabled={isStopping}>
          {isStopping ? 'Stopping…' : 'Stop Run'}
        </button>
      </div>

      <section className="grid">
        <div className="panel span-4">
          <div className="badge">Throughput</div>
          <h2 style={{ marginTop: 12 }}>Pages per second</h2>
          <LineChart data={throughput} label="Throughput" color="var(--accent)" />
        </div>
        <div className="panel span-8">
          <div className="badge">Queues</div>
          <h2 style={{ marginTop: 12 }}>Backpressure breathing</h2>
          <div className="queue-grid">
            <LineChart data={queueSeries[0]} label="Frontier" color="var(--accent-2)" />
            <LineChart data={queueSeries[1]} label="Fetch" color="var(--accent)" />
            <LineChart data={queueSeries[2]} label="Parse" color="var(--accent-3)" />
          </div>
        </div>
      </section>

      <section className="grid">
        <div className="panel span-7">
          <div className="badge">Domain Graph</div>
          <h2 style={{ marginTop: 12 }}>Cross-host map</h2>
          <GraphCanvas nodes={nodes} edges={edges} />
        </div>
        <div className="span-5">
          <ErrorsPanel errors={errors} />
        </div>
      </section>

      <PersonalityCards hosts={hosts} />

      <HostsTable hosts={hosts} />
    </div>
  );
}
